package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) GenerateKeyPair(ctx context.Context, request *pb.GenerateKeyPairRequest) (*pb.GenerateKeyPairResponse, error) {
	mnemonic, err := srv.sv.GenerateKeyPair(ctx, false)
	if err != nil {
		return nil, err
	}

	return &pb.GenerateKeyPairResponse{
		Mnemonic: mnemonic,
	}, nil

}

func (srv *grpcServer) GenerateKeyPairWithForce(ctx context.Context, request *pb.GenerateKeyPairRequest) (*pb.GenerateKeyPairResponse, error) {
	mnemonic, err := srv.sv.GenerateKeyPair(ctx, true)
	if err != nil {
		return nil, err
	}

	return &pb.GenerateKeyPairResponse{
		Mnemonic: mnemonic,
	}, nil
}

func (srv *grpcServer) GetPublicKey(ctx context.Context, request *pb.GetPublicKeyRequest) (*pb.GetPublicKeyResponse, error) {
	pub, err := srv.sv.GetPublicKey(ctx)
	if err != nil {
		return nil, err
	}

	// tok, err := srv.sv.GetHubAuthToken(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	return &pb.GetPublicKeyResponse{
		PublicKey:    pub,
		HubAuthToken: "",
	}, nil
}

func (srv *grpcServer) DeleteKeyPair(ctx context.Context, request *pb.DeleteKeyPairRequest) (*pb.DeleteKeyPairResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) RestoreKeyPairViaMnemonic(ctx context.Context, request *pb.RestoreKeyPairViaMnemonicRequest) (*pb.RestoreKeyPairViaMnemonicResponse, error) {
	if err := srv.sv.RestoreKeyPairFromMnemonic(ctx, request.Mnemonic); err != nil {
		return nil, err
	}

	return &pb.RestoreKeyPairViaMnemonicResponse{}, nil
}

func (srv *grpcServer) GetStoredMnemonic(ctx context.Context, request *pb.GetStoredMnemonicRequest) (*pb.GetStoredMnemonicResponse, error) {
	mnemonic, err := srv.sv.GetMnemonic(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetStoredMnemonicResponse{
		Mnemonic: mnemonic,
	}, nil
}
