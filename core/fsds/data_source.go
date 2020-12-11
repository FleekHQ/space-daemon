package fsds

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FileReadWriterCloser implements interfaces to read, copy, seek and close.
type FileReadWriterCloser interface {
	Read(ctx context.Context, data []byte, offset int64) (int, error)
	Write(ctx context.Context, data []byte, offset int64) (int, error)
	Close(ctx context.Context) error
	Stats(ctx context.Context) (*DirEntry, error)
	Truncate(ctx context.Context, size uint64) error
}

// FSDataSource is data source of file/directories and their information
// It is used as a local/remote cache for looking up information about the directories.
// It should also ensure that the user in the context has permission to data that is being request
type FSDataSource interface {
	// Get a single node, this can be called on either a file or folder entry
	// This is typically used by the OS for lookup of the information about the entry at path
	Get(ctx context.Context, path string) (*DirEntry, error)
	// GetChildren returns child entries in the directory/folder
	GetChildren(ctx context.Context, path string) ([]*DirEntry, error)
	// OpenReader returns a file reader
	Open(ctx context.Context, path string) (FileReadWriterCloser, error)
	// CreateEntry should create a directory or file based on the mode at the path
	CreateEntry(ctx context.Context, path string, mode os.FileMode) (*DirEntry, error)
	// RenameEntry should rename the directory entry from old to new
	RenameEntry(ctx context.Context, oldPath, newPath string) error
	// DeleteEntry should delete the item at the path
	DeleteEntry(ctx context.Context, path string) error
}

// TLFDataSource represents a data source handler for a particular top level file.
type TLFDataSource struct {
	name     string
	basePath string
	FSDataSource
}

// Returns child path inside data source
func (t *TLFDataSource) ChildPath(path string) string {
	return strings.TrimPrefix(path, t.basePath)
}

// returns the path with the datasource base path prefixed
func (t *TLFDataSource) ParentPath(path string) string {
	return filepath.Join(t.basePath, path)
}
