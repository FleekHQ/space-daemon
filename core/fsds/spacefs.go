package fsds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"

	"github.com/FleekHQ/space-daemon/core/space"
)

// EntryNotFound error when a directory is not found
var EntryNotFound = errors.New("Directory entry not found")

// SpaceFSDataSource is an implementation of the FSDataSource
// It interacts with the Space Service Layer to provide data
type SpaceFSDataSource struct {
	service space.Service
}

func isBaseDirectory(path string) bool {
	return path == "/"
}

func isNotExistError(err error) bool {
	// Example of current error representing file not found:
	// error: code = Unknown desc = no link named ".localized" under bafybeievqvkeo2ycggt4lino45pj3olv7yo2e6sybcmyphicejsvq2vimi[]
	return strings.Contains(err.Error(), "no link named")
}

func NewSpaceFSDataSource(service space.Service) *SpaceFSDataSource {
	return &SpaceFSDataSource{
		service: service,
	}
}

// Get returns the DirEntry information for item at path
func (d *SpaceFSDataSource) Get(ctx context.Context, path string) (*DirEntry, error) {
	log.Debug("FSDS.Get", "path="+path)
	// handle quick lookup of home directory
	if isBaseDirectory(path) {
		return d.baseDirEntry(), nil
	}

	baseName := filepath.Base(path)
	parentPath := filepath.Dir(strings.TrimRight(path, "/") + "/..")
	parentEntries, err := d.service.ListDir(ctx, parentPath)
	if err != nil {
		if isNotExistError(err) {
			return nil, syscall.ENOENT
		}
		return nil, err
	}

	log.Debug(fmt.Sprintf("Parent Entries: %+v", parentEntries))

	for _, entry := range parentEntries {
		if entry.Name == baseName {
			return NewDirEntry(entry.DirEntry), nil
		}
	}

	return nil, EntryNotFound
}

// Helper function to construct entry for the home directory
func (d *SpaceFSDataSource) baseDirEntry() *DirEntry {
	return NewDirEntry(domain.DirEntry{
		Path:        "", // filepath.Base("/"),
		IsDir:       true,
		Name:        "",
		SizeInBytes: "0",
	})
}

// GetChildren returns list of entries in a path
func (d *SpaceFSDataSource) GetChildren(ctx context.Context, path string) ([]*DirEntry, error) {
	domainEntries, err := d.service.ListDir(ctx, path)
	if err != nil {
		return nil, err
	}

	dirEntries := make([]*DirEntry, len(domainEntries))
	for i, domainEntries := range domainEntries {
		dirEntries[i] = NewDirEntry(domainEntries.DirEntry)
	}

	return dirEntries, nil
}

// Open is invoked to read the content of a file
func (d *SpaceFSDataSource) Open(ctx context.Context, path string) (ReadSeekCloser, error) {
	openFileInfo, err := d.service.OpenFile(ctx, path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(openFileInfo.Location)
	return file, nil
}

// CreateEntry creates a directory or file based on the mode at the path
func (d *SpaceFSDataSource) CreateEntry(ctx context.Context, path string, mode os.FileMode) (*DirEntry, error) {
	if mode.IsDir() {
		err := d.service.CreateFolder(ctx, path)
		if err != nil {
			return nil, err
		}
	} else {
		// TODO: Handle creating a file in service layer
	}
	return nil, nil
}
