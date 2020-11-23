package fsds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FleekHQ/space-daemon/core/space/domain"

	"github.com/FleekHQ/space-daemon/core/space"
	"github.com/FleekHQ/space-daemon/log"
)

type sharedFileEntry struct {
	entry  *DirEntry
	dbId   string
	bucket string
}

// Provides content for the 'Shared With Me' content managed by the space user
// Requests for items in this path are dispatched to this datasource from SpaceFSDataSource
type sharedWithMeDataSource struct {
	service     space.Service
	maxDirLimit int
	cache       map[string]*sharedFileEntry
}

// Maybe consider caching at the level of SpaceFSDataSource when results are returned from top level file
func (f *sharedWithMeDataSource) Get(ctx context.Context, path string) (*DirEntry, error) {
	baseName := filepath.Base(path)

	if isBaseDirectory(path) || path == "" {
		// return parent directory info
		return NewDirEntryWithMode(domain.DirEntry{
			Path:    path,
			IsDir:   true,
			Name:    baseName,
			Created: time.Now().Format(time.RFC3339),
			Updated: time.Now().Format(time.RFC3339),
		}, RestrictedDirAccessMode), nil
	}
	log.Debug("SharedWithMeDS Get", fmt.Sprintf("path:%s", path))

	// check cache if Item is already there
	entry, exists := f.cache[path]
	if exists {
		return entry.entry, nil
	}

	itemsInParent, _, err := f.service.GetSharedWithMeFiles(ctx, "", f.maxDirLimit)
	if err != nil {
		if !isNotExistError(err) {
			return nil, EntryNotFound
		}
		return nil, err
	}

	f.cacheResults(itemsInParent)

	// find item matching path
	for _, entry := range itemsInParent {
		if entry.Path == path {
			return NewDirEntry(entry.DirEntry), nil
		}
	}

	return nil, EntryNotFound
}

// GetChildren should only be called on the parent folder and will always return
func (f *sharedWithMeDataSource) GetChildren(ctx context.Context, path string) ([]*DirEntry, error) {
	log.Debug("SharedWithMeDS GetChildren", fmt.Sprintf("path:%s", path))
	if !isBaseDirectory(path) && path != "" {
		// just return empty directory since shared with me currently only supports files
		return []*DirEntry{}, nil
	}

	entries, _, err := f.service.GetSharedWithMeFiles(ctx, "", f.maxDirLimit)
	if err != nil {
		return nil, err
	}

	// this ensure it always refreshes the cache whenever operating system calls list directory
	f.cacheResults(entries)

	dirEntries := make([]*DirEntry, len(entries))
	for i, entry := range entries {
		dirEntries[i] = NewDirEntry(entry.DirEntry)
	}

	return dirEntries, nil
}

func (f *sharedWithMeDataSource) Open(ctx context.Context, path string) (FileReadWriterCloser, error) {
	log.Debug("SharedWithMeDS Open", fmt.Sprintf("path:%s", path))
	entry, exists := f.cache[path]
	if !exists {
		return nil, EntryNotFound
	}

	openFileInfo, err := f.service.OpenFile(ctx, path, entry.bucket, entry.dbId)
	if err != nil {
		return nil, err
	}

	return OpenSpaceFilesHandler(f.service, openFileInfo.Location, path, entry.bucket), nil
}

// CreateEntry is not supported for shared with me files.
func (f *sharedWithMeDataSource) CreateEntry(ctx context.Context, path string, mode os.FileMode) (*DirEntry, error) {
	// not allowed so just return error
	return nil, syscall.ENOTSUP
}

func (f *sharedWithMeDataSource) RenameEntry(ctx context.Context, oldPath, newPath string) error {
	// Renaming items in the shared directory is not supported
	return syscall.ENOTSUP
}

func (f *sharedWithMeDataSource) DeleteEntry(ctx context.Context, path string) error {
	// Deleting items in the shared directory is not supported
	return syscall.ENOTSUP
}

func (f *sharedWithMeDataSource) cacheResults(items []*domain.SharedDirEntry) {
	for _, item := range items {
		f.cache[item.Path] = &sharedFileEntry{
			entry:  NewDirEntryWithMode(item.DirEntry, StandardFileAccessMode),
			dbId:   item.DbID,
			bucket: item.Bucket,
		}
	}
}
