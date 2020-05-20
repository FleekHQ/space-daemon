package libfuse

import (
	"context"
	"io"
	"log"
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
	path := vfile.vfs.mirrorPath + vfile.path
	log.Printf("Getting File Attr %s", path)
	osFile, err := os.Open(path)

	if err != nil {
		log.Printf("Error Getting Open File Attr %s", err.Error())
		return err
	}

	fileStat, err := osFile.Stat()
	if err != nil {
		log.Printf("Error Getting File State %s ", err.Error())
		return err
	}

	attr.Size = uint64(fileStat.Size())
	attr.Mode = fileStat.Mode()
	attr.Mtime = fileStat.ModTime()
	attr.Ctime = fileStat.ModTime()
	attr.Crtime = fileStat.ModTime()

	log.Printf("Successful File Attr %s : %+v", path, attr)

	return nil
}

// Open create a handle responsible for reading the file and also closing the file after reading
func (vfile *VFSFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	path := vfile.vfs.mirrorPath + vfile.path
	log.Printf("Attempting to open a File %s", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &VFSFileHandler{
		path:   path,
		reader: f,
	}, nil
}

// VFSFileHandler manages readings and closing access to a VFSFile
type VFSFileHandler struct {
	path   string
	reader io.ReadCloser
}

// Read reads the content of the reader
// Ideally, decryption of the content of the file should be happening here
func (vfh *VFSFileHandler) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// TODO: Handle offset reading
	log.Printf("Reading content of file %s", vfh.path)
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
