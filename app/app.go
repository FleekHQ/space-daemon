package app

import (
	"context"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/core/watcher"
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
	watcher.Start(ctx, cfg)
	// TODO: add listener services for bucket changes
}
