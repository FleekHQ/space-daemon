package grpc

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/golang/protobuf/ptypes/empty"
)

var errNotImplemented = errors.New("Not implemented")

func (srv *grpcServer) sendFileEvent(event *pb.FileEventResponse) {
	if srv.fileEventStream != nil {
		log.Info("sending events to client", event.String())
		srv.fileEventStream.Send(event)
	}
}

func (srv *grpcServer) SendFileEvent(event events.FileEvent) {
	dirEntries := mapFileInfoToDirectoryEntry([]domain.FileInfo{event.Info})
	entry := dirEntries[0]

	pe := &pb.FileEventResponse{
		Type:   mapFileEventToPb(event.Type),
		Entry:  entry,
		Bucket: event.Bucket,
		DbId:   event.DbID,
	}

	srv.sendFileEvent(pe)
}

func mapFileEventToPb(eventType events.FileEventType) pb.EventType {
	switch eventType {
	case events.FileAdded:
		return pb.EventType_ENTRY_ADDED
	case events.FileDeleted:
		return pb.EventType_ENTRY_DELETED
	case events.FileUpdated:
		return pb.EventType_ENTRY_UPDATED
	case events.FileBackupInProgress:
		return pb.EventType_ENTRY_BACKUP_IN_PROGRESS
	case events.FileBackupReady:
		return pb.EventType_ENTRY_BACKUP_READY
	case events.FileRestoring:
		return pb.EventType_ENTRY_RESTORE_IN_PROGRESS
	case events.FileRestored:
		return pb.EventType_ENTRY_RESTORE_READY
	case events.FolderAdded:
		return pb.EventType_FOLDER_ADDED
	case events.FolderDeleted:
		return pb.EventType_FOLDER_DELETED
	case events.FolderUpdated:
		return pb.EventType_FOLDER_UPDATED
	default:
		return pb.EventType_ENTRY_ADDED
	}
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
	bucketName := request.Bucket
	listMembers := !request.OmitMembers

	entries, err := srv.sv.ListDirs(ctx, "", bucketName, listMembers)
	if err != nil {
		return nil, err
	}

	dirEntries := mapFileInfoToDirectoryEntry(entries)

	res := &pb.ListDirectoriesResponse{
		Entries: dirEntries,
	}

	return res, nil
}

func (srv *grpcServer) ListDirectory(
	ctx context.Context,
	request *pb.ListDirectoryRequest,
) (*pb.ListDirectoryResponse, error) {
	listMembers := !request.OmitMembers

	entries, err := srv.sv.ListDir(ctx, request.GetPath(), request.GetBucket(), listMembers)
	if err != nil {
		return nil, err
	}

	dirEntries := mapFileInfoToDirectoryEntry(entries)

	res := &pb.ListDirectoryResponse{
		Entries: dirEntries,
	}

	return res, nil
}

func mapFileInfoToDirectoryEntry(entries []domain.FileInfo) []*pb.ListDirectoryEntry {
	dirEntries := make([]*pb.ListDirectoryEntry, 0)

	for _, e := range entries {
		members := make([]*pb.FileMember, 0)

		for _, m := range e.Members {
			members = append(members, &pb.FileMember{
				Address:   m.Address,
				PublicKey: m.PublicKey,
			})
		}

		var backupCount = 0
		if e.BackedUp {
			backupCount = 1
		}

		dirEntry := &pb.ListDirectoryEntry{
			Path:                e.Path,
			IsDir:               e.IsDir,
			Name:                e.Name,
			SizeInBytes:         e.SizeInBytes,
			Created:             e.Created,
			Updated:             e.Updated,
			FileExtension:       e.FileExtension,
			IpfsHash:            e.IpfsHash,
			Members:             members,
			BackupCount:         int64(backupCount),
			IsLocallyAvailable:  e.LocallyAvailable,
			IsBackupInProgress:  e.BackupInProgress,
			IsRestoreInProgress: e.RestoreInProgress,
		}
		dirEntries = append(dirEntries, dirEntry)
	}
	return dirEntries
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
	fi, err := srv.sv.OpenFile(ctx, request.Path, request.Bucket, request.DbId)
	if err != nil {
		return nil, err
	}

	return &pb.OpenFileResponse{Location: fi.Location}, nil
}

func (srv *grpcServer) AddItems(request *pb.AddItemsRequest, stream pb.SpaceApi_AddItemsServer) error {
	ctx := stream.Context()

	results, totals, err := srv.sv.AddItems(ctx, request.SourcePaths, request.TargetPath, request.Bucket)
	if err != nil {
		return err
	}

	notifications := make(chan domain.AddItemResult)

	done := make(chan struct{})

	// push notification stream from out
	go func() {
		var completedBytes int64
		var completedFiles int64
		for res := range notifications {
			completedFiles++
			var r *pb.AddItemsResponse
			if res.Error != nil {
				r = &pb.AddItemsResponse{
					Result: &pb.AddItemResult{
						SourcePath: res.SourcePath,
						Error:      res.Error.Error(),
					},
					TotalFiles:     totals.TotalFiles,
					TotalBytes:     totals.TotalBytes,
					CompletedFiles: completedFiles,
					CompletedBytes: completedBytes,
				}
			} else {
				completedBytes += res.Bytes
				r = &pb.AddItemsResponse{
					Result: &pb.AddItemResult{
						SourcePath: res.SourcePath,
						BucketPath: res.BucketPath,
					},
					TotalFiles:     totals.TotalFiles,
					TotalBytes:     totals.TotalBytes,
					CompletedFiles: completedFiles,
					CompletedBytes: completedBytes,
				}
			}
			stream.Send(r)
		}
		done <- struct{}{}
	}()

	// receive results from service
	for in := range results {
		select {
		case notifications <- in:
		case <-stream.Context().Done():
			break
		}
	}

	// close out channel and stream
	close(notifications)
	// wait for all notifications to finish
	<-done
	log.Printf("closing stream for addFiles")

	return nil
}

func (srv *grpcServer) CreateFolder(ctx context.Context, request *pb.CreateFolderRequest) (*pb.CreateFolderResponse, error) {
	err := srv.sv.CreateFolder(ctx, request.Path, request.Bucket)
	if err != nil {
		return nil, err
	}

	return &pb.CreateFolderResponse{}, nil
}
