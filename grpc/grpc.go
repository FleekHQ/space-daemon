package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rakyll/statik/fs"
	"github.com/rs/cors"

	"github.com/improbable-eng/grpc-web/go/grpcweb"

	fuse "github.com/FleekHQ/space-daemon/core/space/fuse"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/FleekHQ/space-daemon/core/space"

	_ "github.com/FleekHQ/space-daemon/swagger/bin_ui" // required by statik/fs

	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/log"
	"google.golang.org/grpc"
)

const (
	DefaultGrpcPort = 9999
)

var defaultServerOptions = serverOptions{
	port: DefaultGrpcPort,
}

type serverOptions struct {
	port          int
	proxyPort     int // port for grpcweb proxy
	restProxyPort int // port for rest api proxy
}

type grpcServer struct {
	opts       *serverOptions
	s          *grpc.Server
	rpcProxy   *http.Server
	restServer *http.Server
	sv         space.Service
	fc         *fuse.Controller
	// TODO: see if we need to clean this up by gc or handle an array
	fileEventStream       pb.SpaceApi_SubscribeServer
	txlEventStream        pb.SpaceApi_TxlSubscribeServer
	invitationEventStream pb.SpaceApi_InvitationSubscribeServer
	isStarted             bool
	readyCh               chan bool
}

// Idea taken from here https://medium.com/soon-london/variadic-configuration-functions-in-go-8cef1c97ce99

type ServerOption func(o *serverOptions)

// gRPC server uses Service from core to handle requests
func New(sv space.Service, fc *fuse.Controller, opts ...ServerOption) *grpcServer {
	o := defaultServerOptions
	for _, opt := range opts {
		opt(&o)
	}
	srv := &grpcServer{
		opts:    &o,
		sv:      sv,
		fc:      fc,
		readyCh: make(chan bool, 1),
	}

	return srv
}

// Start grpc and api server with provided options
func (srv *grpcServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", srv.opts.port))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen on port : %v", srv.opts.port), err)
		return err
	}

	log.Info(fmt.Sprintf("listening on address %s", lis.Addr().String()))

	srv.s = grpc.NewServer()
	pb.RegisterSpaceApiServer(srv.s, srv)

	if err = srv.startRestProxy(ctx, lis); err != nil {
		return err
	}
	srv.startGrpcWebProxy()

	log.Info(fmt.Sprintf("gRPC server started on Port %v", srv.opts.port))
	srv.isStarted = true
	srv.readyCh <- true
	// this is a blocking function
	return srv.s.Serve(lis)
}

func (srv *grpcServer) startRestProxy(ctx context.Context, lis net.Listener) error {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := pb.RegisterSpaceApiHandlerFromEndpoint(ctx, mux, lis.Addr().String(), opts); err != nil {
		log.Error("Failed to start REST server", err)
		return err
	}

	swaggerPrefix := "/swaggerui/"
	swaggerHandler, err := srv.getSwaggerHandler(swaggerPrefix)
	if err != nil {
		// QQ: Should we fail launch if error in mounting swagger docs?
		// For now, just log a warning
		log.Warn("Swagger UI failed to load", "err:"+err.Error())
	}

	srv.restServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", srv.opts.restProxyPort),
		Handler: mux,
	}

	srv.restServer.Handler = cors.AllowAll().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Incoming REST Proxy Request", "path:"+r.URL.Path, "method:"+r.Method)
		if swaggerHandler != nil && strings.HasPrefix(r.URL.Path, swaggerPrefix) && r.Method == "GET" {
			log.Debug("Serving swagger ui")
			swaggerHandler.ServeHTTP(w, r)
		} else {
			mux.ServeHTTP(w, r)
		}
	}))

	log.Info("REST server is starting", fmt.Sprintf("port:%v", srv.opts.restProxyPort))
	go func() {
		if err := srv.restServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("REST server failed to start on port %d", srv.opts.restProxyPort), err)
		}
	}()

	return nil
}

func (srv *grpcServer) getSwaggerHandler(prefix string) (http.Handler, error) {
	var swaggerHandler http.Handler

	statikFS, err := fs.New()
	if err != nil {
		return nil, err
	}

	swaggerHandler = http.StripPrefix(prefix, http.FileServer(statikFS))
	return swaggerHandler, nil
}

func (srv *grpcServer) startGrpcWebProxy() {
	webrpcProxy := grpcweb.WrapServer(
		srv.s,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithWebsockets(true),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool {
			return true
		}),
	)

	srv.rpcProxy = &http.Server{
		Addr: fmt.Sprintf(":%d", srv.opts.proxyPort),
	}
	srv.rpcProxy.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if webrpcProxy.IsGrpcWebRequest(r) ||
			webrpcProxy.IsAcceptableGrpcCorsRequest(r) ||
			webrpcProxy.IsGrpcWebSocketRequest(r) {
			webrpcProxy.ServeHTTP(w, r)
		}
	})

	go func() {
		if err := srv.rpcProxy.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Space grpcweb proxy error", err)
		}
	}()
	log.Info(fmt.Sprintf("gRPC-web proxy server started on Port %d", srv.opts.proxyPort))
}

// Helper function for setting port
func WithPort(port int) ServerOption {
	return func(o *serverOptions) {
		if port != 0 {
			o.port = port
		}
	}
}

func WithProxyPort(port int) ServerOption {
	return func(o *serverOptions) {
		if port != 0 {
			o.proxyPort = port
		}
	}
}

// WithRestProxyPort configures the REST Proxy port
func WithRestProxyPort(port int) ServerOption {
	return func(o *serverOptions) {
		if port != 0 {
			o.restProxyPort = port
		}
	}
}

func (srv *grpcServer) Shutdown() error {
	if !srv.isStarted {
		return nil

	}
	close(srv.readyCh)

	defer func() {
		srv.rpcProxy = nil
		srv.restServer = nil
		srv.s = nil
		srv.isStarted = false
	}()

	srv.s.GracefulStop()

	shutdownCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	if err := srv.rpcProxy.Shutdown(shutdownCtx); err != nil {
		return err
	}

	if err := srv.restServer.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return nil
}

func (srv *grpcServer) WaitForReady() chan bool {
	return srv.readyCh
}
