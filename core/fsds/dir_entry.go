package fsds

import (
	"os"
	"strconv"
	"time"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

// DirEntry implements the DirEntryOps
type DirEntry struct {
	entry domain.DirEntry
}

func NewDirEntry(entry domain.DirEntry) *DirEntry {
	return &DirEntry{
		entry: entry,
	}
}

func (d *DirEntry) Path() string {
	return d.entry.Path
}

// IsDir implement DirEntryAttribute
// And returns if the directory is a boolean or not
func (d *DirEntry) IsDir() bool {
	return d.entry.IsDir
}

// Name implements the DirEntryAttribute Interface
func (d *DirEntry) Name() string {
	return d.entry.Name
}

// Size implements the DirEntryAttribute Interface and return the size of the item
func (d *DirEntry) Size() uint64 {
	intSize, err := strconv.ParseUint(d.entry.SizeInBytes, 10, 64)
	if err != nil {
		log.Error("Error getting direntry size", err)
		// error, so returning 0 in the meantime
		return 0
	}
	return intSize
}

// Mode implements the DirEntryAttribute Interface
// Currently if it is a file, returns all access permission 0766
// but ideally should restrict the permission if owner is not the same as file
func (d *DirEntry) Mode() os.FileMode {
	if d.IsDir() {
		return os.ModeDir
	}

	return 0766 // -rwxrw-rw-
}

// Ctime implements the DirEntryAttribute Interface
// It returns the time the directory was created
func (d *DirEntry) Ctime() time.Time {
	layout := "2006-01-02T15:04:05.000Z"
	t, err := time.Parse(layout, d.entry.Created)

	if err != nil {
		log.Error("Error parsing direntry created time", err)
		return time.Time{}
	}

	return t
}

// ModTime returns the modification time
func (d *DirEntry) ModTime() time.Time {
	layout := "2006-01-02T15:04:05.000Z"
	t, err := time.Parse(layout, d.entry.Updated)

	if err != nil {
		log.Error("Error parsing direntry updated time", err)
		return time.Time{}
	}

	return t
}
