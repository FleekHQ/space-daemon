package grpc

import (
	"context"
	"encoding/hex"

	"github.com/FleekHQ/space-poc/core/keychain"
	"github.com/FleekHQ/space-poc/grpc/pb"
)

func (sv *grpcServer) GenerateKeyPair(ctx context.Context, request *pb.GenerateKeyPairRequest) (*pb.GenerateKeyPairResponse, error) {
	kc := keychain.New(sv.db)
	if pub, priv, err := kc.GenerateKeyPair(); err != nil {
		return nil, err
	} else {
		return &pb.GenerateKeyPairResponse{
			PublicKey:  hex.EncodeToString(pub),
			PrivateKey: hex.EncodeToString(priv),
		}, nil
	}
}

func (sv *grpcServer) GenerateKeyPairWithForce(ctx context.Context, request *pb.GenerateKeyPairRequest) (*pb.GenerateKeyPairResponse, error) {
	kc := keychain.New(sv.db)
	if pub, priv, err := kc.GenerateKeyPairWithForce(); err != nil {
		return nil, err
	} else {
		return &pb.GenerateKeyPairResponse{
			PublicKey:  hex.EncodeToString(pub),
			PrivateKey: hex.EncodeToString(priv),
		}, nil
	}
}
