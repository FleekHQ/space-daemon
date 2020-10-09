package fsds

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/FleekHQ/space-daemon/core/fsds/files"

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

	return files.OpenSpaceFilesHandler(f.service, openFileInfo.Location, path, DefaultBucketName)
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
	newFilePath := fmt.Sprintf("%s%s", os.TempDir(), path)
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
