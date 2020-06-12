package textile

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/FleekHQ/space/config"
	"github.com/libp2p/go-libp2p-core/crypto"

	"github.com/FleekHQ/space/core/keychain"
	db "github.com/FleekHQ/space/core/store"
	"github.com/FleekHQ/space/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
)

type textileClient struct {
	store         db.Store
	threads       *threadsClient.Client
	bucketsClient *bucketsClient.Client
	isRunning     bool
	Ready         chan bool

	bucketsLock sync.RWMutex
	buckets     map[string]*bucket
}

func (tc *textileClient) WaitForReady() chan bool {
	return tc.Ready
}

// Keep file is added to empty directories
var keepFileName = ".keep"

// Creates a new Textile Client
func NewClient(store db.Store) Client {
	return &textileClient{
		store:         store,
		threads:       nil,
		bucketsClient: nil,
		isRunning:     false,
		Ready:         make(chan bool),
		buckets:       make(map[string]*bucket),
	}
}

func getThreadName(userPubKey []byte, bucketSlug string) string {
	return hex.EncodeToString(userPubKey) + "-" + bucketSlug
}

func getThreadIDStoreKey(bucketSlug string) []byte {
	return []byte(threadIDStoreKey + "_" + bucketSlug)
}

func (tc *textileClient) findOrCreateThreadID(ctx context.Context, threads *threadsClient.Client, bucketSlug string) (*thread.ID, error) {
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

func (tc *textileClient) IsRunning() bool {
	return tc.isRunning
}

func (tc *textileClient) requiresRunning() error {
	if tc.isRunning == false {
		return errors.New("ran an operation that requires starting textileClient first")
	}
	return nil
}

func (tc *textileClient) GetBaseThreadsContext(ctx context.Context) (context.Context, error) {
	// TODO: this should be happening in an auth lambda
	// only needed for hub connections
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	if key == "" || secret == "" {
		return nil, errors.New("Couldn't get Textile key or secret from envs")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = common.NewAPIKeyContext(ctx, key)

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

// Returns a thread client connection. Requires the client to be running.
func (tc *textileClient) GetThreadsConnection() (*threadsClient.Client, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.threads, nil
}

// Starts the Textile Client
func (tc *textileClient) start(cfg config.Config) error {
	auth := common.Credentials{}
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	var threads *threadsClient.Client
	var buckets *bucketsClient.Client

	finalHubTarget := hubTarget
	finalThreadsTarget := threadsTarget

	hubTargetFromCfg := cfg.GetString(config.TextileHubTarget, "")
	threadsTargetFromCfg := cfg.GetString(config.TextileThreadsTarget, "")

	if hubTargetFromCfg != "" {
		finalHubTarget = hubTargetFromCfg
	}

	if threadsTargetFromCfg != "" {
		finalThreadsTarget = threadsTargetFromCfg
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

	tc.bucketsClient = buckets
	tc.threads = threads

	tc.isRunning = true
	tc.Ready <- true
	return nil
}

// Closes connection to Textile
func (tc *textileClient) Stop() error {
	tc.isRunning = false
	close(tc.Ready)
	if err := tc.bucketsClient.Close(); err != nil {
		return err
	}

	if err := tc.threads.Close(); err != nil {
		return err
	}

	tc.bucketsClient = nil
	tc.threads = nil

	return nil
}

// StartAndBootstrap starts a Textile Client and also initializes default resources for it like a key pair and default bucket.
func (tc *textileClient) StartAndBootstrap(ctx context.Context, cfg config.Config) error {
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
	if err := tc.start(cfg); err != nil {
		log.Error("Error starting Textile Client", err)
		return err
	}

	log.Debug("Creating default bucket...")
	_, err := tc.CreateBucket(ctx, defaultPersonalBucketSlug)
	if err != nil {
		log.Error("Error creating default bucket", err)
		return err
	}

	log.Debug("Textile Client initialized successfully")
	return nil
}
