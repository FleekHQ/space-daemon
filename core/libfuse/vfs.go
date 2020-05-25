package libfuse

import (
	"context"
	"errors"
	"log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/FleekHQ/space-poc/core/spacefs"
)

var _ fs.FS = (*VFS)(nil)

var (
	errorNotMounted = errors.New("VFS not mounted yet")
)

// VFS represent Virtual System
type VFS struct {
	ctx             context.Context
	mountPath       string
	fsOps           spacefs.FSOps
	mountConnection *fuse.Conn
}

// NewVFileSystem creates a new Virtual FileSystem object
func NewVFileSystem(ctx context.Context, mountPath string, fsOps spacefs.FSOps) VFS {
	return VFS{
		// storing ctx here to be used in the Root request
		// as FUSE doesn't provide one there
		ctx:             ctx,
		mountPath:       mountPath,
		fsOps:           fsOps,
		mountConnection: nil,
	}
}

// Mount mounts the file system, if it is not already mounted
func (vfs *VFS) Mount(fsName string) error {
	c, err := fuse.Mount(
		vfs.mountPath,
		fuse.FSName(fsName),
		fuse.VolumeName(fsName),
		fuse.NoAppleDouble(),
		fuse.NoAppleXattr(),
	)
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
	rootDirEntry, err := vfs.fsOps.Root(vfs.ctx)
	if err != nil {
		return nil, err
	}

	rootDir, ok := rootDirEntry.(spacefs.DirOps)
	if !ok {
		log.Fatal("Root directory is not a spacefs.DirOps")
	}

	log.Printf("Root Dir: %+v", rootDir)

	node := &VFSDir{
		vfs:    vfs,
		dirOps: rootDir,
	}
	return node, nil
}
