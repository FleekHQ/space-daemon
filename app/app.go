package app

import (
	"context"
	"encoding/hex"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/keychain"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/core/watcher"
	"github.com/FleekHQ/space-poc/grpc"
	"github.com/FleekHQ/space-poc/log"
)

// Entry point for the app
func Start(ctx context.Context, cfg config.Config) {
	// init store
	store := store.New(
		store.WithPath(cfg.GetString(config.SpaceStorePath, "")),
	)

	// Generating key pair
	log.Info("Generating key pair...")
	kc := keychain.New(store)
	if pub, err := kc.GenerateKeyPair(); err != nil {
		log.Error("Error while generating key pair", err)
		panic(err)
	} else {
		log.Info(hex.EncodeToString(pub))
	}

	// TODO: inject store and kc to gRPC params?
	// starting the RPC server
	srv := grpc.New(
		grpc.WithPort(cfg.GetInt(config.SpaceServerPort, 0)),
	)
	srv.Start(ctx)
	watcher.Start(ctx, cfg)
	// TODO: add listener services for bucket changes
}
