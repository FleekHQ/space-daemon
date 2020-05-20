package libfuse

import (
	"context"
	"syscall"

	"github.com/FleekHQ/space-poc/examples/fleek-fs-sync/spacefs"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.Node = (*VFSDir)(nil)
var _ = fs.NodeRequestLookuper(&VFSDir{})
var _ = fs.HandleReadDirAller(&VFSDir{})

// VFSDir represents a directory in the Virtual file system
type VFSDir struct {
	vfs    *VFS // pointer to the parent file system
	dirOps spacefs.DirOps
}

// Attr returns fuse.Attr for the directory
func (dir *VFSDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	dirAttribute, err := dir.dirOps.Attribute()
	if err != nil {
		return err
	}

	attr.Mode = dirAttribute.Mode()
	// attr.Mode = os.ModeDir | 0755
	return nil
}

// ReadDirAll reads all the content of a directory
// In this mirror drive case, we just return items in the mirror path
func (dir *VFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dirList, err := dir.dirOps.ReadDir()
	if err != nil {
		return nil, err
	}

	var res []fuse.Dirent
	for _, dirEntry := range dirList {
		entryAttribute, err := dirEntry.Attribute()
		if err != nil {
			return nil, err
		}

		entry := fuse.Dirent{
			Name: entryAttribute.Name(),
		}
		if entryAttribute.IsDir() {
			entry.Type = fuse.DT_Dir
		} else {
			entry.Type = fuse.DT_File
		}

		res = append(res, entry)
	}

	return res, nil
}

// Lookup finds entry Node within a directory
// Seems to be called when not enough information is gotten from the ReadDirAll
func (dir *VFSDir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	path := dir.dirOps.Path() + req.Name
	entry, err := dir.vfs.fsOps.LookupPath(path)
	if err != nil {
		return nil, err
	}

	entryAttribute, err := entry.Attribute()
	if err != nil {
		return nil, err
	}

	if entryAttribute.IsDir() {
		dirOps, ok := entry.(spacefs.DirOps)
		if !ok {
			// TODO: Return a better syscall error
			return nil, syscall.ENOENT
		}
		return &VFSDir{
			vfs:    dir.vfs,
			dirOps: dirOps,
		}, nil
	}

	fileOps, ok := entry.(spacefs.FileOps)
	if !ok {
		// TODO: Return a better syscall error
		return nil, syscall.ENOENT
	}

	return &VFSFile{
		vfs:     dir.vfs,
		fileOps: fileOps,
	}, nil
}
