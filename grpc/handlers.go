package grpc

import (
	"context"
	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/grpc/pb"
	"github.com/FleekHQ/space-poc/log"
	"github.com/golang/protobuf/ptypes/empty"
	"strconv"
	"time"
)

func (srv *grpcServer) sendFileEvent(event *pb.FileEventResponse) {
	if srv.fileEventStream != nil {
		log.Info("sending events to client")
		srv.fileEventStream.Send(event)
	}
}

func (srv *grpcServer) SendFileEvent(event events.FileEvent) {
	pe := &pb.FileEventResponse{
		Path: event.Path,
	}

	srv.sendFileEvent(pe)
}

func (srv *grpcServer) GetPathInfo(ctx context.Context, req *pb.PathInfoRequest) (*pb.PathInfoResponse, error) {
	var res domain.PathInfo
	var err error
	res, err = srv.sv.GetPathInfo(ctx, req.Path)
	if err != nil {
		return nil, err
	}

	return &pb.PathInfoResponse{
		Path:     res.Path,
		IpfsHash: res.IpfsHash,
		IsDir:    res.IsDir,
	}, nil
}

// TODO: implement
func (srv *grpcServer) ListDirectories(ctx context.Context, request *pb.ListDirectoriesRequest) (*pb.ListDirectoriesResponse, error) {
	panic("implement me")
}

func (srv *grpcServer) GetConfigInfo(ctx context.Context, e *empty.Empty) (*pb.ConfigInfoResponse, error) {
	appCfg := srv.sv.GetConfig(ctx)

	res := &pb.ConfigInfoResponse{
		FolderPath: appCfg.FolderPath,
		Port:       strconv.Itoa(appCfg.Port),
		AppPath:    appCfg.AppPath,
	}

	return res, nil
}

func (srv *grpcServer) Subscribe(empty *empty.Empty, stream pb.SpaceApi_SubscribeServer) error {
	srv.registerStream(stream)
	c := time.Tick(1 * time.Second)
	for i := 0; i < 10; i++ {
		<-c
		mockFileResponse := &pb.FileEventResponse{Path: "test/path"}
		srv.sendFileEvent(mockFileResponse)
	}

	log.Info("closing stream")
	return nil
}

func (srv *grpcServer) registerStream(stream pb.SpaceApi_SubscribeServer) {
	srv.fileEventStream = stream
}
