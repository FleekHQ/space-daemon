package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/grpc/pb"
)

// Search files based on query fields
// NOTE: This is still a TODO the current implementation just returns a list of files in the base personal bucket
func (srv *grpcServer) SearchFiles(ctx context.Context, request *pb.SearchFilesRequest) (*pb.SearchFilesResponse, error) {
	if request.Query == "" {
		return &pb.SearchFilesResponse{
			Entries: []*pb.ListDirectoryEntry{},
			Query:   request.Query,
		}, nil
	}

	entries, err := srv.sv.ListDir(ctx, "", "", false)
	if err != nil {
		return nil, err
	}

	dirEntries := mapFileInfoToDirectoryEntry(entries)

	return &pb.SearchFilesResponse{
		Entries: dirEntries,
		Query:   request.Query,
	}, nil
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
