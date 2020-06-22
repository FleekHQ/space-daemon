package textile

import (
	"context"
	"fmt"
	"os"

	"github.com/FleekHQ/space-poc/log"
	"github.com/textileio/textile/cmd"
	"github.com/textileio/textile/core"
)

type TextileBuckd struct {
	isRunning bool
	ready     chan bool
}

func NewBuckd() Buckd {
	return &TextileBuckd{
		ready: make(chan bool),
	}
}

func (tb *TextileBuckd) Start() error {
	addrAPI := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3006")
	addrAPIProxy := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3007")
	addrThreadsHost := cmd.AddrFromStr("/ip4/0.0.0.0/tcp/4006")
	// TODO: replace with local blockstore
	// TODO: get value from build time
	addrIpfsAPI := cmd.AddrFromStr("/ip4/34.223.251.246/tcp/5001")

	addrGatewayHost := cmd.AddrFromStr("/ip4/127.0.0.1/tcp/8006")
	addrGatewayURL := "http://127.0.0.1:8006"

	// PLACEHOLDER: filecoin settings

	// TODO: replace with embedded store
	// TODO: get value from build time
	pw := os.Getenv("MONGOPW")
	addrMongoURI := "mongodb+srv://root:" + pw + "@textile-bucksd-dev-eg4f5.mongodb.net"
	// <dbname>?retryWrites=true&w=majority"

	// TODO: setup logging
	// if logFile != "" {
	// 	util.SetupDefaultLoggingConfig(logFile)
	// }

	// TODO: on shared bucket creation, add hub as replicator
	// use dbinfo to get keys to give to host, to get hostid
	// use textile client.GetHostId (against hub we want to
	// replicate). it will give back a couple just dont use
	// local one

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	textile, err := core.NewTextile(ctx, core.Config{
		RepoPath:        "~/.buckd/repo",
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
	defer textile.Close()
	textile.Bootstrap()

	fmt.Println("Welcome to Buckets!")
	fmt.Println("Your peer ID is " + textile.HostID().String())
	tb.isRunning = true
	tb.ready <- true
	return nil
}

func (tb *TextileBuckd) WaitForReady() chan bool {
	return tb.ready
}

func (tb *TextileBuckd) Stop() error {
	tb.isRunning = false
	close(tb.ready)
	// TODO: what else
	return nil
}
