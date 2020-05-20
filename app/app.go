package app

import "github.com/FleekHQ/space-poc/api"

// Entry point for the app
func Start() {
	// starting the RPC server
	api.Start()

	// TODO: add watcher service for local FS
	// TODO: add listener service for bucket changes
}