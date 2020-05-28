package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ipfs/interface-go-ipfs-core/path"

	buckets_pb "github.com/textileio/textile/api/buckets/pb"

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
	defaultPersonalBucketSlug = "personal"
)

type TextileBucketRoot buckets_pb.Root
type TextileDirEntries buckets_pb.ListPathReply

type TextileClient struct {
	store     *db.Store
	threads   *threadsClient.Client
	buckets   *bucketsClient.Client
	isRunning bool
	ctx       context.Context
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
		ctx:       nil,
	}
}

func getThreadName(userPubKey []byte, bucketSlug string) string {
	return hex.EncodeToString(userPubKey) + "-" + bucketSlug
}

func getThreadIDStoreKey(bucketSlug string) []byte {
	return []byte(threadIDStoreKey + "_" + bucketSlug)
}

func (tc *TextileClient) findOrCreateThreadID(threads *threadsClient.Client, bucketSlug string) (*thread.ID, error) {
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

func (tc *TextileClient) initContext() error {
	// TODO: this should be happening in an auth lambda
	// only needed for hub connections
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	if key == "" || secret == "" {
		return errors.New("Couldn't get Textile key or secret from envs")
	}

	ctx := context.Background()
	ctx = common.NewAPIKeyContext(ctx, key)

	var err error
	var apiSigCtx context.Context

	if apiSigCtx, err = common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), secret); err != nil {
		return err
	}
	ctx = apiSigCtx

	log.Debug("Obtaining user key pair from local store")
	kc := keychain.New(tc.store)
	var privateKey crypto.PrivKey
	if privateKey, _, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return err
	}

	// TODO: CTX has to be made from session key received from lambda
	log.Debug("Creating libp2p identity")
	var tok thread.Token
	if tok, err = tc.threads.GetToken(ctx, thread.NewLibp2pIdentity(privateKey)); err != nil {
		return err
	}
	ctx = thread.NewTokenContext(ctx, tok)

	tc.ctx = ctx

	return nil
}

// Creates a bucket.
func (tc *TextileClient) CreateBucket(bucketSlug string) (*TextileBucketRoot, error) {
	log.Debug("Creating a new bucket with slug" + bucketSlug)

	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	var err error
	var publicKey crypto.PubKey
	kc := keychain.New(tc.store)
	if _, publicKey, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, err
	}

	// create thread (each bucket belongs to a different thread)
	log.Debug("Creating thread")
	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return nil, err
	}

	ctx := tc.ctx
	ctx = common.NewThreadNameContext(ctx, getThreadName(pubKeyInBytes, bucketSlug))

	var dbID *thread.ID
	log.Debug("Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(tc.threads, bucketSlug); err != nil {
		return nil, err
	}

	ctx = common.NewThreadIDContext(ctx, *dbID)

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

	log.Debug("Creating Thread DB")
	if err := tc.threads.NewDB(ctx, *dbID); err != nil {
		return nil, err
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

	if err := tc.initContext(); err != nil {
		return err
	}

	tc.isRunning = true
	return nil
}

// Closes connection to Textile
func (tc *TextileClient) Stop() error {
	tc.isRunning = false
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
func (tc *TextileClient) StartAndBootstrap() (*TextileBucketRoot, error) {
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
		return nil, err
	}

	// Create default bucket
	log.Debug("Creating default bucket...")
	bucketRoot, err := tc.CreateBucket(defaultPersonalBucketSlug)
	if err != nil {
		log.Error("Error creating default bucket", err)
		return nil, err
	}

	log.Debug("Textile Client initialized successfully")
	return bucketRoot, nil
}

// UploadFile uploads a file to path on textile
// path should include the file name as the last path segment
// also nested path not existing yet would be created automatically
func (tc *TextileClient) UploadFile(
	ctx context.Context,
	bucketKey string,
	path string,
	reader io.Reader,
) (result path.Resolved, root path.Path, err error) {
	return tc.buckets.PushPath(ctx, bucketKey, path, reader)
}

// CreateDirectory creates an empty directory
// Because textile doesn't support empty directory an empty .keep file is created
// in the directory
func (tc *TextileClient) CreateDirectory(
	ctx context.Context,
	bucketKey string,
	path string,
) (result path.Resolved, root path.Path, err error) {
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
	return tc.buckets.RemovePath(ctx, bucketKey, path)
}
