package services

import (
	"context"
	"path"

	"github.com/FleekHQ/space-daemon/core/textile/model"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

func (s *Space) SearchFiles(ctx context.Context, query string) ([]domain.SearchFileEntry, error) {
	searchResult, err := s.tc.GetModel().QuerySearchIndex(ctx, query)
	if err != nil {
		return nil, err
	}

	resultEntries := make([]domain.SearchFileEntry, len(searchResult))

	for i, result := range searchResult {
		resultEntries[i] = domain.SearchFileEntry{
			FileInfo: domain.FileInfo{
				DirEntry: domain.DirEntry{
					Path:          result.ItemPath,
					IsDir:         result.ItemType == string(model.DirectoryItem),
					Name:          result.ItemName,
					FileExtension: path.Ext(result.ItemName),
				},
			},
			Bucket: result.BucketSlug,
			DbID:   result.DbId,
		}
	}

	return resultEntries, nil
}
