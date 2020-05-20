package filesystem

import (
	"os"
	"time"
)

// DirEntryAttribute similar to the FileInfo in the os.Package
type DirEntryAttribute interface {
	Name() string       // base name of the file
	Size() uint64       // length in bytes for files; can be anything for directories
	Mode() os.FileMode  // file mode bits
	Ctime() time.Time   // creation time
	ModTime() time.Time // modification time
	IsDir() bool
}

// Implements the DirEntryOps
type DirEntry struct {
	attr DirEntryAttribute
	path string
}

// DirEntryOps are the list of actions to be invoked on a directry entry
// A directory entry is either a file or a folder.
// See DirOps and FileOps for operations specific to those types
type DirEntryOps interface {
	// Path should return the absolute path string for directory
	Path() string
	// Attribute should return the metadata information for the file
	Attribute() DirEntryAttribute
}

// DirOps are the list of actions that can be done on a directory
type DirOps interface {
}

// FileOps are the list of actions that can be done on a file
type FileOps interface {
}

// FSOps represents the filesystem operations
type FSOps interface {
	LookupPath(path string) (DirEntryOps, error)
}
