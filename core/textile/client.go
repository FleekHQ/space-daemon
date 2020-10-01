package textile

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FleekHQ/space-daemon/config"

	"github.com/FleekHQ/space-daemon/core/keychain"
	db "github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/core/textile/notifier"
	synchronizer "github.com/FleekHQ/space-daemon/core/textile/sync"
	"github.com/FleekHQ/space-daemon/core/util/address"
	"github.com/FleekHQ/space-daemon/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	nc "github.com/textileio/go-threads/net/api/client"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/api/common"
	uc "github.com/textileio/textile/api/users/client"
	"github.com/textileio/textile/cmd"
	mail "github.com/textileio/textile/mail/local"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const healthcheckFailuresBeforeUnhealthy = 3

var HealthcheckMaxRetriesReachedErr = errors.New(fmt.Sprintf("textile client not initialized after %d attempts", healthcheckFailuresBeforeUnhealthy))

type textileClient struct {
	store              db.Store
	kc                 keychain.Keychain
	threads            *threadsClient.Client
	ht                 *threadsClient.Client
	bucketsClient      *bucketsClient.Client
	mb                 Mailbox
	hb                 *bucketsClient.Client
	isRunning          bool
	isInitialized      bool
	Ready              chan bool
	keypairDeleted     chan bool
	shuttingDown       chan bool
	onHealthy          chan error
	onInitialized      chan bool
	cfg                config.Config
	isConnectedToHub   bool
	netc               *nc.Client
	uc                 UsersClient
	mailEvents         chan mail.MailboxEvent
	hubAuth            hub.HubAuth
	mbNotifier         GrpcMailboxNotifier
	failedHealthchecks int
	sync               synchronizer.Synchronizer
	notifier           bucket.Notifier
}

// Creates a new Textile Client
func NewClient(store db.Store, kc keychain.Keychain, hubAuth hub.HubAuth, uc UsersClient, mb Mailbox) *textileClient {
	return &textileClient{
		store:              store,
		kc:                 kc,
		threads:            nil,
		bucketsClient:      nil,
		mb:                 mb,
		netc:               nil,
		uc:                 uc,
		ht:                 nil,
		hb:                 nil,
		isRunning:          false,
		isInitialized:      false,
		Ready:              make(chan bool),
		keypairDeleted:     make(chan bool),
		shuttingDown:       make(chan bool),
		onHealthy:          make(chan error),
		onInitialized:      make(chan bool),
		mailEvents:         make(chan mail.MailboxEvent),
		isConnectedToHub:   false,
		hubAuth:            hubAuth,
		mbNotifier:         nil,
		failedHealthchecks: 0,
		sync:               nil,
		notifier:           nil,
	}
}

func (tc *textileClient) WaitForReady() chan bool {
	return tc.Ready
}

func (tc *textileClient) WaitForInitialized() chan bool {
	return tc.onInitialized
}

// Returns an error if it exceeds the max amount of attempts
func (tc *textileClient) WaitForHealthy() chan error {
	return tc.onHealthy
}

func (tc *textileClient) IsInitialized() bool {
	return tc.isInitialized
}

