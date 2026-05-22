package api

import (
	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/config"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterRoutes sets up all API route groups on the given Gin engine.
// Routes are organized into: public, PXE, internal, and protected groups.
func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	// ── Health & Info ──
	registerHealthRoutes(r, cfg)

	// ── API v1 ──
	apiGroup := r.Group("/api/v1")

	apiGroup.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	apiGroup.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ── Public ──
	registerPublicRoutes(apiGroup, cfg)

	// ── PXE (unauthenticated, for booting machines) ──
	registerPXERoutes(apiGroup)

	// ── Internal (post-install scripts, heartbeat) ──
	registerInternalRoutes(apiGroup, cfg)

	// ── Protected (requires JWT) ──
	registerProtectedRoutes(apiGroup, cfg)
}

func registerHealthRoutes(r *gin.Engine, cfg *config.Config) {
	r.Static("/tftp", "/tftpboot")

	// Liveness probe — always returns 200 if the process is running
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "alive"})
	})

	// Readiness probe — checks DB connectivity
	readinessCheck := func(c *gin.Context) {
		dbStatus := "down"
		httpStatus := 503
		if db := getDB(); db != nil {
			sqlDB, err := db.DB()
			if err == nil && sqlDB.Ping() == nil {
				dbStatus = "up"
				httpStatus = 200
			}
		}
		c.JSON(httpStatus, gin.H{
			"status":   map[bool]string{true: "ready", false: "not_ready"}[httpStatus == 200],
			"database": dbStatus,
			"mode":     cfg.Server.Mode,
		})
	}

	r.GET("/health/ready", readinessCheck)
	r.GET("/health", readinessCheck) // backwards-compatible alias

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OS Baka API (Go)",
			"version": "1.0.0",
			"docs":    "/api/v1/docs/index.html",
		})
	})
}

func registerPublicRoutes(apiGroup *gin.RouterGroup, cfg *config.Config) {
	authHandler := NewAuthHandler(cfg)
	apiGroup.POST("/auth/login", LoginRateLimitMiddleware(), authHandler.Login)

	sshHandler := NewSSHHandler(cfg)
	apiGroup.GET("/ws/ssh", sshHandler.HandleSSH)
}

func registerPXERoutes(apiGroup *gin.RouterGroup) {
	pxeHandler := NewPXEHandler()
	apiGroup.GET("/pxe/init", pxeHandler.InitScript)
	apiGroup.GET("/pxe/boot/:mac", pxeHandler.BootScript)
	apiGroup.GET("/pxe/preseed/:mac", pxeHandler.Preseed)
	apiGroup.GET("/pxe/postinstall/:mac", pxeHandler.PostInstall)
}

func registerInternalRoutes(apiGroup *gin.RouterGroup, cfg *config.Config) {
	nodeHandler := NewNodeHandler()
	heartbeatHandler := NewHeartbeatHandler()

	internalGroup := apiGroup.Group("/internal")
	internalGroup.Use(InternalAPIMiddleware(cfg))
	{
		internalGroup.PUT("/nodes/:id/status", nodeHandler.UpdateNodeStatus)
		internalGroup.POST("/nodes/:id/heartbeat", heartbeatHandler.Heartbeat)
	}
}

func registerProtectedRoutes(apiGroup *gin.RouterGroup, cfg *config.Config) {
	protected := apiGroup.Group("/")
	protected.Use(AuthMiddleware(cfg))
	protected.Use(AuditMiddleware())

	authHandler := NewAuthHandler(cfg)
	protected.GET("/auth/me", authHandler.Me)

	registerNodeRoutes(protected)
	registerUserRoutes(protected)
	registerDHCPRoutes(protected)
	registerSystemRoutes(protected)
	registerAuditRoutes(protected)
}

