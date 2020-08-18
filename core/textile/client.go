package textile

import (
	"context"
	"errors"
	"strings"

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
	"github.com/textileio/textile/cmd"
	"google.golang.org/grpc"
)

type textileClient struct {
	store            db.Store
	kc               keychain.Keychain
	threads          *threadsClient.Client
	bucketsClient    *bucketsClient.Client
	isRunning        bool
	Ready            chan bool
	cfg              config.Config
	isConnectedToHub bool
	netc             *nc.Client
}

// Creates a new Textile Client
func NewClient(store db.Store, kc keychain.Keychain) *textileClient {
	return &textileClient{
		store:            store,
		kc:               kc,
		threads:          nil,
		bucketsClient:    nil,
		netc:             nil,
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

func (tc *textileClient) getHubCtx(ctx context.Context) (context.Context, error) {
	log.Debug("Authenticating with Textile Hub")

	// TODO: Use hub.GetHubToken instead
	tokStr, err := hub.GetHubTokenUsingTextileKeys(ctx, tc.store, tc.kc, tc.threads)
	if err != nil {
		return nil, err
	}

	tok := thread.Token(tokStr)

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

	tc.isRunning = true

	// Attempt to connect to the Hub
	_, err := tc.getHubCtx(ctx)
	if err != nil {
		log.Error("Could not connect to Textile Hub. Starting in offline mode.", err)
	} else {
		tc.isConnectedToHub = true
	}

	tc.Ready <- true
	return nil
}

// StartAndBootstrap starts a Textile Client and also initializes default resources for it like a key pair and default bucket.
func (tc *textileClient) Start(ctx context.Context, cfg config.Config) error {
	// Create key pair if not present
	log.Debug("Starting Textile Client: Getting key status...")
	_, err := tc.kc.GetStoredPublicKey()
	if err != nil {
		log.Debug("Key pair not found. Generating a new one")
		if mnemonic, err := tc.kc.GenerateKeyFromMnemonic(); err != nil {

			log.Error("Error generating key pair. Cannot continue Textile initialization", err)
			return err
		} else {
			words := strings.Split(mnemonic, " ")
			log.Debug("Generated initial key pair using mnemonic using seed: " + words[0] + ", " + words[1] + "...")
		}
	} else {
		log.Debug("Starting Textile Client: Key pair found.")
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

func (tc *textileClient) getThreadContext(parentCtx context.Context, threadName string, dbID thread.ID) (context.Context, error) {
	var err error
	ctx := parentCtx

	if err = tc.requiresRunning(); err != nil {
		return nil, err
	}

	// If we are connected to the Hub, add the keys to the context so we can replicate
	if tc.isConnectedToHub == true {
		ctx, err = tc.getHubCtx(ctx)
		if err != nil {
			return nil, err
		}
	}

	var publicKey crypto.PubKey
	if publicKey, err = tc.kc.GetStoredPublicKey(); err != nil {
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
