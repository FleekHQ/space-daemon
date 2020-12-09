package textile

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/FleekHQ/space-daemon/core/search"

	"github.com/FleekHQ/space-daemon/config"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	iface "github.com/ipfs/interface-go-ipfs-core"

	"github.com/FleekHQ/space-daemon/core/keychain"
	db "github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/core/textile/notifier"
	synchronizer "github.com/FleekHQ/space-daemon/core/textile/sync"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/core/util/address"
	"github.com/FleekHQ/space-daemon/log"
	ma "github.com/multiformats/go-multiaddr"
	threadsClient "github.com/textileio/go-threads/api/client"
	nc "github.com/textileio/go-threads/net/api/client"
	bucketsClient "github.com/textileio/textile/v2/api/bucketsd/client"
	"github.com/textileio/textile/v2/api/common"
	uc "github.com/textileio/textile/v2/api/usersd/client"
	"github.com/textileio/textile/v2/cmd"
	mail "github.com/textileio/textile/v2/mail/local"
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
	filesSearchEngine  search.FilesSearchEngine
	isRunning          bool
	isInitialized      bool
	isSyncInitialized  bool
	Ready              chan bool
	keypairDeleted     chan bool
	shuttingDown       chan bool
	onHealthy          chan error
	onInitialized      chan bool
	cfg                config.Config
	isConnectedToHub   bool
	netc               *nc.Client
	hnetc              *nc.Client
	uc                 UsersClient
	mailEvents         chan mail.MailboxEvent
	hubAuth            hub.HubAuth
	mbNotifier         GrpcMailboxNotifier
	failedHealthchecks int
	sync               synchronizer.Synchronizer
	notifier           bucket.Notifier
	ipfsClient         iface.CoreAPI
	dbListeners        map[string]Listener
	shouldForceRestore bool
	healthcheckMutex   *sync.Mutex
}

// Creates a new Textile Client
func NewClient(
	store db.Store,
	kc keychain.Keychain,
	hubAuth hub.HubAuth,
	uc UsersClient,
	mb Mailbox,
	search search.FilesSearchEngine,
) *textileClient {
	return &textileClient{
		store:              store,
		kc:                 kc,
		threads:            nil,
		bucketsClient:      nil,
		mb:                 mb,
		netc:               nil,
		hnetc:              nil,
		uc:                 uc,
		ht:                 nil,
		hb:                 nil,
		isRunning:          false,
		isInitialized:      false,
		isSyncInitialized:  false,
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
		dbListeners:        make(map[string]Listener),
		shouldForceRestore: false,
		healthcheckMutex:   &sync.Mutex{},
		filesSearchEngine:  search,
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
		tc.store,
		tc.GetModel(),
		tc.kc,
		tc.hubAuth,
		tc.hb,
		tc.ht,
		tc.netc,
		tc.cfg,
		getMirrorBucketFn,
		getLocalBucketFn,
		tc.getBucketContext,
		tc.addListener,
	)

	tc.notifier = notifier.New(tc.sync)

	if err := tc.sync.RestoreQueue(); err != nil {
		log.Warn("Could not restore Textile synchronizer queue. Queue will start fresh.")
	}

	tc.isSyncInitialized = true
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
	addrAPI := cmd.AddrFromStr(tc.cfg.GetString(config.BuckdApiMaAddr, "/ip4/127.0.0.1/tcp/3006"))
	_, host, err := manet.DialArgs(addrAPI)
	if err != nil {
		return errors.New("invalid bucket daemon host provided: " + err.Error())
	}

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

	ipfsNodeAddr := cfg.GetString(config.Ipfsnodeaddr, "/ip4/127.0.0.1/tcp/5001")
	if ipfsNodeAddr == "" {
		ipfsNodeAddr = "/ip4/127.0.0.1/tcp/5001"
	}

	multiAddr, err := ma.NewMultiaddr(ipfsNodeAddr)
	if err != nil {
		cmd.Fatal(err)
	}

	if ic, err := httpapi.NewApi(multiAddr); err != nil {
		cmd.Fatal(err)
	} else {
		tc.ipfsClient = ic
	}

	tc.bucketsClient = buckets
	tc.threads = threads
	tc.netc = netc
	tc.ht = getHubThreadsClient(tc.cfg.GetString(config.TextileHubTarget, ""))
	tc.hb = getHubBucketClient(tc.cfg.GetString(config.TextileHubTarget, ""))
	tc.hnetc = getHubNetworkClient(tc.cfg.GetString(config.TextileHubTarget, ""))

	tc.isRunning = true

	tc.healthcheck(ctx)

	tc.Ready <- true

	// Repeating healthcheck
	for {
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
			return nil
		case <-tc.shuttingDown:
			return nil
		}
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

func getHubTargetOpts(host string) []grpc.DialOption {
	auth := common.Credentials{}
	var opts []grpc.DialOption

	if strings.Contains(host, "443") {
		creds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(creds))
		auth.Secure = true
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithPerRPCCredentials(auth))

	return opts
}

