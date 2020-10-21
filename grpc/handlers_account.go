package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/pkg/errors"
)

func (srv *grpcServer) DeleteAccount(ctx context.Context, request *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {

	if err := srv.fc.Unmount(); err != nil {
		return nil, errors.Wrap(err, "failed to unmount fuse drive")
	}

	if err := srv.sv.TurnOffComponents(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to turn OFF components")
	}

	if err := srv.sv.DeleteKeypair(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to remove keypair")
	}

	if err := srv.sv.TruncateData(ctx); err != nil {
		return nil, errors.Wrap(err, "error during clean up")
	}

	if err := srv.sv.TurnOnComponents(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to turn ON components")
	}

	return &pb.DeleteAccountResponse{}, nil
}
