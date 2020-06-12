package grpc

import (
	"context"

	"github.com/FleekHQ/space/grpc/pb"
)

func (srv *grpcServer) GenerateKeyPair(ctx context.Context, request *pb.GenerateKeyPairRequest) (*pb.GenerateKeyPairResponse, error) {
	if kp, err := srv.sv.GenerateKeyPair(ctx, false); err != nil {
		return nil, err
	} else {
		return &pb.GenerateKeyPairResponse{
			PublicKey:  kp.PublicKey,
			PrivateKey: kp.PrivateKey,
		}, nil
	}
}

func (srv *grpcServer) GenerateKeyPairWithForce(ctx context.Context, request *pb.GenerateKeyPairRequest) (*pb.GenerateKeyPairResponse, error) {
	if kp, err := srv.sv.GenerateKeyPair(ctx, true); err != nil {
		return nil, err
	} else {
		return &pb.GenerateKeyPairResponse{
			PublicKey:  kp.PublicKey,
			PrivateKey: kp.PrivateKey,
		}, nil
	}
}
