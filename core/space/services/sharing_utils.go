package services

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"time"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

var EmptyFileSharingInfo = domain.FileSharingInfo{}

func generateFilesSharingZip() string {
	return fmt.Sprintf("space_shared_files-%d.zip", time.Now().UnixNano())
}

// EncryptedFileWriter writes while encrypting the written bytes using
// the encryption key provided
type EncryptedFileWriter struct {
	writer io.Writer
	stream cipher.Stream
	IV     []byte
	Key    []byte
}

func newEncryptedFileWriter(writer io.Writer, key []byte) (*EncryptedFileWriter, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &EncryptedFileWriter{
		writer: writer,
		stream: cipher.NewCTR(block, FileSharingIV),
		IV:     FileSharingIV,
		Key:    key,
	}, nil
}

func (e *EncryptedFileWriter) Write(p []byte) (n int, err error) {
	e.stream.XORKeyStream(p[:], p[:])
	// To be considered: Also compute mac of encrypted hash for further verification
	// not sure how useful it could be though
	return e.writer.Write(p)
}

// EncryptedFileReader reads an encrypted stream and decrypts them from the decoded stream
// decoded bytes are then written to the writer
type EncryptedFileReader struct {
	reader io.Reader
	iv     []byte
	key    []byte
	stream cipher.Stream
}

func newEncryptedFileReader(reader io.Reader, key []byte) (*EncryptedFileReader, error) {
	iv := FileSharingIV

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &EncryptedFileReader{
		reader: reader,
		iv:     FileSharingIV,
		key:    key,
		stream: cipher.NewCTR(block, iv),
	}, nil
}

func (e *EncryptedFileReader) Read(b []byte) (int, error) {
	n, err := e.reader.Read(b)
	e.stream.XORKeyStream(b[:], b[:])
	return n, err
}
