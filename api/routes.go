package api

import (
	"github.com/FleekHQ/space-poc/api/controllers/health"
	"github.com/gin-gonic/gin"
)

var (
	router = gin.Default()
)


func setupRoutes() {
	router.GET("/health", health.Ping)

}