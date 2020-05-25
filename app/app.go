package app

import (
	"context"
	"log"
	"os"

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
	err = watcher.Watch(ctx, func(e w.UpdateEvent, fileInfo os.FileInfo, newPath, oldPath string) {
		log.Printf(
			"Event: %s\nNewPath: %s\nOldPath: %s\nFile Name: %s\n",
			e.String(),
			newPath,
			oldPath,
			fileInfo.Name(),
		)
	})
	if err != nil {
		log.Fatal(err)
	}
	// TODO: add listener services for bucket changes
}
