package grpc

import (
	"context"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/pkg/errors"
	"os"
	"os/user"
	"path/filepath"
)

func (srv *grpcServer) DeleteAccount(ctx context.Context, request *pb.DeleteAccountRequest) (*pb.DeleteAccountResponse, error) {

	usr, err := user.Current()

	if err != nil {
		return nil, err
	}

	if err := srv.fc.Unmount(); err != nil {
		return nil, errors.Wrap(err, "failed to unmount fuse drive")
	}

	if err := srv.sv.DeleteKeypair(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to remove keypair")
	}

	// do we need to shutdown store here?

	// remove data dirs
	buckdDir := filepath.Join(usr.HomeDir, ".buckd")
	os.RemoveAll(buckdDir)

	fleekDir := filepath.Join(usr.HomeDir, ".fleek-space")
	os.RemoveAll(fleekDir)

	// should we also remove .ipfs?
	// ipfsDir := filepath.Join(usr.HomeDir, ".ipfs")
	// os.RemoveAll(ipfsDir)

	return &pb.DeleteAccountResponse{}, nil
}
