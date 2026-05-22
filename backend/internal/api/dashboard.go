package api

import (
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// Summary godoc
// @Summary      Dashboard summary
// @Description  Aggregated statistics for the dashboard overview
// @Tags         dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /dashboard/summary [get]
func (h *DashboardHandler) Summary(c *gin.Context) {
	var nodesTotal int64
	var nodesActive int64
	var nodesError int64
	var nodesInstalling int64
	var nodesEncrypted int64

	getDB().Model(&model.Node{}).Count(&nodesTotal)
	getDB().Model(&model.Node{}).Where("status = ?", "active").Count(&nodesActive)
	getDB().Model(&model.Node{}).Where("status = ?", "error").Count(&nodesError)
	getDB().Model(&model.Node{}).Where("status = ?", "installing").Count(&nodesInstalling)
	getDB().Model(&model.Node{}).Where("encryption_enabled = ?", true).Count(&nodesEncrypted)

	// User count
	var usersTotal int64
	getDB().Model(&model.User{}).Count(&usersTotal)

	// Check if dnsmasq is running (best-effort)
	dnsmasqRunning := isDnsmasqRunning()

	// Recent audit logs (last 10 entries)
	var recentLogs []model.AuditLog
	getDB().Order("created_at DESC").Limit(10).Find(&recentLogs)

	// dnsmasq scheduler state — last_run is RFC3339 (omitted when zero so
	// the UI can render "—" instead of an epoch).
	lastRun, lastError, runCount := DnsmasqStats()
	dnsmasqLastRun := ""
	if !lastRun.IsZero() {
		dnsmasqLastRun = lastRun.UTC().Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes_total":         nodesTotal,
		"nodes_active":        nodesActive,
		"nodes_error":         nodesError,
		"nodes_installing":    nodesInstalling,
		"nodes_encrypted":     nodesEncrypted,
		"users_total":         usersTotal,
		"dnsmasq_running":     dnsmasqRunning,
		"dnsmasq_last_run":    dnsmasqLastRun,
		"dnsmasq_last_error":  lastError,
		"dnsmasq_run_count":   runCount,
		"recent_activity":     recentLogs,
	})
}

// isDnsmasqRunning checks if a dnsmasq process is active.
func isDnsmasqRunning() bool {
	err := exec.Command("pgrep", "-x", "dnsmasq").Run()
	return err == nil
}
