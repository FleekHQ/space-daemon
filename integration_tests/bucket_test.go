package integration_tests

import (
	"context"
	"flag"
	"os"
	"sync"
	"testing"

	"github.com/FleekHQ/space-daemon/config"
	spaceEnv "github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

var (
	textileClient textile.Client
	ctx           context.Context
)

var (
	devMode        = flag.Bool("dev", true, "defaulting to true for testing")
	ipfsaddr       = flag.String("ipfs_addr", "", "Set IPFS_ADDR for this value")
	mongousr       = flag.String("mongo_usr", "", "Required. Set MONGO_USR for this value")
	mongopw        = flag.String("mongo_pw", "", "Required. MONGO_PW")
	mongohost      = flag.String("mongo_host", "", "Required")
	mongorepset    = flag.String("mongo_replica_set", "", "Required")
	spaceapi       = flag.String("service_api_url", "", "Required")
	spacehubauth   = flag.String("service_hub_auth_url", "", "Required")
	textilehub     = flag.String("txl_hub_target", "", "Required. Set TXL_HUB_TARGET")
	textilehubma   = flag.String("txl_hub_ma", "", "Required")
	textilethreads = flag.String("txl_threads_target", "", "Required")
	appInstances   atomic.Int32
	appIsRunning   bool
	appFixtureLock sync.Mutex
)

func GetTestConfig() (*config.Flags, config.Config, spaceEnv.SpaceEnv) {
	flags := &config.Flags{
		Ipfsaddr:             *ipfsaddr,
		Mongousr:             *mongousr,
		Mongopw:              *mongopw,
		Mongohost:            *mongohost,
		Mongorepset:          *mongorepset,
		ServicesAPIURL:       *spaceapi,
		ServicesHubAuthURL:   *spacehubauth,
		DevMode:              *devMode == true,
		TextileHubTarget:     *textilehub,
		TextileHubMa:         *textilehubma,
		TextileThreadsTarget: *textilethreads,
	}

	// env
	env := spaceEnv.New()

	// load configs
	return flags, config.NewMap(env, flags), env
}

func setup() {
	ctx = context.Background()

	_, cfg, _ := GetTestConfig()
	st := store.New(
		store.WithPath("/tmp"),
	)
	if err := st.Open(); err != nil {
		return
	}

	buckd := textile.NewBuckd(cfg)
	err := buckd.Start(ctx)
	if err != nil {
		return
	}

	textileClient = textile.NewClient(st)
	go func() {
		textileClient.Start(ctx, cfg)
	}()
	<-textileClient.WaitForReady()
}

func shutdown() {
	textileClient.Shutdown()
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestCopyFiles(t *testing.T) {
	b1, e := textileClient.CreateBucket(ctx, "testbucket1")
	if e != nil {
		t.Error("error creating bucket: ", e)
		return
	}
	assert.NotNil(t, b1)

	b1.CreateDirectory(ctx, "folderA")

	f, _ := os.Open("./test_files/file1")
	p1, p2, _ := b1.UploadFile(ctx, "folderA/file1", f)
	assert.NotNil(t, p1)
	assert.NotNil(t, p2)

	ls, _ := b1.ListDirectory(ctx, "")
	for _, l := range ls.Item.Items {
		t.Log(l.GetName())
	}

	ls1, _ := b1.ListDirectory(ctx, "folderA")
	for _, l := range ls1.Item.Items {
		t.Log(l.GetName())
	}
}
