package api

import (
	"log/slog"
	"net/http"
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
	result := getDB().Model(&model.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_heartbeat": now,
		"cpu_usage":      req.CPUUsage,
		"memory_usage":   req.MemoryUsage,
		"disk_usage":     req.DiskUsage,
		"uptime":         req.Uptime,
		"status":         "active", // Heartbeat means node is alive
	})

	if result.RowsAffected == 0 {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat recorded"})
}

// CheckStaleNodes marks nodes as offline if they haven't sent a heartbeat in the configured time.
// This should be called periodically (e.g., via a background goroutine or cron).
func CheckStaleNodes(staleMinutes int) {
	if staleMinutes <= 0 {
		staleMinutes = 10
	}

	threshold := time.Now().Add(-time.Duration(staleMinutes) * time.Minute)

	result := getDB().Model(&model.Node{}).
		Where("last_heartbeat IS NOT NULL AND last_heartbeat < ? AND status = ?", threshold, "active").
		Update("status", "offline")

	if result.RowsAffected > 0 {
		slog.Info("Marked stale nodes as offline", "count", result.RowsAffected, "threshold_minutes", staleMinutes)
	}
}

// StartStaleNodeChecker starts a background goroutine that periodically checks for stale nodes.
func StartStaleNodeChecker(intervalMinutes, staleMinutes int) {
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}

	go func() {
		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CheckStaleNodes(staleMinutes)
		}
	}()

	slog.Info("Stale node checker started",
		"interval_minutes", intervalMinutes,
		"stale_threshold_minutes", staleMinutes)
}
