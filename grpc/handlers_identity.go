package grpc

import (
	"context"

	"github.com/FleekHQ/space-poc/grpc/pb"

	"github.com/pkg/errors"
)

var errNotImplemented = errors.New("Not implemented")

func (srv *grpcServer) GetIdentityByUsername(ctx context.Context, request *pb.GetIdentityByUsernameRequest) (*pb.GetIdentityByUsernameResponse, error) {
	if request.Username == "" {
		return nil, errors.New("Username is required")
	}

	result, err := srv.sv.GetIdentityByUsername(ctx, request.Username)
	if err != nil {
		return nil, err
	}

	return &pb.GetIdentityByUsernameResponse{
		Identity: &pb.Identity{
			Address:   result.Address,
			PublicKey: result.PublicKey,
			Username:  result.Username,
		},
	}, nil
}

func (srv *grpcServer) CreateUsernameAndEmail(ctx context.Context, request *pb.CreateUsernameAndEmailRequest) (*pb.CreateUsernameAndEmailResponse, error) {
	if request.Username == "" {
		return nil, errors.New("Username is required")
	}

	_, err := srv.sv.CreateIdentity(ctx, request.Username)
	if err != nil {
		return nil, err
	}

	return &pb.CreateUsernameAndEmailResponse{}, nil
}
