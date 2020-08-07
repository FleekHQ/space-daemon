package textile

import (
	"context"
	"crypto/tls"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/FleekHQ/space-daemon/config"
	crypto "github.com/libp2p/go-libp2p-crypto"

	"github.com/FleekHQ/space-daemon/core/keychain"
	db "github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	nc "github.com/textileio/go-threads/net/api/client"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
	uc "github.com/textileio/textile/api/users/client"
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type textileClient struct {
	store            db.Store
	threads          *threadsClient.Client
	ht               *threadsClient.Client
	bucketsClient    *bucketsClient.Client
	isRunning        bool
	Ready            chan bool
	cfg              config.Config
	isConnectedToHub bool
	netc             *nc.Client
	uc               *uc.Client
}

// Creates a new Textile Client
func NewClient(store db.Store) *textileClient {
	return &textileClient{
		store:            store,
		threads:          nil,
		bucketsClient:    nil,
		netc:             nil,
		uc:               nil,
		ht:               nil,
		isRunning:        false,
		Ready:            make(chan bool),
		isConnectedToHub: false,
	}
}

func (tc *textileClient) WaitForReady() chan bool {
	return tc.Ready
}

func (tc *textileClient) requiresRunning() error {
	if tc.isRunning == false {
		return errors.New("ran an operation that requires starting textileClient first")
	}
	return nil
}

func (tc *textileClient) getHubCtxViaChallenge(ctx context.Context) (context.Context, error) {
	log.Debug("Authenticating with Textile Hub")
	tokStr, err := hub.GetHubToken(ctx, tc.store, tc.cfg)
	if err != nil {
		log.Error("Token Challenge Error:", err)
		return nil, err
	}

	tok := thread.Token(tokStr)

	return thread.NewTokenContext(ctx, tok), nil
}

// This method is just for testing purposes. Keys shouldn't be bundled in the daemon
// NOTE: Being used right now instead of the challenge flow
// TODO: Use getHubCtxViaChallenge instead once that's fixed
func (tc *textileClient) getHubCtx(ctx context.Context) (context.Context, error) {
	log.Debug("Authenticating with Textile Hub")

	key := os.Getenv("TXL_USER_KEY")
	secret := os.Getenv("TXL_USER_SECRET")

	if key == "" || secret == "" {
		return nil, errors.New("Couldn't get Textile key or secret from envs")
	}

	ctx = common.NewAPIKeyContext(ctx, key)
	var apiSigCtx context.Context
	var err error
	if apiSigCtx, err = common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), secret); err != nil {
		return nil, err
	}
	ctx = apiSigCtx

	kc := keychain.New(tc.store)
	var privateKey crypto.PrivKey
	if privateKey, _, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, err
	}

	// TODO: CTX has to be made from session key received from lambda
	tok, err := tc.ht.GetToken(ctx, thread.NewLibp2pIdentity(privateKey))

	if err != nil {
		log.Info("error getting token")
		return nil, err
	}

	ctx = thread.NewTokenContext(ctx, tok)
	return ctx, nil
}

// Starts the Textile Client
func (tc *textileClient) start(ctx context.Context, cfg config.Config) error {
	tc.cfg = cfg
	auth := common.Credentials{}
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	var threads *threadsClient.Client
	var buckets *bucketsClient.Client
	var netc *nc.Client

	// by default it goes to local threads now
	host := "127.0.0.1:3006"

	log.Debug("Creating buckets client in " + host)
	if b, err := bucketsClient.NewClient(host, opts...); err != nil {
		cmd.Fatal(err)
	} else {
		buckets = b
	}

	log.Debug("Creating threads client in " + host)
	if t, err := threadsClient.NewClient(host, opts...); err != nil {
		cmd.Fatal(err)
	} else {
		threads = t
	}

	if n, err := nc.NewClient(host, opts...); err != nil {
		cmd.Fatal(err)
	} else {
		netc = n
	}

	tc.bucketsClient = buckets
	tc.threads = threads
	tc.netc = netc
	tc.uc = getUserClient()
	tc.ht = getHubThreadsClient()

	tc.isRunning = true

	// Attempt to connect to the Hub
	hubctx, err := tc.getHubCtx(ctx)
	if err != nil {
		log.Error("Could not connect to Textile Hub. Starting in offline mode.", err)
	} else {
		tc.isConnectedToHub = true

		// setup mailbox
		mid, err := tc.uc.SetupMailbox(hubctx)
		if err != nil {
			log.Error("Unable to setup mailbox", err)
			return err
		}
		log.Info("Mailbox id: " + mid.String())
	}

	tc.Ready <- true
	return nil
}

