package textile

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/textileio/powergate/util"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/config"

	"github.com/pkg/errors"

	"github.com/textileio/powergate/api/server"
)

type PowergateDaemon struct {
	server *server.Server
	config server.Config
}

func StartPowd(cfg config.Config) (*PowergateDaemon, error) {
	repPath, err := getRepoPath(
		filepath.Join(cfg.GetString(config.SpaceStorePath, ""), ".powergate"),
		true,
	)

	if err != nil {
		return nil, err
	}

	// because we are running in Devnet === true, all config are default set to work with it
	// make sure you have run `make localnet-up` first.
	// TODO: get addresses and ports from the config variable
	serverConfig := server.Config{
		Devnet:             true,
		WalletInitialFunds: *big.NewInt(4000000000000000),
		//IpfsAPIAddr:          util.MustParseAddr(cfg.GetString(config.Ipfsaddr, "/ip4/127.0.0.1/tcp/5001")),
		IpfsAPIAddr:          util.MustParseAddr("/ip4/127.0.0.1/tcp/5001"),
		LotusAddress:         util.MustParseAddr("/ip4/127.0.0.1/tcp/7777"),
		LotusAuthToken:       "",
		LotusMasterAddr:      "",
		AutocreateMasterAddr: false,
		GrpcServerOpts:       nil,
		GrpcHostNetwork:      "tcp",
		GrpcHostAddress:      util.MustParseAddr("/ip4/0.0.0.0/tcp/5005"),
		GrpcWebProxyAddress:  "0.0.0.0:6005",
		GatewayHostAddr:      "0.0.0.0:7001",
		RepoPath:             repPath,
		MaxMindDBFolder:      "./iplocation",
	}
	newServer, err := server.NewServer(serverConfig)

	if err != nil {
		return nil, errors.Wrap(err, "powergate server failed to start")
	}

	log.Info("Powergate server started")

	return &PowergateDaemon{
		server: newServer,
		config: serverConfig,
	}, nil
}

func (p *PowergateDaemon) Shutdown() error {
	p.server.Close()
	if p.config.Devnet {
		if err := os.RemoveAll(p.config.RepoPath); err != nil {
			log.Warn("Error cleaning up temporary powergate repo path", "err:"+err.Error())
		}
	}
	return nil
}

func getRepoPath(repoPath string, devnet bool) (string, error) {
	if devnet {
		repoPath, err := ioutil.TempDir("", ".powergate-*")
		if err != nil {
			return "", fmt.Errorf("generating temp for repo folder: %s", err)
		}
		return repoPath, nil
	}

	if strings.Contains(repoPath, "~") {
		expandedPath, err := homedir.Expand(repoPath)
		if err != nil {
			log.Error("expanding homedir: %s", err)
			return "", err
		}
		repoPath = expandedPath
	}
	return repoPath, nil
}
