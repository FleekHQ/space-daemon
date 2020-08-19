package spacefs

import (
	"context"
	"os"
	"time"
)

type FileHandlerMode uint8

const (
	ReadMode = FileHandlerMode(0)
	WriteMode
)

// DirEntryAttribute similar to the FileInfo in the os.Package
type DirEntryAttribute interface {
	Name() string       // base name of the file
	Size() uint64       // length in bytes for files; can be anything for directories
	Mode() os.FileMode  // file mode bits
	Uid() string        // user id of owner of entry
	Gid() string        // group id of owner of entry
	Ctime() time.Time   // creation time
	ModTime() time.Time // modification time
	IsDir() bool
}

// DirEntryOps are the list of actions to be invoked on a directry entry
// A directory entry is either a file or a folder.
// See DirOps and FileOps for operations specific to those types
type DirEntryOps interface {
	// Path should return the absolute path string for directory or file
	// Directory path's should end in `/`
	Path() string
	// Attribute should return the metadata information for the file
	Attribute() (DirEntryAttribute, error)
}

// DirOps are the list of actions that can be done on a directory
type DirOps interface {
	DirEntryOps
	ReadDir(ctx context.Context) ([]DirEntryOps, error)
}

// FileHandler is in charge of reading, writing and closing access to a file
// It should handle locking and track read and write offsets till it is closed
type FileHandler interface {
	Read(ctx context.Context, data []byte, offset int64) (int, error)
	Write(ctx context.Context, data []byte, offset int64) (int, error)
	Close(ctx context.Context) error
}

// FileOps are the list of actions that can be done on a file
type FileOps interface {
	DirEntryOps
	Open(ctx context.Context, mode FileHandlerMode) (FileHandler, error)
}

type CreateDirEntry struct {
	Path string
	Mode os.FileMode
}

// FSOps represents the filesystem operations
type FSOps interface {
	// Root should return the root directory entry
	Root(ctx context.Context) (DirEntryOps, error)
	// LookupPath should return the directory entry at that particular path
	LookupPath(ctx context.Context, path string) (DirEntryOps, error)
	// Open a file at specific path, with specified mode
	Open(ctx context.Context, path string, mode FileHandlerMode) (FileHandler, error)
	// CreateEntry should create an directory entry and return either a FileOps or DirOps entry
	// depending on the mode
	CreateEntry(ctx context.Context, req CreateDirEntry) (DirEntryOps, error)
}
