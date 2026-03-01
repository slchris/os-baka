package api

import (
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type NodeHandler struct{}

func NewNodeHandler() *NodeHandler {
	return &NodeHandler{}
}

// ListNodes godoc
// @Summary      List all assets
// @Description  Get a list of all registered server nodes
// @Tags         assets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /nodes [get]
func (h *NodeHandler) ListNodes(c *gin.Context) {
	var nodes []model.Node
	getDB().Find(&nodes)

	// Map to frontend expectation if needed, or return raw
	// Frontend expects: id, hostname, mac_address, ip_address, status, ...
	// GORM field names match JSON defaults loosely, but might need explicit tags if camelCase is strict
	// Gin Gonic JSON serializer usually handles struct fields well enough.

	type NodeView struct {
		ID                uint   `json:"id"`
		Hostname          string `json:"hostname"`
		IPAddress         string `json:"ip_address"`
		MACAddress        string `json:"mac_address"`
		AssetTag          string `json:"asset_tag"`
		Status            string `json:"status"`
		OSType            string `json:"os_type"`
		OSVersion         string `json:"os_version"`
		MirrorURL         string `json:"mirror_url"`
		Timezone          string `json:"timezone"`
		SSHEnabled        bool   `json:"ssh_enabled"`
		SSHRootLogin      bool   `json:"ssh_root_login"`
		EncryptionEnabled bool   `json:"encryption_enabled"`
		TPMEnabled        bool   `json:"tpm_enabled"`
		USBKeyRequired    bool   `json:"usb_key_required"`
		PCRBinding        string `json:"pcr_binding"`
		CreatedAt         string `json:"created_at"`
		UpdatedAt         string `json:"updated_at"`
	}

	response := []NodeView{}
	for _, n := range nodes {
		slog.Debug("Node status from DB", "id", n.ID, "hostname", n.Hostname, "status", n.Status)
		response = append(response, NodeView{
			ID:                n.ID,
			Hostname:          n.Hostname,
			IPAddress:         n.IPAddress,
			MACAddress:        n.MACAddress,
			AssetTag:          n.AssetTag,
			SSHEnabled:        n.SSHEnabled,
			SSHRootLogin:      n.SSHRootLogin,
			Status:            n.Status,
			OSType:            n.OSType,
			OSVersion:         n.OSVersion,
			MirrorURL:         n.MirrorURL,
			Timezone:          n.Timezone,
			EncryptionEnabled: n.EncryptionEnabled,
			CreatedAt:         n.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         n.UpdatedAt.Format(time.RFC3339),
			TPMEnabled:        n.TPMEnabled,
			USBKeyRequired:    n.USBKeyRequired,
			PCRBinding:        n.PCRBinding,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": response, "total": len(response)})
}

func (h *NodeHandler) CreateNode(c *gin.Context) {
	var req struct {
		Hostname             string `json:"hostname" binding:"required"`
		IPAddress            string `json:"ip_address" binding:"required"`
		MACAddress           string `json:"mac_address" binding:"required"`
		AssetTag             string `json:"asset_tag"`
		Status               string `json:"status"`
		OSType               string `json:"os_type"`
		OSVersion            string `json:"os_version"`
		MirrorURL            string `json:"mirror_url"`
		Timezone             string `json:"timezone"`
		RootPassword         string `json:"root_password"`
		SSHEnabled           bool   `json:"ssh_enabled"`
		SSHRootLogin         bool   `json:"ssh_root_login"`
		EncryptionEnabled    bool   `json:"encryption_enabled"`
		EncryptionPassphrase string `json:"encryption_passphrase"`
		TPMEnabled           bool   `json:"tpm_enabled"`
		USBKeyRequired       bool   `json:"usb_key_required"`
		PCRBinding           string `json:"pcr_binding"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate MAC address format
	macRegex := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	if !macRegex.MatchString(req.MACAddress) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid MAC address format (expected XX:XX:XX:XX:XX:XX)")
		return
	}

	// Validate IP address format
	if net.ParseIP(req.IPAddress) == nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid IP address format")
		return
	}

	if req.EncryptionEnabled && strings.TrimSpace(req.EncryptionPassphrase) == "" {
		ErrorResponse(c, http.StatusBadRequest, "encryption_passphrase is required when encryption_enabled is true")
		return
	}

	// Hash root password before storing (preseed uses the hashed form)
	var hashedRootPassword string
	if req.RootPassword != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.RootPassword), 10)
		if err != nil {
			ErrorResponse(c, http.StatusInternalServerError, "Failed to hash root password")
			return
		}
		hashedRootPassword = string(hashed)
	}

	node := model.Node{
		Hostname:             req.Hostname,
		IPAddress:            req.IPAddress,
		MACAddress:           req.MACAddress,
		AssetTag:             req.AssetTag,
		Status:               req.Status,
		OSType:               req.OSType,
		OSVersion:            req.OSVersion,
		MirrorURL:            req.MirrorURL,
		Timezone:             req.Timezone,
		RootPassword:         hashedRootPassword,
		SSHEnabled:           req.SSHEnabled,
		SSHRootLogin:         req.SSHRootLogin,
		EncryptionEnabled:    req.EncryptionEnabled,
		EncryptionPassphrase: req.EncryptionPassphrase, // Kept recoverable for LUKS key recovery
		TPMEnabled:           req.TPMEnabled,
		USBKeyRequired:       req.USBKeyRequired,
		PCRBinding:           req.PCRBinding,
	}

	if result := getDB().Create(&node); result.Error != nil {
		ErrorResponse(c, http.StatusInternalServerError, result.Error.Error())
		return
	}

	// Sync to DHCP Reservation
	var reservation model.DHCPReservation
	if result := getDB().Where("mac_address = ?", node.MACAddress).First(&reservation); result.Error != nil {
		// Create new reservation
		reservation = model.DHCPReservation{
			MACAddress:  node.MACAddress,
			IPAddress:   node.IPAddress,
			Hostname:    node.Hostname,
			Description: "Auto-synced from node asset",
			IsActive:    true,
		}
		getDB().Create(&reservation)
	} else {
		// Update existing
		reservation.IPAddress = node.IPAddress
		reservation.Hostname = node.Hostname
		getDB().Save(&reservation)
	}

	// Regenerate dnsmasq config for DHCP reservations
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("dnsmasq config regeneration failed", "error", err)
	}

	c.JSON(http.StatusCreated, node)
}

func (h *NodeHandler) UpdateNode(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	var node model.Node
	if result := getDB().First(&node, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	var req struct {
		Hostname             string  `json:"hostname"`
		IPAddress            string  `json:"ip_address"`
		MACAddress           string  `json:"mac_address"`
		Status               string  `json:"status"`
		OSType               string  `json:"os_type"`
		OSVersion            string  `json:"os_version"`
		MirrorURL            string  `json:"mirror_url"`
		Timezone             *string `json:"timezone"`
		RootPassword         *string `json:"root_password"`
		SSHEnabled           *bool   `json:"ssh_enabled"`
		SSHRootLogin         *bool   `json:"ssh_root_login"`
		EncryptionEnabled    bool    `json:"encryption_enabled"`
		EncryptionPassphrase *string `json:"encryption_passphrase"`
		TPMEnabled           *bool   `json:"tpm_enabled"`
		USBKeyRequired       *bool   `json:"usb_key_required"`
		PCRBinding           *string `json:"pcr_binding"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.EncryptionEnabled && req.EncryptionPassphrase != nil && strings.TrimSpace(*req.EncryptionPassphrase) == "" {
		ErrorResponse(c, http.StatusBadRequest, "encryption_passphrase cannot be empty when provided")
		return
	}

	node.Hostname = req.Hostname
	node.IPAddress = req.IPAddress
	node.MACAddress = req.MACAddress
	node.Status = req.Status
	node.OSType = req.OSType
	node.OSVersion = req.OSVersion
	node.MirrorURL = req.MirrorURL
	if req.Timezone != nil {
		node.Timezone = *req.Timezone
	}
	if req.RootPassword != nil {
		node.RootPassword = *req.RootPassword
	}
	if req.SSHEnabled != nil {
		node.SSHEnabled = *req.SSHEnabled
	}
	if req.SSHRootLogin != nil {
		node.SSHRootLogin = *req.SSHRootLogin
	}
	node.EncryptionEnabled = req.EncryptionEnabled
	if req.EncryptionPassphrase != nil {
		node.EncryptionPassphrase = *req.EncryptionPassphrase
	}
	if req.TPMEnabled != nil {
		node.TPMEnabled = *req.TPMEnabled
	}
	if req.USBKeyRequired != nil {
		node.USBKeyRequired = *req.USBKeyRequired
	}
	if req.PCRBinding != nil {
		node.PCRBinding = *req.PCRBinding
	}

	getDB().Save(&node)

	// Sync to DHCP Reservation
	var reservation model.DHCPReservation
	if result := getDB().Where("mac_address = ?", node.MACAddress).First(&reservation); result.Error == nil {
		// Update existing
		reservation.IPAddress = node.IPAddress
		reservation.Hostname = node.Hostname
		getDB().Save(&reservation)
	} else {
		// Create if missing (user might have deleted reservation manually)
		reservation = model.DHCPReservation{
			MACAddress:  node.MACAddress,
			IPAddress:   node.IPAddress,
			Hostname:    node.Hostname,
			Description: "Auto-synced from node asset",
			IsActive:    true,
		}
		getDB().Create(&reservation)
	}

	// Regenerate dnsmasq config for DHCP reservations
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("dnsmasq config regeneration failed", "error", err)
	}

	c.JSON(http.StatusOK, node)
}

func (h *NodeHandler) DeleteNode(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	var node model.Node
	if result := getDB().First(&node, id); result.Error == nil {
		// Delete associated DHCP reservation
		getDB().Where("mac_address = ?", node.MACAddress).Delete(&model.DHCPReservation{})
	}

	getDB().Delete(&model.Node{}, id)

	// Regenerate dnsmasq config for DHCP reservations
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("dnsmasq config regeneration failed", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetPassphrase returns the stored encryption passphrase for a node.
// This endpoint is protected and should only be used by authorized users (e.g., to download recovery keys).
func (h *NodeHandler) GetPassphrase(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	var node model.Node
	if result := getDB().First(&node, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	if !node.EncryptionEnabled {
		ErrorResponse(c, http.StatusBadRequest, "Encryption not enabled for this node")
		return
	}

	if node.EncryptionPassphrase == "" {
		ErrorResponse(c, http.StatusNotFound, "No passphrase stored for this node")
		return
	}

	c.JSON(http.StatusOK, gin.H{"passphrase": node.EncryptionPassphrase})
}

// UpdateNodeStatus updates only the status field of a node.
// This is an internal endpoint used by postinstall scripts.
// @Summary      Update node status
// @Description  Update the status of a node (internal API)
// @Tags         internal
// @Accept       json
// @Produce      json
// @Param        id path int true "Node ID"
// @Param        body body object true "Status update"
// @Success      200  {object} map[string]interface{}
// @Router       /internal/nodes/{id}/status [put]
func (h *NodeHandler) UpdateNodeStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid node ID")
		return
	}

	var node model.Node
	if result := getDB().First(&node, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Validate status
	validStatuses := map[string]bool{
		"pending":     true,
		"installing":  true,
		"active":      true,
		"maintenance": true,
		"error":       true,
	}

	if !validStatuses[req.Status] {
		ErrorResponse(c, http.StatusBadRequest, "Invalid status value")
		return
	}

	slog.Debug("UpdateNodeStatus", "id", node.ID, "hostname", node.Hostname, "from", node.Status, "to", req.Status)
	node.Status = req.Status
	getDB().Save(&node)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  node.Status,
		"message": "Node status updated successfully",
	})
}

// RebuildNode triggers a rebuild of a node by setting its status to 'installing'
// @Summary      Rebuild a node
// @Description  Set node status to 'installing' to trigger reinstallation on next boot
// @Tags         assets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Success      200  {object} map[string]interface{}
// @Router       /nodes/{id}/rebuild [post]
func (h *NodeHandler) RebuildNode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid node ID")
		return
	}

	var node model.Node
	if result := getDB().First(&node, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Node not found")
		return
	}

	// Set status to installing to trigger reinstallation
	node.Status = "installing"
	getDB().Save(&node)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Node rebuild initiated. System will reinstall on next boot.",
		"status":  node.Status,
	})
}
