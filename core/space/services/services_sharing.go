package services

import (
	"archive/zip"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	t "github.com/FleekHQ/space-daemon/core/textile"
	"github.com/ipfs/go-cid"
	"github.com/textileio/dcrypto"
)

func (s *Space) GenerateFileSharingLink(
	ctx context.Context,
	encryptionPassword string,
	path string,
	bucketName string,
	dbID string,
) (domain.FileSharingInfo, error) {
	_, fileName := filepath.Split(path)

	var bucket t.Bucket
	var err error

	if dbID != "" {
		bucket, err = s.getBucketForRemoteFile(ctx, bucketName, dbID, path)
	} else {
		bucket, err = s.getBucketWithFallback(ctx, bucketName)
	}
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

// uploads the shared file to ipfs through users public bucket in hub
func (s *Space) uploadSharedFileToIpfs(
	ctx context.Context,
	password string,
	sharedContent io.Reader,
	fileName string,
	bucketName string,
) (domain.FileSharingInfo, error) {
	b, err := s.tc.GetPublicShareBucket(ctx)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "failed to get public files bucket")
	}

	timestamp := time.Now().UnixNano()
	uploadResult, _, err := b.UploadFile(ctx, fmt.Sprintf("%s-%d", fileName, timestamp), sharedContent)
	if err != nil {
		return EmptyFileSharingInfo, errors.Wrap(err, "publishing shared file failed")
	}

	encryptedFileHash := uploadResult.Cid().String()

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
func (s *Space) GenerateFilesSharingLink(ctx context.Context, encryptionPassword string, paths []string, bucketName, dbID string) (domain.FileSharingInfo, error) {
	if len(paths) == 0 {
		return EmptyFileSharingInfo, errors.New("no file passed to share link")
	}
	if len(paths) == 1 {
		return s.GenerateFileSharingLink(ctx, encryptionPassword, paths[0], bucketName, dbID)
	}

	var bucket t.Bucket
	var err error

	if dbID != "" {
		// Safe to use the first path to get the bucket as all shared files should be under the same dbID
		bucket, err = s.getBucketForRemoteFile(ctx, bucketName, dbID, paths[0])
	} else {
		bucket, err = s.getBucketWithFallback(ctx, bucketName)
	}
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
// files location. NOTE: This only opens public link shared files and not those shared via direct invites.
func (s *Space) OpenSharedFile(ctx context.Context, hash, password, filename string) (domain.OpenFileInfo, error) {
	parsedCid, err := cid.Parse(hash)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}

	err = s.waitForTextileHub(ctx)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}

	if password == "" {
		// try to fetch password from shared files
		_, password, err = s.tc.GetPublicReceivedFile(ctx, hash, true)
		if err != nil {
			return domain.OpenFileInfo{}, errors.Wrap(err, "password is required to open this file")
		}
	}

	encryptedFile, err := s.tc.DownloadPublicGatewayItem(ctx, parsedCid)
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
		log.Error("initializing decrypter failed", err)
		return domain.OpenFileInfo{}, errors.New("incorrect password")
	}

	decryptedFileSize, err := io.Copy(decryptedFile, reader)
	if err != nil {
		return domain.OpenFileInfo{}, errors.Wrap(err, "decryption failed")
	}

	// Add accessed file to shared with me list
	_, err = s.tc.AcceptSharedFileLink(ctx, hash, password, filename, strconv.FormatInt(decryptedFileSize, 10))
	if err != nil {
		return domain.OpenFileInfo{}, errors.Wrap(err, "accepting shared link failed")
	}

	return domain.OpenFileInfo{
		Location: decryptedFile.Name(),
	}, nil
}

