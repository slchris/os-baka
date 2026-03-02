package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
	"github.com/os-baka/backend/internal/sysutil"
)

type DHCPHandler struct{}

func NewDHCPHandler() *DHCPHandler {
	return &DHCPHandler{}
}

// System Handler for system-level ops
type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// ListInterfaces godoc
// @Summary      List network interfaces on the server
// @Tags         system
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} []sysutil.NetworkInterface
// @Router       /system/interfaces [get]
func (h *SystemHandler) ListInterfaces(c *gin.Context) {
	ifaces, err := sysutil.ListInterfaces()
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, ifaces)
}

// ===== DHCP Configuration =====

// ListConfigs godoc
// @Summary      List all DHCP configurations
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /dhcp/configs [get]
func (h *DHCPHandler) ListConfigs(c *gin.Context) {
	configs := []model.DHCPConfig{}
	getDB().Find(&configs)

	c.JSON(http.StatusOK, gin.H{"items": configs, "total": len(configs)})
}

// GetConfig godoc
// @Summary      Get a DHCP configuration by ID
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Config ID"
// @Success      200  {object} model.DHCPConfig
// @Router       /dhcp/configs/{id} [get]
func (h *DHCPHandler) GetConfig(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	var config model.DHCPConfig
	if result := getDB().First(&config, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Configuration not found")
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetActiveConfig godoc
// @Summary      Get the active DHCP configuration
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} model.DHCPConfig
// @Router       /dhcp/config/active [get]
func (h *DHCPHandler) GetActiveConfig(c *gin.Context) {
	var config model.DHCPConfig
	if result := getDB().Where("is_active = ?", true).First(&config); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "No active configuration found")
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateConfig godoc
// @Summary      Create a new DHCP configuration
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      201  {object} model.DHCPConfig
// @Router       /dhcp/configs [post]
func (h *DHCPHandler) CreateConfig(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Interface    string `json:"interface"`
		RangeStart   string `json:"range_start" binding:"required"`
		RangeEnd     string `json:"range_end" binding:"required"`
		SubnetMask   string `json:"subnet_mask"`
		Gateway      string `json:"gateway"`
		DNSServer    string `json:"dns_server"`
		LeaseTime    string `json:"lease_time"`
		Domain       string `json:"domain"`
		TFTPServer   string `json:"tftp_server"`
		BootFile     string `json:"boot_file"`
		NextServer   string `json:"next_server"`
		KernelParams string `json:"kernel_params"`
		IsActive     bool   `json:"is_active"`
		EnablePXE    bool   `json:"enable_pxe"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	config := model.DHCPConfig{
		Name:         req.Name,
		Interface:    req.Interface,
		RangeStart:   req.RangeStart,
		RangeEnd:     req.RangeEnd,
		SubnetMask:   req.SubnetMask,
		Gateway:      req.Gateway,
		DNSServer:    req.DNSServer,
		LeaseTime:    req.LeaseTime,
		Domain:       req.Domain,
		TFTPServer:   req.TFTPServer,
		BootFile:     req.BootFile,
		NextServer:   req.NextServer,
		KernelParams: req.KernelParams,
		IsActive:     req.IsActive,
		EnablePXE:    req.EnablePXE,
	}

	// Set defaults
	// Interface defaults to empty (listen on all)
	if config.SubnetMask == "" {
		config.SubnetMask = "255.255.255.0"
	}
	if config.LeaseTime == "" {
		config.LeaseTime = "12h"
	}
	if config.Domain == "" {
		config.Domain = "os-baka.local"
	}
	if config.BootFile == "" {
		config.BootFile = "undionly.kpxe"
	}

	// If this is marked as active, deactivate others
	if config.IsActive {
		getDB().Model(&model.DHCPConfig{}).Where("is_active = ?", true).Update("is_active", false)
	}

	if result := getDB().Create(&config); result.Error != nil {
		ErrorResponse(c, http.StatusInternalServerError, result.Error.Error())
		return
	}

	// Regenerate dnsmasq config
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Failed to regenerate dnsmasq config", "error", err)
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateConfig godoc
// @Summary      Update a DHCP configuration
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Config ID"
// @Success      200  {object} model.DHCPConfig
// @Router       /dhcp/configs/{id} [put]
func (h *DHCPHandler) UpdateConfig(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	var config model.DHCPConfig
	if result := getDB().First(&config, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Configuration not found")
		return
	}

	var req struct {
		Name         string `json:"name"`
		Interface    string `json:"interface"`
		RangeStart   string `json:"range_start"`
		RangeEnd     string `json:"range_end"`
		SubnetMask   string `json:"subnet_mask"`
		Gateway      string `json:"gateway"`
		DNSServer    string `json:"dns_server"`
		LeaseTime    string `json:"lease_time"`
		Domain       string `json:"domain"`
		TFTPServer   string `json:"tftp_server"`
		BootFile     string `json:"boot_file"`
		NextServer   string `json:"next_server"`
		KernelParams string `json:"kernel_params"`
		IsActive     bool   `json:"is_active"`
		EnablePXE    bool   `json:"enable_pxe"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// If this is marked as active, deactivate others
	if req.IsActive && !config.IsActive {
		getDB().Model(&model.DHCPConfig{}).Where("is_active = ? AND id != ?", true, id).Update("is_active", false)
	}

	config.Name = req.Name
	config.Interface = req.Interface
	config.RangeStart = req.RangeStart
	config.RangeEnd = req.RangeEnd
	config.SubnetMask = req.SubnetMask
	config.Gateway = req.Gateway
	config.DNSServer = req.DNSServer
	config.LeaseTime = req.LeaseTime
	config.Domain = req.Domain
	config.TFTPServer = req.TFTPServer
	config.BootFile = req.BootFile
	config.NextServer = req.NextServer
	config.KernelParams = req.KernelParams
	config.IsActive = req.IsActive
	config.EnablePXE = req.EnablePXE

	getDB().Save(&config)

	// Regenerate dnsmasq config
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Failed to regenerate dnsmasq config", "error", err)
	}

	c.JSON(http.StatusOK, config)
}

