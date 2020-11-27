//+build !windows

package libfuse

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"syscall"

	"github.com/FleekHQ/space-daemon/core/spacefs"
	"github.com/FleekHQ/space-daemon/log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var (
	_ fs.Node         = (*VFSDir)(nil)
	_ fs.NodeAccesser = (*VFSDir)(nil)
	_                 = fs.NodeRequestLookuper(&VFSDir{})
	_                 = fs.HandleReadDirAller(&VFSDir{})
	_                 = fs.NodeCreater(&VFSDir{})
	_                 = fs.NodeMkdirer(&VFSDir{})
	_                 = fs.NodeRenamer(&VFSDir{})
	_                 = fs.NodeRemover(&VFSDir{})
)

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
	dirAttribute, err := dir.dirOps.Attribute(ctx)
	if err != nil {
		return err
	}

	attr.Mode = dirAttribute.Mode()
	attr.Uid = dirAttribute.Uid()
	attr.Gid = dirAttribute.Gid()

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
		entryAttribute, err := dirEntry.Attribute(ctx)
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
	//log.Debug("VFSDir.Lookup", "name:"+req.Name)

	path := dir.dirOps.Path() + req.Name
	entry, err := dir.vfs.fsOps.LookupPath(ctx, path)
	if err != nil {
		return nil, err
	}

	entryAttribute, err := entry.Attribute(ctx)
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

// Mkdir implements the fs.NodeMkdirer interface
func (dir *VFSDir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	path := dir.dirOps.Path()
	log.Debug(fmt.Sprintf("Mkdir a file/directory: %+v with name %s, in path: %s", *req, req.Name, path))
	dirEntry, err := dir.vfs.fsOps.CreateEntry(ctx, spacefs.CreateDirEntry{
		Path: fmt.Sprintf("%s%c%s", strings.TrimSuffix(path, "/"), '/', req.Name),
		Mode: req.Mode,
	})
	if err != nil {
		return nil, err
	}

	if dirOps, ok := dirEntry.(spacefs.DirOps); ok {
		return NewVFSDir(dir.vfs, dirOps), nil
	}

	log.Error("should not happen", errors.New("created directory is not a directory"))
	return nil, fuse.ENOTSUP
}

// Rename implements the fs.NodeRenamer
// Rename is only implemented for VFSDir and not VFSFile, because we currently don't support renaming files
// and rename on fsOps should only work empty folders.
func (dir *VFSDir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	parentPath := dir.dirOps.Path()
	log.Debug("Renaming node", "oldName:"+req.OldName, "newName:"+req.NewName, "parentPath:"+parentPath)

	return dir.vfs.fsOps.RenameEntry(ctx, spacefs.RenameDirEntry{
		OldPath: path.Join(parentPath, req.OldName),
		NewPath: path.Join(parentPath, req.NewName),
	})
}

// Remove implements the fs.NodeRemover
func (dir *VFSDir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	parentPath := dir.dirOps.Path()
	return dir.vfs.fsOps.DeleteEntry(ctx, path.Join(parentPath, req.Name))
}

func (dir *VFSDir) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return nil
}
