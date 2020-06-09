package grpc

import (
	"context"
	"strconv"

	"github.com/FleekHQ/space-poc/core/events"
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

func (srv *grpcServer) sendTextileEvent(event *pb.TextileEventResponse) {
	if srv.txlEventStream != nil {
		log.Info("sending events to client")
		srv.txlEventStream.Send(event)
	}
}

func (srv *grpcServer) SendTextileEvent(event events.TextileEvent) {
	pe := &pb.TextileEventResponse{}

	srv.sendTextileEvent(pe)
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
	// waits until request is done
	select {
	case <-stream.Context().Done():
		break
	}
	// clean up stream
	srv.registerStream(nil)
	log.Info("closing stream")
	return nil
}

func (srv *grpcServer) registerStream(stream pb.SpaceApi_SubscribeServer) {
	srv.fileEventStream = stream
}

func (srv *grpcServer) TxlSubscribe(empty *empty.Empty, stream pb.SpaceApi_TxlSubscribeServer) error {
	srv.registerTxlStream(stream)
	// waits until request is done
	select {
	case <-stream.Context().Done():
		break
	}
	// clean up stream
	srv.registerTxlStream(nil)
	log.Info("closing stream")
	return nil
}

func (srv *grpcServer) registerTxlStream(stream pb.SpaceApi_TxlSubscribeServer) {
	srv.txlEventStream = stream
}

func (srv *grpcServer) OpenFile(ctx context.Context, request *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	fi, err := srv.sv.OpenFile(ctx, request.Path, "")
	if err != nil {
		return nil, err
	}

	return &pb.OpenFileResponse{Location: fi.Location}, nil
}

func (srv *grpcServer) AddItems(ctx context.Context, request *pb.AddItemsRequest) (*pb.AddItemsResponse, error) {
	res, err := srv.sv.AddItems(ctx, request.SourcePaths, request.TargetPath)
	if err != nil {
		return nil, err
	}

	results := make([]*pb.AddItemResult, 0)
	errors := make([]*pb.AddItemError, 0)

	for _, r := range res.Results {
		results = append(results, &pb.AddItemResult{
			SourcePath: r.SourcePath,
			BucketPath: r.BucketPath,
		})
	}

	for _, e := range res.Errors {
		errors = append(errors, &pb.AddItemError{
			SourcePath: e.SourcePath,
			Error:     e.Error.Error(),
		})
	}

	return &pb.AddItemsResponse{
		Results: results,
		Errors:  errors,
	}, nil

}

func (srv *grpcServer) CreateFolder(ctx context.Context, request *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	err := srv.sv.CreateFolder(ctx, request.Path)
	if err != nil {
		return nil, err
	}

	return &pb.CreateFolderResponse{}, nil
}
