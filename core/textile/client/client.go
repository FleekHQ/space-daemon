package client

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/FleekHQ/space-poc/core/keychain"
	db "github.com/FleekHQ/space-poc/core/store"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
)

const hubTarget = "127.0.0.1:3006"
const threadstarget = "127.0.0.1:3006"
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

	if b, err := bucketsClient.NewClient(hubTarget, opts...); err != nil {
		cmd.Fatal(err)
	} else {
		buckets = b
	}

	if t, err := threadsClient.NewClient(threadstarget, opts...); err != nil {
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

func (tc *TextileClient) findOrCreateThreadID(threads *threadsClient.Client) (*thread.ID, error) {
	if val, err := tc.store.Get([]byte(threadIDStoreKey)); err != nil {
		newErr := errors.New("error while retrieving thread id: check your local space db accessibility")
		return nil, newErr
	} else if val != nil {
		// Cast the stored dbID from bytes to thread.ID
		if dbID, err := thread.Cast(val); err != nil {
			return nil, err
		} else {
			return &dbID, nil
		}
	}

	// thread id does not exist yet
	dbID := thread.NewIDV1(thread.Raw, 32)
	dbIDInBytes := dbID.Bytes()

	if err := tc.store.Set([]byte(threadIDStoreKey), dbIDInBytes); err != nil {
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
	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")
	ctx := context.Background()
	ctx = common.NewAPIKeyContext(ctx, key)

	if apiSigCtx, err := common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), secret); err != nil {
		return err
	} else {
		ctx = apiSigCtx
	}

	kc := keychain.New(tc.store)
	var privateKey crypto.PrivKey
	var publicKey crypto.PubKey
	if priv, pub, err := kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return err
	} else {
		privateKey = priv
		publicKey = pub
	}

	// TODO: CTX has to be made from session key received from lambda
	if tok, err := tc.threads.GetToken(ctx, thread.NewLibp2pIdentity(privateKey)); err != nil {
		return err
	} else {
		ctx = thread.NewTokenContext(ctx, tok)
	}

	// create thread
	if pub, err := publicKey.Bytes(); err != nil {
		return err
	} else {
		ctx = common.NewThreadNameContext(ctx, getThreadName(pub, bucketSlug))
	}

	var dbID *thread.ID
	if val, err := tc.findOrCreateThreadID(tc.threads); err != nil {
		return err
	} else {
		dbID = val
	}
	if err := tc.threads.NewDB(ctx, *dbID); err != nil {
		return err
	}
	ctx = common.NewThreadIDContext(ctx, *dbID)

	// create bucket
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
