package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// HeartbeatHandler handles node health reporting.
type HeartbeatHandler struct{}

func NewHeartbeatHandler() *HeartbeatHandler {
	return &HeartbeatHandler{}
}

// Heartbeat godoc
// @Summary      Report node heartbeat
// @Description  Called by node agents to report health status. Internal API.
// @Tags         internal
// @Accept       json
// @Produce      json
// @Param        id path int true "Node ID"
// @Param        body body object true "Health metrics"
// @Success      200 {object} map[string]string
// @Router       /internal/nodes/{id}/heartbeat [post]
func (h *HeartbeatHandler) Heartbeat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid node ID")
		return
	}

	var req struct {
		CPUUsage    float64 `json:"cpu_usage"`
		MemoryUsage float64 `json:"memory_usage"`
		DiskUsage   float64 `json:"disk_usage"`
		Uptime      int64   `json:"uptime"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	now := time.Now()

	// Heartbeats always update the health metrics, but they only promote
	// status from offline -> active. Nodes in installing/pending/maintenance
	// are operator/system-managed; a chatty agent in those states must NOT
	// race the install-timeout watcher or undo a maintenance lockout.
	// Active stays active; offline gets promoted back to active.
	updates := map[string]any{
		"last_heartbeat": now,
		"cpu_usage":      req.CPUUsage,
		"memory_usage":   req.MemoryUsage,
		"disk_usage":     req.DiskUsage,
		"uptime":         req.Uptime,
	}
	result := getDB().Model(&model.Node{}).
		Where("id = ? AND status IN ?", id, []string{NodeStatusActive, NodeStatusOffline}).
		Updates(map[string]any{
			"last_heartbeat": now,
			"cpu_usage":      req.CPUUsage,
			"memory_usage":   req.MemoryUsage,
			"disk_usage":     req.DiskUsage,
			"uptime":         req.Uptime,
			"status":         NodeStatusActive,
		})

	if result.RowsAffected == 0 {
		// Either the node doesn't exist, or it's in a state where heartbeat
		// must not change `status`. Update health metrics only.
		fallback := getDB().Model(&model.Node{}).Where("id = ?", id).Updates(updates)
		if fallback.RowsAffected == 0 {
			ErrorResponse(c, http.StatusNotFound, "Node not found")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat recorded"})
}

// CheckStaleNodes marks nodes as offline if they haven't sent a heartbeat
// in the configured time. Bulk UPDATE for throughput — per-node SetNodeStatus
// would be N queries each periodic tick. A coarse audit log captures the
// batch.
func CheckStaleNodes(staleMinutes int) {
	if staleMinutes <= 0 {
		staleMinutes = 10
	}

	threshold := time.Now().Add(-time.Duration(staleMinutes) * time.Minute)

	result := getDB().Model(&model.Node{}).
		Where("last_heartbeat IS NOT NULL AND last_heartbeat < ? AND status = ?",
			threshold, NodeStatusActive).
		Update("status", NodeStatusOffline)

	if result.RowsAffected > 0 {
		slog.Info("Marked stale nodes as offline",
			"count", result.RowsAffected,
			"threshold_minutes", staleMinutes)
		writeSystemAuditLog(getDB(), "node.bulk_status_change", "node", "",
			fmt.Sprintf("%d node(s) active -> offline: stale heartbeat (threshold=%dm)",
				result.RowsAffected, staleMinutes))
	}
}

// CheckInstallTimeout flips nodes that have been provisioning for too long
// into the error state. Caller is responsible for picking the threshold;
// the in-process default lives in StartStaleNodeChecker.
//
// Bulk UPDATE for throughput. The single `installing_started_at IS NOT NULL`
// guard is what makes this safe to re-run — once flipped to error, the
// SetNodeStatus path that nulls the timestamp prevents re-firing on the
// next tick. The bulk path here also clears it explicitly.
func CheckInstallTimeout(timeoutMinutes int) {
	if timeoutMinutes <= 0 {
		timeoutMinutes = 60
	}

	threshold := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)

	result := getDB().Model(&model.Node{}).
		Where(`status = ?
		         AND installing_started_at IS NOT NULL
		         AND installing_started_at < ?`,
			NodeStatusInstalling, threshold).
		Updates(map[string]any{
			"status":                NodeStatusError,
			"installing_started_at": nil,
		})

	if result.RowsAffected > 0 {
		slog.Warn("Marked installing nodes as error due to timeout",
			"count", result.RowsAffected,
			"timeout_minutes", timeoutMinutes)
		writeSystemAuditLog(getDB(), "node.bulk_status_change", "node", "",
			fmt.Sprintf("%d node(s) installing -> error: install timeout %dm",
				result.RowsAffected, timeoutMinutes))
	}
}

// StartStaleNodeChecker starts the background goroutines that watch for
// stale nodes and stuck installs. Returns immediately; the goroutines run
// for the process lifetime.
//
// intervalMinutes  : how often to scan (both checks share the interval)
// staleMinutes     : threshold for active -> offline
// installTimeout   : threshold for installing -> error (read from env
//                    INSTALL_TIMEOUT_MINUTES if positive, falls back to 60)
func StartStaleNodeChecker(intervalMinutes, staleMinutes int) {
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}
	installTimeout := installTimeoutFromEnv()

	go func() {
		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CheckStaleNodes(staleMinutes)
			CheckInstallTimeout(installTimeout)
		}
	}()

	slog.Info("Stale node checker started",
		"interval_minutes", intervalMinutes,
		"stale_threshold_minutes", staleMinutes,
		"install_timeout_minutes", installTimeout)
}

// defaultInstallTimeoutMinutes is the default budget for a node to go
// from `installing` to a postinstall-callback-driven `active`. Bumped
// from 60 to 120 because 60 was a close call on the happy path:
// BIOS POST (3 min) + d-i (15-30 min) + base packages + TPM enroll
// (3-5 min) + first boot + callback (1-2 min) easily exceeds 60 min
// on slow hardware or congested intranet mirrors.
const defaultInstallTimeoutMinutes = 120

// installTimeoutFromEnv returns the install-timeout in minutes, configurable
// via INSTALL_TIMEOUT_MINUTES. Also consulted by pxeTokenTTL to derive a
// sensible default token TTL (= 2x install timeout) when the token TTL
// isn't explicitly set.
func installTimeoutFromEnv() int {
	def := defaultInstallTimeoutMinutes
	if v := os.Getenv("INSTALL_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
