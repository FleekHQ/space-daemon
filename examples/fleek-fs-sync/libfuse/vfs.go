package libfuse

import (
	"errors"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.FS = (*VFS)(nil)

var (
	errorNotMounted = errors.New("VFS not mounted yet")
)

// VFS represent Virtual System
type VFS struct {
	mountPath       string
	mirrorPath      string
	mountConnection *fuse.Conn
}

// NewVFileSystem creates a new Virtual FileSystem object
func NewVFileSystem(mountPath, mirrorPath string) VFS {
	return VFS{
		mountPath:       mountPath,
		mirrorPath:      mirrorPath,
		mountConnection: nil,
	}
}

// Mount mounts the file system, if it is not already mounted
// This is a blocking operation
func (vfs *VFS) Mount() error {
	c, err := fuse.Mount(vfs.mountPath)
	if err != nil {
		return err
	}

	vfs.mountConnection = c
	return nil
}

// IsMounted returns true if the vfs still has a valid connection to the mounted path
func (vfs *VFS) IsMounted() bool {
	return vfs.mountConnection != nil
}

// Serve start the FUSE server that handles requests from the mounted connection
// This is a blocking operation
func (vfs *VFS) Serve() error {
	if !vfs.IsMounted() {
		return errorNotMounted
	}

	if err := fs.Serve(vfs.mountConnection, vfs); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-vfs.mountConnection.Ready
	if err := vfs.mountConnection.MountError; err != nil {
		return err
	}

	return nil
}

// UnMount closes connection
func (vfs *VFS) Unmount() error {
	if !vfs.IsMounted() {
		return errorNotMounted
	}

	err := vfs.mountConnection.Close()
	return err
}

// Root complies with the Fuse Interface that returns the Root Node of our file system
func (vfs *VFS) Root() (fs.Node, error) {
	node := &VFSDir{
		vfs:  vfs,
		path: "/",
	}
	return node, nil
}
