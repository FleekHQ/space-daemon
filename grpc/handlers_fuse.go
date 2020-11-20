package grpc

import (
	"context"

	"github.com/opentracing/opentracing-go"

	"github.com/FleekHQ/space-daemon/core/space/fuse"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
)

// ToggleFuseDrive switching on or off a mounted fuse drive
func (srv *grpcServer) ToggleFuseDrive(ctx context.Context, request *pb.ToggleFuseRequest) (*pb.FuseDriveResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ToggleFuseDrive")
	defer span.Finish()

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
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetFuseDriveStatus")
	defer span.Finish()

	state, err := srv.fc.GetFuseState(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.FuseDriveResponse{
		State: fuseStateToRpcState(state),
	}, nil
}

var fuseStateToRpcStateMap = map[fuse.State]pb.FuseState{
	fuse.UNSUPPORTED:   pb.FuseState_UNSUPPORTED,
	fuse.NOT_INSTALLED: pb.FuseState_NOT_INSTALLED,
	fuse.UNMOUNTED:     pb.FuseState_UNMOUNTED,
	fuse.MOUNTED:       pb.FuseState_MOUNTED,
}

func fuseStateToRpcState(state fuse.State) pb.FuseState {
	return fuseStateToRpcStateMap[state]
}
