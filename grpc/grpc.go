package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/FleekHQ/space-poc/core/store"
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
	db   *store.Store
}

func (sv *grpcServer) GetPathInfo(ctx context.Context, request *pb.PathInfoRequest) (*pb.PathInfoResponse, error) {
	return &pb.PathInfoResponse{
		Path:     "test.txt",
		IpfsHash: "testhash",
		IsDir:    false,
	}, nil
}

// Idea taken from here https://medium.com/soon-london/variadic-configuration-functions-in-go-8cef1c97ce99

type ServerOption func(o *serverOptions)

func New(db *store.Store, opts ...ServerOption) *grpcServer {
	o := defaultServerOptions
	for _, opt := range opts {
		opt(&o)
	}
	srv := &grpcServer{
		opts: &o,
		db:   db,
	}

	return srv
}

// Start grpc server with provided options
func (sv *grpcServer) Start(ctx context.Context) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", sv.opts.port))
	if err != nil {
		log.Error(fmt.Sprintf("failed to listen on port : %v", sv.opts.port), err)
		panic(err)
	}

	sv.s = grpc.NewServer()
	pb.RegisterSpaceApiServer(sv.s, sv)

	go func() {
		log.Info(fmt.Sprintf("grpc server starting in Port %v", sv.opts.port))
		sv.s.Serve(lis)
	}()
}

// Helper function for setting port
func WithPort(port int) ServerOption {
	return func(o *serverOptions) {
		if port != 0 {
			o.port = port
		}
	}
}
