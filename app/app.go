package app

import (
	"context"
	"log"

	"github.com/FleekHQ/space-poc/core/synchronizers/bucketsync"
	tc "github.com/FleekHQ/space-poc/core/textile/client"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/store"
	w "github.com/FleekHQ/space-poc/core/watcher"
	"github.com/FleekHQ/space-poc/grpc"
)

// Entry point for the app
func Start(ctx context.Context, cfg config.Config) {
	// init store
	store := store.New(
		store.WithPath(cfg.GetString(config.SpaceStorePath, "")),
	)

	if err := store.Open(); err != nil {
		panic(err)
	}

	// TODO: Add `defer store.Close()` inside server shutdown

	// starting the RPC server
	srv := grpc.New(
		store,
		grpc.WithPort(cfg.GetInt(config.SpaceServerPort, 0)),
	)
	srv.Start(ctx)
	watcher, err := w.New(w.WithPaths(cfg.GetString(config.SpaceFolderPath, "")))
	if err != nil {
		log.Fatal(err)
		return
	}

	textileClient := tc.New(store)

	// Testing bucket creation here
	if err := textileClient.CreateBucket("my-bucket"); err != nil {
		log.Fatal("error creating bucket", err)
	} else {
		log.Printf("Created bucket successfully")
	}

	sync := bucketsync.New(watcher, textileClient)

	err = sync.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = watcher.Watch(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// TODO: add listener services for bucket changes
}
