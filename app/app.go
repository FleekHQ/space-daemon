package app

import (
	"context"
	"github.com/FleekHQ/space-poc/api"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/watcher"
)

// Entry point for the app
func Start(ctx context.Context, cfg config.Config) {
	// starting the RPC server
	api.Start(ctx, cfg)
	watcher.Start(ctx, cfg)
	// TODO: add watcher service for local FS
	// TODO: add listener service for bucket changes
}