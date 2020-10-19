//+build !windows

package libfuse

import (
	"context"
	"log"
	"os"
	"syscall"

	"github.com/FleekHQ/space-daemon/core/spacefs"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.Node = (*VFSFile)(nil)
var _ = fs.NodeAccesser(&VFSFile{})
var _ = fs.NodeOpener(&VFSFile{})
var _ = fs.HandleReader(&VFSFileHandler{})
var _ = fs.HandleWriter(&VFSFileHandler{})
var _ = fs.HandleReleaser(&VFSFileHandler{})

// VFSFile represents a file in the Virtual file system
type VFSFile struct {
	vfs     *VFS // pointer to the parent file system
	fileOps spacefs.FileOps
}

func NewVFSFile(vfs *VFS, fileOps spacefs.FileOps) *VFSFile {
	return &VFSFile{
		vfs:     vfs,
		fileOps: fileOps,
	}
}

// Attr returns fuse.Attr for the directory or file
func (vfile *VFSFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	path := vfile.fileOps.Path()
	log.Printf("Getting File Attr %s", path)
	fileAttribute, err := vfile.fileOps.Attribute()
	if err != nil {
		log.Printf("Error Getting Open File Attr %s", err.Error())
		return err
	}

	attr.Size = fileAttribute.Size()
	attr.Mode = fileAttribute.Mode()
	attr.Mtime = fileAttribute.ModTime()
	attr.Ctime = fileAttribute.Ctime()
	attr.Crtime = fileAttribute.Ctime()

	log.Printf("Successful File Attr %s : %+v", path, attr)

	return nil
}

// Access implements the fs.NodeAccesser interface for File. This is necessary
// for macOS to correctly identify plaintext files as plaintext. If not
// implemented, bazil-fuse returns a nil error for every call, so when macOS
// checks for executable bit using Access (instead of Attr!), it gets a
// success, which makes it think the file is executable, yielding a "Unix
// executable" UTI.
func (vfile *VFSFile) Access(ctx context.Context, r *fuse.AccessRequest) (err error) {
	if int(r.Uid) != os.Getuid() &&
		// Finder likes to use UID 0 for some operations. osxfuse already allows
		// ACCESS and GETXATTR requests from root to go through. This allows root
		// in ACCESS handler.
		int(r.Uid) != 0 {
		// short path: not accessible by anybody other than root or the current user
		return syscall.EPERM
	}

	if r.Mask&03 == 0 {
		// Since we only check for w and x bits, we can return nil early here.
		return nil
	}

	// check is executable mask enable
	if r.Mask&01 != 0 {
		_, err := vfile.fileOps.Attribute()
		if err != nil {
			return err
		}
		// for now always return permission error for executable calls
		// we are not supporting executable at the moment
		return syscall.EPERM
	}

	return nil
}

// Open create a handle responsible for reading the file and also closing the file after reading
func (vfile *VFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	log.Printf("Opening content of file %s", vfile.fileOps.Path())
	return NewVFSFileHandler(ctx, vfile)
}

// VFSFileHandler manages readings and closing access to a VFSFile
type VFSFileHandler struct {
	path         string
	readWriteOps spacefs.FileHandler
}

func NewVFSFileHandler(ctx context.Context, vfile *VFSFile) (*VFSFileHandler, error) {
	readWriteOps, err := vfile.fileOps.Open(ctx, spacefs.ReadMode)
	if err != nil {
		return nil, err
	}

	return &VFSFileHandler{
		path:         vfile.fileOps.Path(),
		readWriteOps: readWriteOps,
	}, nil
}

// Read reads the content of the reader
// Ideally, decryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	log.Printf("Reading content of file %s, and size: %d", vfh.path, req.Size)
	buf := make([]byte, req.Size)
	n, err := vfh.readWriteOps.Read(ctx, buf, req.Offset)
	if err != nil {
		log.Printf("Reading error: %s", err.Error())
		return err
	}

	resp.Data = buf[:n]
	return nil
}

// Write writes content from request into the underlying file. Keeping track of offset and all
// Ideally, encryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Printf("Writing content to file %s", vfh.path)
	n, err := vfh.readWriteOps.Write(ctx, req.Data, req.Offset)
	if err != nil {
		log.Printf("Writing error: %s", err.Error())
		return err
	}

	resp.Size = n
	return nil
}

// Release closes the reader on this file handler
func (vfh *VFSFileHandler) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return vfh.readWriteOps.Close(ctx)
}
