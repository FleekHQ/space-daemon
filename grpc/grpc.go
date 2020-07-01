package grpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	fuse "github.com/FleekHQ/space-poc/core/space/fuse"
	"github.com/improbable-eng/grpc-web/go/grpcweb"

	"github.com/FleekHQ/space-poc/core/space"

	"github.com/FleekHQ/space-poc/grpc/pb"
	"github.com/FleekHQ/space-poc/log"
	"google.golang.org/grpc"
)

const (
	DefaultGrpcPort = 9999
)

var defaultServerOptions = serverOptions{
	port: DefaultGrpcPort,
}

type serverOptions struct {
	port      int
	proxyPort int // port for grpcweb proxy
}

type grpcServer struct {
	opts     *serverOptions
	s        *grpc.Server
	rpcProxy *http.Server
	sv       space.Service
	fc       *fuse.Controller
	// TODO: see if we need to clean this up by gc or handle an array
	fileEventStream pb.SpaceApi_SubscribeServer
	txlEventStream  pb.SpaceApi_TxlSubscribeServer
	isStarted       bool
	readyCh         chan bool
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

// Start grpc server with provided options
func (srv *grpcServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", srv.opts.port))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen on port : %v", srv.opts.port), err)
		return err
	}

	srv.s = grpc.NewServer()
	pb.RegisterSpaceApiServer(srv.s, srv)

	srv.isStarted = true
	srv.readyCh <- true

	log.Info(fmt.Sprintf("Starting gRPC-web proxy on Port %d", srv.opts.proxyPort))
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

	log.Info(fmt.Sprintf("grpc server started in Port %v", srv.opts.port))

	// this is a blocking function
	return srv.s.Serve(lis)
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

func (srv *grpcServer) Shutdown() error {
	if !srv.isStarted {
		return nil

	}
	close(srv.readyCh)

	defer func() {
		srv.rpcProxy = nil
		srv.s = nil
		srv.isStarted = false
	}()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := srv.rpcProxy.Shutdown(ctx); err != nil {
		return err
	}
	srv.s.GracefulStop()
	return nil
}

func (srv *grpcServer) WaitForReady() chan bool {
	return srv.readyCh
}
