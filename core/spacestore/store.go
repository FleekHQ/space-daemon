package spacestore

import (
	"context"
	"io"
)

// ReadSeekCloser implements interfaces to read, copy, seek and close.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Writer
	io.Closer
}

// SpaceStore is local database of files and their information
// It is used as a local/remote cache for looking up information about the directories.
// It should also ensure that the user in the context has permission to data that is being request
type SpaceStore interface {
	// Get a single node
	Get(ctx context.Context, path string) (*DirEntry, error)
	// GetChildren returns child entries in the file
	GetChildren(ctx context.Context, path string) ([]*DirEntry, error)
	// OpenReader returns a file reader
	Open(ctx context.Context, path string) (ReadSeekCloser, error)
}
