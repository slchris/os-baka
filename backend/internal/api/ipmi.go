package api

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// IPMIHandler handles IPMI/BMC power management endpoints.
type IPMIHandler struct{}

func NewIPMIHandler() *IPMIHandler {
	return &IPMIHandler{}
}

type PowerActionRequest struct {
	Action string `json:"action" binding:"required"` // on, off, reset, cycle, status
}

// PowerAction godoc
// @Summary      Execute IPMI power action
// @Description  Send a power command (on/off/reset/cycle/status) to a node via IPMI
// @Tags         ipmi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Param        action body PowerActionRequest true "Power action"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/{id}/power [post]
func (h *IPMIHandler) PowerAction(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var node model.Node
	if err := getDB().First(&node, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	if node.IPMIAddress == "" {
		ErrorResponse(c, http.StatusBadRequest, "IPMI not configured for this node")
		return
	}

	var req PowerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	validActions := map[string]string{
		"on":     "on",
		"off":    "off",
		"reset":  "reset",
		"cycle":  "cycle",
		"status": "status",
	}

	ipmiAction, ok := validActions[req.Action]
	if !ok {
		ErrorResponse(c, http.StatusBadRequest, "Invalid action. Must be one of: on, off, reset, cycle, status")
		return
	}

	// Build ipmitool command
	args := []string{
		"-I", "lanplus",
		"-H", node.IPMIAddress,
		"-U", node.IPMIUsername,
	}

	if node.IPMIPassword != "" {
		args = append(args, "-P", node.IPMIPassword)
	}

	if node.IPMIAllowUntrusted {
		args = append(args, "-C", "0") // No cipher suite enforcement
	}

	if req.Action == "status" {
		args = append(args, "power", "status")
	} else {
		args = append(args, "power", ipmiAction)
	}

	// Execute with timeout
	ctx := c.Request.Context()
	cmd := exec.CommandContext(ctx, "ipmitool", args...)

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		WriteAuditLog(c, "ipmi.power_error", "node", fmt.Sprintf("%d", id),
			fmt.Sprintf("Action: %s, Error: %v, Output: %s", req.Action, err, outputStr))
		ErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("IPMI command failed: %s", outputStr))
		return
	}

	WriteAuditLog(c, "ipmi.power_"+req.Action, "node", fmt.Sprintf("%d", id),
		fmt.Sprintf("Action: %s, Result: %s", req.Action, outputStr))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"action":  req.Action,
		"output":  outputStr,
		"node_id": id,
	})
}

// TestIPMI godoc
// @Summary      Test IPMI connectivity
// @Description  Test if the node's IPMI/BMC is reachable and credentials are valid
// @Tags         ipmi
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Success      200 {object} map[string]interface{}
// @Router       /nodes/{id}/ipmi/test [get]
func (h *IPMIHandler) TestIPMI(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var node model.Node
	if err := getDB().First(&node, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	if node.IPMIAddress == "" {
		ErrorResponse(c, http.StatusBadRequest, "IPMI not configured for this node")
		return
	}

	// Try a simple power status check as connectivity test
	args := []string{
		"-I", "lanplus",
		"-H", node.IPMIAddress,
		"-U", node.IPMIUsername,
	}
	if node.IPMIPassword != "" {
		args = append(args, "-P", node.IPMIPassword)
	}
	args = append(args, "power", "status")

	start := time.Now()
	cmd := exec.CommandContext(c.Request.Context(), "ipmitool", args...)
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"reachable":  false,
			"error":      strings.TrimSpace(string(output)),
			"latency_ms": elapsed.Milliseconds(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reachable":    true,
		"power_status": strings.TrimSpace(string(output)),
		"latency_ms":   elapsed.Milliseconds(),
	})
}