func getUserClient() *uc.Client {
	hubTarget := os.Getenv("TXL_HUB_HOST")
	auth := common.Credentials{}
	var opts []grpc.DialOption

	if strings.Contains(hubTarget, "443") {
		creds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(creds))
		auth.Secure = true
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	users, err := uc.NewClient(hubTarget, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	return users
}

func getHubThreadsClient() *threadsClient.Client {
	hubTarget := os.Getenv("TXL_HUB_HOST")
	auth := common.Credentials{}
	var opts []grpc.DialOption

	if strings.Contains(hubTarget, "443") {
		creds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(creds))
		auth.Secure = true
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	tc, err := threadsClient.NewClient(hubTarget, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	return tc
}

// StartAndBootstrap starts a Textile Client and also initializes default resources for it like a key pair and default bucket.
func (tc *textileClient) Start(ctx context.Context, cfg config.Config) error {
	// Create key pair if not present
	kc := keychain.New(tc.store)
	log.Debug("Starting Textile Client: Generating key pair...")
	if _, _, err := kc.GenerateKeyPair(); err != nil {
		log.Debug("Starting Textile Client: Error generating key pair, key might already exist")
		log.Debug(err.Error())
		// Not returning err since it can error if keys already exist
	}

	// Start Textile Client
	if err := tc.start(ctx, cfg); err != nil {
		log.Error("Error starting Textile Client", err)
		return err
	}

	log.Debug("Listing buckets...")
	buckets, err := tc.ListBuckets(ctx)
	if err != nil {
		log.Error("Error listing buckets on Textile client start", err)
		return err
	}

	// Create default bucket if it doesnt exist
	defaultBucketExists := false
	for _, b := range buckets {
		if b.Slug() == defaultPersonalBucketSlug {
			defaultBucketExists = true
		}
	}
	if defaultBucketExists == false {
		log.Debug("Creating default bucket...")
		_, err := tc.CreateBucket(ctx, defaultPersonalBucketSlug)
		if err != nil {
			log.Error("Error creating default bucket", err)
			return err
		}
	}

	log.Debug("Textile Client initialized successfully")
	return nil
}

// Closes connection to Textile
func (tc *textileClient) Shutdown() error {
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

// Returns a thread client connection. Requires the client to be running.
func (tc *textileClient) GetThreadsConnection() (*threadsClient.Client, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.threads, nil
}

func (tc *textileClient) IsRunning() bool {
	return tc.isRunning
}

func (tc *textileClient) getThreadContext(parentCtx context.Context, threadName string, dbID thread.ID, hub bool) (context.Context, error) {
	var err error
	ctx := parentCtx

	if err = tc.requiresRunning(); err != nil {
		return nil, err
	}

	// Some threads will be on the hub and some will be local, this flag lets you specify
	// where it is, perhaps an improvement could be to save this flag in the store and load it from there
	if hub {
		ctx, err = tc.getHubCtx(ctx)
		if err != nil {
			return nil, err
		}
	}

	var publicKey crypto.PubKey
	kc := keychain.New(tc.store)
	if _, publicKey, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, err
	}

	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return nil, err
	}

	ctx = common.NewThreadNameContext(ctx, getThreadName(pubKeyInBytes, threadName))
	ctx = common.NewThreadIDContext(ctx, dbID)

	return ctx, nil
}
