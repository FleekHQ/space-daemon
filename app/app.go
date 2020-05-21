package app

import (
	"context"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/watcher"
	"github.com/FleekHQ/space-poc/grpc"
)

// Entry point for the app
func Start(ctx context.Context, cfg config.Config) {
	// api.Start(ctx, cfg)

	// starting the RPC server
	srv := grpc.New(grpc.WithPort(cfg.GetInt(config.SpaceServerPort, 0)))
	srv.Start(ctx)
	watcher.Start(ctx, cfg)
	// TODO: add watcher services for local FS
	// TODO: add listener services for bucket changes
}