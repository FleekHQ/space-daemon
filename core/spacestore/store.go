package spacestore

import (
	"context"
)

// SpaceStore is local database of files and their information
// It is used as a local/remote cache for looking up information about the directories.
// It should also ensure that the user in the context has permission to data that is being request
type SpaceStore interface {
	// Get a single node
	Get(ctx context.Context, path string) (*DirEntry, error)
	// Get children
	GetChildren(ctx context.Context, path string) ([]*DirEntry, error)
}
