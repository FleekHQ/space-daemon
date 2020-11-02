package textile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/common"
	"github.com/FleekHQ/space-daemon/core/textile/utils"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/textile/bucket/crypto"

	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	threadsClient "github.com/textileio/go-threads/api/client"
	bc "github.com/textileio/textile/v2/api/buckets/client"
	bucketsClient "github.com/textileio/textile/v2/api/buckets/client"
	bucketspb "github.com/textileio/textile/v2/api/buckets/pb"
	"github.com/textileio/textile/v2/buckets"
)

var textileRelPathRegex = regexp.MustCompile(`/ip[f|n]s/[^/]*(?P<relPath>/.*)`)

// SecureBucketClient implements the BucketsClient Interface
// It encrypts data being pushed to the underlying textile client
// and also decrypts response from the underlying textile client
type SecureBucketClient struct {
	client     *bucketsClient.Client
	kc         keychain.Keychain
	st         store.Store
	threads    *threadsClient.Client
	ipfsClient iface.CoreAPI
	isRemote   bool
}

func NewSecureBucketsClient(
	client *bucketsClient.Client,
	kc keychain.Keychain,
	st store.Store,
	threads *threadsClient.Client,
	ipfsClient iface.CoreAPI,
	isRemote bool,
) *SecureBucketClient {
	return &SecureBucketClient{
		client:     client,
		kc:         kc,
		st:         st,
		threads:    threads,
		ipfsClient: ipfsClient,
		isRemote:   isRemote,
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

	_, filename := filepath.Split(path)
	encryptedPath, _, err := s.encryptPathData(ctx, encryptionKey, filename, nil)
	if err != nil {
		return err
	}

	errs := make(chan error)
	pipeReader, pipeWriter := io.Pipe()

	// pipe the writes from buckets to reader to be decrypted

	go func() {
		defer pipeWriter.Close()
		if err := s.racePullFile(ctx, key, encryptedPath, pipeWriter, opts...); err != nil {
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
	err = <-errs
	return err
}

func (s *SecureBucketClient) overwriteDecryptedItem(ctx context.Context, item *bucketspb.PathItem) error {
	encryptionKey, err := s.getBucketEncryptionKey(ctx)
	if err != nil {
		return err
	}
	if utils.IsMetaFileName(item.Name) {
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
			log.Debug(fmt.Sprintf("Error decrypting a file: %s", err.Error()))
		}
	}

	// decrypt root item
	err = s.overwriteDecryptedItem(ctx, result.Item)
	if err != nil {
		// Don't error on a single file not decrypted
		log.Debug(fmt.Sprintf("Error decrypting a file: %s", err.Error()))
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

type pathPullingFn func(context.Context, string, string, io.Writer, ...bc.Option) (bool, error)

type pullSuccessResponse struct {
	file        *os.File
	shouldCache bool
}

func (s *SecureBucketClient) racePullFile(ctx context.Context, key, encPath string, w io.Writer, opts ...bc.Option) error {
	pullers := []pathPullingFn{s.pullFileFromLocal, s.pullFileFromClient, s.pullFileFromDHT}

	pullSuccess := make(chan *pullSuccessResponse)
	errc := make(chan error)
	defer close(pullSuccess)

	ctxWithCancel, cancelPulls := context.WithCancel(ctx)
	pendingFns := len(pullers)
	erroredFns := 0

	for _, fn := range pullers {
		f, err := ioutil.TempFile("", "*-"+encPath)

		if err != nil {
			cancelPulls()
			return err
		}
		defer f.Close()
		defer os.Remove(f.Name())

		go func(fn pathPullingFn, f *os.File) {
			shouldCache, err := fn(ctxWithCancel, key, encPath, f, opts...)
			if err != nil {
				errc <- err
				return
			}

			chanRes := &pullSuccessResponse{
				file:        f,
				shouldCache: shouldCache,
			}

			pullSuccess <- chanRes
			errc <- nil
		}(fn, f)
	}

	var pullErr error

	// Wait for either all pullers to fail or for one to succeed
	go func() {
		for {
			select {
			case err := <-errc:
				pendingFns--

				if err != nil {
					erroredFns++
					pullErr = err
				}
				if pendingFns <= 0 && erroredFns >= len(pullers) {
					// All functions failed. Stop waiting
					pullSuccess <- nil
				}

				if pendingFns <= 0 {
					close(errc)
					return
				}
			}
		}
	}()

	pullResponse := <-pullSuccess
	cancelPulls()

	// Return error if all pull functions failed
	if erroredFns >= len(pullers) || pullResponse == nil {
		return pullErr
	}

	finalFile := pullResponse.file
	shouldCache := pullResponse.shouldCache

	// Copy pulled file to upstream writer
	resErrc := make(chan error)
	defer close(resErrc)
	go func() {
		from, err := os.Open(finalFile.Name())
		if err != nil {
			resErrc <- err
			return
		}
		defer from.Close()

		_, err = io.Copy(w, from)
		resErrc <- err
	}()

	// Copy pulled file to local cache
	cacheErrc := make(chan error)
	defer close(cacheErrc)
	go func() {
		var err error
		if !shouldCache {
			cacheErrc <- nil
			return
		}
		from, err := os.Open(finalFile.Name())
		if err != nil {
			cacheErrc <- err
			return
		}
		defer from.Close()

		p, err := s.ipfsClient.Unixfs().Add(
			ctx,
			ipfsfiles.NewReaderFile(from),
			options.Unixfs.Pin(false), // Turn to true when we enable DHT discovery
			options.Unixfs.Progress(false),
			options.Unixfs.CidVersion(1),
		)
		if err != nil {
			cacheErrc <- err
			return
		}

		cidBinary := p.Cid().Bytes()
		err = s.st.Set(getFileCacheKey(encPath), cidBinary)

		cacheErrc <- err
	}()

	if err := <-resErrc; err != nil {
		return err
	}

	if err := <-cacheErrc; err != nil {
		return err
	}

	return nil
}

const fileCachePrefix = "file_cache"

func getFileCacheKey(encPath string) []byte {
	return []byte(fileCachePrefix + ":" + encPath)
}

func (s *SecureBucketClient) pullFileFromClient(ctx context.Context, key, encPath string, w io.Writer, opts ...bc.Option) (shouldCache bool, err error) {
	shouldCache = true
	if s.isRemote == false {
		// File already in local bucket
		shouldCache = false
	}

	if err = s.client.PullPath(ctx, key, encPath, w, opts...); err != nil {
		return false, err
	}
	return shouldCache, nil
}

var errNoLocalClient = errors.New("No cache client available")

func (s *SecureBucketClient) pullFileFromLocal(ctx context.Context, key, encPath string, w io.Writer, opts ...bc.Option) (shouldCache bool, err error) {
	shouldCache = false

	cidBinary, err := s.st.Get(getFileCacheKey(encPath))
	if cidBinary == nil || err != nil {
		return false, errors.New("CID not stored in local cache")
	}

	_, c, err := cid.CidFromBytes(cidBinary)
	if err != nil {
		return false, err
	}

	node, err := s.ipfsClient.Unixfs().Get(ctx, path.New(c.String()))
	if err != nil {
		return false, err
	}
	defer node.Close()

	file := ipfsfiles.ToFile(node)
	if file == nil {
		return false, errors.New("File is a directory")
	}

	if _, err := io.Copy(w, file); err != nil {
		return false, err
	}

	return shouldCache, nil
}

func (s *SecureBucketClient) pullFileFromDHT(ctx context.Context, key, encPath string, w io.Writer, opts ...bc.Option) (shouldCache bool, err error) {
	shouldCache = true

	// return shouldCache, nil
	return false, errors.New("Not implemented")
}

const cacheBucketThreadName = "cache_bucket"
