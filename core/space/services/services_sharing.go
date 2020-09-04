package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"

	"github.com/textileio/dcrypto"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/ipfs/go-cid"
)

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

	// tempFile is written from textile before encryption
	tempFile, err := s.createTempFileForPath(ctx, path, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer func() {
		tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	// encrypted file is the final encrypted file
	encryptedFile, err := s.createTempFileForPath(ctx, path, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer encryptedFile.Close()

	err = bucket.GetFile(ctx, path, tempFile)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}
	encryptedReader, err := dcrypto.NewEncrypterWithPassword(tempFile, []byte(encryptionPassword))
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}

	log.Printf("Copying encrypted file to disk")
	_, err = io.Copy(encryptedFile, encryptedReader)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}

	log.Printf("Copy successful")
	_, err = encryptedFile.Seek(0, 0)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}

	log.Printf("Uploading shared file")
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
		SpaceDownloadLink: "https://app.space.storage/files/share?" + urlQuery.Encode(),
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
	// tempFile is written from textile before encryption
	tempFile, err := s.createTempFileForPath(ctx, filename, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer func() {
		tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()

	encryptedFile, err := s.createTempFileForPath(ctx, filename, true)
	if err != nil {
		return domain.FileSharingInfo{}, err
	}
	defer encryptedFile.Close()

	zipper := zip.NewWriter(tempFile)
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

	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}
	encryptedReader, err := dcrypto.NewEncrypterWithPassword(tempFile, []byte(encryptionPassword))
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "file encryption failed")
	}

	_, err = io.Copy(encryptedFile, encryptedReader)
	if err != nil {
		return EmptyFileSharingInfo, err
	}

	_, err = encryptedFile.Seek(0, 0)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "encryption failed")
	}

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

	reader, err := dcrypto.NewDecrypterWithPassword(encryptedFile, []byte(password))
	if err != nil {
		return domain.OpenFileInfo{}, err
	}

	if _, err := io.Copy(decryptedFile, reader); err != nil {
		return domain.OpenFileInfo{}, errors.Wrap(err, "decryption failed")
	}

	return domain.OpenFileInfo{
		Location: decryptedFile.Name(),
	}, nil
}

func (s *Space) ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey) error {
	err := s.tc.ShareFilesViaPublicKey(ctx, paths, pubkeys)
	if err != nil {
		return err
	}

	for _, path := range paths {
		// this handles personal bucket since for shared-with-me files
		// the dbid will be preset
		if path.DbId == "" {
			b, err := s.tc.GetDefaultBucket(ctx)
			if err != nil {
				return err
			}

			bs, err := s.tc.GetBucket(ctx, b.Slug())
			if err != nil {
				return err
			}
			threadID, err := bs.GetThreadID(ctx)
			if err != nil {
				return err
			}

			path.DbId = utils.CastDbIDToString(*threadID)
		}

		if path.Bucket == "" {
			b, err := s.tc.GetDefaultBucket(ctx)
			if err != nil {
				return err
			}
			path.Bucket = b.Slug()
		}
	}

	for _, pk := range pubkeys {

		d := &domain.Invitation{
			ItemPaths: paths,
			// Key: TODO - get from keys thread for each file
		}

		i, err := json.Marshal(d)
		if err != nil {
			return err
		}

		b := &domain.MessageBody{
			Type: domain.INVITATION,
			Body: i,
		}

		j, err := json.Marshal(b)
		if err != nil {
			return err
		}

		_, err = s.tc.SendMessage(ctx, pk, j)
		if err != nil {
			return err
		}
	}
	return nil
}
