package grpc

import (
	"context"
	"strconv"
	"time"

	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/grpc/pb"
	"github.com/FleekHQ/space-poc/log"
	"github.com/golang/protobuf/ptypes/empty"
)

func (srv *grpcServer) sendFileEvent(event *pb.FileEventResponse) {
	if srv.fileEventStream != nil {
		log.Info("sending events to client")
		srv.fileEventStream.Send(event)
	}
}

func (srv *grpcServer) SendFileEvent(event events.FileEvent) {
	pe := &pb.FileEventResponse{}

	srv.sendFileEvent(pe)
}

func (srv *grpcServer) GetPathInfo(ctx context.Context, req *pb.PathInfoRequest) (*pb.PathInfoResponse, error) {
	var res domain.FileInfo
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

func (srv *grpcServer) ListDirectories(ctx context.Context, request *pb.ListDirectoriesRequest) (*pb.ListDirectoriesResponse, error) {
	entries, err := srv.sv.ListDir(ctx)
	if err != nil {
		return nil, err
	}

	dirEntries := make([]*pb.ListDirectoryEntry, 0)

	for _, e := range entries {
		dirEntry := &pb.ListDirectoryEntry{
			Path:          e.Path,
			IsDir:         e.IsDir,
			Name:          e.Name,
			SizeInBytes:   e.SizeInBytes,
			Created:       e.Created,
			Updated:       e.Updated,
			FileExtension: e.FileExtension,
			IpfsHash:      e.IpfsHash,
		}
		dirEntries = append(dirEntries, dirEntry)
	}

	res := &pb.ListDirectoriesResponse{
		Entries: dirEntries,
	}

	return res, nil
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
		mockFileResponse := &pb.FileEventResponse{Type: pb.EventType_ENTRY_ADDED, Entry: &pb.ListDirectoryEntry{
			Path:          "temp/path",
			IsDir:         false,
			Name:          "myPath.txt",
			SizeInBytes:   "600",
			Created:       "",
			Updated:       "",
			FileExtension: "txt",
		}}
		srv.sendFileEvent(mockFileResponse)
	}

	log.Info("closing stream")
	return nil
}

func (srv *grpcServer) registerStream(stream pb.SpaceApi_SubscribeServer) {
	srv.fileEventStream = stream
}

func (srv *grpcServer) OpenFile(ctx context.Context, request *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	fi, err := srv.sv.OpenFile(ctx, request.Path, "")
	if err != nil {
		return nil, err
	}

	return &pb.OpenFileResponse{Location: fi.Location}, nil
}
