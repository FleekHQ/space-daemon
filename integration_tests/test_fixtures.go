package integration_tests

import (
	"context"
	"fmt"
	"sync"
	"testing"

	spaceApp "github.com/FleekHQ/space-daemon/app"

	"go.uber.org/atomic"

	"github.com/FleekHQ/space-daemon/grpc/pb"

	"google.golang.org/grpc"

	"github.com/FleekHQ/space-daemon/config"
	spaceEnv "github.com/FleekHQ/space-daemon/core/env"
	"github.com/namsral/flag"
	"github.com/stretchr/testify/assert"
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
	app            *spaceApp.App
)

// GetTestConfig returns a ConfigMap instance instantiated using the env variables
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
	return flags, config.NewMap(flags), env
}

type AppFixture struct {
	errChan chan error
}

func NewAppFixture() *AppFixture {
	return &AppFixture{
		errChan: make(chan error),
	}
}

// StartApp ensures that only a single global app instance is launched
// it uses a mutex lock to ensure everything is initialized properly
// so all tests can call NewAppFixture().StartApp(t) without worrying
// about clashes with multiple app instances launching
// Also cleanup isi done seamlessly
func (a *AppFixture) StartApp(t *testing.T) {
	appFixtureLock.Lock()
	defer appFixtureLock.Unlock()

	if appIsRunning {
		return
	}

	_, cfg, env := GetTestConfig()
	app = spaceApp.New(cfg, env)

	ctx, cancelCtx := context.WithCancel(context.Background())
	var err error
	go func() {
		a.errChan <- app.Start(ctx)
		a.errChan = nil
	}()

	select {
	case <-app.WaitForReady():
	case err = <-a.errChan:
	}

	assert.NoError(t, err, "app.Start() Failed")

	appInstances.Inc()
	appIsRunning = true

	t.Cleanup(func() {
		instances := appInstances.Dec()
		if instances == 0 {
			cancelCtx()
			if a.errChan != nil {
				<-a.errChan
			}
			appIsRunning = false
		}
	})
}

func (a *AppFixture) GrpcClient(t *testing.T) *grpc.ClientConn {
	_, cfg, _ := GetTestConfig()
	conn, err := grpc.Dial(
		fmt.Sprintf(":%d", cfg.GetInt(config.SpaceServerPort, 9999)),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	assert.Nil(t, err, "Error connecting to grpc")

	t.Cleanup(func() {
		err := conn.Close()
		assert.NoError(t, err, "Error closing grpc connection")
	})

	return conn
}

func (a *AppFixture) SpaceApiClient(t *testing.T) pb.SpaceApiClient {
	conn := a.GrpcClient(t)
	return pb.NewSpaceApiClient(conn)
}
