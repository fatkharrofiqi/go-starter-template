package config

import (
	"github.com/gin-gonic/gin"
)

// NewGin initializes a Gin engine with Logrus-based logging
func NewGin() *gin.Engine {
	// Create a new Gin instance
	engine := gin.Default()

	// gin.SetMode(gin.ReleaseMode)

	return engine
}
