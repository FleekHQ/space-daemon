package app

import (
	"github.com/FleekHQ/space-poc/api"
	"github.com/FleekHQ/space-poc/core/watcher"
)

// Entry point for the app
func Start() {
	// starting the RPC server
	api.Start()
	watcher.StartWatcher()
	// TODO: add watcher service for local FS
	// TODO: add listener service for bucket changes
}