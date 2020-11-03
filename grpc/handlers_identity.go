package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) ImportIdentity(ctx context.Context, request *pb.ImportIdentityRequest) (*pb.ImportIdentityResponse, error) {
	// TODO

	return &pb.ImportIdentityResponse{}, nil
}

func (srv *grpcServer) EncryptKeyPairWithIdentity(ctx context.Context, request *pb.EncryptKeyPairWithIdentityRequest) (*pb.EncryptKeyPairWithIdentityResponse, error) {
	// TODO

	return &pb.EncryptKeyPairWithIdentityResponse{}, nil
}
