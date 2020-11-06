package search

import (
	"context"
)

// Represents Search Engines for File and Folders
// Can be used for indexing and querying of File/Folders
type FilesSearchEngine interface {
	Start() error
	InsertFileData(ctx context.Context, data *InsertIndexRecord) (*IndexRecord, error)
	DeleteFileData(ctx context.Context, data *DeleteIndexRecord) error
	QueryFileData(ctx context.Context, query string, limit int) ([]*IndexRecord, error)
}
