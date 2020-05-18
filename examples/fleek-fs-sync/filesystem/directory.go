package filesystem

import (
	"context"
	"io/ioutil"
	"log"
	"os"

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
	// TODO: Handle isFile
	// return fuse.Attr{
	// 	Size:   f.UncompressedSize64,
	// 	Mode:   f.Mode(),
	// 	Mtime:  f.ModTime(),
	// 	Ctime:  f.ModTime(),
	// 	Crtime: f.ModTime(),
	// }
	attr.Mode = os.ModeDir | 0755
	return nil
}

// ReadDirAll reads all the content of a directory
func (dir *VFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dirItems, err := ioutil.ReadDir(dir.vfs.mountPath)
	if err != nil {
		return nil, err
	}

	var res []fuse.Dirent
	for _, dirItem := range dirItems {
		entry := fuse.Dirent{
			Name: dirItem.Name(),
		}

		if dirItem.IsDir() {
			entry.Type = fuse.DT_Dir
		}

		res = append(res, entry)
	}

	return res, nil
}

// Lookup finds entry Nodes withing a directory
func (dir *VFSDir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	path := dir.vfs.mountPath + dir.path + req.Name
	log.Printf("Looking up directory entry by path: %s", path)
	// ideal logic would be:
	// - Fetch server entry (synced) and merge with local entry
	// - Local files not on the server are not synced
	// - perhaps maintain a local map of sycing directory and folders and use that instead
	//
	// for now just read the actual os file directory
	dirItems, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, dirItem := range dirItems {
		if dirItem.Name() == req.Name {

			if dirItem.IsDir() {
				return &VFSDir{
					vfs:  dir.vfs,
					path: dir.path + "/" + req.Name,
				}, nil
			}

			return &VFSFile{
				vfs:  dir.vfs,
				path: dir.path + "/" + req.Name,
			}, nil

		}
	}
	return nil, fuse.ENOENT
}
