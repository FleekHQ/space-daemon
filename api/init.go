package api

import (
	"context"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/log"
)

func Start(ctx context.Context, cfg config.Config) {
	setupRoutes()

	log.Info("about to start the application")
	go func() {
		router.Run(":8080")
	}()
}
