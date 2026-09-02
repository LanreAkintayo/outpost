package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/LanreAkintayo/outpost/internal/config"
)

// New initializes and configures a *gin.Engine with middlewares and route groups.
// Because it returns a pure *gin.Engine, it can be tested directly with httptest
// without needing a live network listener.
func New(cfg *config.Config, log zerolog.Logger) *gin.Engine {
	// Set Gin mode (debug, release, test) from configuration
	gin.SetMode(cfg.Server.GinMode)

	r := gin.New()

	// Global Middlewares
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "outpost",
		})
	})

	// API v1 Route Group (domain routes will be registered here)
	v1 := r.Group("/api/v1")
	{
		// Placeholder for v1 routes
		_ = v1
	}

	return r
}

// corsMiddleware sets standard Cross-Origin Resource Sharing headers.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
