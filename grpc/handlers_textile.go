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

func (srv *grpcServer) ListBuckets(ctx context.Context, request *pb.ListBucketsRequest) (*pb.ListBucketsResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) ShareBucket(ctx context.Context, request *pb.ShareBucketRequest) (*pb.ShareBucketResponse, error) {
	i, err := srv.sv.ShareBucket(ctx, request.bucket)
	if err != nil {
		return nil, err
	}
	ti := &pb.ThreadInfo{
		Addresses: i.addresses,
		Key:       i.key,
	}

	return &pb.ShareBucketResponse{
		ThreadInfo: threadinfo,
	}, nil
}

func (srv *grpcServer) JoinBucket(ctx context.Context, request *pb.JoinBucketRequest) (*pb.JoinBucketResponse, error) {
	r, err := srv.sv.JoinBucket(ctx, request.bucket, request.threadinfo)
	if err != nil {
		return nil, err
	}

	return &pb.ShareBucketResponse{
		Result: r,
	}, nil
}
