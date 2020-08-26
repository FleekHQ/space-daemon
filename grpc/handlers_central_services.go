package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) GetAPISessionTokens(ctx context.Context, request *pb.GetAPISessionTokensRequest) (*pb.GetAPISessionTokensResponse, error) {
	return &pb.GetAPISessionTokensResponse{
		HubToken: "",
		// TODO: Connect to token challenge
		ServicesToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwdWJrZXkiOiJhZTRiMmFiNjU4ZmJiNzcyMjE0MDRkNjU3YzZiNzQyZDJlZjdjNTI2YjZhNWE5YzIwMGNjZjkzZmNhMWRjZTYzIiwidXVpZCI6ImM5MDdlN2VmLTdiMzYtNGFiMS04YTU2LWY3ODhkNzUyNmEyYyIsImlhdCI6MTU5ODI4NTA0MSwiZXhwIjoxNjAwODc3MDQxfQ.dgp8UhWCLjsU0SjxXwSb3g0jEurt2jAKPaY3B_eO-qE",
	}, nil

}
