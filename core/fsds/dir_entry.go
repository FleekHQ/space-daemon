package fsds

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

var StandardFileAccessMode os.FileMode = 0600   // -rw-------
var StandardDirAccessMode = os.ModeDir | 0700   // drwx------
var RestrictedDirAccessMode = os.ModeDir | 0500 // dr-x------ only allow reading and opening directory for user

// DirEntry implements the DirEntryOps
type DirEntry struct {
	entry domain.DirEntry
	mode  os.FileMode
}

func NewDirEntry(entry domain.DirEntry) *DirEntry {
	return NewDirEntryWithMode(entry, 0)
}

func NewDirEntryWithMode(entry domain.DirEntry, mode os.FileMode) *DirEntry {
	return &DirEntry{
		entry: entry,
		mode:  mode,
	}
}

func (d *DirEntry) Path() string {
	if d.IsDir() {
		return fmt.Sprintf(
			"%s%c",
			strings.TrimRight(d.entry.Path, fmt.Sprintf("%c", os.PathSeparator)),
			os.PathSeparator,
		)
	}

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
	if d.mode != 0 {
		return d.mode
	}

	if d.IsDir() {
		return StandardDirAccessMode
	}

	return StandardFileAccessMode
}

func (d *DirEntry) Uid() string {
	// for now return id of currently logged in user
	currentUser, err := user.Current()
	if err != nil {
		log.Error("Uid() Error fetching user.Current()", err)
		return "0"
	}

	return currentUser.Uid
}

func (d *DirEntry) Gid() string {
	currentUser, err := user.Current()
	if err != nil {
		log.Error("Gid() Error fetching user.Current()", err)
		return "0"
	}

	return currentUser.Gid
}

// Ctime implements the DirEntryAttribute Interface
// It returns the time the directory was created
func (d *DirEntry) Ctime() time.Time {
	t, err := time.Parse(time.RFC3339, d.entry.Created)

	if err != nil {
		log.Error("Error parsing direntry created time", err)
		return time.Time{}
	}

	return t
}

// ModTime returns the modification time
func (d *DirEntry) ModTime() time.Time {
	t, err := time.Parse(time.RFC3339, d.entry.Updated)

	if err != nil {
		log.Error("Error parsing direntry updated time", err)
		return time.Time{}
	}

	return t
}
