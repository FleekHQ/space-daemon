package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/url"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	cid "github.com/ipfs/go-cid"
)

func (s *Space) GenerateFileSharingLink(
	ctx context.Context,
	path string,
	bucketName string,
) (domain.FileSharingInfo, error) {
	_, fileName := filepath.Split(path)
	key, err := s.keychain.GenerateTempKey()
	if err != nil {
		return domain.FileSharingInfo{}, err
	}

	bucket, err := s.getBucketWithFallback(ctx, bucketName)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}

	encryptedFile, err := s.createTempFileForPath(ctx, path, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer encryptedFile.Close()

	writer, err := newEncryptedFileWriter(encryptedFile, key)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}

	err = bucket.GetFile(ctx, path, writer)
	if err != nil {
		return domain.FileSharingInfo{}, errors.Wrap(err, "file encryption failed")
	}

	_, err = encryptedFile.Seek(0, 0)
	if err != nil {
		return domain.FileSharingInfo{}, errors.Wrap(err, "file encryption failed")
	}

	fileUploadResult := s.ic.AddItem(ctx, encryptedFile)
	if err := fileUploadResult.Error; err != nil {
		return domain.FileSharingInfo{}, errors.Wrap(err, "encrypted file upload failed")
	}
	encryptedFileHash := fileUploadResult.Resolved.Cid().String()

	urlQuery := url.Values{}
	urlQuery.Add("fname", fileName)
	urlQuery.Add("hash", encryptedFileHash)
	urlQuery.Add("key", writer.EncodeKey())

	return domain.FileSharingInfo{
		Bucket:            bucketName,
		Path:              path,
		SharedFileCid:     encryptedFileHash,
		SharedFileKey:     writer.EncodeKey(),
		SpaceDownloadLink: "https://space.storage/files/share?" + urlQuery.Encode(),
	}, nil
}

// OpenSharedFile fetched the ipfs file and decrypts it with the key. Then returns the decrypted
// files location.
func (s *Space) OpenSharedFile(ctx context.Context, hash, key, filename string) (domain.OpenFileInfo, error) {
	parsedCid, err := cid.Parse(hash)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}

	encryptedFile, err := s.ic.PullItem(ctx, parsedCid)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}
	defer encryptedFile.Close()

	decryptedFile, err := s.createTempFileForPath(ctx, filename, true)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}
	defer decryptedFile.Close()

	reader, err := newEncryptedFileReader(encryptedFile, key)
	if _, err := io.Copy(decryptedFile, reader); err != nil {
		return domain.OpenFileInfo{}, errors.Wrap(err, "decryption failed")
	}

	return domain.OpenFileInfo{
		Location: decryptedFile.Name(),
	}, nil
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

	iv := make([]byte, block.BlockSize())
	_, err = rand.Read(iv)

	return &EncryptedFileWriter{
		writer: writer,
		stream: cipher.NewCTR(block, iv),
		IV:     iv,
		Key:    key,
	}, nil
}

func (e *EncryptedFileWriter) Write(p []byte) (n int, err error) {
	e.stream.XORKeyStream(p[:], p[:])
	// To be considered: Also compute mac of encrypted hash for further verification
	// not sure how useful it could be though
	return e.writer.Write(p)
}

// EncodeKey returns a base64 hash of the IV+Key used for encryption
func (e *EncryptedFileWriter) EncodeKey() string {
	return base64.StdEncoding.EncodeToString(append(e.IV[:], e.Key[:]...))
}

// EncryptedFileReader reads an encrypted stream and decrypts them from the decoded stream
// decoded bytes are then written to the writer
type EncryptedFileReader struct {
	reader io.Reader
	iv     []byte
	key    []byte
	stream cipher.Stream
}

// Initialization vector size is 16 bytes
// this corresponds to the block size of the cipher key used in encrypted writer
var IVBlockSize = 16

func newEncryptedFileReader(reader io.Reader, encodedKey string) (*EncryptedFileReader, error) {
	encodedKeyBytes, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, err
	}

	if len(encodedKeyBytes) < IVBlockSize {
		return nil, errors.New("file key is wrong")
	}

	iv := encodedKeyBytes[:IVBlockSize]
	key := encodedKeyBytes[IVBlockSize:]

	block, err := aes.NewCipher(key)

	return &EncryptedFileReader{
		reader: reader,
		iv:     iv,
		key:    iv,
		stream: cipher.NewCTR(block, iv),
	}, nil
}

func (e *EncryptedFileReader) Read(b []byte) (int, error) {
	n, err := e.reader.Read(b)
	e.stream.XORKeyStream(b[:], b[:])
	return n, err
}
