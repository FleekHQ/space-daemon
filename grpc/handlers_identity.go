package grpc

import (
	"context"

	"github.com/FleekHQ/space-poc/grpc/pb"

	"github.com/pkg/errors"
)

var errNotImplemented = errors.New("Not implemented")

func (srv *grpcServer) GetIdentityByUsername(ctx context.Context, request *pb.GetIdentityByUsernameRequest) (*pb.GetIdentityByUsernameResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) CreateUsernameAndEmail(ctx context.Context, request *pb.CreateUsernameAndEmailRequest) (*pb.CreateUsernameAndEmailResponse, error) {
	return nil, errNotImplemented
}
