package grpc

import (
	"context"
	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) ToggleBucketBackup(ctx context.Context, request *pb.ToggleBucketBackupRequest) (*pb.ToggleBucketBackupResponse, error) {
	bucketName := request.Bucket
	bucketBackup := request.Backup

	err := srv.sv.ToggleBucketBackup(ctx, bucketName, bucketBackup)
	if err != nil {
		return nil, err
	}

	return &pb.ToggleBucketBackupResponse{}, nil
}

func (srv *grpcServer) GetUsageInfo(ctx context.Context, request *pb.GetUsageInfoRequest) (*pb.GetUsageInfoResponse, error) {
	return nil, errNotImplemented
}
