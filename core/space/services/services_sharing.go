package services

import (
	"archive/zip"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/ipfs/go-cid"
)

var FileSharingPbkdfSalt, _ = hex.DecodeString("372e33a7d44242202b85e9cee132e0cc4b91b9a1b77a4ea82becb66ae7713dbd")

// Initialization vector size is 16 bytes
var FileSharingIV, _ = hex.DecodeString("81ff6d0c33a6f13e363c34c5ceddf529 ")

func (s *Space) GenerateFileSharingLink(
	ctx context.Context,
	encryptionPassword string,
	path string,
	bucketName string,
) (domain.FileSharingInfo, error) {
	_, fileName := filepath.Split(path)

	bucket, err := s.getBucketWithFallback(ctx, bucketName)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}

	encryptedFile, err := s.createTempFileForPath(ctx, path, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer encryptedFile.Close()

	key := s.keychain.GeneratePasswordBasedKey(encryptionPassword, FileSharingPbkdfSalt)
	writer, err := newEncryptedFileWriter(encryptedFile, key)
	if err != nil {
		return EmptyFileSharingInfo, err
	}

	err = bucket.GetFile(ctx, path, writer)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}

	_, err = encryptedFile.Seek(0, 0)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}

	return s.uploadSharedFileToIpfs(
		ctx,
		encryptionPassword,
		encryptedFile,
		fileName,
		bucketName,
	)
}

func (s *Space) uploadSharedFileToIpfs(
	ctx context.Context,
	password string,
	sharedContent io.Reader,
	fileName string,
	bucketName string,
) (domain.FileSharingInfo, error) {
	fileUploadResult := s.ic.AddItem(ctx, sharedContent)
	if err := fileUploadResult.Error; err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "encrypted file upload failed")
	}
	encryptedFileHash := fileUploadResult.Resolved.Cid().String()

	urlQuery := url.Values{}
	urlQuery.Add("fname", fileName)
	urlQuery.Add("hash", encryptedFileHash)

	return domain.FileSharingInfo{
		Bucket:            bucketName,
		SharedFileCid:     encryptedFileHash,
		SharedFileKey:     password,
		SpaceDownloadLink: "https://space.storage/files/share?" + urlQuery.Encode(),
	}, nil
}

// GenerateFilesSharingLink zips multiple files together
func (s *Space) GenerateFilesSharingLink(ctx context.Context, encryptionPassword string, paths []string, bucketName string) (domain.FileSharingInfo, error) {
	if len(paths) == 0 {
		return EmptyFileSharingInfo, errors.New("no file passed to share link")
	}
	if len(paths) == 1 {
		return s.GenerateFileSharingLink(ctx, encryptionPassword, paths[0], bucketName)
	}

	bucket, err := s.getBucketWithFallback(ctx, bucketName)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}

	// create zip file output
	filename := generateFilesSharingZip()
	encryptedFile, err := s.createTempFileForPath(ctx, filename, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer encryptedFile.Close()

	key := s.keychain.GeneratePasswordBasedKey(encryptionPassword, FileSharingPbkdfSalt)
	writer, err := newEncryptedFileWriter(encryptedFile, key)
	if err != nil {
		return EmptyFileSharingInfo, err
	}

	zipper := zip.NewWriter(writer)
	// write each file to zip
	for _, path := range paths {
		_, fileName := filepath.Split(path)
		writer, err := zipper.Create(fileName)
		if err != nil {
			return EmptyFileSharingInfo, errors.Wrap(err, fmt.Sprintf("failed to compress item: %s", path))
		}

		err = bucket.GetFile(ctx, path, writer)
		if err != nil {
			return EmptyFileSharingInfo, errors.Wrap(err, fmt.Sprintf("failed to compress item: %s", path))
		}
	}

	err = zipper.Close()
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "creating compressed file failed")
	}

	encryptedFile.Seek(0, 0)
	return s.uploadSharedFileToIpfs(
		ctx,
		encryptionPassword,
		encryptedFile,
		filename,
		bucketName,
	)
}

// OpenSharedFile fetched the ipfs file and decrypts it with the key. Then returns the decrypted
// files location.
func (s *Space) OpenSharedFile(ctx context.Context, hash, password, filename string) (domain.OpenFileInfo, error) {
	key := s.keychain.GeneratePasswordBasedKey(password, FileSharingPbkdfSalt)
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
