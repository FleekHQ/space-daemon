package fs_data_source

import (
	"context"
	"io"
	"os"
)

// ReadSeekCloser implements interfaces to read, copy, seek and close.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Writer
	io.Closer
}

// FSDataSource is data source of file/directories and their information
// It is used as a local/remote cache for looking up information about the directories.
// It should also ensure that the user in the context has permission to data that is being request
type FSDataSource interface {
	// Get a single node
	Get(ctx context.Context, path string) (*DirEntry, error)
	// GetChildren returns child entries in the file
	GetChildren(ctx context.Context, path string) ([]*DirEntry, error)
	// OpenReader returns a file reader
	Open(ctx context.Context, path string) (ReadSeekCloser, error)
	// CreateEntry should create a directory or file based on the mode at the path
	CreateEntry(ctx context.Context, path string, mode os.FileMode) (*DirEntry, error)
}
