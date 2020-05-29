package grpc

import (
	"context"
	"fmt"
	"github.com/FleekHQ/space-poc/core/space"
	"net"

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
	// TODO: see if we need to clean this up by gc or handle an array
	fileEventStream pb.SpaceApi_SubscribeServer
}

// Idea taken from here https://medium.com/soon-london/variadic-configuration-functions-in-go-8cef1c97ce99

type ServerOption func(o *serverOptions)
// gRPC server uses Service from core to handle requests
func New(sv space.Service, opts ...ServerOption) *grpcServer {
	o := defaultServerOptions
	for _, opt := range opts {
		opt(&o)
	}
	srv := &grpcServer{
		opts: &o,
		sv:   sv,
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

	log.Info(fmt.Sprintf("grpc server started in Port %v", srv.opts.port))
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
	srv.s.GracefulStop()
}
