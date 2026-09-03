package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/LanreAkintayo/outpost/internal/config"
	"github.com/LanreAkintayo/outpost/internal/database"
	"github.com/LanreAkintayo/outpost/internal/handler"
	"github.com/LanreAkintayo/outpost/internal/logger"
	"github.com/LanreAkintayo/outpost/internal/middleware"
	"github.com/LanreAkintayo/outpost/internal/repository"
	"github.com/LanreAkintayo/outpost/internal/router"
	"github.com/LanreAkintayo/outpost/internal/server"
	"github.com/LanreAkintayo/outpost/internal/service"
)

func main() {
	// Load local .env file (if present)
	_ = godotenv.Load()

	// Load and validate application configuration
	cfg, err := config.Load()
	if err != nil {
		panic("fatal: failed to load configuration: " + err.Error())
	}

	// Initialize structured logger (Dependency Injected)
	log := logger.New(cfg.Environment)
	log.Info().
		Str("environment", cfg.Environment).
		Str("port", cfg.Server.Port).
		Str("gin_mode", cfg.Server.GinMode).
		Str("db", cfg.Database.MaskedConnectionString()).
		Int("workers", cfg.Engine.WorkerCount).
		Msg("configuration loaded successfully")

	// Initialize PostgreSQL connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := database.NewPool(ctx, cfg.Database.ConnectionString(), database.DefaultPoolConfig())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to PostgreSQL")
	}
	defer dbPool.Close()

	log.Info().Msg("database connection pool initialized and ping verified")

	// Wire Dependencies (Dependency Injection)
	appRepo := repository.NewPostgresApplicationRepository(dbPool)
	appService := service.NewApplicationService(appRepo)
	appHandler := handler.NewApplicationHandler(appService)
	authHandler := handler.NewAuthHandler()
	authMiddleware := middleware.AuthenticateAPIKey(appService)

	// Build HTTP Router (Routing & Middlewares)
	r := router.New(router.RouterParams{
		Config:          cfg,
		Logger:          log,
		AuthMiddleware:  authMiddleware,
		PublicRoutes:    []router.RouteRegistrar{appHandler},
		ProtectedRoutes: []router.RouteRegistrar{authHandler},
	})

	// Initialize HTTP Server (Transport Lifecycle)
	srv := server.New(cfg, log, r)

	// Start HTTP Server in a background goroutine
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("starting HTTP server")
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("HTTP server failed to listen")
		}
	}()

	// 4. Graceful Shutdown: Listen for OS termination signals (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("shutdown signal received, closing server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown due to timeout")
	}

	log.Info().Msg("server exited cleanly")
}
