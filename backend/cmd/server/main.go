package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/os-baka/backend/internal/api"
	"github.com/os-baka/backend/internal/config"
	"github.com/os-baka/backend/internal/model"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	_ "github.com/os-baka/backend/docs"
)

// @title           OS Baka API
// @version         1.0.0
// @description     PXE Boot Provisioning and Node Management System API.

// @contact.name   OS Baka Project
// @contact.url    https://github.com/os-baka

// @license.name  MIT

// @host      localhost:8000
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// Set Gin mode
	// Load Config
	cfg := config.Load()

	// Init DB
	model.InitDB(cfg)

	// Inject DB into API handlers
	api.InitHandlers(model.DB)

	// Regenerate dnsmasq config synchronously on startup to ensure the
	// on-disk state matches the DB before we start serving PXE requests.
	// After this, all mutations go through the scheduler.
	if err := api.GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Startup dnsmasq config generation failed", "error", err)
	}
	api.StartDnsmasqScheduler()

	r := gin.Default()

	// CORS Configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.Cors.AllowedOrigins
	if len(corsConfig.AllowOrigins) == 0 {
		// Fallback defaults if config is empty
		corsConfig.AllowOrigins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:8000",
			"http://127.0.0.1:8000",
		}
	}

	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Security Headers
	r.Use(api.SecurityHeadersMiddleware())

	// Request ID + Structured Logging
	r.Use(api.RequestIDMiddleware())
	r.Use(api.RequestLoggerMiddleware())

	// ── Routes ──
	api.RegisterRoutes(r, cfg)

	// Start background stale node checker (every 5 min, threshold 10 min)
	api.StartStaleNodeChecker(5, 10)

	// Configure HTTP server with timeouts (prevent slow-loris attacks)
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to run server", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown: wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("Shutdown signal received, gracefully stopping server...", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Drain the dnsmasq scheduler so any in-flight regen completes before
	// process exit. Time-bound to avoid hanging shutdown if regen wedges.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := api.StopDnsmasqScheduler(stopCtx); err != nil {
		slog.Warn("dnsmasq scheduler stop timed out", "error", err)
	}
	stopCancel()

	slog.Info("Server exited gracefully")
}
