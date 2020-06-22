package textile

import (
	"context"
	"fmt"
	"os"
	"os/user"

	"github.com/FleekHQ/space-poc/log"
	"github.com/textileio/textile/cmd"
	"github.com/textileio/textile/core"
)

var IpfsAddr string
var MongoUsr string
var MongoPw string
var MongoHost string

type TextileBuckd struct {
	textile   *core.Textile
	isRunning bool
	ready     chan bool
}

func NewBuckd() Buckd {
	return &TextileBuckd{
		ready: make(chan bool),
	}
}

func (tb *TextileBuckd) Start(ctx context.Context) error {
	// TODO: get value from build time instead
	IpfsAddr = os.Getenv("IPFS_ADDR")
	MongoUsr = os.Getenv("MONGO_USR")
	MongoPw = os.Getenv("MONGO_PW")
	MongoHost = os.Getenv("MONGO_HOST")

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

	fmt.Println("Welcome to Buckets!")
	fmt.Println("Your peer ID is " + textile.HostID().String())
	tb.textile = textile
	tb.isRunning = true
	tb.ready <- true
	return nil
}

func (tb *TextileBuckd) WaitForReady() chan bool {
	return tb.ready
}

func (tb *TextileBuckd) Stop() error {
	tb.isRunning = false
	tb.textile.Close()
	close(tb.ready)
	// TODO: what else
	return nil
}
