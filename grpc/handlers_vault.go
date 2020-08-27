package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

func (srv *grpcServer) BackupKeysByPassphrase(ctx context.Context, request *pb.BackupKeysByPassphraseRequest) (*pb.BackupKeysByPassphraseResponse, error) {
	resp := &pb.BackupKeysByPassphraseResponse{}
	err := srv.sv.BackupKeysByPassphrase(ctx, request.Uuid, request.Passphrase)

	return resp, err
}

func (srv *grpcServer) RecoverKeysByPassphrase(ctx context.Context, request *pb.RecoverKeysByPassphraseRequest) (*pb.RecoverKeysByPassphraseResponse, error) {
	resp := &pb.RecoverKeysByPassphraseResponse{}
	err := srv.sv.RecoverKeysByPassphrase(ctx, request.Uuid, request.Passphrase)

	return resp, err
}

func (srv *grpcServer) CreateLocalKeysBackup(ctx context.Context, request *pb.CreateLocalKeysBackupRequest) (*pb.CreateLocalKeysBackupResponse, error) {
	resp := &pb.CreateLocalKeysBackupResponse{}
	err := srv.sv.CreateLocalKeysBackup(ctx, request.PathToKeyBackup)

	return resp, err
}

func (srv *grpcServer) RecoverKeysByLocalBackup(ctx context.Context, request *pb.RecoverKeysByLocalBackupRequest) (*pb.RecoverKeysByLocalBackupResponse, error) {
	resp := &pb.RecoverKeysByLocalBackupResponse{}
	err := srv.sv.RecoverKeysByLocalBackup(ctx, request.PathToKeyBackup)

	return resp, err
}
