package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) GetAPISessionTokens(ctx context.Context, request *pb.GetAPISessionTokensRequest) (*pb.GetAPISessionTokensResponse, error) {
	tokens, err := srv.sv.GetAPISessionTokens(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetAPISessionTokensResponse{
		HubToken:      tokens.HubToken,
		ServicesToken: tokens.ServicesToken,
	}, nil
}
