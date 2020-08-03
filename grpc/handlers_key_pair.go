package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
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

func (srv *grpcServer) GetPublicKey(ctx context.Context, request *pb.GetPublicKeyRequest) (*pb.GetPublicKeyResponse, error) {
	pub, err := srv.sv.GetPublicKey(ctx)
	if err != nil {
		return nil, err
	}

	tok, err := srv.sv.GetHubAuthToken(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetPublicKeyResponse{
		PublicKey:    pub,
		HubAuthToken: tok,
	}, nil
}

func (srv *grpcServer) DeleteKeyPair(ctx context.Context, request *pb.DeleteKeyPairRequest) (*pb.DeleteKeyPairResponse, error) {
	return nil, errNotImplemented
}