func registerNodeRoutes(rg *gin.RouterGroup) {
	h := NewNodeHandler()
	rg.GET("/nodes", h.ListNodes)
	rg.POST("/nodes", RequireRole("admin", "operator"), h.CreateNode)
	rg.PUT("/nodes/:id", RequireRole("admin", "operator"), h.UpdateNode)
	rg.POST("/nodes/:id/rebuild", RequireRole("admin", "operator"), h.RebuildNode)
	rg.GET("/nodes/:id/passphrase", RequireRole("admin"), h.GetPassphrase)
	rg.POST("/nodes/:id/rotate-passphrase", RequireRole("admin"), h.RotatePassphrase)
	rg.DELETE("/nodes/:id", RequireRole("admin"), h.DeleteNode)

	// IPMI
	ipmi := NewIPMIHandler()
	rg.POST("/nodes/:id/power", RequireRole("admin", "operator"), ipmi.PowerAction)
	rg.GET("/nodes/:id/ipmi/test", RequireRole("admin", "operator"), ipmi.TestIPMI)

	// Groups & Tags
	g := NewGroupHandler()
	rg.GET("/groups", g.ListGroups)
	rg.POST("/groups", RequireRole("admin"), g.CreateGroup)
	rg.PUT("/groups/:id", RequireRole("admin"), g.UpdateGroup)
	rg.DELETE("/groups/:id", RequireRole("admin"), g.DeleteGroup)
	rg.PUT("/nodes/:id/group", RequireRole("admin", "operator"), g.AssignGroup)
	rg.GET("/nodes/:id/tags", g.ListNodeTags)
	rg.PUT("/nodes/:id/tags", RequireRole("admin", "operator"), g.SetNodeTags)

	// Bulk
	b := NewBulkHandler()
	rg.POST("/nodes/bulk/rebuild", RequireRole("admin", "operator"), b.BulkRebuild)
	rg.POST("/nodes/bulk/delete", RequireRole("admin"), b.BulkDelete)
	rg.PUT("/nodes/bulk/group", RequireRole("admin", "operator"), b.BulkAssignGroup)
}

func registerUserRoutes(rg *gin.RouterGroup) {
	h := NewUserHandler()
	rg.GET("/users", h.ListUsers)
	rg.POST("/users", h.CreateUser)
	rg.PUT("/users/:id", h.UpdateUser)
	rg.PUT("/users/:id/password", h.ChangePassword)
	rg.DELETE("/users/:id", h.DeleteUser)
}

func registerDHCPRoutes(rg *gin.RouterGroup) {
	h := NewDHCPHandler()
	rg.GET("/dhcp/configs", h.ListConfigs)
	rg.GET("/dhcp/configs/:id", h.GetConfig)
	rg.GET("/dhcp/config/active", h.GetActiveConfig)
	rg.POST("/dhcp/configs", h.CreateConfig)
	rg.PUT("/dhcp/configs/:id", h.UpdateConfig)
	rg.DELETE("/dhcp/configs/:id", h.DeleteConfig)
	rg.POST("/dhcp/service/restart", h.RestartService)

	// Reservations
	rg.GET("/dhcp/reservations", h.ListReservations)
	rg.POST("/dhcp/reservations", h.CreateReservation)
	rg.PUT("/dhcp/reservations/:id", h.UpdateReservation)
	rg.DELETE("/dhcp/reservations/:id", h.DeleteReservation)
	rg.POST("/dhcp/reservations/sync", h.SyncFromNodes)
}

func registerSystemRoutes(rg *gin.RouterGroup) {
	sys := NewSystemHandler()
	rg.GET("/system/interfaces", sys.ListInterfaces)

	asset := NewAssetHandler()
	rg.GET("/assets/boot", asset.ListAssets)
	rg.POST("/assets/boot", asset.UploadAsset)
	rg.DELETE("/assets/boot/:id", asset.DeleteAsset)

	dash := NewDashboardHandler()
	rg.GET("/dashboard/summary", dash.Summary)

	notif := NewNotificationHandler()
	rg.GET("/notifications", notif.List)
	rg.POST("/notifications/:id/read", notif.MarkRead)
}

func registerAuditRoutes(rg *gin.RouterGroup) {
	h := NewAuditHandler()
	rg.GET("/audit-logs", h.ListAuditLogs)

	ak := NewAPIKeyHandler()
	rg.GET("/api-keys", RequireRole("admin"), ak.ListAPIKeys)
	rg.POST("/api-keys", RequireRole("admin"), ak.CreateAPIKey)
	rg.DELETE("/api-keys/:id", RequireRole("admin"), ak.RevokeAPIKey)
	rg.POST("/api-keys/:id/rotate", RequireRole("admin"), ak.RotateAPIKey)
}
