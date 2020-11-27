package fsds

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FleekHQ/space-daemon/core/space/domain"

	"github.com/FleekHQ/space-daemon/core/space"
	"github.com/FleekHQ/space-daemon/log"
)

// Provides content for the 'Files' content managed by the space user
// Requests for items in this path are dispatched to this datasource from SpaceFSDataSource
type filesDataSource struct {
	service space.Service
}

// Maybe consider caching at the level of SpaceFSDataSource when results are returned from top level file
func (f *filesDataSource) Get(ctx context.Context, path string) (*DirEntry, error) {
	baseName := filepath.Base(path)
	if isBaseDirectory(path) || path == "" {
		return NewDirEntryWithMode(domain.DirEntry{
			Path:    path,
			IsDir:   true,
			Name:    baseName,
			Created: time.Now().Format(time.RFC3339),
			Updated: time.Now().Format(time.RFC3339),
		}, RestrictedDirAccessMode), nil
	}

	log.Debug("FileDS Get", fmt.Sprintf("path:%s", path))

	itemsInParent, err := f.service.ListDir(ctx, path, DefaultBucketName, true)
	if err != nil {
		if !isNotExistError(err) {
			return nil, EntryNotFound
		}

		return nil, err
	}

	// If space service.ListDir on path is empty, then it is a file
	// if it is not empty, then it is a directory
	if len(itemsInParent) != 0 {
		// is a directory because space directory cannot be empty (must at least contain a .keep file)
		return NewDirEntry(domain.DirEntry{
			Path:    path,
			IsDir:   true,
			Name:    baseName,
			Created: time.Now().Format(time.RFC3339),
			Updated: time.Now().Format(time.RFC3339),
		}), nil
	}

	// OpenFile to get Size information of file
	// TODO: Verify service.OpenFile() logic to ensure that multiple open file doesn't recreate multiple local copies for the same file without cleanup
	r, err := f.service.OpenFile(ctx, path, DefaultBucketName, "")
	if err != nil {
		//if isNotExistError(err) {
		return nil, EntryNotFound
		//}
	}

	fileStat, err := os.Stat(r.Location)
	if err != nil {
		return nil, err
	}

	//is a file, so return file entry
	return NewDirEntry(domain.DirEntry{
		Path:          path,
		IsDir:         false,
		Name:          baseName,
		SizeInBytes:   fmt.Sprintf("%d", fileStat.Size()),
		Created:       time.Now().Format(time.RFC3339),
		Updated:       time.Now().Format(time.RFC3339),
		FileExtension: filepath.Ext(path),
	}), nil
}

func (f *filesDataSource) GetChildren(ctx context.Context, path string) ([]*DirEntry, error) {
	log.Debug("FileDS GetChildren", fmt.Sprintf("path:%s", path))
	domainEntries, err := f.service.ListDir(ctx, path, DefaultBucketName, true)
	if err != nil {
		return nil, err
	}

	dirEntries := make([]*DirEntry, len(domainEntries))
	for i, domainEntries := range domainEntries {
		dirEntries[i] = NewDirEntry(domainEntries.DirEntry)
	}

	return dirEntries, nil
}

func (f *filesDataSource) Open(ctx context.Context, path string) (FileReadWriterCloser, error) {
	log.Debug("FileDS Open", fmt.Sprintf("path:%s", path))
	openFileInfo, err := f.service.OpenFile(ctx, path, DefaultBucketName, "")
	if err != nil {
		return nil, err
	}

	return OpenSpaceFilesHandler(f.service, openFileInfo.Location, path, DefaultBucketName), nil
}

// Create the entry at the specified path and return a DirEntry representing it.
// The DirEntry would be used to write/copy the items necessary at the point
func (f *filesDataSource) CreateEntry(ctx context.Context, path string, mode os.FileMode) (*DirEntry, error) {
	log.Debug("FileDS CreateEntry", fmt.Sprintf("path:%s", path), fmt.Sprintf("mode:%v", mode))
	entryName := filepath.Base(path)
	parentDir := filepath.Dir(path)

	if mode.IsDir() {
		err := f.service.CreateFolder(ctx, path, DefaultBucketName)
		if err != nil {
			return nil, err
		}

		return NewDirEntry(domain.DirEntry{
			Path:    path,
			IsDir:   true,
			Name:    entryName,
			Created: time.Now().Format(time.RFC3339),
			Updated: time.Now().Format(time.RFC3339),
		}), nil
	}

	// create an empty file to uploaded to the specified path
	newFilePath := filepath.Join(os.TempDir(), path)
	_ = os.MkdirAll(filepath.Dir(newFilePath), os.ModePerm)
	err := ioutil.WriteFile(newFilePath, []byte{}, mode)
	if err != nil {
		log.Error("Error creating empty file", err, "newFilePath:"+newFilePath)
		return nil, err
	}

	waitChan, _, err := f.service.AddItems(
		ctx,
		[]string{
			newFilePath,
		},
		parentDir,
		DefaultBucketName,
	)
	if err != nil {
		return nil, err
	}

	r := <-waitChan
	if r.Error != nil {
		log.Error("FileDS Failed to upload file", r.Error)
		return nil, err
	}

	return NewDirEntry(domain.DirEntry{
		Path:    path,
		IsDir:   false,
		Name:    entryName,
		Created: time.Now().Format(time.RFC3339),
		Updated: time.Now().Format(time.RFC3339),
	}), nil
}

// RenameEntry for now only supports renaming of empty folders
// Depending on user request and textile support, non-empty folders and file renames will be supported
func (f *filesDataSource) RenameEntry(ctx context.Context, oldPath, newPath string) error {
	log.Debug("FileDS RenameEntry", "oldPath:"+oldPath, "newPath:"+newPath)
	entry, err := f.Get(ctx, oldPath)
	if err != nil {
		return err
	}

	if !entry.IsDir() {
		log.Warn("FileDS trying to rename an entry that is not a directory")
		return syscall.ENOTSUP
	}

	childEntries, err := f.GetChildren(ctx, oldPath)
	if err != nil {
		log.Error("failed to get children of old path", err, "oldPath:"+oldPath)
		return err
	}

	if len(childEntries) != 0 && !areAllEntriesHidden(childEntries) {
		log.Warn("FileDS renaming directory that is not empty")
		// folder is not empty, so just error out
		return syscall.ENOTSUP
		// in the future, we should do a recursive copy to the newPath and then delete old path
	}

	if err = f.service.CreateFolder(ctx, newPath, DefaultBucketName); err != nil {
		log.Error("failed to new path create folder", err, "newPath:"+newPath)
		return syscall.ENOTSUP
	}

	return f.service.RemoveDirOrFile(ctx, oldPath, DefaultBucketName)
}

func areAllEntriesHidden(entries []*DirEntry) bool {
	for _, entry := range entries {
		baseName := filepath.Base(entry.Name())
		if !blackListedDirEntryNames[baseName] {
			return false
		}
	}
	return true
}

func (f *filesDataSource) DeleteEntry(ctx context.Context, path string) error {
	log.Debug("FileDS DeletEntry", "path:"+path)
	return f.service.RemoveDirOrFile(ctx, path, DefaultBucketName)
}
