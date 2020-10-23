package integration_tests_test

import (
	"context"
	"testing"

	"github.com/FleekHQ/space-daemon/app"
	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	spaceApp     *app.App
	appErrChan   chan error
	appCtx       context.Context
	cancelAppCtx context.CancelFunc
)

func TestIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IntegrationTests Suite")
}

// GetTestConfig returns a ConfigMap instance instantiated using the env variables
func GetTestConfig() (*config.Flags, config.Config, env.SpaceEnv) {
	flags := &config.Flags{
		Ipfsaddr:             "/ip4/127.0.0.1/tcp/5001",
		Ipfsnode:             true,
		Ipfsnodeaddr:         "/ip4/127.0.0.1/tcp/5001",
		Ipfsnodepath:         "",
		Mongousr:             "root",
		Mongopw:              "Jkvv7Cua6A3Ysji",
		Mongohost:            "textile-bucksd-dev-shard-00-00-eg4f5.mongodb.net:27017,textile-bucksd-dev-shard-00-01-eg4f5.mongodb.net:27017,textile-bucksd-dev-shard-00-02-eg4f5.mongodb.net:27017",
		Mongorepset:          "textile-bucksd-dev-shard-0",
		DevMode:              false,
		ServicesAPIURL:       "https://td4uiovozc.execute-api.us-west-2.amazonaws.com/dev",
		VaultAPIURL:          "https://f4nmmmkstb.execute-api.us-west-2.amazonaws.com/dev",
		VaultSaltSecret:      "WXpKd2JrMUlUbXhhYW10M1RWUkNlV0Z0YkhCYU1tUn",
		ServicesHubAuthURL:   "wss://gqo1oqz055.execute-api.us-west-2.amazonaws.com/dev",
		TextileHubTarget:     "textile-hub-dev.fleek.co:3006",
		TextileHubMa:         "/dns4/textile-hub-dev.fleek.co/tcp/4006/p2p/12D3KooWEYHGowTJYj2fA8c17DPD5wTXJ8dpZ4XCuMCtwCxGDVpx",
		TextileThreadsTarget: "textile-hub-dev.fleek.co:3006",
		TextileHubGatewayUrl: "https://hub-dev.space.storage",
		TextileUserKey:       "bqrbervcjp6hwza43y2bohig23e",
		TextileUserSecret:    "bgalcfm2stf7b3du5zw26bw4y66gmnd64gfphg4i",
	}

	// env
	spaceEnv := env.New()

	// load configs
	return flags, config.NewMap(flags), spaceEnv
}

var _ = BeforeSuite(func() {
	appCtx, cancelAppCtx = context.WithCancel(context.Background())
	_, cfg, env := GetTestConfig()
	spaceApp = app.New(cfg, env)
	var err error
	appErrChan = make(chan error)
	go func() {
		appErrChan <- spaceApp.Start(appCtx)
	}()

	select {
	case <-spaceApp.WaitForReady():
	case err = <-appErrChan:
	}

	Expect(err).NotTo(HaveOccurred(), "space app failed to start")
})

var _ = AfterSuite(func() {
	cancelAppCtx()
	err := <-appErrChan
	Expect(err).NotTo(HaveOccurred(), "failed to shutdown app")
	Expect(spaceApp.IsRunning).Should(Equal(false), "app did not shutdown in time")
})
