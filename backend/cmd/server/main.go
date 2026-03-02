package main

import (
	"log/slog"
	"os"

	"github.com/os-baka/backend/internal/api"
	"github.com/os-baka/backend/internal/config"
	"github.com/os-baka/backend/internal/model"
	"github.com/os-baka/backend/internal/vault"

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

	// Init secret store (Vault or DB fallback)
	vaultCfg := &vault.Config{
		Enabled:    cfg.Vault.Enabled,
		Address:    cfg.Vault.Address,
		Token:      cfg.Vault.Token,
		MountPath:  cfg.Vault.MountPath,
		PathPrefix: cfg.Vault.PathPrefix,
	}
	secretStore := vault.NewFromConfig(vaultCfg)

	// Inject DB and secret store into API handlers
	api.InitHandlers(model.DB, secretStore)

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
			"message":      "OS Baka API (Go)",
			"version":      "1.0.0",
			"docs":         "/api/v1/docs/index.html",
			"secret_store": secretStore.Type(),
		})
	})

	authHandler := api.NewAuthHandler(cfg)
	nodeHandler := api.NewNodeHandler()
	sshHandler := api.NewSSHHandler(cfg)
	userHandler := api.NewUserHandler()

	// API Group
	apiGroup := r.Group("/api/v1")
	{
		apiGroup.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// Swagger Docs
		apiGroup.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		// Auth (with login-specific rate limiting)
		apiGroup.POST("/auth/login", api.LoginRateLimitMiddleware(), authHandler.Login)

		// PXE Boot Services (Publicly accessible by booting machines)
		pxeHandler := api.NewPXEHandler()
		apiGroup.GET("/pxe/init", pxeHandler.InitScript)
		apiGroup.GET("/pxe/boot/:mac", pxeHandler.BootScript)
		apiGroup.GET("/pxe/preseed/:mac", pxeHandler.Preseed)
		apiGroup.GET("/pxe/postinstall/:mac", pxeHandler.PostInstall)

		// WebSocket SSH proxy (auth via first WS message, not HTTP header)
		apiGroup.GET("/ws/ssh", sshHandler.HandleSSH)

		// Internal API (used by postinstall scripts, restricted access)
		internalGroup := apiGroup.Group("/internal")
		internalGroup.Use(api.InternalAPIMiddleware(cfg))
		{
			internalGroup.PUT("/nodes/:id/status", nodeHandler.UpdateNodeStatus)
			heartbeatHandler := api.NewHeartbeatHandler()
			internalGroup.POST("/nodes/:id/heartbeat", heartbeatHandler.Heartbeat)
		}

		// Protected Routes
		protected := apiGroup.Group("/")
		protected.Use(api.AuthMiddleware(cfg))
		protected.Use(api.AuditMiddleware())
		{
			protected.GET("/auth/me", authHandler.Me)

			// Nodes
			protected.GET("/nodes", nodeHandler.ListNodes)
			protected.POST("/nodes", api.RequireRole("admin", "operator"), nodeHandler.CreateNode)
			protected.PUT("/nodes/:id", api.RequireRole("admin", "operator"), nodeHandler.UpdateNode)
			protected.POST("/nodes/:id/rebuild", api.RequireRole("admin", "operator"), nodeHandler.RebuildNode)
			protected.GET("/nodes/:id/passphrase", api.RequireRole("admin"), nodeHandler.GetPassphrase)
			protected.POST("/nodes/:id/rotate-passphrase", api.RequireRole("admin"), nodeHandler.RotatePassphrase)
			protected.DELETE("/nodes/:id", api.RequireRole("admin"), nodeHandler.DeleteNode)

			// IPMI Power Management
			ipmiHandler := api.NewIPMIHandler()
			protected.POST("/nodes/:id/power", api.RequireRole("admin", "operator"), ipmiHandler.PowerAction)
			protected.GET("/nodes/:id/ipmi/test", api.RequireRole("admin", "operator"), ipmiHandler.TestIPMI)

			// Node Groups & Tags
			groupHandler := api.NewGroupHandler()
			protected.GET("/groups", groupHandler.ListGroups)
			protected.POST("/groups", api.RequireRole("admin"), groupHandler.CreateGroup)
			protected.PUT("/groups/:id", api.RequireRole("admin"), groupHandler.UpdateGroup)
			protected.DELETE("/groups/:id", api.RequireRole("admin"), groupHandler.DeleteGroup)
			protected.PUT("/nodes/:id/group", api.RequireRole("admin", "operator"), groupHandler.AssignGroup)
			protected.GET("/nodes/:id/tags", groupHandler.ListNodeTags)
			protected.PUT("/nodes/:id/tags", api.RequireRole("admin", "operator"), groupHandler.SetNodeTags)

			// Bulk Operations
			bulkHandler := api.NewBulkHandler()
			protected.POST("/nodes/bulk/rebuild", api.RequireRole("admin", "operator"), bulkHandler.BulkRebuild)
			protected.POST("/nodes/bulk/delete", api.RequireRole("admin"), bulkHandler.BulkDelete)
			protected.PUT("/nodes/bulk/group", api.RequireRole("admin", "operator"), bulkHandler.BulkAssignGroup)

			// Users (all user management is admin-only, enforced in handler too)
			protected.GET("/users", userHandler.ListUsers)
			protected.POST("/users", userHandler.CreateUser)
			protected.PUT("/users/:id", userHandler.UpdateUser)
			protected.PUT("/users/:id/password", userHandler.ChangePassword)
			protected.DELETE("/users/:id", userHandler.DeleteUser)

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

			// Audit Logs
			auditHandler := api.NewAuditHandler()
			protected.GET("/audit-logs", auditHandler.ListAuditLogs)

			// API Keys (admin-only)
			apiKeyHandler := api.NewAPIKeyHandler()
			protected.GET("/api-keys", api.RequireRole("admin"), apiKeyHandler.ListAPIKeys)
			protected.POST("/api-keys", api.RequireRole("admin"), apiKeyHandler.CreateAPIKey)
			protected.DELETE("/api-keys/:id", api.RequireRole("admin"), apiKeyHandler.RevokeAPIKey)
			protected.POST("/api-keys/:id/rotate", api.RequireRole("admin"), apiKeyHandler.RotateAPIKey)
		}
	}

	// Start background stale node checker (every 5 min, threshold 10 min)
	api.StartStaleNodeChecker(5, 10)

	slog.Info("Server starting", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		slog.Error("Failed to run server", "error", err)
		os.Exit(1)
	}
}
