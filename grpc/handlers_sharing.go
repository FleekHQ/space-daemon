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
	res, err := srv.sv.GenerateFileSharingLink(ctx, request.FilePath, request.Bucket)
	if err != nil {
		return nil, err
	}

	return &pb.GenerateFileShareLinkResponse{
		Link:    res.SpaceDownloadLink,
		FileCid: res.SharedFileCid,
		FileKey: res.SharedFileKey,
	}, nil
}

func (srv *grpcServer) OpenPublicSharedFile(ctx context.Context, request *pb.OpenSharedFileRequest) (*pb.OpenSharedFileResponse, error) {
	res, err := srv.sv.OpenSharedFile(ctx, request.FileCid, request.FileKey, request.Filename)
	if err != nil {
		return nil, err
	}

	return &pb.OpenSharedFileResponse{
		Location: res.Location,
	}, nil
}
