package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) ShareBucketViaEmail(ctx context.Context, request *pb.ShareBucketViaEmailRequest) (*pb.ShareBucketViaEmailResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) ShareBucketViaIdentity(ctx context.Context, request *pb.ShareBucketViaIdentityRequest) (*pb.ShareBucketViaIdentityResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) GenerateFileShareLink(ctx context.Context, request *pb.GenerateFileShareLinkRequest) (*pb.GenerateFileShareLinkResponse, error) {
	return nil, errNotImplemented
}
