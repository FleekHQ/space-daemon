package textile

import (
	"context"
	"fmt"

	"github.com/FleekHQ/space-poc/log"
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
	addrApi := "/ip4/127.0.0.1/tcp/3006"
	addrApiProxy := "/ip4/127.0.0.1/tcp/3007"
	addrThreadsHost := "/ip4/0.0.0.0/tcp/4006"
	// TODO: replace with local blockstore
	addrIpfsApi := "TODO: get from build process env var"

	addrGatewayHost := "/ip4/127.0.0.1/tcp/8006"
	addrGatewayUrl := "http://127.0.0.1:8006"

	// PLACEHOLDER: filecoin settings

	// TODO: replace with embedded db
	addrMongoUri := "TODO: get from build process env var"

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
		RepoPath:        config.Viper.GetString("repo"),
		AddrAPI:         addrApi,
		AddrAPIProxy:    addrApiProxy,
		AddrThreadsHost: addrThreadsHost,
		AddrIPFSAPI:     addrIpfsApi,
		AddrGatewayHost: addrGatewayHost,
		AddrGatewayURL:  addrGatewayUrl,
		//AddrPowergateAPI: addrPowergateApi,
		AddrMongoURI: addrMongoUri,
		//UseSubdomains:    config.Viper.GetBool("gateway.subdomains"),
		MongoName: "buckets",
		//DNSDomain:        dnsDomain,
		//DNSZoneID:        dnsZoneID,
		//DNSToken:         dnsToken,
		Debug: config.Viper.GetBool("log.debug"),
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
	close(tt.ready)
	// TODO: what else
	return nil
}
