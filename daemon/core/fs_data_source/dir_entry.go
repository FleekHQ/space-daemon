package fs_data_source

import (
	"log"
	"os"
	"time"

	format "github.com/ipfs/go-ipld-format"
)

// DirEntry implements the DirEntryOps
type DirEntry struct {
	//attr DirEntryAttribute
	path  string
	name  string
	node  format.Node
	stats *format.NodeStat
}

func NewDirEntry(path, name string, node format.Node, stats *format.NodeStat) *DirEntry {
	return &DirEntry{
		path:  path,
		name:  name,
		node:  node,
		stats: stats,
	}
}

func (d *DirEntry) Path() string {
	if d.path == "/" {
		return d.path
	}

	if d.IsDir() {
		return d.path + "/"
	}

	return d.path
}

func (d *DirEntry) Stats() *format.NodeStat {
	if d.stats != nil {
		return d.stats
	}

	stats, err := d.node.Stat()
	if err != nil {
		log.Printf("Unhandled error fetching Dir stats")
	}
	d.stats = stats
	return d.stats
}

// IsDir implement DirEntryAttribute
// And returns if the directory is a boolean or not
func (d *DirEntry) IsDir() bool {
	return d.Stats().DataSize == 2
}

// Name implements the DirEntryAttribute Interface
func (d *DirEntry) Name() string {
	return d.name
}

// Size implements the DirEntryAttribute Interface and return the size of the item
func (d *DirEntry) Size() uint64 {
	size := len(d.node.RawData())
	if size > 12 {
		// Seems there is some extra 12 bytes metadata,
		// so removing that from rawdata
		size -= 11
	}
	return uint64(size)
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
	return time.Time{}
}

// ModTime returns the modification time
func (d *DirEntry) ModTime() time.Time {
	return d.Ctime()
}
