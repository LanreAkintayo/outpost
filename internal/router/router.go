package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/LanreAkintayo/outpost/internal/config"
	"github.com/LanreAkintayo/outpost/internal/middleware"
)

// RouteRegistrar defines a component capable of mounting its endpoints onto a Gin router group.
type RouteRegistrar interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

// RouterParams encapsulates dependencies for building the HTTP router.
type RouterParams struct {
	Config          *config.Config
	Logger          zerolog.Logger
	AuthMiddleware  gin.HandlerFunc
	PublicRoutes    []RouteRegistrar
	ProtectedRoutes []RouteRegistrar
}

// New initializes and configures a *gin.Engine with middlewares and route groups.
func New(params RouterParams) *gin.Engine {
	gin.SetMode(params.Config.Server.GinMode)

	r := gin.New()

	// Global Middlewares (Executed in order)
	r.Use(middleware.Recovery(params.Logger))
	r.Use(middleware.RequestLogger(params.Logger))
	r.Use(middleware.CORS())

	// Health Check (Public, unauthenticated)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "outpost",
		})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Public Routes
		for _, registrar := range params.PublicRoutes {
			registrar.RegisterRoutes(v1)
		}

		// Protected Route Group (Requires Bearer API key authentication)
		if params.AuthMiddleware != nil {
			protected := v1.Group("")
			protected.Use(params.AuthMiddleware)
			for _, registrar := range params.ProtectedRoutes {
				registrar.RegisterRoutes(protected)
			}
		}
	}

	return r
}
