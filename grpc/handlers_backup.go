package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) ToggleBucketBackup(ctx context.Context, request *pb.ToggleBucketBackupRequest) (*pb.ToggleBucketBackupResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) GetUsageInfo(ctx context.Context, request *pb.GetUsageInfoRequest) (*pb.GetUsageInfoResponse, error) {
	return nil, errNotImplemented
}
