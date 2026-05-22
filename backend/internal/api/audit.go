package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
	"gorm.io/gorm"
)

// auditWriteTimeout caps how long an audit insert may take before we give
// up and log a failure. Picked to be larger than any reasonable Postgres
// round-trip but small enough that a wedged DB doesn't stall every request
// for seconds.
const auditWriteTimeout = 2 * time.Second

// AuditHandler handles audit log endpoints.
type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// WriteAuditLog records an audit event to the database synchronously.
//
// Originally fire-and-forget via a goroutine per call. That was wrong for
// two reasons:
//  1. Bulk paths (e.g. CSV import of 500 nodes through CreateNode) spawned
//     500 concurrent goroutines all racing for DB connections.
//  2. A goroutine leaking the gin context could read c.ClientIP after the
//     request had ended, with undefined behavior.
//
// A single audit insert is sub-millisecond against a properly indexed
// table; making it synchronous is correct. A bounded timeout protects
// against a wedged DB stalling the request.
//
// On failure we log slog.Error and return — the user's action already
// succeeded by the time we get here, so propagating the audit-write
// error would only confuse them.
func WriteAuditLog(c *gin.Context, action, resource, resourceID, details string) {
	userID, _ := GetAuthUserID(c)
	username := GetAuthUsername(c)
	if username == "" {
		username = "system"
	}
	// Capture client IP now — c may be reused / pooled after the request
	// completes, so we must not call methods on it past this point.
	clientIP := c.ClientIP()

	log := model.AuditLog{
		Action:     action,
		UserID:     userID,
		Username:   username,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  clientIP,
	}

	writeAuditLogSync(getDB(), &log)
}

// WriteAuditLogs persists multiple audit entries in batches. Used by
// high-volume paths (bulk delete, CSV import) to avoid N round-trips.
// Each entry must already have its fields filled — this function does
// not consult the gin context.
func WriteAuditLogs(logs []model.AuditLog) {
	if len(logs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
	defer cancel()
	// Batch size 100 chosen to keep each round-trip well under a TCP MTU's
	// worth of bind parameters (PG's 16-bit param count gives us plenty of
	// headroom but smaller batches mean smaller retries on partial failure).
	const batchSize = 100
	if err := getDB().WithContext(ctx).CreateInBatches(logs, batchSize).Error; err != nil {
		slog.Error("audit: batch write failed",
			"count", len(logs),
			"error", err)
	}
}

// writeAuditLogSync is the shared synchronous path with a bounded timeout.
// Errors are logged but never returned — audit logging must not break the
// user-facing action that triggered it. A nil db is also a no-op rather
// than a panic: it can happen in tests that exercise handlers without
// initializing storage, or if a goroutine outlives InitHandlers shutdown
// somehow.
func writeAuditLogSync(db *gorm.DB, log *model.AuditLog) {
	if db == nil {
		slog.Warn("audit: write skipped (db not initialized)",
			"action", log.Action,
			"resource", log.Resource,
			"resource_id", log.ResourceID)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
	defer cancel()
	if err := db.WithContext(ctx).Create(log).Error; err != nil {
		slog.Error("audit: write failed",
			"action", log.Action,
			"resource", log.Resource,
			"resource_id", log.ResourceID,
			"error", err)
	}
}

// nodeDeleteSnapshot is the structured payload embedded into audit log
// details when a node row is destroyed. After deletion the node table
// no longer has any record of the physical machine, so the audit log is
// the only place to find out what was removed. Keep it small and stable —
// readers may parse it.
type nodeDeleteSnapshot struct {
	ID         uint   `json:"id"`
	Hostname   string `json:"hostname"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	AssetTag   string `json:"asset_tag,omitempty"`
	GroupID    *uint  `json:"group_id,omitempty"`
	Status     string `json:"status,omitempty"`
}

// makeNodeDeleteSnapshot builds the structured details string for a node
// deletion audit entry. Returns valid JSON even for a zero-value node so
// callers don't have to nil-check.
func makeNodeDeleteSnapshot(n *model.Node) string {
	snap := nodeDeleteSnapshot{
		ID:         n.ID,
		Hostname:   n.Hostname,
		MACAddress: n.MACAddress,
		IPAddress:  n.IPAddress,
		AssetTag:   n.AssetTag,
		GroupID:    n.GroupID,
		Status:     n.Status,
	}
	b, err := json.Marshal(snap)
	if err != nil {
		// Shouldn't happen — the struct is JSON-safe. Fall back to a
		// minimal text representation so we don't lose ALL traceability.
		return fmt.Sprintf("node id=%d hostname=%q mac=%q (json marshal failed: %v)",
			n.ID, n.Hostname, n.MACAddress, err)
	}
	return string(b)
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
