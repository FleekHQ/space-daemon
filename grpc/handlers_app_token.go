package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) InitializeMasterAppToken(ctx context.Context, request *pb.InitializeMasterAppTokenRequest) (*pb.InitializeMasterAppTokenResponse, error) {
	appToken, err := srv.sv.InitializeMasterAppToken(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.InitializeMasterAppTokenResponse{
		AppToken: appToken.GetAccessToken(),
	}, nil
}

func (srv *grpcServer) GenerateAppToken(ctx context.Context, request *pb.GenerateAppTokenRequest) (*pb.GenerateAppTokenResponse, error) {
	// TODO: Implement this when we prioritize adding a third-party app marketplace
	return nil, errNotImplemented
}
