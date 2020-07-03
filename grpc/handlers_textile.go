package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) CreateBucket(ctx context.Context, request *pb.CreateBucketRequest) (*pb.CreateBucketResponse, error) {
	b, err := srv.sv.CreateBucket(ctx, request.Slug)
	if err != nil {
		return nil, err
	}
	bd := b.GetData()
	br := &pb.Bucket{
		Key:       bd.Key,
		Name:      bd.Name,
		Path:      bd.Path,
		CreatedAt: bd.CreatedAt,
		UpdatedAt: bd.UpdatedAt,
	}

	return &pb.CreateBucketResponse{
		Bucket: br,
	}, nil
}
