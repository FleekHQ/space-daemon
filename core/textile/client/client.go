package client

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"time"

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

const hubTarget = "127.0.0.1:3006"
const threadsTarget = "127.0.0.1:3006"
const threadIDStoreKey = "thread_id"

type TextileClient struct {
	store     *db.Store
	threads   *threadsClient.Client
	buckets   *bucketsClient.Client
	isRunning bool
}

// Creates a new Textile Client
func New(store *db.Store) *TextileClient {
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

	return &TextileClient{
		store:     store,
		threads:   threads,
		buckets:   buckets,
		isRunning: false,
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

// Creates a bucket.
func (tc *TextileClient) CreateBucket(bucketSlug string) error {
	// TODO: this should be happening in an auth lambda
	// only needed for hub connections
	log.Debug("Creating a new bucket with slug" + bucketSlug)

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
	var publicKey crypto.PubKey
	if privateKey, publicKey, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return err
	}

	// TODO: CTX has to be made from session key received from lambda
	log.Debug("Creating libp2p identity")
	var tok thread.Token
	if tok, err = tc.threads.GetToken(ctx, thread.NewLibp2pIdentity(privateKey)); err != nil {
		return err
	}
	ctx = thread.NewTokenContext(ctx, tok)

	// create thread
	log.Debug("Creating thread")
	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return err
	}

	ctx = common.NewThreadNameContext(ctx, getThreadName(pubKeyInBytes, bucketSlug))

	var dbID *thread.ID
	log.Debug("Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(tc.threads, bucketSlug); err != nil {
		return err
	}

	log.Debug("Creating Thread DB")
	if err := tc.threads.NewDB(ctx, *dbID); err != nil {
		return err
	}
	ctx = common.NewThreadIDContext(ctx, *dbID)

	// create bucket
	log.Debug("Generating bucket")
	if _, err := tc.buckets.Init(ctx, bucketSlug); err != nil {
		return err
	}

	return nil
}

// Starts the Textile Client
func (tc *TextileClient) Start() error {

	tc.isRunning = true
	// TODO: Listen for changes

	return nil
}
