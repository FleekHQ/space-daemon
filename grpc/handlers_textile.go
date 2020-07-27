package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func parseMember(m *domain.Member) *pb.BucketMember {
	parsed := &pb.BucketMember{
		Address: m.PublicKey, // there's no address on domain.Member?
		PublicKey: m.PublicKey,
		IsOwner: m.IsOwner,
		HasJoined: m.Joined,
	}

	return parsed;
}

func parseBucket(b textile.Bucket) *pb.Bucket {
	bd := b.GetData()
	members := b.GetMembers();

	parsedMembers := []*pb.BucketMember{}

	for _, m := range members {
		member := parseMember(m)
		parsedMembers = append(parsedMembers, member)
	}

	br := &pb.Bucket{
		Key:       bd.Key,
		Name:      bd.Name,
		Path:      bd.Path,
		CreatedAt: bd.CreatedAt,
		UpdatedAt: bd.UpdatedAt,
		Members:  parsedMembers,
		IsPersonalBucket: false,
	}

	return br
}

func (srv *grpcServer) CreateBucket(ctx context.Context, request *pb.CreateBucketRequest) (*pb.CreateBucketResponse, error) {
	b, err := srv.sv.CreateBucket(ctx, request.Slug)
	if err != nil {
		return nil, err
	}

	parsedBucket := parseBucket(b)
	return &pb.CreateBucketResponse{
		Bucket: parsedBucket,
	}, nil
}

func (srv *grpcServer) ListBuckets(ctx context.Context, request *pb.ListBucketsRequest) (*pb.ListBucketsResponse, error) {
	buckets, err := srv.sv.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	parsedBuckets := []*pb.Bucket{}

	for _, b := range buckets {
		parsedBucket := parseBucket(b)
		parsedBuckets = append(parsedBuckets, parsedBucket)
	}

	return &pb.ListBucketsResponse{
		Buckets: parsedBuckets,
	}, nil
}

func (srv *grpcServer) ShareBucket(ctx context.Context, request *pb.ShareBucketRequest) (*pb.ShareBucketResponse, error) {
	i, err := srv.sv.ShareBucket(ctx, request.Bucket)
	if err != nil {
		return nil, err
	}
	ti := &pb.ThreadInfo{
		Addresses: i.Addresses,
		Key:       i.Key,
	}

	return &pb.ShareBucketResponse{
		Threadinfo: ti,
	}, nil
}

func (srv *grpcServer) JoinBucket(ctx context.Context, request *pb.JoinBucketRequest) (*pb.JoinBucketResponse, error) {
	ti := &domain.ThreadInfo{
		Addresses: request.Threadinfo.Addresses,
		Key:       request.Threadinfo.Key,
	}
	r, err := srv.sv.JoinBucket(ctx, request.Bucket, ti)
	if err != nil {
		return nil, err
	}

	return &pb.JoinBucketResponse{
		Result: r,
	}, nil
}
