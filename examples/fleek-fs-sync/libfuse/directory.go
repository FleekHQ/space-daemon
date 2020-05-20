package libfuse

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.Node = (*VFSDir)(nil)
var _ = fs.NodeRequestLookuper(&VFSDir{})
var _ = fs.HandleReadDirAller(&VFSDir{})

// VFSDir represents a directory in the Virtual file system
type VFSDir struct {
	vfs  *VFS // pointer to the parent file system
	path string
}

// Attr returns fuse.Attr for the directory
func (dir *VFSDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = os.ModeDir | 0755
	log.Printf("Attr of dir %s is %+v", dir.path, attr)
	return nil
}

// ReadDirAll reads all the content of a directory
// In this mirror drive case, we just return items in the mirror path
func (dir *VFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	path := dir.vfs.mirrorPath + dir.path
	log.Printf("Directory List started %s", path)
	dirItems, err := ioutil.ReadDir(path)
	if err != nil {
		log.Printf("Error readdirAll %s", path)
		return nil, err
	}

	var res []fuse.Dirent
	for _, dirItem := range dirItems {
		entry := fuse.Dirent{
			Name: dirItem.Name(),
		}

		if dirItem.IsDir() {
			entry.Type = fuse.DT_Dir
		} else {
			// assume it is a file in this case, but not always the case
			entry.Type = fuse.DT_File
		}

		res = append(res, entry)
	}

	log.Printf("Directory list result %s : %+v", path, res)
	return res, nil
}

// Lookup finds entry Node within a directory
// Seems to be called when not enough information is gotten from the ReadDirAll
func (dir *VFSDir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	path := dir.vfs.mirrorPath + dir.path + req.Name
	log.Printf("Looking up directory entry by path: %s", path)
	// ideal logic would be:
	// - Fetch server entry (synced) and merge with local entry
	// - Local files not on the server are not synced
	// - perhaps maintain a local map of sycing directory and folders and use that instead
	//
	// for now just read the actual os file directory
	osFile, err := os.Open(path)

	if err != nil {
		log.Printf("Error looking up directory %s : %s", path, err.Error())
		if os.IsNotExist(err) {
			return nil, syscall.ENOENT
		}
		return nil, err
	}

	fileStat, err := osFile.Stat()
	if err != nil {
		log.Printf("Error getting file/directory state %s : %s", path, err.Error())
		return nil, err
	}

	if fileStat.IsDir() {
		log.Printf("Lookup %s is Directory", path)
		return &VFSDir{
			vfs:  dir.vfs,
			path: dir.path + req.Name + "/",
		}, nil
	}

	log.Printf("Lookup %s is File", path)
	return &VFSFile{
		vfs:  dir.vfs,
		path: dir.path + req.Name,
	}, nil
}
