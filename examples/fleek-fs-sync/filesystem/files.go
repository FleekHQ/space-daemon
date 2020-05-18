package filesystem

import (
	"context"
	"io"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

var _ fs.Node = (*VFSFile)(nil)
var _ = fs.NodeOpener(&VFSFile{})
var _ = fs.HandleReader(&VFSFileHandler{})
var _ = fs.HandleWriter(&VFSFileHandler{})
var _ = fs.HandleReleaser(&VFSFileHandler{})

// VFSFile represents a file in the Virtual file system
type VFSFile struct {
	vfs  *VFS // pointer to the parent file system
	path string
}

// Attr returns fuse.Attr for the directory or file
func (vfile *VFSFile) Attr(ctx context.Context, attr *fuse.Attr) error {
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

// Open create a handle responsible for reading the file and also closing the file after reading
func (vfile *VFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	path := vfile.vfs.mountPath + vfile.path
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &VFSFileHandler{
		reader: f,
	}, nil
}

// VFSFileHandler manages readings and closing access to a VFSFile
type VFSFileHandler struct {
	reader io.ReadCloser
}

// Read reads the content of the reader
// Ideally, decryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// TODO: Handle offset reading
	buf := make([]byte, req.Size)
	n, err := vfh.reader.Read(buf)
	resp.Data = buf[:n]
	return err
}

// Write writes content from request into the underly file. Keeping track of offset and all
// Ideally, encryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	// TODO
	return nil
}

// Release closes the reader on this file handler
func (vfh *VFSFileHandler) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return vfh.reader.Close()
}