// DeleteConfig godoc
// @Summary      Delete a DHCP configuration
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Config ID"
// @Success      200  {object} map[string]interface{}
// @Router       /dhcp/configs/{id} [delete]
func (h *DHCPHandler) DeleteConfig(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var config model.DHCPConfig
	if result := getDB().First(&config, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Configuration not found")
		return
	}

	getDB().Delete(&model.DHCPConfig{}, id)

	message := "Configuration deleted"
	if config.IsActive {
		message = "Active configuration deleted. Please activate another configuration."
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

// RestartService godoc
// @Summary      Restart the DHCP service (regenerate config)
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /dhcp/service/restart [post]
func (h *DHCPHandler) RestartService(c *gin.Context) {
	if err := GenerateDnsmasqConfig(); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to regenerate config: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Service configuration regenerated"})
}

// ===== DHCP Reservations =====

// ListReservations godoc
// @Summary      List all DHCP reservations
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /dhcp/reservations [get]
func (h *DHCPHandler) ListReservations(c *gin.Context) {
	reservations := []model.DHCPReservation{}
	getDB().Find(&reservations)

	c.JSON(http.StatusOK, gin.H{"items": reservations, "total": len(reservations)})
}

// CreateReservation godoc
// @Summary      Create a DHCP reservation
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      201  {object} model.DHCPReservation
// @Router       /dhcp/reservations [post]
func (h *DHCPHandler) CreateReservation(c *gin.Context) {
	var req struct {
		MACAddress  string `json:"mac_address" binding:"required"`
		IPAddress   string `json:"ip_address" binding:"required"`
		Hostname    string `json:"hostname" binding:"required"`
		Description string `json:"description"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	reservation := model.DHCPReservation{
		MACAddress:  req.MACAddress,
		IPAddress:   req.IPAddress,
		Hostname:    req.Hostname,
		Description: req.Description,
		IsActive:    req.IsActive,
	}

	if result := getDB().Create(&reservation); result.Error != nil {
		ErrorResponse(c, http.StatusInternalServerError, result.Error.Error())
		return
	}

	// Regenerate dnsmasq config
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Failed to regenerate dnsmasq config", "error", err)
	}

	c.JSON(http.StatusCreated, reservation)
}

// UpdateReservation godoc
// @Summary      Update a DHCP reservation
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Reservation ID"
// @Success      200  {object} model.DHCPReservation
// @Router       /dhcp/reservations/{id} [put]
func (h *DHCPHandler) UpdateReservation(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	var reservation model.DHCPReservation
	if result := getDB().First(&reservation, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Reservation not found")
		return
	}

	var req struct {
		MACAddress  string `json:"mac_address"`
		IPAddress   string `json:"ip_address"`
		Hostname    string `json:"hostname"`
		Description string `json:"description"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	reservation.MACAddress = req.MACAddress
	reservation.IPAddress = req.IPAddress
	reservation.Hostname = req.Hostname
	reservation.Description = req.Description
	reservation.IsActive = req.IsActive

	getDB().Save(&reservation)

	// Regenerate dnsmasq config
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Failed to regenerate dnsmasq config", "error", err)
	}

	c.JSON(http.StatusOK, reservation)
}

// DeleteReservation godoc
// @Summary      Delete a DHCP reservation
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Reservation ID"
// @Success      200  {object} map[string]interface{}
// @Router       /dhcp/reservations/{id} [delete]
func (h *DHCPHandler) DeleteReservation(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	getDB().Delete(&model.DHCPReservation{}, id)

	// Regenerate dnsmasq config
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Failed to regenerate dnsmasq config", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// SyncFromNodes godoc
// @Summary      Sync DHCP reservations from registered nodes
// @Tags         dhcp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /dhcp/reservations/sync [post]
func (h *DHCPHandler) SyncFromNodes(c *gin.Context) {
	var nodes []model.Node
	getDB().Find(&nodes)

	created := 0
	updated := 0

	for _, node := range nodes {
		if node.MACAddress == "" || node.IPAddress == "" {
			continue
		}

		var existing model.DHCPReservation
		result := getDB().Where("mac_address = ?", node.MACAddress).First(&existing)

		if result.Error != nil {
			// Create new reservation
			reservation := model.DHCPReservation{
				MACAddress:  node.MACAddress,
				IPAddress:   node.IPAddress,
				Hostname:    node.Hostname,
				Description: "Auto-synced from node",
				IsActive:    true,
			}
			getDB().Create(&reservation)
			created++
		} else {
			// Update existing
			existing.IPAddress = node.IPAddress
			existing.Hostname = node.Hostname
			getDB().Save(&existing)
			updated++
		}
	}

	// Regenerate dnsmasq config
	if err := GenerateDnsmasqConfig(); err != nil {
		slog.Warn("Failed to regenerate dnsmasq config", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"created": created,
		"updated": updated,
	})
}
