package textile

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"

	"github.com/FleekHQ/space-daemon/core/textile/common"
	"github.com/FleekHQ/space-daemon/core/textile/utils"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/textile/bucket/crypto"

	"github.com/ipfs/interface-go-ipfs-core/path"
	bc "github.com/textileio/textile/api/buckets/client"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	bucketspb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/buckets"
)

var textileRelPathRegex = regexp.MustCompile(`/ip[f|n]s/[^/]*(?P<relPath>/.*)`)

// SecureBucketClient implements the BucketsClient Interface
// It encrypts data being pushed to the underlying textile client
// and also decrypts response from the underlying textile client
type SecureBucketClient struct {
	client     *bucketsClient.Client
	bucketSlug string
}

func NewSecureBucketsClient(
	client *bucketsClient.Client,
	bucketSlug string,
) *SecureBucketClient {
	return &SecureBucketClient{
		client:     client,
		bucketSlug: bucketSlug,
	}
}

func (s *SecureBucketClient) PushPath(ctx context.Context, key, path string, reader io.Reader, opts ...bc.Option) (result path.Resolved, root path.Resolved, err error) {
	path = cleanBucketPath(path)
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return nil, nil, err
	}

	// encrypt path before uploading
	encryptedPath, encryptedReader, err := s.encryptPathData(ctx, encryptionKey, path, reader)
	if err != nil {
		return nil, nil, err
	}

	return s.client.PushPath(ctx, key, encryptedPath, encryptedReader)

	// For now ignoring parsing results since it not being used downstream
	// but putting a TODO here in the meantime
}

func (s *SecureBucketClient) PushPathAccessRoles(ctx context.Context, key, path string, roles map[string]buckets.Role) error {
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return err
	}

	encryptedPath, _, err := s.encryptPathData(ctx, encryptionKey, path, nil)
	if err != nil {
		return err
	}

	return s.client.PushPathAccessRoles(ctx, key, encryptedPath, roles)
}

func (s *SecureBucketClient) PullPathAccessRoles(ctx context.Context, key, path string) (map[string]buckets.Role, error) {
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return nil, err
	}

	encryptedPath, _, err := s.encryptPathData(ctx, encryptionKey, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.PullPathAccessRoles(ctx, key, encryptedPath)
}

func (s *SecureBucketClient) PullPath(ctx context.Context, key, path string, writer io.Writer, opts ...bc.Option) error {
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return err
	}

	encryptedPath, _, err := s.encryptPathData(ctx, encryptionKey, path, nil)
	if err != nil {
		return err
	}

	errs := make(chan error)
	pipeReader, pipeWriter := io.Pipe()

	// pipe the writes from buckets to reader to be decrypted

	go func() {
		defer pipeWriter.Close()
		if err := s.client.PullPath(ctx, key, encryptedPath, pipeWriter, opts...); err != nil {
			errs <- err
		}
	}()
	go func() {
		defer close(errs)
		_, r, err := s.decryptPathData(ctx, encryptionKey, "", pipeReader)
		if err != nil {
			errs <- err
			return
		}
		defer r.Close()
		// copy decrypted reads to original writer
		if _, err := io.Copy(writer, r); err != nil {
			errs <- err
			return
		}
	}()
	return <-errs
}

func (s *SecureBucketClient) overwriteDecryptedItem(ctx context.Context, item *bucketspb.PathItem) error {
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return err
	}
	log.Debug("Processing Result Item", "name:"+item.Name, "path:"+item.Path)
	if utils.IsSpecialFileName(item.Name) {
		return nil
	}
	// decrypt file name
	item.Name, _, err = s.decryptPathData(ctx, encryptionKey, item.Name, nil)
	if err != nil {
		return err
	}

	// decrypts file path
	matchedPaths := textileRelPathRegex.FindStringSubmatch(item.Path)
	if len(matchedPaths) > 1 {
		item.Path, _, err = s.decryptPathData(ctx, encryptionKey, matchedPaths[1], nil)
		if err != nil {
			return err
		}
	}
	log.Debug("Processed Result Item", "name:"+item.Name, "path:"+item.Path)

	// Item size is generally (content size + hmac (64 bytes))
	if item.Size >= 64 {
		item.Size = item.Size - 64
	}

	return nil
}

func (s *SecureBucketClient) ListIpfsPath(ctx context.Context, pth path.Path) (*bucketspb.ListIpfsPathResponse, error) {
	return s.client.ListIpfsPath(ctx, pth)
}

func (s *SecureBucketClient) ListPath(ctx context.Context, key, path string) (*bucketspb.ListPathResponse, error) {
	path = cleanBucketPath(path)
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return nil, err
	}

	encryptedPath, _, err := s.encryptPathData(ctx, encryptionKey, path, nil)
	if err != nil {
		return nil, err
	}

	result, err := s.client.ListPath(ctx, key, encryptedPath)
	if err != nil {
		return nil, err
	}

	// decrypt result items
	for _, item := range result.Item.Items {
		err = s.overwriteDecryptedItem(ctx, item)
		if err != nil {
			// Don't error on a single file not decrypted
			log.Error("Error decrypting a file", err)
		}
	}

	// decrypt root item
	err = s.overwriteDecryptedItem(ctx, result.Item)
	if err != nil {
		// Don't error on a single file not decrypted
		log.Error("Error decrypting a file", err)
	}

	return result, nil
}

func (s *SecureBucketClient) RemovePath(ctx context.Context, key, path string, opts ...bc.Option) (path.Resolved, error) {
	path = cleanBucketPath(path)
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return nil, err
	}

	// encrypt path before submitting delete
	encryptedPath, _, err := s.encryptPathData(ctx, encryptionKey, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.RemovePath(ctx, key, encryptedPath, opts...)
}

func (s *SecureBucketClient) getBucketEncryptionKey(ctx context.Context) ([]byte, error) {
	if key, exists := common.BucketEncryptionKeyFromContext(ctx); exists {
		return key, nil
	}
	return nil, errors.New("bucket encryption key missing")
}

func (s *SecureBucketClient) encryptPathData(
	ctx context.Context,
	key []byte,
	path string,
	dataReader io.Reader,
) (string, io.Reader, error) {
	return crypto.EncryptPathItems(key, path, dataReader)
}

func (s *SecureBucketClient) decryptPathData(
	ctx context.Context,
	key []byte,
	path string,
	dataReader io.Reader,
) (string, io.ReadCloser, error) {
	return crypto.DecryptPathItems(key, path, dataReader)
}

// Cleans path used to access data in buckets
// Currently only removes prefix path if exists.
// would later include logic to normalize paths from other operating systems like windows
func cleanBucketPath(path string) string {
	return strings.TrimPrefix(path, "/")
}
