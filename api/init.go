package api

import "github.com/FleekHQ/space-poc/logger"

func Start() {
	setupRoutes()

	logger.Info("about to start the application")
	router.Run(":8080")
}
