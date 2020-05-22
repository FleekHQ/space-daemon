package spacestore

import (
	"github.com/ipfs/go-unixfs/io"
)

type IPFSFileHandler struct {
	isRead bool
	reader io.ReadSeekCloser
}

func NewIPFSReadHandler(reader io.ReadSeekCloser) *IPFSFileHandler {
	return &IPFSFileHandler{
		isRead: true,
		reader: reader,
	}
}

func (i IPFSFileHandler) Read(p []byte) (n int, err error) {
	return i.reader.Read(p)
}

func (i IPFSFileHandler) Write(p []byte) (n int, err error) {
	panic("Write not implemented")

}

func (i IPFSFileHandler) Seek(offset int64, whence int) (int64, error) {
	return i.reader.Seek(offset, whence)
}

func (i IPFSFileHandler) Close() error {
	return i.reader.Close()
}
