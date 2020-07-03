package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
)

// ToggleFuseDrive switching on or off a mounted fuse drive
func (srv *grpcServer) ToggleFuseDrive(ctx context.Context, request *pb.ToggleFuseRequest) (*pb.FuseDriveResponse, error) {
	if srv.fc.IsMounted() == request.MountDrive {
		return &pb.FuseDriveResponse{
			FuseDriveMounted: request.MountDrive,
		}, nil
	}

	if request.MountDrive {
		if err := srv.fc.Mount(); err != nil {
			return nil, errors.Wrap(err, "failed to mount fuse drive")
		}
	} else {
		if err := srv.fc.Unmount(); err != nil {
			return nil, errors.Wrap(err, "failed to unmount fuse drive")
		}
	}

	return srv.GetFuseDriveStatus(ctx, nil)
}

// GetFuseDriveStatus returns the current mounted state
func (srv *grpcServer) GetFuseDriveStatus(ctx context.Context, empty *empty.Empty) (*pb.FuseDriveResponse, error) {
	return &pb.FuseDriveResponse{
		FuseDriveMounted: srv.fc.IsMounted(),
	}, nil
}
