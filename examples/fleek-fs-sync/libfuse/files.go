package libfuse

import (
	"context"
	"log"

	"github.com/FleekHQ/space-poc/examples/fleek-fs-sync/spacefs"

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
	vfs     *VFS // pointer to the parent file system
	fileOps spacefs.FileOps
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

// Open create a handle responsible for reading the file and also closing the file after reading
func (vfile *VFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	path := vfile.fileOps.Path()
	reader, err := vfile.fileOps.Open()
	if err != nil {
		return nil, err
	}

	return &VFSFileHandler{
		path:   path,
		readWriteOps: reader,
	}, nil
}

// VFSFileHandler manages readings and closing access to a VFSFile
type VFSFileHandler struct {
	path         string
	readWriteOps spacefs.FileHandler
}

// Read reads the content of the reader
// Ideally, decryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	log.Printf("Reading content of file %s", vfh.path)
	err := vfh.readWriteOps.SetReadOffset(req.Offset)
	if err != nil {
		return err
	}

	buf := make([]byte, req.Size)
	n, err := vfh.readWriteOps.Read(buf)
	if err != nil {
		return err
	}

	resp.Data = buf[:n]
	return nil
}

// Write writes content from request into the underly file. Keeping track of offset and all
// Ideally, encryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	log.Printf("Reading content to file %s", vfh.path)
	err := vfh.readWriteOps.SetWriteOffset(req.Offset)
	if err != nil {
		return err
	}

	_, err = vfh.readWriteOps.Write(req.Data)
	if err != nil {
		return err
	}

	return nil
}

// Release closes the reader on this file handler
func (vfh *VFSFileHandler) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return vfh.readWriteOps.Close()
}
