package main

import (
	"context"
	"log"
	"time"

	"github.com/textileio/textile/cmd"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/textileio/textile/core"
)

func StartTextile(ctx context.Context) (*core.Textile, error) {
	textile, err := core.NewTextile(ctx, core.Config{
		RepoPath:           ".buckd/repo",
		AddrAPI:            cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3006"),
		AddrAPIProxy:       cmd.AddrFromStr("/ip4/127.0.0.1/tcp/3007"),
		AddrThreadsHost:    cmd.AddrFromStr("/ip4/0.0.0.0/tcp/4006"),
		AddrIPFSAPI:        cmd.AddrFromStr("/ip4/127.0.0.1/tcp/5001"),
		AddrGatewayHost:    cmd.AddrFromStr("/ip4/127.0.0.1/tcp/8006"),
		AddrGatewayURL:     "http://127.0.0.1:8006",
		AddrMongoURI:       "mongodb://127.0.0.1:27027",
		MongoName:          "buckets",
		ThreadsConnManager: connmgr.NewConnManager(10, 50, time.Second*20),
		Debug:              false,
	})
	if err != nil {
		return nil, err
	}

	textile.Bootstrap()

	return textile, nil
}

func main() {
	ctx, _ := context.WithCancel(context.Background())
	textile, err := StartTextile(ctx)
	if err != nil {
		log.Fatalf("Failed to start textile: %+v", err)
	}

	if err != textile.Close() {
		log.Fatalf("Error stopping textile: %+v", err)
	}

	textile, err = StartTextile(ctx)
	if err != nil {
		log.Fatalf("Failed to start textile a second time: %+v", err)
	}

	if err != textile.Close() {
		log.Fatalf("Error stopping textile second time: %+v", err)
	}
}
