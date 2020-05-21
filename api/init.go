package api

import (
	"context"

	"encoding/hex"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/keychain"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/log"
)

func Start(ctx context.Context, cfg config.Config) {
	setupRoutes()

	var db *store.Store

	if storePath, err := cfg.GetString(config.SpaceStorePath, nil); err != nil {
		log.Info("space.storePath not found in space.json. Defaulting store folder.")
		db = store.New()
	} else {
		db = store.New(storePath)
	}

	if err := db.Set([]byte("A"), []byte("B")); err != nil {
		log.Error("error", err)
		return
	}

	if val, err := db.Get([]byte("A")); err != nil {
		log.Error("error", err)
	} else {
		log.Info("Got store response")
		log.Info(string(val))
	}

	log.Info("Generating key pair...")
	kc := keychain.New(db)
	if pub, err := kc.GenerateKeyPair(); err != nil {
		log.Error("Error while generating key pair", err)
	} else {
		log.Info(hex.EncodeToString(pub))
	}

	log.Info("about to start the application")
	go func() {
		router.Run(":8080")
	}()
}
