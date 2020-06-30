package grpc

import (
	"context"
	"fmt"
	"net"

	fuse "github.com/FleekHQ/space-poc/core/space/fuse"

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
	port int
}

type grpcServer struct {
	opts *serverOptions
	s    *grpc.Server
	sv   space.Service
	fc   *fuse.Controller
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
		panic(err)
	}

	srv.s = grpc.NewServer()
	pb.RegisterSpaceApiServer(srv.s, srv)

	srv.isStarted = true
	srv.readyCh <- true
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

func (srv *grpcServer) Stop() {
	if srv.isStarted {
		srv.s.GracefulStop()
	}
}

func (srv *grpcServer) Shutdown() error {
	close(srv.readyCh)
	srv.Stop()
	return nil
}

func (srv *grpcServer) WaitForReady() chan bool {
	return srv.readyCh
}
