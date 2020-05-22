package spacefs

import (
	"context"
	"syscall"

	"github.com/FleekHQ/space-poc/core/spacestore"
)

// SpaceFS is represents the filesystem
// It implements the FSOps interface
// And is responsible for managing file access, encryption and decryption
type SpaceFS struct {
	ctx   context.Context
	store spacestore.SpaceStore
}

// NewSpaceFS initializes a SpaceFS instance with IPFS peer listening
func NewSpaceFS(ctx context.Context, store spacestore.SpaceStore) (*SpaceFS, error) {
	return &SpaceFS{
		ctx:   ctx,
		store: store,
	}, nil
}

// Root implements the FSOps Root function
// It returns the root directory of the file
func (fs *SpaceFS) Root() (DirEntryOps, error) {
	// TODO: fetch the root block node and cache their information locally
	entry, err := fs.store.Get(fs.ctx, "/")
	if err != nil {
		return nil, err
	}

	return &SpaceDirectory{
		fs:    fs,
		entry: entry,
	}, nil
}

// LookupPath implements the FsOps interface for looking up information
// in a directory
func (fs *SpaceFS) LookupPath(path string) (DirEntryOps, error) {
	entry, err := fs.store.Get(fs.ctx, path)

	if err != nil {
		return nil, syscall.ENOENT
	}
	if entry.IsDir() {
		return &SpaceDirectory{
			fs:    fs,
			entry: entry,
		}, nil
	}

	return &SpaceFile{
		fs:    fs,
		entry: entry,
	}, nil
}

// Open a file at specified path
func (fs *SpaceFS) Open(path string, mode FileHandlerMode) (FileHandler, error) {
	result, err := fs.store.Open(fs.ctx, path)
	return result, err
}

// SpaceDirectory is a directory managed by space
type SpaceDirectory struct {
	fs    *SpaceFS
	entry *spacestore.DirEntry
}

// Path implements DirEntryOps Path() and return the path of the directory
func (dir *SpaceDirectory) Path() string {
	return dir.entry.Path()
}

// Attribute implements DirEntryOps Attribute() and fetches the metadata of the directory
func (dir *SpaceDirectory) Attribute() (DirEntryAttribute, error) {
	return dir.entry, nil
}

// ReadDir implements DirOps ReadDir and returns the list of entries in a directory
func (dir *SpaceDirectory) ReadDir() ([]DirEntryOps, error) {
	childrenEntries, err := dir.fs.store.GetChildren(dir.fs.ctx, dir.entry.Path())
	if err != nil {
		return nil, syscall.ENOENT
	}

	var result []DirEntryOps
	for _, entry := range childrenEntries {
		if entry.IsDir() {
			result = append(result, &SpaceDirectory{
				fs:    dir.fs,
				entry: entry,
			})
		} else {
			result = append(result, &SpaceFile{
				fs:    dir.fs,
				entry: entry,
			})
		}
	}

	return result, nil
}

// SpaceFile is a file managed by space
type SpaceFile struct {
	fs    *SpaceFS
	entry *spacestore.DirEntry
}

// Path implements DirEntryOps Path() and return the path of the directory
func (f *SpaceFile) Path() string {
	return f.entry.Path()
}

// Attribute implements DirEntryOps Attribute() and fetches the metadata of the directory
func (f *SpaceFile) Attribute() (DirEntryAttribute, error) {
	return f.entry, nil
}

// Open implements FileOps Open
// It should download/cache the content of the file and return a fileHandler
// that can read the cached content.
func (f *SpaceFile) Open(mode FileHandlerMode) (FileHandler, error) {
	fileHandler, err := f.fs.Open(f.entry.Path(), mode)
	return fileHandler, err
}
