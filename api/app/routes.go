package app

import (
	"github.com/FleekHQ/space-poc/api/controllers/health"
)

func setupRoutes() {
	router.GET("/health", health.Ping)
}