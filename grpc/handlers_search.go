package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

// Search files based on query fields
func (srv *grpcServer) SearchFiles(ctx context.Context, request *pb.SearchFilesRequest) (*pb.SearchFilesResponse, error) {
	if request.Query == "" {
		return &pb.SearchFilesResponse{
			Entries: []*pb.SearchFilesDirectoryEntry{},
			Query:   request.Query,
		}, nil
	}

	entries, err := srv.sv.SearchFiles(ctx, request.Query)
	if err != nil {
		return nil, err
	}

	searchResponseEntries := make([]*pb.SearchFilesDirectoryEntry, len(entries))
	for i, e := range entries {
		searchResponseEntries[i] = &pb.SearchFilesDirectoryEntry{
			Entry: &pb.ListDirectoryEntry{
				Path:                e.Path,
				IsDir:               e.IsDir,
				Name:                e.Name,
				SizeInBytes:         e.SizeInBytes,
				Created:             e.Created,
				Updated:             e.Updated,
				FileExtension:       e.FileExtension,
				IpfsHash:            e.IpfsHash,
				IsLocallyAvailable:  e.LocallyAvailable,
				IsBackupInProgress:  e.BackupInProgress,
				IsRestoreInProgress: e.RestoreInProgress,
			},
			DbId:   e.DbID,
			Bucket: e.Bucket,
		}
	}

	return &pb.SearchFilesResponse{
		Entries: searchResponseEntries,
		Query:   request.Query,
	}, nil
}
