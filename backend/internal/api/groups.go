package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// GroupHandler handles node group and tag endpoints.
type GroupHandler struct{}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{}
}

// ListGroups godoc
// @Summary      List all node groups
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /groups [get]
func (h *GroupHandler) ListGroups(c *gin.Context) {
	var groups []model.NodeGroup
	getDB().Order("name ASC").Find(&groups)

	// Count nodes per group
	type groupCount struct {
		GroupID uint
		Count   int64
	}
	var counts []groupCount
	getDB().Model(&model.Node{}).
		Select("group_id, COUNT(*) as count").
		Where("group_id IS NOT NULL").
		Group("group_id").
		Find(&counts)

	countMap := make(map[uint]int64)
	for _, gc := range counts {
		countMap[gc.GroupID] = gc.Count
	}

	type GroupView struct {
		model.NodeGroup
		NodeCount int64 `json:"node_count"`
	}

	views := make([]GroupView, len(groups))
	for i, g := range groups {
		views[i] = GroupView{NodeGroup: g, NodeCount: countMap[g.ID]}
	}

	c.JSON(http.StatusOK, gin.H{"items": views, "total": len(views)})
}

// CreateGroup godoc
// @Summary      Create a node group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        group body object true "Group details"
// @Success      201 {object} model.NodeGroup
// @Router       /groups [post]
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Color == "" {
		req.Color = "#6366f1" // default indigo
	}

	group := model.NodeGroup{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	}

	if err := getDB().Create(&group).Error; err != nil {
		ErrorResponse(c, http.StatusConflict, "Group name already exists")
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateGroup godoc
// @Summary      Update a node group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Group ID"
// @Param        group body object true "Group details"
// @Success      200 {object} model.NodeGroup
// @Router       /groups/{id} [put]
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var group model.NodeGroup
	if err := getDB().First(&group, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "Group not found")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Color       *string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Description != nil {
		group.Description = *req.Description
	}
	if req.Color != nil {
		group.Color = *req.Color
	}

	if err := getDB().Save(&group).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to update group")
		return
	}

	c.JSON(http.StatusOK, group)
}

// DeleteGroup godoc
// @Summary      Delete a node group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Group ID"
// @Success      200 {object} map[string]string
// @Router       /groups/{id} [delete]
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	// Unset group_id for all nodes in this group
	getDB().Model(&model.Node{}).Where("group_id = ?", id).Update("group_id", nil)

	if err := getDB().Delete(&model.NodeGroup{}, id).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to delete group")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted"})
}

// AssignGroup godoc
// @Summary      Assign a node to a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Param        body body object true "Group assignment"
// @Success      200 {object} map[string]string
// @Router       /nodes/{id}/group [put]
func (h *GroupHandler) AssignGroup(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		GroupID *uint `json:"group_id"` // nil to unassign
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

	if err := getDB().Model(&model.Node{}).Where("id = ?", id).Update("group_id", req.GroupID).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to assign group")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group assigned"})
}

// ---- Tags ----

// ListNodeTags godoc
// @Summary      List tags for a node
// @Tags         tags
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/{id}/tags [get]
func (h *GroupHandler) ListNodeTags(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var tags []model.NodeTag
	getDB().Where("node_id = ?", id).Order("key ASC").Find(&tags)

	c.JSON(http.StatusOK, gin.H{"items": tags, "total": len(tags)})
}

// SetNodeTags godoc
// @Summary      Set tags for a node (replaces all existing tags)
// @Tags         tags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Param        tags body object true "Tags to set"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/{id}/tags [put]
func (h *GroupHandler) SetNodeTags(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		Tags []struct {
			Key   string `json:"key" binding:"required"`
			Value string `json:"value"`
		} `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Delete existing tags and replace
	tx := getDB().Begin()
	tx.Where("node_id = ?", id).Delete(&model.NodeTag{})

	for _, t := range req.Tags {
		tag := model.NodeTag{
			NodeID: uint(id),
			Key:    t.Key,
			Value:  t.Value,
		}
		tx.Create(&tag)
	}

	tx.Commit()

	// Return updated tags
	var tags []model.NodeTag
	getDB().Where("node_id = ?", id).Order("key ASC").Find(&tags)

	c.JSON(http.StatusOK, gin.H{"items": tags, "total": len(tags)})
}
