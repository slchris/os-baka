package api

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

type DashboardHandler struct{}

func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	var nodesTotal int64
	var nodesActive int64
	var nodesError int64
	var nodesInstalling int64
	
	getDB().Model(&model.Node{}).Count(&nodesTotal)
	getDB().Model(&model.Node{}).Where("status = ?", "active").Count(&nodesActive)
	getDB().Model(&model.Node{}).Where("status = ?", "error").Count(&nodesError)
	getDB().Model(&model.Node{}).Where("status = ?", "installing").Count(&nodesInstalling)
	
	// Mock deployment stats for now as we don't have a deployments table yet
	// But in real world, this would query a Deployment table.
	
	c.JSON(http.StatusOK, gin.H{
		"nodes_total":           nodesTotal,
		"nodes_active":          nodesActive,
		"nodes_error":           nodesError,
		"nodes_installing":      nodesInstalling,
		"deployments_total":     0,
		"deployments_pending":   0,
		"deployments_in_progress": 0,
		"deployments_completed": 0,
		"deployments_failed":    0,
		"pxe_service_running":   true, // Mock status
	})
}
