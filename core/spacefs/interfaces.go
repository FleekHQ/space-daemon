package spacefs

import (
	"io"
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
	ReadDir() ([]DirEntryOps, error)
}

// FileHandler is in charge of reading, writing and closing access to a file
// It should handle locking and track read and write offsets till it is closed
type FileHandler interface {
	io.ReadWriteSeeker
	io.Closer
}

// FileOps are the list of actions that can be done on a file
type FileOps interface {
	DirEntryOps
	Open() (FileHandler, error)
}

// FSOps represents the filesystem operations
type FSOps interface {
	// Root should return the root directory entry
	Root() (DirEntryOps, error)
	// LookupPath should return the directory entry at that particular path
	LookupPath(path string) (DirEntryOps, error)
}
