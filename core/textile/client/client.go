package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ipfs/interface-go-ipfs-core/path"

	buckets_pb "github.com/textileio/textile/api/buckets/pb"

	"github.com/FleekHQ/space-poc/core/ipfs"
	"github.com/FleekHQ/space-poc/core/keychain"
	db "github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/log"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
)

const (
	hubTarget                 = "127.0.0.1:3006"
	threadsTarget             = "127.0.0.1:3006"
	threadIDStoreKey          = "thread_id"
	defaultPersonalBucketSlug = "personal3"
)

type TextileBucketRoot buckets_pb.Root
type TextileDirEntries buckets_pb.ListPathReply

type TextileClient struct {
	store     *db.Store
	threads   *threadsClient.Client
	buckets   *bucketsClient.Client
	isRunning bool
	Ready     chan bool
}

// Keep file is added to empty directories
var keepFileName = ".keep"

// Creates a new Textile Client
func New(store *db.Store) *TextileClient {
	return &TextileClient{
		store:     store,
		threads:   nil,
		buckets:   nil,
		isRunning: false,
		Ready:     make(chan bool),
	}
}

func getThreadName(userPubKey []byte, bucketSlug string) string {
	return hex.EncodeToString(userPubKey) + "-" + bucketSlug
}

func getThreadIDStoreKey(bucketSlug string) []byte {
	return []byte(threadIDStoreKey + "_" + bucketSlug)
}

func (tc *TextileClient) findOrCreateThreadID(ctx context.Context, threads *threadsClient.Client, bucketSlug string) (*thread.ID, error) {
	if val, _ := tc.store.Get([]byte(getThreadIDStoreKey(bucketSlug))); val != nil {
		log.Debug("Thread ID found in local store")
		// Cast the stored dbID from bytes to thread.ID
		if dbID, err := thread.Cast(val); err != nil {
			return nil, err
		} else {
			return &dbID, nil
		}
	}

	// thread id does not exist yet
	log.Debug("Thread ID not found in local store. Generating a new one...")
	dbID := thread.NewIDV1(thread.Raw, 32)
	dbIDInBytes := dbID.Bytes()

	log.Debug("Creating Thread DB")
	if err := tc.threads.NewDB(ctx, dbID); err != nil {
		return nil, err
	}

	if err := tc.store.Set([]byte(getThreadIDStoreKey(bucketSlug)), dbIDInBytes); err != nil {
		newErr := errors.New("error while storing thread id: check your local space db accessibility")
		return nil, newErr
	}

	return &dbID, nil

}

func (tc *TextileClient) requiresRunning() error {
	if tc.isRunning == false {
		return errors.New("ran an operation that requires starting TextileClient first")
	}
	return nil
}

func (tc *TextileClient) GetBaseThreadsContext() (context.Context, error) {
	// TODO: this should be happening in an auth lambda
	// only needed for hub connections
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	if key == "" || secret == "" {
		return nil, errors.New("Couldn't get Textile key or secret from envs")
	}

	ctx := common.NewAPIKeyContext(context.Background(), key)

	var err error
	var apiSigCtx context.Context

	if apiSigCtx, err = common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), secret); err != nil {
		return nil, err
	}
	ctx = apiSigCtx

	log.Debug("Obtaining user key pair from local store")
	kc := keychain.New(tc.store)
	var privateKey crypto.PrivKey
	if privateKey, _, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, err
	}

	// TODO: CTX has to be made from session key received from lambda
	log.Debug("Creating libp2p identity")
	tok, err := tc.threads.GetToken(ctx, thread.NewLibp2pIdentity(privateKey))
	if err != nil {
		return nil, err
	}

	ctx = thread.NewTokenContext(ctx, tok)

	return ctx, nil
}

// Returns a context that works for accessing a bucket
func (tc *TextileClient) GetBucketContext(bucketSlug string) (context.Context, *thread.ID, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, nil, err
	}

	ctx, err := tc.GetBaseThreadsContext()
	if err != nil {
		return nil, nil, err
	}
	var publicKey crypto.PubKey
	kc := keychain.New(tc.store)
	if _, publicKey, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, nil, err
	}

	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return nil, nil, err
	}

	ctx = common.NewThreadNameContext(ctx, getThreadName(pubKeyInBytes, bucketSlug))

	var dbID *thread.ID
	log.Debug("Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(ctx, tc.threads, bucketSlug); err != nil {
		return nil, nil, err
	}

	ctx = common.NewThreadIDContext(ctx, *dbID)

	return ctx, dbID, nil
}

// Returns a thread client connection. Requires the client to be running.
func (tc *TextileClient) GetThreadsConnection() (*threadsClient.Client, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.threads, nil
}

func (tc *TextileClient) ListBuckets() ([]*TextileBucketRoot, error) {
	threadsCtx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)

	bucketList, err := tc.buckets.List(threadsCtx)
	if err != nil {
		return nil, err
	}

	result := make([]*TextileBucketRoot, 0)
	for _, r := range bucketList.Roots {
		result = append(result, (*TextileBucketRoot)(r))
	}

	return result, nil
}

