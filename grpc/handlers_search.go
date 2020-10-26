package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
)

// Search files based on query fields
// NOTE: This is still a TODO the current implementation just returns a list of files in the base personal bucket
func (srv *grpcServer) SearchFiles(ctx context.Context, request *pb.SearchFilesRequest) (*pb.SearchFilesResponse, error) {
	if request.Query == "" {
		return &pb.SearchFilesResponse{
			Entries: []*pb.SearchFilesDirectoryEntry{},
			Query:   request.Query,
		}, nil
	}

	entries, err := srv.sv.ListDir(ctx, "", "", false)
	if err != nil {
		return nil, err
	}

	dirEntries := mapFileInfoToDirectoryEntry(entries)
	searchResponseEntries := make([]*pb.SearchFilesDirectoryEntry, len(dirEntries))
	for i, e := range dirEntries {
		searchResponseEntries[i] = &pb.SearchFilesDirectoryEntry{
			Entry:  e,
			DbId:   "", // TODO: To be filled
			Bucket: "", // TODO: To be filled
		}
	}

	return &pb.SearchFilesResponse{
		Entries: searchResponseEntries,
		Query:   request.Query,
	}, nil
}
