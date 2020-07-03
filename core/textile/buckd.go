package textile

import (
	"context"
	"fmt"
	"os/user"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/textile/cmd"
	"github.com/textileio/textile/core"
)

var IpfsAddr string
var MongoUsr string
var MongoPw string
var MongoHost string

type TextileBuckd struct {
	textile   *core.Textile
	IsRunning bool
	ready     chan bool
	cfg       config.Config
}

func NewBuckd(cfg config.Config) *TextileBuckd {
	return &TextileBuckd{
		ready: make(chan bool),
		cfg:   cfg,
	}
}

func (tb *TextileBuckd) Start(ctx context.Context) error {
	IpfsAddr = tb.cfg.GetString(config.Ipfsaddr, "/ip4/127.0.0.1/tcp/5001")
	MongoUsr = tb.cfg.GetString(config.Mongousr, "")
	MongoPw = tb.cfg.GetString(config.Mongopw, "")
	MongoHost = tb.cfg.GetString(config.Mongohost, "")

	addrAPI := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3006")
	addrAPIProxy := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3007")
	addrThreadsHost := cmd.AddrFromStr("/ip4/0.0.0.0/tcp/4006")
	// TODO: replace with local blockstore
	addrIpfsAPI := cmd.AddrFromStr(IpfsAddr)

	addrGatewayHost := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/8006")
	addrGatewayURL := "http://127.0.0.1:8006"

	// PLACEHOLDER: filecoin settings

	// TODO: replace with embedded store
	addrMongoURI := "mongodb+srv://" + MongoUsr + ":" + MongoPw + "@" + MongoHost

	// TODO: setup logging
	// if logFile != "" {
	// 	util.SetupDefaultLoggingConfig(logFile)
	// }

	// TODO: on shared bucket creation, add hub as replicator
	// use dbinfo to get keys to give to host, to get hostid
	// use textile client.GetHostId (against hub we want to
	// replicate). it will give back a couple just dont use
	// local one

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	textile, err := core.NewTextile(ctx, core.Config{
		RepoPath:        usr.HomeDir + "/.buckd/repo",
		AddrAPI:         addrAPI,
		AddrAPIProxy:    addrAPIProxy,
		AddrThreadsHost: addrThreadsHost,
		AddrIPFSAPI:     addrIpfsAPI,
		AddrGatewayHost: addrGatewayHost,
		AddrGatewayURL:  addrGatewayURL,
		//AddrPowergateAPI: addrPowergateApi,
		AddrMongoURI: addrMongoURI,
		//UseSubdomains:    config.Viper.GetBool("gateway.subdomains"),
		MongoName: "buckets",
		//DNSDomain:        dnsDomain,
		//DNSZoneID:        dnsZoneID,
		//DNSToken:         dnsToken,
		Debug: false,
	})
	if err != nil {
		log.Fatal(err)
	}

	textile.Bootstrap()

	log.Info("Welcome to bucket", fmt.Sprintf("peerID:%s", textile.HostID().String()))
	tb.textile = textile
	tb.IsRunning = true
	return nil
}

func (tb *TextileBuckd) Stop() error {
	tb.IsRunning = false
	tb.textile.Close()
	close(tb.ready)
	// TODO: what else
	return nil
}

func (tb *TextileBuckd) Shutdown() error {
	return tb.Stop()
}