func CreateUserClient(host string) UsersClient {
	opts := getHubTargetOpts(host)

	users, err := uc.NewClient(host, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	return users
}

func getHubThreadsClient(host string) *threadsClient.Client {
	opts := getHubTargetOpts(host)

	tc, err := threadsClient.NewClient(host, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	return tc
}

func getHubNetworkClient(host string) *nc.Client {
	opts := getHubTargetOpts(host)

	n, err := nc.NewClient(host, opts...)
	if err != nil {
		cmd.Fatal(err)
	}

	return n
}

func getHubBucketClient(host string) *bucketsClient.Client {
	opts := getHubTargetOpts(host)

	tc, err := bucketsClient.NewClient(host, opts...)
	if err != nil {
		cmd.Fatal(err)
	}
	return tc
}

func (tc *textileClient) initialize(ctx context.Context) error {
	err := tc.restoreBuckets(ctx)
	if err != nil {
		return err
	}

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

	if err = tc.initSearchIndex(ctx); err != nil {
		log.Error("Error initializing files search index", err)
		return err
	}

	if tc.sync != nil {
		tc.sync.NotifyBucketStartup(defaultPersonalBucketSlug)
	}

	_, err = tc.createDefaultPublicBucket(ctx)
	if err != nil {
		log.Warn("Failed to create default public bucket", "err:"+err.Error())
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

// Used by delete account so we can disable it so it gets
// enabled again during startup
func (tc *textileClient) DisableSync() {
	tc.isSyncInitialized = false
}

// Closes connection to Textile
func (tc *textileClient) Shutdown() error {
	tc.shuttingDown <- true
	tc.isRunning = false
	tc.isInitialized = false
	tc.isSyncInitialized = false
	tc.shouldForceRestore = false

	// Close channels
	close(tc.mailEvents)
	close(tc.Ready)
	close(tc.onHealthy)
	close(tc.keypairDeleted)
	close(tc.shuttingDown)

	tc.closeListeners()

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
	tc.healthcheckMutex.Lock()
	defer tc.healthcheckMutex.Unlock()

	log.Debug("Textile Client healthcheck... Start.")

	if tc.isSyncInitialized == false {
		tc.initializeSync(ctx)
	}

	if tc.isInitialized == false {
		// NOTE: Initialize does not need a hub connection as remote syncing is done in a background process
		tc.initialize(ctx)
	}

	tc.checkHubConnection(ctx)

	if len(tc.dbListeners) == 0 {
		tc.initializeListeners(ctx)
	}

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

func (tc *textileClient) RemoveKeys(ctx context.Context) error {
	if err := tc.hubAuth.ClearCache(); err != nil {
		return err
	}

	if err := tc.clearLocalMailbox(); err != nil {
		return err
	}

	tc.isInitialized = false
	tc.isConnectedToHub = false
	tc.keypairDeleted <- true

	metathreadID, err := utils.NewDeterministicThreadID(tc.kc, utils.MetathreadThreadVariant)
	if err != nil {
		return err
	}

	err = tc.threads.DeleteDB(ctx, metathreadID)
	if err != nil {
		return err
	}

	return nil
}

func (tc *textileClient) GetModel() model.Model {
	return model.New(
		tc.store,
		tc.kc,
		tc.threads,
		tc.ht,
		tc.hubAuth,
		tc.cfg,
		tc.netc,
		tc.hnetc,
		tc.shouldForceRestore,
		tc.filesSearchEngine,
	)
}

func (tc *textileClient) getSecureBucketsClient(baseClient *bucketsClient.Client) *SecureBucketClient {
	isRemote := baseClient == tc.hb
	return NewSecureBucketsClient(baseClient, tc.kc, tc.store, tc.threads, tc.ipfsClient, isRemote, tc.cfg)
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

func (tc *textileClient) AttachSynchronizerNotifier(notif synchronizer.EventNotifier) {
	tc.sync.AttachNotifier(notif)
}

// Initializes dbs from a backup. Returns error if it can't initialize
func (tc *textileClient) RestoreDB(ctx context.Context) error {
	tc.healthcheckMutex.Lock()
	defer tc.healthcheckMutex.Unlock()

	tc.shouldForceRestore = true
	err := tc.initialize(ctx)
	tc.shouldForceRestore = false
	if err != nil {
		tc.kc.DeleteKeypair()
		return err
	}
	return nil
}