// Creates a bucket.
func (tc *TextileClient) CreateBucket(bucketSlug string) (*TextileBucketRoot, error) {
	log.Debug("Creating a new bucket with slug " + bucketSlug)

	ctx := context.Background()
	var err error

	if ctx, _, err = tc.GetBucketContext(bucketSlug); err != nil {
		return nil, err
	}

	// return if bucket aready exists
	// TODO: see if threads.find would be faster
	bucketList, err := tc.buckets.List(ctx)
	if err != nil {
		log.Error("error while fetching bucket list ", err)
		return nil, err
	}
	for _, r := range bucketList.Roots {
		if r.Name == bucketSlug {
			log.Info("Bucket '" + bucketSlug + "' already exists")
			return (*TextileBucketRoot)(r), nil
		}
	}

	// create bucket
	log.Debug("Generating bucket")
	bucket, err := tc.buckets.Init(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	return (*TextileBucketRoot)(bucket.Root), nil
}

// Starts the Textile Client
func (tc *TextileClient) Start() error {
	auth := common.Credentials{}
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	var threads *threadsClient.Client
	var buckets *bucketsClient.Client

	finalHubTarget := hubTarget
	finalThreadsTarget := threadsTarget

	hubTargetFromEnv := os.Getenv("TXL_HUB_TARGET")
	threadsTargetFromEnv := os.Getenv("TXL_THREADS_TARGET")

	if hubTargetFromEnv != "" {
		finalHubTarget = hubTargetFromEnv
	}

	if threadsTargetFromEnv != "" {
		finalThreadsTarget = threadsTargetFromEnv
	}

	log.Debug("Creating buckets client in " + finalHubTarget)
	if b, err := bucketsClient.NewClient(finalHubTarget, opts...); err != nil {
		cmd.Fatal(err)
	} else {
		buckets = b
	}

	log.Debug("Creating threads client in " + finalThreadsTarget)
	if t, err := threadsClient.NewClient(finalThreadsTarget, opts...); err != nil {
		cmd.Fatal(err)
	} else {
		threads = t
	}

	tc.buckets = buckets
	tc.threads = threads

	tc.isRunning = true
	tc.Ready <- true
	return nil
}

// Closes connection to Textile
func (tc *TextileClient) Stop() error {
	tc.isRunning = false
	close(tc.Ready)
	if err := tc.buckets.Close(); err != nil {
		return err
	}

	if err := tc.threads.Close(); err != nil {
		return err
	}

	tc.buckets = nil
	tc.threads = nil

	return nil
}

// StartAndBootstrap starts a Textile Client and also initializes default resources for it like a key pair and default bucket.
func (tc *TextileClient) StartAndBootstrap() error {
	// Create key pair if not present
	kc := keychain.New(tc.store)
	log.Debug("Generating key pair...")
	if _, _, err := kc.GenerateKeyPair(); err != nil {
		log.Debug("Error generating key pair, key might already exist")
		log.Debug(err.Error())
		// Not returning err since it can error if keys already exist
	}

	// Start Textile Client
	log.Debug("Starting Textile Client...")
	if err := tc.Start(); err != nil {
		log.Error("Error starting Textile Client", err)
		return err
	}

	log.Debug("Creating default bucket...")
	_, err := tc.CreateBucket(defaultPersonalBucketSlug)
	if err != nil {
		log.Error("Error creating default bucket", err)
		return err
	}

	log.Debug("Textile Client initialized successfully")
	return nil
}

// UploadFile uploads a file to path on textile
// path should include the file name as the last path segment
// also nested path not existing yet would be created automatically
func (tc *TextileClient) UploadFile(
	bucketKey string,
	path string,
	reader io.Reader,
) (result path.Resolved, root path.Path, err error) {
	ctx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)
	return tc.buckets.PushPath(ctx, bucketKey, path, reader)
}

// CreateDirectory creates an empty directory
// Because textile doesn't support empty directory an empty .keep file is created
// in the directory
func (tc *TextileClient) CreateDirectory(
	bucketKey string,
	path string,
) (result path.Resolved, root path.Path, err error) {
	ctx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)

	// append .keep file to the end of the directory
	emptyDirPath := strings.TrimRight(path, "/") + "/" + keepFileName
	return tc.buckets.PushPath(ctx, bucketKey, emptyDirPath, &bytes.Buffer{})
}

// ListDirectory returns a list of items in a particular directory
func (tc *TextileClient) ListDirectory(
	ctx context.Context,
	bucketKey string,
	path string,
) (*TextileDirEntries, error) {
	result, err := tc.buckets.ListPath(ctx, bucketKey, path)

	return (*TextileDirEntries)(result), err
}

// DeleteDirOrFile will delete file or directory at path
func (tc *TextileClient) DeleteDirOrFile(
	ctx context.Context,
	bucketKey string,
	path string,
) error {
	tc.buckets.RemovePath(ctx, bucketKey, path)

	return nil
}

func (tc *TextileClient) FolderExists(key string, path string) (bool, error) {
	ctx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)
	if err != nil {
		return false, nil
	}

	_, err = tc.buckets.ListPath(ctx, key, path)

	log.Info("returned from bucket call")

	if err != nil {
		// NOTE: not sure if this is the best approach but didnt
		// want to loop over items each time
		match, _ := regexp.MatchString(".*no link named.*under.*", err.Error())
		if match {
			return false, nil
		}
		log.Info("error doing list path on non existent directoy: ", err.Error())
		// Since a nil would be interpreted as a false
		return false, err
	}
	return true, nil
}

func (tc *TextileClient) FileExists(key string, path string, r io.Reader) (bool, error) {
	ctx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)
	if err != nil {
		return false, nil
	}

	lp, err := tc.buckets.ListPath(ctx, key, path)
	if err != nil {
		match, _ := regexp.MatchString(".*no link named.*under.*", err.Error())
		if match {
			return false, nil
		}
		log.Info("error doing list path on non existent directoy: ", err.Error())
		// Since a nil would be interpreted as a false
		return false, err
	}

	var fsHash string
	if _, err := ipfs.GetFileHash(r); err != nil {
		log.Error("Unable to get filehash: ", err)
		return false, err
	}

	item := lp.GetItem()
	if item.Cid == fsHash {
		return true, nil
	}

	return false, nil
}