func (s *Space) ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey) error {
	err := s.waitForTextileHub(ctx)
	if err != nil {
		return err
	}

	m := s.tc.GetModel()

	enhancedPaths := make([]domain.FullPath, len(paths))
	enckeys := make([][]byte, len(paths))
	for i, path := range paths {
		ep := domain.FullPath{
			DbId:      path.DbId,
			Bucket:    path.Bucket,
			Path:      path.Path,
			BucketKey: path.BucketKey,
		}

		// this handles personal bucket since for shared-with-me files
		// the dbid will be preset
		if ep.DbId == "" {
			b, err := s.tc.GetDefaultBucket(ctx)
			if err != nil {
				return err
			}

			bs, err := m.FindBucket(ctx, b.Slug())
			if err != nil {
				return err
			}

			ep.DbId = bs.RemoteDbID
		}

		if ep.Bucket == "" || ep.Bucket == t.GetDefaultBucketSlug() {
			b, err := s.tc.GetDefaultBucket(ctx)
			if err != nil {
				return err
			}
			bs, err := m.FindBucket(ctx, b.GetData().Name)
			if err != nil {
				return err
			}
			ep.Bucket = t.GetDefaultMirrorBucketSlug()
			ep.BucketKey = bs.RemoteBucketKey
			enckeys[i] = bs.EncryptionKey
		} else {
			r, err := m.FindReceivedFile(ctx, path.DbId, path.Bucket, path.Path)
			if err != nil {
				return err
			}
			ep.Bucket = r.Bucket
			ep.BucketKey = r.BucketKey
			enckeys[i] = r.EncryptionKey
		}

		enhancedPaths[i] = ep
	}

	err = s.tc.ShareFilesViaPublicKey(ctx, enhancedPaths, pubkeys, enckeys)
	if err != nil {
		return err
	}

	for _, pk := range pubkeys {
		inviter, err := s.keychain.GetStoredPublicKey()
		if err != nil {
			return err
		}

		inviterRaw, err := inviter.Raw()
		if err != nil {
			return err
		}

		pkRaw, err := pk.Raw()
		if err != nil {
			return err
		}

		d := &domain.Invitation{
			InviterPublicKey: hex.EncodeToString(inviterRaw),
			InviteePublicKey: hex.EncodeToString(pkRaw),
			ItemPaths:        enhancedPaths,
			Keys:             enckeys,
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

var errInvitationNotFound = errors.New("invitation not found")
var errFailedToNotifyInviter = errors.New("failed to notify inviter of invitation status")

// HandleSharedFilesInvitation accepts or rejects an invitation based on the invitation id
func (s *Space) HandleSharedFilesInvitation(ctx context.Context, invitationId string, accept bool) error {
	err := s.waitForTextileHub(ctx)
	if err != nil {
		return err
	}

	n, err := s.tc.GetMailAsNotifications(ctx, invitationId, 1)
	if err != nil {
		log.Error("failed to get invitation", err)
		return errInvitationNotFound
	}

	if len(n) == 0 {
		log.Debug("shared file invitation not found", "invitationId:"+invitationId)
		return errInvitationNotFound
	}

	invitation, err := extractInvitation(n[0])
	if err != nil {
		return err
	}

	if accept {
		invitation, err = s.tc.AcceptSharedFilesInvitation(ctx, invitation)

		// notify inviter,  it was accepted
		invitersPk, err := decodePublicKey(err, invitation.InviterPublicKey)
		if err != nil {
			log.Error("should not happen, but inviters public key is invalid", err)
			return errFailedToNotifyInviter
		}

		messageBody, err := json.Marshal(&invitation)
		if err != nil {
			log.Error("error encoding invitation response body", err)
			return errFailedToNotifyInviter
		}

		message, err := json.Marshal(&domain.MessageBody{
			Type: domain.INVITATION_REPLY,
			Body: messageBody,
		})

		if err != nil {
			log.Error("error encoding invitation response", err)
			return errFailedToNotifyInviter
		}

		_, err = s.tc.SendMessage(ctx, invitersPk, message)
	} else {
		invitation, err = s.tc.RejectSharedFilesInvitation(ctx, invitation)
	}
	if err != nil {
		return err
	}

	return err
}

func (s *Space) AddRecentlySharedPublicKeys(ctx context.Context, pubkeys []crypto.PubKey) error {
	err := s.waitForTextileInit(ctx)
	if err != nil {
		return err
	}

	var ps string

	for _, pk := range pubkeys {
		b, err := pk.Raw()
		if err != nil {
			return err
		}

		ps = hex.EncodeToString(b)

		// TODO: transaction
		_, err = s.tc.GetModel().CreateSharedPublicKey(ctx, ps)
		if err != nil {
			return nil
		}
	}

	return nil
}

func (s *Space) RecentlySharedPublicKeys(ctx context.Context) ([]crypto.PubKey, error) {
	err := s.waitForTextileInit(ctx)
	if err != nil {
		return nil, err
	}

	ret := []crypto.PubKey{}

	keys, err := s.tc.GetModel().ListSharedPublicKeys(ctx)
	if err != nil {
		return nil, err
	}

	for _, schema := range keys {
		b, err := hex.DecodeString(schema.PublicKey)
		if err != nil {
			return nil, err
		}
		p, err := crypto.UnmarshalEd25519PublicKey([]byte(b))
		if err != nil {
			return nil, err
		}

		ret = append(ret, p)
	}

	return ret, nil
}

// Returns a list of shared files the user has received and accepted
func (s *Space) GetSharedWithMeFiles(ctx context.Context, seek string, limit int) ([]*domain.SharedDirEntry, string, error) {
	err := s.waitForTextileInit(ctx)
	if err != nil {
		return nil, "", err
	}

	items, offset, err := s.tc.GetReceivedFiles(ctx, true, seek, limit)

	return items, offset, err
}
