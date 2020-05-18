package filesystem

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.FS = (*VFS)(nil)

// VFS represent Virtual System
type VFS struct {
	mountPath string
	isMounted bool
}

// NewVFileSystem creates a new Virtual FileSystem object
func NewVFileSystem(mountPath string) VFS {
	return VFS{
		mountPath: mountPath,
		isMounted: false,
	}
}

// Mount mounts the file system, if it is not already mounted
func (vfs *VFS) Mount() error {
	c, err := fuse.Mount(vfs.mountPath)
	if err != nil {
		return err
	}

	if err := fs.Serve(c, vfs); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}

	return nil
}

// Root complies with the Fuse Interface that returns the Root Node of our file system
func (vfs *VFS) Root() (fs.Node, error) {
	node := &VFSDir{
		vfs:  vfs,
		path: "/",
	}
	return node, nil
}
