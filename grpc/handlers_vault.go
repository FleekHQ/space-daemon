package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) BackupKeysByPassphrase(ctx context.Context, request *pb.BackupKeysByPassphraseRequest) (*pb.BackupKeysByPassphraseResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) RecoverKeysByPassphrase(ctx context.Context, request *pb.RecoverKeysByPassphraseRequest) (*pb.RecoverKeysByPassphraseResponse, error) {
	return nil, errNotImplemented
}
