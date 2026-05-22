package api

import (
	"fmt"
	"net/http"
	"time"

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

	// Bulk path: skip per-row legal-transition checking since the operator
	// explicitly selected these nodes. Both columns updated in one shot so
	// installing_started_at stays in sync with status for the timeout
	// watcher. A single audit log captures the batch — per-node entries
	// would flood the audit table for large rebuilds.
	result := getDB().Model(&model.Node{}).
		Where("id IN ?", req.IDs).
		Updates(map[string]any{
			"status":                NodeStatusInstalling,
			"installing_started_at": time.Now().UTC(),
		})

	WriteAuditLog(c, "node.bulk_rebuild", "node", fmt.Sprintf("%v", req.IDs),
		fmt.Sprintf("Rebuilt %d nodes", result.RowsAffected))

	// Auto power-cycle: re-fetch the rows so we have IPMI fields, then
	// fan out cycles through the bounded semaphore. Caller can opt out
	// of cycling with ?no_power_cycle=1 (e.g. during incident response
	// when you want to control reboots manually).
	powerCycleTally := map[string]int{}
	if c.Query("no_power_cycle") != "1" {
		var nodes []model.Node
		getDB().Where("id IN ?", req.IDs).Find(&nodes)
		powerCycleTally = TriggerPowerCycleBatch(nodes)
	} else {
		powerCycleTally["skipped:opted_out"] = len(req.IDs)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"affected":    result.RowsAffected,
		"message":     fmt.Sprintf("%d nodes set to rebuilding", result.RowsAffected),
		"power_cycle": powerCycleTally,
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

	// Read all nodes BEFORE deletion so the audit log retains
	// hostname/MAC/IP after the rows are gone. This is the only place
	// that survives the destructive action.
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

	// One audit entry per deleted node, written in a single batch INSERT.
	// Each row carries the full snapshot in details so we can answer
	// "which physical machine was that?" after the fact.
	auditUserID, _ := GetAuthUserID(c)
	auditUsername := GetAuthUsername(c)
	if auditUsername == "" {
		auditUsername = "system"
	}
	clientIP := c.ClientIP()
	auditLogs := make([]model.AuditLog, 0, len(nodes))
	for i := range nodes {
		auditLogs = append(auditLogs, model.AuditLog{
			Action:     "node.delete",
			UserID:     auditUserID,
			Username:   auditUsername,
			Resource:   "node",
			ResourceID: fmt.Sprintf("%d", nodes[i].ID),
			Details:    makeNodeDeleteSnapshot(&nodes[i]),
			IPAddress:  clientIP,
		})
	}
	WriteAuditLogs(auditLogs)

	// Rollup entry — useful for "show me bulk operations" queries.
	WriteAuditLog(c, "node.bulk_delete", "node", fmt.Sprintf("%v", req.IDs),
		fmt.Sprintf("Deleted %d nodes", result.RowsAffected))

	ScheduleDnsmasqRegen()

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
