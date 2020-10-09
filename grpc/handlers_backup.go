package grpc

import (
	"context"
	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) ToggleBucketBackup(ctx context.Context, request *pb.ToggleBucketBackupRequest) (*pb.ToggleBucketBackupResponse, error) {
	bucketSlug := request.Bucket
	bucketBackup := request.Backup

	err := srv.sv.ToggleBucketBackup(ctx, bucketSlug, bucketBackup)
	if err != nil {
		return nil, err
	}

	return &pb.ToggleBucketBackupResponse{}, nil
}

func (srv *grpcServer) BucketBackupRestore(ctx context.Context, request *pb.BucketBackupRestoreRequest) (*pb.BucketBackupRestoreResponse, error) {
	bucketSlug := request.Bucket

	err := srv.sv.BucketBackupRestore(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	return &pb.BucketBackupRestoreResponse{}, nil
}

func (srv *grpcServer) GetUsageInfo(ctx context.Context, request *pb.GetUsageInfoRequest) (*pb.GetUsageInfoResponse, error) {
	return nil, errNotImplemented
}
