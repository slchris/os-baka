package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// BulkHandler handles batch operations on nodes.
type BulkHandler struct{}

func NewBulkHandler() *BulkHandler {
	return &BulkHandler{}
}

type BulkIDsRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BulkRebuild godoc
// @Summary      Bulk rebuild nodes
// @Description  Set multiple nodes to 'installing' status for reinstallation
// @Tags         bulk
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body BulkIDsRequest true "Node IDs to rebuild"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/bulk/rebuild [post]
func (h *BulkHandler) BulkRebuild(c *gin.Context) {
	var req BulkIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	result := getDB().Model(&model.Node{}).
		Where("id IN ?", req.IDs).
		Update("status", "installing")

	WriteAuditLog(c, "node.bulk_rebuild", "node", fmt.Sprintf("%v", req.IDs),
		fmt.Sprintf("Rebuilt %d nodes", result.RowsAffected))

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"affected": result.RowsAffected,
		"message":  fmt.Sprintf("%d nodes set to rebuilding", result.RowsAffected),
	})
}

// BulkDelete godoc
// @Summary      Bulk delete nodes
// @Description  Delete multiple nodes and their DHCP reservations
// @Tags         bulk
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body BulkIDsRequest true "Node IDs to delete"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/bulk/delete [post]
func (h *BulkHandler) BulkDelete(c *gin.Context) {
	var req BulkIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Get MACs for DHCP cleanup
	var nodes []model.Node
	getDB().Where("id IN ?", req.IDs).Find(&nodes)

	macs := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if n.MACAddress != "" {
			macs = append(macs, n.MACAddress)
		}
	}

	// Delete DHCP reservations
	if len(macs) > 0 {
		getDB().Where("mac_address IN ?", macs).Delete(&model.DHCPReservation{})
	}

	// Delete tags
	getDB().Where("node_id IN ?", req.IDs).Delete(&model.NodeTag{})

	// Delete nodes
	result := getDB().Delete(&model.Node{}, req.IDs)

	WriteAuditLog(c, "node.bulk_delete", "node", fmt.Sprintf("%v", req.IDs),
		fmt.Sprintf("Deleted %d nodes", result.RowsAffected))

	// Regen dnsmasq
	if err := GenerateDnsmasqConfig(); err != nil {
		// Non-fatal
		_ = err
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"affected": result.RowsAffected,
		"message":  fmt.Sprintf("%d nodes deleted", result.RowsAffected),
	})
}

// BulkAssignGroup godoc
// @Summary      Bulk assign nodes to a group
// @Description  Assign multiple nodes to a group (or unassign by passing null group_id)
// @Tags         bulk
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "Node IDs and group ID"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/bulk/group [put]
func (h *BulkHandler) BulkAssignGroup(c *gin.Context) {
	var req struct {
		IDs     []uint `json:"ids" binding:"required,min=1"`
		GroupID *uint  `json:"group_id"` // nil to unassign
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Verify group exists if assigning
	if req.GroupID != nil && *req.GroupID > 0 {
		var group model.NodeGroup
		if err := getDB().First(&group, *req.GroupID).Error; err != nil {
			ErrorResponse(c, http.StatusNotFound, "Group not found")
			return
		}
	}

	result := getDB().Model(&model.Node{}).
		Where("id IN ?", req.IDs).
		Update("group_id", req.GroupID)

	WriteAuditLog(c, "node.bulk_group", "node", fmt.Sprintf("%v", req.IDs),
		fmt.Sprintf("Assigned %d nodes to group %v", result.RowsAffected, req.GroupID))

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"affected": result.RowsAffected,
		"message":  fmt.Sprintf("%d nodes updated", result.RowsAffected),
	})
}
