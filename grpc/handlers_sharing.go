package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) ShareBucketViaPublicKey(ctx context.Context, request *pb.ShareBucketViaPublicKeyRequest) (*pb.ShareBucketViaPublicKeyResponse, error) {
	err := srv.sv.ShareBucketViaPublicKey(ctx, request.PublicKeys, request.Bucket, nil)
	return &pb.ShareBucketViaPublicKeyResponse{}, err
}

func (srv *grpcServer) CopyAndShareFiles(ctx context.Context, request *pb.CopyAndShareFilesRequest) (*pb.CopyAndShareFilesResponse, error) {
	err := srv.sv.CopyAndShareFiles(ctx, request.Bucket, request.ItemPaths, request.PublicKeys, request.CustomMessage)
	if err != nil {
		return nil, err
	}

	return &pb.CopyAndShareFilesResponse{}, nil
}

func (srv *grpcServer) GeneratePublicFileLink(ctx context.Context, request *pb.GeneratePublicFileLinkRequest) (*pb.GeneratePublicFileLinkResponse, error) {
	// TODO: Generalize for multiple file upload
	res, err := srv.sv.GenerateFileSharingLink(ctx, request.ItemPaths[0], request.Bucket)
	if err != nil {
		return nil, err
	}

	return &pb.GeneratePublicFileLinkResponse{
		Link:    res.SpaceDownloadLink,
		FileCid: res.SharedFileCid,
	}, nil
}

func (srv *grpcServer) OpenPublicFile(ctx context.Context, request *pb.OpenPublicFileRequest) (*pb.OpenPublicFileResponse, error) {
	res, err := srv.sv.OpenSharedFile(ctx, request.FileCid, request.FileKey, request.Filename)
	if err != nil {
		return nil, err
	}

	return &pb.OpenPublicFileResponse{
		Location: res.Location,
	}, nil
}
