//+build !windows

package libfuse

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"syscall"

	"github.com/FleekHQ/space-daemon/core/spacefs"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.Node = (*VFSDir)(nil)
var _ = fs.NodeRequestLookuper(&VFSDir{})
var _ = fs.HandleReadDirAller(&VFSDir{})
var _ = fs.NodeCreater(&VFSDir{})

// VFSDir represents a directory in the Virtual file system
type VFSDir struct {
	vfs    *VFS // pointer to the parent file system
	dirOps spacefs.DirOps
}

func NewVFSDir(vfs *VFS, dirOps spacefs.DirOps) *VFSDir {
	return &VFSDir{
		vfs:    vfs,
		dirOps: dirOps,
	}
}

// Attr returns fuse.Attr for the directory
func (dir *VFSDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	dirAttribute, err := dir.dirOps.Attribute()
	if err != nil {
		return err
	}

	attr.Mode = dirAttribute.Mode()

	if uid, err := strconv.Atoi(dirAttribute.Uid()); err == nil {
		attr.Uid = uint32(uid)
	}

	if gid, err := strconv.Atoi(dirAttribute.Gid()); err == nil {
		attr.Gid = uint32(gid)
	}

	return nil
}

// ReadDirAll reads all the content of a directory
// In this mirror drive case, we just return items in the mirror path
func (dir *VFSDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	dirList, err := dir.dirOps.ReadDir(ctx)
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
	entry, err := dir.vfs.fsOps.LookupPath(ctx, path)
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
		return NewVFSDir(dir.vfs, dirOps), nil
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

// Create is invoked when a new directory is to be created
// It implements the fs.NodeCreator interface
func (dir *VFSDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	path := dir.dirOps.Path()
	log.Printf("Creating a file/directory: %+v in path: %s", *req, path)
	dirEntry, err := dir.vfs.fsOps.CreateEntry(ctx, spacefs.CreateDirEntry{
		Path: fmt.Sprintf("%s%c%s", strings.TrimSuffix(path, "/"), '/', req.Name),
		Mode: req.Mode,
	})
	if err != nil {
		return nil, nil, err
	}

	if dirOps, ok := dirEntry.(spacefs.DirOps); ok {
		return NewVFSDir(dir.vfs, dirOps), nil, nil
	}

	if fileOps, ok := dirEntry.(spacefs.FileOps); ok {
		vfsFile := NewVFSFile(dir.vfs, fileOps)
		handler, err := NewVFSFileHandler(ctx, vfsFile)
		if err != nil {
			return nil, nil, err
		}

		return vfsFile, handler, nil
	}

	return nil, nil, syscall.EACCES
}
