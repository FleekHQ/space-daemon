package fixtures

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"
	. "github.com/onsi/gomega"
	"github.com/phayes/freeport"
)

// GetTestConfig returns a ConfigMap instance instantiated using the env variables
func GetTestConfig() (*config.Flags, config.Config, env.SpaceEnv) {
	homeDir, err := os.UserHomeDir()
	Expect(err).NotTo(HaveOccurred())

	freePorts, err := freeport.GetFreePorts(8)
	Expect(err).NotTo(HaveOccurred())

	flags := config.Flags{
		Ipfsaddr:               "/ip4/127.0.0.1/tcp/5001",
		Ipfsnode:               false, // use external ipfs node
		Ipfsnodeaddr:           "/ip4/127.0.0.1/tcp/5001",
		Ipfsnodepath:           filepath.Join(homeDir, ".fleek-space-ipfs-node-test"),
		DevMode:                false,
		ServicesAPIURL:         os.Getenv("SERVICES_API_URL"),
		SpaceStorageSiteUrl:    os.Getenv("SPACE_STORAGE_SITE_URL"),
		VaultAPIURL:            os.Getenv("VAULT_API_URL"),
		VaultSaltSecret:        os.Getenv("VAULT_SALT_SECRET"),
		ServicesHubAuthURL:     os.Getenv("SERVICES_HUB_AUTH_URL"),
		TextileHubTarget:       os.Getenv("TXL_HUB_TARGET"),
		TextileHubMa:           os.Getenv("TXL_HUB_MA"),
		TextileThreadsTarget:   os.Getenv("TXL_THREADS_TARGET"),
		TextileHubGatewayUrl:   os.Getenv("TXL_HUB_GATEWAY_URL"),
		TextileUserKey:         os.Getenv("TXL_USER_KEY"),
		TextileUserSecret:      os.Getenv("TXL_USER_SECRET"),
		SpaceStorePath:         filepath.Join(homeDir, ".fleek-space-"+RandomPathName()),
		RpcServerPort:          freePorts[1],
		RpcProxyServerPort:     freePorts[2],
		RestProxyServerPort:    freePorts[3],
		BuckdPath:              filepath.Join(homeDir, ".fleek-space-buckd-"+RandomPathName()),
		BuckdApiMaAddr:         fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", freePorts[4]),
		BuckdApiProxyMaAddr:    fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", freePorts[5]),
		BuckdThreadsHostMaAddr: fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", freePorts[6]),
		BuckdGatewayPort:       freePorts[7],
		LogLevel:               "info",
	}

	// env
	spaceEnv := env.New()

	// load configs
	return &flags, config.NewMap(&flags), spaceEnv
}
