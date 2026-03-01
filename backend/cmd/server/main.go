package main

import (
	"log/slog"
	"os"

	"github.com/os-baka/backend/internal/api"
	"github.com/os-baka/backend/internal/config"
	"github.com/os-baka/backend/internal/model"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	_ "github.com/os-baka/backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// Regenerate dnsmasq config to ensure consistency on startup
	if err := api.GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Startup dnsmasq config generation failed", "error", err)
	}

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

	// Serve TFTP files via HTTP
	r.Static("/tftp", "/tftpboot")

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "down"
		if model.DB != nil {
			sqlDB, err := model.DB.DB()
			if err == nil && sqlDB.Ping() == nil {
				dbStatus = "up"
			}
		}
		c.JSON(200, gin.H{
			"status":   "healthy",
			"database": dbStatus,
			"mode":     cfg.Server.Mode,
		})
	})

	// Root Endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OS Baka API (Go)",
			"version": "0.1.0",
			"docs":    "/api/v1/docs/index.html",
		})
	})

	authHandler := api.NewAuthHandler(cfg)
	nodeHandler := api.NewNodeHandler()

	// API Group
	apiGroup := r.Group("/api/v1")
	{
		apiGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// Swagger Docs
		apiGroup.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		// Auth
		apiGroup.POST("/auth/login", authHandler.Login)

		// PXE Boot Services (Publicly accessible by booting machines)
		pxeHandler := api.NewPXEHandler()
		apiGroup.GET("/pxe/init", pxeHandler.InitScript)
		apiGroup.GET("/pxe/boot/:mac", pxeHandler.BootScript)
		apiGroup.GET("/pxe/preseed/:mac", pxeHandler.Preseed)
		apiGroup.GET("/pxe/kickstart/:mac", pxeHandler.Kickstart)
		apiGroup.GET("/pxe/postinstall/:mac", pxeHandler.PostInstall)

		// Internal API (used by postinstall scripts, restricted access)
		internalGroup := apiGroup.Group("/internal")
		internalGroup.Use(api.InternalAPIMiddleware(cfg))
		{
			internalGroup.PUT("/nodes/:id/status", nodeHandler.UpdateNodeStatus)
		}

		// Protected Routes
		protected := apiGroup.Group("/")
		protected.Use(api.AuthMiddleware(cfg))
		{
			protected.GET("/auth/me", authHandler.Me)

			// Nodes
			protected.GET("/nodes", nodeHandler.ListNodes)
			protected.POST("/nodes", nodeHandler.CreateNode)
			protected.PUT("/nodes/:id", nodeHandler.UpdateNode)
			protected.POST("/nodes/:id/rebuild", nodeHandler.RebuildNode)
			protected.GET("/nodes/:id/passphrase", nodeHandler.GetPassphrase)
			protected.DELETE("/nodes/:id", nodeHandler.DeleteNode)

			// DHCP Configuration
			dhcpHandler := api.NewDHCPHandler()
			protected.GET("/dhcp/configs", dhcpHandler.ListConfigs)
			protected.GET("/dhcp/configs/:id", dhcpHandler.GetConfig)
			protected.GET("/dhcp/config/active", dhcpHandler.GetActiveConfig)
			protected.POST("/dhcp/configs", dhcpHandler.CreateConfig)
			protected.PUT("/dhcp/configs/:id", dhcpHandler.UpdateConfig)
			protected.DELETE("/dhcp/configs/:id", dhcpHandler.DeleteConfig)
			protected.POST("/dhcp/service/restart", dhcpHandler.RestartService)

			// DHCP Reservations
			protected.GET("/dhcp/reservations", dhcpHandler.ListReservations)
			protected.POST("/dhcp/reservations", dhcpHandler.CreateReservation)
			protected.PUT("/dhcp/reservations/:id", dhcpHandler.UpdateReservation)
			protected.DELETE("/dhcp/reservations/:id", dhcpHandler.DeleteReservation)
			protected.POST("/dhcp/reservations/sync", dhcpHandler.SyncFromNodes)

			// System
			systemHandler := api.NewSystemHandler()
			protected.GET("/system/interfaces", systemHandler.ListInterfaces)

			// Boot Assets
			assetHandler := api.NewAssetHandler()
			protected.GET("/assets/boot", assetHandler.ListAssets)
			protected.POST("/assets/boot", assetHandler.UploadAsset)
			protected.DELETE("/assets/boot/:id", assetHandler.DeleteAsset)

			// Dashboard
			dashboardHandler := api.NewDashboardHandler()
			protected.GET("/dashboard/summary", dashboardHandler.Summary)

			// Notifications
			notificationHandler := api.NewNotificationHandler()
			protected.GET("/notifications", notificationHandler.List)
			protected.POST("/notifications/:id/read", notificationHandler.MarkRead)
		}
	}

	slog.Info("Server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		slog.Error("Failed to run server", "error", err)
		os.Exit(1)
	}
}
