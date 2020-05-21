package api

import (
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/logger"
)

func Start() {
	setupRoutes()
	s := store.New()
	if err := s.Set([]byte("A"), []byte("B")); err != nil {
		logger.Error("error", err)
		return
	}

	if val, err := s.Get([]byte("A")); err != nil {
		logger.Error("error", err)
	} else {
		logger.Info("Got store response")
		logger.Info(string(val))
	}

	logger.Info("about to start the application")
	router.Run(":8080")
}
