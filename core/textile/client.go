package textile

import (
	"context"
	"crypto/tls"
	"errors"
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
	kc               keychain.Keychain
	threads          *threadsClient.Client
	ht               *threadsClient.Client
	bucketsClient    *bucketsClient.Client
	isRunning        bool
	isInitialized    bool
	Ready            chan bool
	keypairDeleted   chan bool
	cfg              config.Config
	isConnectedToHub bool
	netc             *nc.Client
	uc               UsersClient
	hubAuth          hub.HubAuth
}

// Creates a new Textile Client
func NewClient(store db.Store, kc keychain.Keychain) *textileClient {
	return &textileClient{
		store:            store,
		kc:               kc,
		threads:          nil,
		bucketsClient:    nil,
		netc:             nil,
		uc:               nil,
		ht:               nil,
		isRunning:        false,
		isInitialized:    false,
		Ready:            make(chan bool),
		keypairDeleted:   make(chan bool),
		isConnectedToHub: false,
		hubAuth:          nil,
	}
}

func (tc *textileClient) WaitForReady() chan bool {
	return tc.Ready
}

func (tc *textileClient) requiresRunning() error {
	if tc.isRunning == false || tc.isInitialized == false {
		return errors.New("ran an operation that requires starting and initializing textileClient first")
	}
	return nil
}

func (tc *textileClient) getHubCtx(ctx context.Context) (context.Context, error) {
	ctx, err := tc.hubAuth.GetHubContext(ctx)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

// Starts the Textile Client
func (tc *textileClient) start(ctx context.Context, cfg config.Config) error {
	tc.cfg = cfg
	tc.hubAuth = hub.New(tc.store, tc.kc, cfg)
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
	tc.uc = getUserClient(tc.cfg.GetString(config.TextileHubTarget, ""))
	tc.ht = getHubThreadsClient(tc.cfg.GetString(config.TextileHubTarget, ""))

	tc.isRunning = true

	tc.ConnectToHub(ctx)

	tc.Ready <- true

	// Healthcheck flow
	tc.healthcheck(ctx)
	for {
		timeAfterNextCheck := 60 * time.Second
		// Do more frequent checks if the client is not initialized/running
		if err := tc.requiresRunning(); err != nil {
			timeAfterNextCheck = 5 * time.Second
		}
		select {
		case <-time.After(timeAfterNextCheck):
			tc.healthcheck(ctx)

		// If we get notified that the keypair got deleted, start checking right away
		case <-tc.keypairDeleted:
			tc.healthcheck(ctx)

		// If it's trying to shutdown we return right away
		case <-ctx.Done():
			return nil
		}
	}
}

// adding for testability but if there is a better
// way to do this plz advise
func (tc *textileClient) SetUc(uc UsersClient) {
	tc.uc = uc
}

// notreturning error rn and this helper does
// the logging if connection to hub fails, and
// we continue with startup
func (tc *textileClient) ConnectToHub(ctx context.Context) error {
	// Get the public key to see if we have any
	// Reject right away if not
	_, err := tc.kc.GetStoredPublicKey()
	if err == keychain.ErrKeyPairNotFound {
		return err
	}

	// Attempt to connect to the Hub
	hubctx, err := tc.getHubCtx(ctx)
	if err != nil {
		log.Error("Could not connect to Textile Hub. Starting in offline mode.", err)
		return err
	}

	wasConnectedToHub := tc.isConnectedToHub

	tc.isConnectedToHub = true

	// setup mailbox
	mid, err := tc.uc.SetupMailbox(hubctx)
	if err != nil {
		log.Error("Unable to setup mailbox", err)
		return err
	}

	if wasConnectedToHub == false {
		log.Debug("Mailbox id: " + mid.String())
	}
	return nil
}

func getUserClient(host string) UsersClient {
	hubTarget := host
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

func getHubThreadsClient(host string) *threadsClient.Client {
	hubTarget := host
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

func (tc *textileClient) initialize(ctx context.Context) error {
	buckets, err := tc.listBuckets(ctx)
	if err != nil {
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
		_, err := tc.createBucket(ctx, defaultPersonalBucketSlug)
		if err != nil {
			log.Error("Error creating default bucket", err)
			return err
		}
	}

	tc.isInitialized = true
	return nil
}

// Starts a Textile Client and also initializes default resources for it like a key pair and default bucket.
// Then leaves the process running to attempt to connect or to initialize if it's not already initialized
func (tc *textileClient) Start(ctx context.Context, cfg config.Config) error {
	// Create key pair if not present
	// Start Textile Client
	if err := tc.start(ctx, cfg); err != nil {
		log.Error("Error starting Textile Client", err)
		return err
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

	// Some threads will be on the hub and some will be local, this flag lets you specify
	// where it is
	if hub {
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

// Checks for connection and initialization needs.
func (tc *textileClient) healthcheck(ctx context.Context) {
	log.Debug("Textile Client healthcheck... Start.")

	if tc.isInitialized == false {
		tc.initialize(ctx)
	}

	tc.ConnectToHub(ctx)

	switch {
	case tc.isInitialized == false:
		log.Debug("Textile Client healthcheck... Not initialized yet.")
	case tc.isConnectedToHub == false:
		log.Debug("Textile Client healthcheck... Not connected to hub.")
	default:
		log.Debug("Textile Client healthcheck... OK.")
	}

}

func (tc *textileClient) RemoveKeys() {
	tc.isInitialized = false
	tc.isConnectedToHub = false
	tc.keypairDeleted <- true
}
