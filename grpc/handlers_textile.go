package grpc

import (
	"context"

	"github.com/FleekHQ/space-poc/grpc/pb"
)

func (srv *grpcServer) CreateBucket(ctx context.Context, request *pb.CreateBucketRequest) (*pb.CreateBucketResponse, error) {
	b, err := srv.sv.CreateBucket(ctx, request.Slug)
	if err != nil {
		return nil, err
	}

	br := &pb.Bucket{
		Key:       b.GetData().Key,
		Name:      b.GetData().Name,
		Path:      b.GetData().Path,
		CreatedAt: b.GetData().CreatedAt,
		UpdatedAt: b.GetData().UpdatedAt,
	}

	return &pb.CreateBucketResponse{
		Bucket: br,
	}, nil
}