// Healthy means initialized and connected to hub
func (tc *textileClient) IsHealthy() bool {
	return tc.isInitialized && tc.isConnectedToHub
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

func (tc *textileClient) initializeSync(ctx context.Context) {
	getLocalBucketFn := func(ctx context.Context, slug string) (bucket.BucketInterface, error) {
		return tc.getBucket(ctx, slug, nil)
	}

	getMirrorBucketFn := func(ctx context.Context, slug string) (bucket.BucketInterface, error) {
		return tc.getBucketForMirror(ctx, slug)
	}

	tc.sync = synchronizer.New(
		tc.store, tc.GetModel(), tc.kc, tc.hubAuth, tc.hb, tc.ht, tc.netc, tc.cfg, getMirrorBucketFn, getLocalBucketFn, tc.getBucketContext,
	)

	tc.notifier = notifier.New(tc.sync)

	if err := tc.sync.RestoreQueue(); err != nil {
		log.Warn("Could not restore Textile synchronizer queue. Queue will start fresh.")
	}

	tc.sync.Start(ctx)
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
	tc.ht = getHubThreadsClient(tc.cfg.GetString(config.TextileHubTarget, ""))
	tc.hb = getHubBucketClient(tc.cfg.GetString(config.TextileHubTarget, ""))

	tc.initializeSync(ctx)

	tc.isRunning = true

	tc.healthcheck(ctx)

	tc.Ready <- true

	// Repeating healthcheck
	healthcheckMutex := &sync.Mutex{}
	for {
		healthcheckMutex.Lock()
		timeAfterNextCheck := 60 * time.Second
		// Do more frequent checks if the client is not initialized/running
		if tc.isConnectedToHub == false || tc.isInitialized == false {
			timeAfterNextCheck = 3 * time.Second
		}

		// If it's trying to shutdown we return right away
		if tc.isRunning == false {
			return nil
		}

		select {
		case <-time.After(timeAfterNextCheck):
			tc.healthcheck(ctx)

		// If we get notified that the keypair got deleted, start checking right away
		case <-tc.keypairDeleted:
			tc.healthcheck(ctx)

		// If it's trying to shutdown we return right away
		case <-ctx.Done():
			healthcheckMutex.Unlock()
			return nil
		case <-tc.shuttingDown:
			healthcheckMutex.Unlock()
			return nil
		}
		healthcheckMutex.Unlock()
	}
}

func (tc *textileClient) checkHubConnection(ctx context.Context) error {
	// Get the public key to see if we have any
	// Reject right away if not
	_, err := tc.kc.GetStoredPublicKey()
	if err != nil {
		tc.isConnectedToHub = false
		return err
	}

	// Attempt to connect to the Hub
	hubctx, err := tc.getHubCtx(ctx)
	if err != nil {
		tc.isConnectedToHub = false
		log.Error("Could not connect to Textile Hub. Starting in offline mode.", err)
		return err
	}

	if tc.isConnectedToHub == false {
		// setup mailbox
		mailbox, err := tc.setupOrCreateMailBox(hubctx)
		if err != nil {
			log.Error("Unable to setup mailbox", err)
			tc.isConnectedToHub = false
			return err
		}
		tc.mb = mailbox

		if err := tc.listenForMessages(hubctx); err != nil {
			tc.isConnectedToHub = false
			log.Error("Could not listen for mailbox messages", err)
			return err
		}
	}

	tc.isConnectedToHub = true

	return nil
}

func CreateUserClient(host string) UsersClient {
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

func getHubBucketClient(host string) *bucketsClient.Client {
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

	tc, err := bucketsClient.NewClient(hubTarget, opts...)
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

	pub, _ := tc.kc.GetStoredPublicKey()
	if pub != nil {
		address := address.DeriveAddress(pub)
		log.Debug("Initializing Textile client", fmt.Sprintf("address:%s", address))
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
	// Non-blocking channel send in case there are no listeners registered
	select {
	case tc.onInitialized <- true:
		log.Debug("Notifying Textile Client init ready")
	default:
		// Do nothing
	}
	log.Debug("Textile Client initialized successfully")
	return nil
}

// Starts a Textile Client and also initializes default resources for it (default bucket and metathread).
// Then leaves the process running to attempt to connect or to initialize if it's not already initialized
func (tc *textileClient) Start(ctx context.Context, cfg config.Config) error {
	// Start Textile Client
	return tc.start(ctx, cfg)
}

// Closes connection to Textile
func (tc *textileClient) Shutdown() error {
	tc.shuttingDown <- true
	tc.isRunning = false

	// Close channels
	close(tc.mailEvents)
	close(tc.Ready)
	close(tc.onHealthy)
	close(tc.keypairDeleted)
	close(tc.shuttingDown)

	if err := tc.bucketsClient.Close(); err != nil {
		return err
	}

	if err := tc.threads.Close(); err != nil {
		return err
	}

	tc.sync.Shutdown()

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

func (tc *textileClient) GetFailedHealthchecks() int {
	return tc.failedHealthchecks
}

// Checks for connection and initialization needs.
func (tc *textileClient) healthcheck(ctx context.Context) {
	log.Debug("Textile Client healthcheck... Start.")

	if tc.isInitialized == false {
		// NOTE: Initialize does not need a hub connection as remote syncing is done in a background process
		tc.initialize(ctx)
	}

	tc.checkHubConnection(ctx)

	switch {
	case tc.isInitialized == false:
		log.Debug("Textile Client healthcheck... Not initialized yet.")
		tc.failedHealthchecks = tc.failedHealthchecks + 1
	case tc.isConnectedToHub == false:
		log.Debug("Textile Client healthcheck... Not connected to hub.")
		tc.failedHealthchecks = tc.failedHealthchecks + 1
	default:
		log.Debug("Textile Client healthcheck... OK.")
		tc.failedHealthchecks = 0
		// Non-blocking channel send in case there are no listeners registered
		select {
		case tc.onHealthy <- nil:
			log.Debug("Notifying health OK")
		default:
			// Do nothing
		}
	}

	if tc.failedHealthchecks >= 3 {
		// Non-blocking channel send in case there are no listeners registered
		select {
		case tc.onHealthy <- HealthcheckMaxRetriesReachedErr:
			log.Debug("Notifying healthcheck: max attempts surpassed")
			tc.failedHealthchecks = 0
		default:
			// Do nothing
		}
	}
}

func (tc *textileClient) RemoveKeys() error {
	if err := tc.hubAuth.ClearCache(); err != nil {
		return err
	}

	if err := tc.clearLocalMailbox(); err != nil {
		return err
	}

	tc.isInitialized = false
	tc.isConnectedToHub = false
	tc.keypairDeleted <- true

	return nil
}

func (tc *textileClient) GetModel() model.Model {
	return model.New(tc.store, tc.kc, tc.threads, tc.hubAuth)
}

func (tc *textileClient) requiresHubConnection() error {
	if err := tc.requiresRunning(); err != nil {
		return err
	}

	if tc.isConnectedToHub == false || tc.mb == nil {
		return errors.New("ran an operation that requires connection to hub")
	}
	return nil
}
