package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// AuditHandler handles audit log endpoints.
type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// WriteAuditLog records an audit event to the database.
// Can be called from any handler to log important actions.
func WriteAuditLog(c *gin.Context, action, resource, resourceID, details string) {
	userID, _ := GetAuthUserID(c)
	username := GetAuthUsername(c)
	if username == "" {
		username = "system"
	}

	log := model.AuditLog{
		Action:     action,
		UserID:     userID,
		Username:   username,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  c.ClientIP(),
	}

	// Fire-and-forget: don't block the request for audit logging
	go func() {
		_ = getDB().Create(&log).Error
	}()
}

// ListAuditLogs godoc
// @Summary      List audit logs
// @Description  Returns paginated audit log entries, newest first
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 50, max 200)"
// @Param        action query string false "Filter by action prefix (e.g. node, user, dhcp)"
// @Success      200 {object} map[string]interface{}
// @Router       /audit-logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	actionFilter := c.Query("action")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	db := getDB().Model(&model.AuditLog{})
	if actionFilter != "" {
		db = db.Where("action LIKE ?", actionFilter+"%")
	}

	var total int64
	db.Count(&total)

	var logs []model.AuditLog
	db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"items": logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// AuditMiddleware creates a Gin middleware that automatically logs
// mutating requests (POST, PUT, DELETE) to the audit log.
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only audit mutating methods
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "DELETE" {
			c.Next()
			return
		}

		// Process the request first
		c.Next()

		// Only log successful mutations (2xx/3xx)
		status := c.Writer.Status()
		if status >= 400 {
			return
		}

		// Determine action from path and method
		path := c.FullPath()
		action := deriveAction(method, path)
		if action == "" {
			return // Skip paths we don't care about (e.g. /ping)
		}

		// Extract resource ID if present
		resourceID := ""
		if id := c.Param("id"); id != "" {
			resourceID = id
		} else if mac := c.Param("mac"); mac != "" {
			resourceID = mac
		}

		details := fmt.Sprintf("%s %s → %d", method, c.Request.URL.Path, status)
		WriteAuditLog(c, action, deriveResource(path), resourceID, details)
	}
}

// deriveAction maps HTTP method + path pattern to a human-readable action.
func deriveAction(method, path string) string {
	// Map of path patterns to resource names
	switch {
	case contains(path, "/nodes") && method == "POST":
		return "node.create"
	case contains(path, "/nodes/:id/rebuild"):
		return "node.rebuild"
	case contains(path, "/nodes/:id") && method == "PUT":
		return "node.update"
	case contains(path, "/nodes/:id") && method == "DELETE":
		return "node.delete"
	case contains(path, "/users") && method == "POST":
		return "user.create"
	case contains(path, "/users/:id/password"):
		return "user.password_change"
	case contains(path, "/users/:id") && method == "PUT":
		return "user.update"
	case contains(path, "/users/:id") && method == "DELETE":
		return "user.delete"
	case contains(path, "/auth/login"):
		return "user.login"
	case contains(path, "/dhcp/configs") && method == "POST":
		return "dhcp.config_create"
	case contains(path, "/dhcp/configs/:id") && method == "PUT":
		return "dhcp.config_update"
	case contains(path, "/dhcp/configs/:id") && method == "DELETE":
		return "dhcp.config_delete"
	case contains(path, "/dhcp/service/restart"):
		return "dhcp.service_restart"
	case contains(path, "/dhcp/reservations") && method == "POST":
		return "dhcp.reservation_create"
	case contains(path, "/dhcp/reservations/sync"):
		return "dhcp.reservation_sync"
	case contains(path, "/assets/boot") && method == "POST":
		return "asset.upload"
	case contains(path, "/assets/boot/:id") && method == "DELETE":
		return "asset.delete"
	case contains(path, "/notifications/:id/read"):
		return "notification.read"
	}
	return ""
}

// deriveResource extracts the resource type from a route path.
func deriveResource(path string) string {
	switch {
	case contains(path, "/nodes"):
		return "node"
	case contains(path, "/users"):
		return "user"
	case contains(path, "/auth"):
		return "auth"
	case contains(path, "/dhcp"):
		return "dhcp"
	case contains(path, "/assets"):
		return "asset"
	case contains(path, "/notifications"):
		return "notification"
	}
	return "system"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
