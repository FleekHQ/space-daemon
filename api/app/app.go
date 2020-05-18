package app


import (
	"github.com/FleekHQ/space-poc/api/logger"
	"github.com/gin-gonic/gin"
)

var (
	router = gin.Default()
)

func Start() {
	setupRoutes()

	logger.Info("about to start the application")
	router.Run(":8080")
}