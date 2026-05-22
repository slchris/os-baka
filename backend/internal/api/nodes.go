package api

import (
	cryptoRand "crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
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
	// ── Pagination params ──
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	search := c.Query("search")
	status := c.Query("status")
	sortParam := c.DefaultQuery("sort", "-created_at")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	db := getDB().Model(&model.Node{})

	// ── Search filter (hostname, IP, MAC) ──
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("hostname ILIKE ? OR ip_address ILIKE ? OR mac_address ILIKE ?", like, like, like)
	}

	// ── Status filter ──
	if status != "" {
		db = db.Where("status = ?", status)
	}

	// ── Count total before pagination ──
	var total int64
	db.Count(&total)

	// ── Sort ──
	allowedSorts := map[string]string{
		"hostname": "hostname", "ip_address": "ip_address",
		"status": "status", "created_at": "created_at", "updated_at": "updated_at",
	}
	orderClause := "created_at DESC"
	if sortParam != "" {
		desc := false
		field := sortParam
		if strings.HasPrefix(field, "-") {
			desc = true
			field = field[1:]
		}
		if col, ok := allowedSorts[field]; ok {
			orderClause = col
			if desc {
				orderClause += " DESC"
			}
		}
	}

	// ── Fetch page ──
	offset := (page - 1) * pageSize
	var nodes []model.Node
	db.Order(orderClause).Offset(offset).Limit(pageSize).Find(&nodes)

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

	response := make([]NodeView, 0, len(nodes))
	for _, n := range nodes {
		response = append(response, NodeView{
			ID: n.ID, Hostname: n.Hostname, IPAddress: n.IPAddress,
			MACAddress: n.MACAddress, AssetTag: n.AssetTag,
			SSHEnabled: n.SSHEnabled, SSHRootLogin: n.SSHRootLogin,
			Status: n.Status, OSType: n.OSType, OSVersion: n.OSVersion,
			MirrorURL: n.MirrorURL, Timezone: n.Timezone,
			EncryptionEnabled: n.EncryptionEnabled,
			CreatedAt:         n.CreatedAt.Format(time.RFC3339),
			UpdatedAt:         n.UpdatedAt.Format(time.RFC3339),
			TPMEnabled:        n.TPMEnabled, USBKeyRequired: n.USBKeyRequired,
			PCRBinding: n.PCRBinding,
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       response,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
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
		SecurityMirrorURL    string `json:"security_mirror_url"`
		Timezone             string `json:"timezone"`
		RootPassword         string `json:"root_password"`
		SSHEnabled           bool   `json:"ssh_enabled"`
		SSHRootLogin         bool   `json:"ssh_root_login"`
		EncryptionEnabled    bool   `json:"encryption_enabled"`
		EncryptionPassphrase string `json:"encryption_passphrase"`
		TPMEnabled           bool   `json:"tpm_enabled"`
		USBKeyRequired       bool   `json:"usb_key_required"`
		PCRBinding           string `json:"pcr_binding"`
		IPMIAddress          string `json:"ipmi_address"`
		IPMIUsername         string `json:"ipmi_username"`
		IPMIPassword         string `json:"ipmi_password"`
		IPMIAllowUntrusted   bool   `json:"ipmi_allow_untrusted"`
		TargetDisk           string `json:"target_disk"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.TargetDisk != "" && !isValidDiskPath(req.TargetDisk) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid target_disk format (must match ^/dev/[a-zA-Z0-9/_.-]+$)")
		return
	}
	if req.MirrorURL != "" && !isValidMirrorURL(req.MirrorURL) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid mirror_url (must be http(s)://host[/path])")
		return
	}
	if req.SecurityMirrorURL != "" && !isValidMirrorURL(req.SecurityMirrorURL) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid security_mirror_url (must be http(s)://host[/path])")
		return
	}

	// Validate hostname: RFC1123 single-label, 1-63 chars, [a-zA-Z0-9-],
	// no leading/trailing hyphen, no dot. Flows into preseed, kernel
	// cmdline, hostnamectl — escape-once-at-edge is simpler than the
	// per-sink quoting fan-out.
	if !isValidHostname(req.Hostname) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid hostname (must be RFC1123 single-label: 1-63 chars, [a-zA-Z0-9-], no leading/trailing hyphen, no dot)")
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
	if err := validatePassphrase(req.EncryptionPassphrase); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "encryption_passphrase: "+err.Error())
		return
	}
	if err := validatePassphrase(req.RootPassword); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "root_password: "+err.Error())
		return
	}

	// Default initial status. Operator-registered nodes are pending until
	// they PXE-boot and run through install. An explicit non-empty status
	// must be one we recognize — otherwise typos like "Pending" silently
	// create unmanaged nodes.
	if req.Status == "" {
		req.Status = NodeStatusPending
	} else if !IsKnownNodeStatus(req.Status) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid status value: "+req.Status)
		return
	}

	// Root password is stored as AES-GCM ciphertext; the installer hashes the
	// plaintext via crypt(3) during preseed. Do NOT bcrypt here — that would
	// produce a hash the installer treats as the plaintext password.
	node := model.Node{
		Hostname:             req.Hostname,
		IPAddress:            req.IPAddress,
		MACAddress:           req.MACAddress,
		AssetTag:             req.AssetTag,
		Status:               req.Status,
		OSType:               req.OSType,
		OSVersion:            req.OSVersion,
		MirrorURL:            req.MirrorURL,
		SecurityMirrorURL:    req.SecurityMirrorURL,
		Timezone:             req.Timezone,
		RootPassword:         req.RootPassword,
		SSHEnabled:           req.SSHEnabled,
		SSHRootLogin:         req.SSHRootLogin,
		EncryptionEnabled:    req.EncryptionEnabled,
		EncryptionPassphrase: req.EncryptionPassphrase, // Kept recoverable for LUKS key recovery
		TPMEnabled:           req.TPMEnabled,
		USBKeyRequired:       req.USBKeyRequired,
		PCRBinding:           req.PCRBinding,
		IPMIAddress:          req.IPMIAddress,
		IPMIUsername:         req.IPMIUsername,
		IPMIPassword:         req.IPMIPassword,
		IPMIAllowUntrusted:   req.IPMIAllowUntrusted,
		TargetDisk:           req.TargetDisk,
	}

	// If the initial status is `installing`, stamp the start time so the
	// install-timeout watcher can catch stuck nodes. `pending` means the
	// node is registered but has not begun provisioning — no clock yet.
	if node.Status == NodeStatusInstalling {
		now := time.Now().UTC()
		node.InstallingStartedAt = &now
	}

	// Encrypt sensitive fields before storage
	node.IPMIPassword = EncryptField(node.IPMIPassword)
	node.RootPassword = EncryptField(node.RootPassword)
	node.EncryptionPassphrase = EncryptField(node.EncryptionPassphrase)

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

	ScheduleDnsmasqRegen()

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
		SecurityMirrorURL    string  `json:"security_mirror_url"`
		Timezone             *string `json:"timezone"`
		RootPassword         *string `json:"root_password"`
		SSHEnabled           *bool   `json:"ssh_enabled"`
		SSHRootLogin         *bool   `json:"ssh_root_login"`
		EncryptionEnabled    bool    `json:"encryption_enabled"`
		EncryptionPassphrase *string `json:"encryption_passphrase"`
		TPMEnabled           *bool   `json:"tpm_enabled"`
		USBKeyRequired       *bool   `json:"usb_key_required"`
		PCRBinding           *string `json:"pcr_binding"`
		IPMIAddress          *string `json:"ipmi_address"`
		IPMIUsername         *string `json:"ipmi_username"`
		IPMIPassword         *string `json:"ipmi_password"`
		IPMIAllowUntrusted   *bool   `json:"ipmi_allow_untrusted"`
		TargetDisk           *string `json:"target_disk"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// target_disk: nil = leave alone; "" = clear (back to auto-detect);
	// non-empty = validate. Same three-state pattern as ipmi_password.
	if req.TargetDisk != nil && *req.TargetDisk != "" && !isValidDiskPath(*req.TargetDisk) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid target_disk format (must match ^/dev/[a-zA-Z0-9/_.-]+$)")
		return
	}
	if req.MirrorURL != "" && !isValidMirrorURL(req.MirrorURL) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid mirror_url (must be http(s)://host[/path])")
		return
	}
	if req.SecurityMirrorURL != "" && !isValidMirrorURL(req.SecurityMirrorURL) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid security_mirror_url (must be http(s)://host[/path])")
		return
	}
	// UpdateNode writes Hostname unconditionally (it's a plain-string
	// field, not the pointer pattern used for partial fields). That
	// means an omitted hostname becomes empty and breaks the node.
	// Reject empty here too — same rule as CreateNode.
	if !isValidHostname(req.Hostname) {
		ErrorResponse(c, http.StatusBadRequest, "Invalid hostname (must be RFC1123 single-label: 1-63 chars, [a-zA-Z0-9-], no leading/trailing hyphen, no dot)")
		return
	}

	if req.EncryptionEnabled && req.EncryptionPassphrase != nil && strings.TrimSpace(*req.EncryptionPassphrase) == "" {
		ErrorResponse(c, http.StatusBadRequest, "encryption_passphrase cannot be empty when provided")
		return
	}
	if req.EncryptionPassphrase != nil {
		if err := validatePassphrase(*req.EncryptionPassphrase); err != nil {
			ErrorResponse(c, http.StatusBadRequest, "encryption_passphrase: "+err.Error())
			return
		}
	}
	if req.RootPassword != nil {
		if err := validatePassphrase(*req.RootPassword); err != nil {
			ErrorResponse(c, http.StatusBadRequest, "root_password: "+err.Error())
			return
		}
	}

	node.Hostname = req.Hostname
	node.IPAddress = req.IPAddress
	node.MACAddress = req.MACAddress
	node.Status = req.Status
	node.OSType = req.OSType
	node.OSVersion = req.OSVersion
	node.MirrorURL = req.MirrorURL
	node.SecurityMirrorURL = req.SecurityMirrorURL
	if req.Timezone != nil {
		node.Timezone = *req.Timezone
	}
	if req.RootPassword != nil && *req.RootPassword != "" {
		// Store AES-GCM encrypted plaintext; installer hashes it during preseed.
		node.RootPassword = EncryptField(*req.RootPassword)
	}
	if req.SSHEnabled != nil {
		node.SSHEnabled = *req.SSHEnabled
	}
	if req.SSHRootLogin != nil {
		node.SSHRootLogin = *req.SSHRootLogin
	}
	node.EncryptionEnabled = req.EncryptionEnabled
	if req.EncryptionPassphrase != nil {
		// Store AES-GCM encrypted; preseed decrypts at provisioning time.
		node.EncryptionPassphrase = EncryptField(*req.EncryptionPassphrase)
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
	if req.IPMIAddress != nil {
		node.IPMIAddress = *req.IPMIAddress
	}
	if req.IPMIUsername != nil {
		node.IPMIUsername = *req.IPMIUsername
	}
	// IPMIPassword semantics: nil = leave untouched, empty string = caller
	// explicitly cleared it (e.g. removing BMC creds), non-empty = new value.
	// We must NOT silently clobber stored ciphertext when the field is absent
	// from the request (Update is partial), and we must NOT re-encrypt
	// existing ciphertext.
	if req.IPMIPassword != nil {
		if *req.IPMIPassword == "" {
			node.IPMIPassword = ""
		} else {
			node.IPMIPassword = EncryptField(*req.IPMIPassword)
		}
	}
	if req.IPMIAllowUntrusted != nil {
		node.IPMIAllowUntrusted = *req.IPMIAllowUntrusted
	}
	// TargetDisk: nil = leave alone; empty = clear (auto-detect);
	// non-empty = set (validated above).
	if req.TargetDisk != nil {
		node.TargetDisk = *req.TargetDisk
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

	ScheduleDnsmasqRegen()

	c.JSON(http.StatusOK, node)
}

func (h *NodeHandler) DeleteNode(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}
	// Capture the snapshot BEFORE deleting — the auto-audit middleware only
	// sees the request path and HTTP status, not what was actually destroyed.
	// After the row is gone there is no recovery path.
	var node model.Node
	found := getDB().First(&node, id).Error == nil
	if found {
		// Delete associated DHCP reservation
		getDB().Where("mac_address = ?", node.MACAddress).Delete(&model.DHCPReservation{})
	}

	if err := getDB().Delete(&model.Node{}, id).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to delete node")
		return
	}

	if found {
		WriteAuditLog(c, "node.delete", "node", strconv.Itoa(id), makeNodeDeleteSnapshot(&node))
	}

	ScheduleDnsmasqRegen()

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

	plaintext := DecryptField(node.EncryptionPassphrase)
	c.JSON(http.StatusOK, gin.H{"passphrase": plaintext})
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

	if err := SetNodeStatus(c, getDB(), &node, req.Status, "internal API callback"); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			ErrorResponse(c, http.StatusBadRequest, err.Error())
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

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

	if err := SetNodeStatus(c, getDB(), &node, NodeStatusInstalling, "operator rebuild"); err != nil {
		if errors.Is(err, ErrIllegalTransition) {
			ErrorResponse(c, http.StatusConflict, err.Error())
			return
		}
		ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Auto power-cycle: send the BMC a `power cycle` so the node actually
	// PXE-boots into the reinstall instead of waiting for someone to walk
	// to the rack. Async — the HTTP response returns "scheduled" and the
	// real outcome lands in the audit log.
	powerCycleOutcome := "skipped:opted_out"
	if c.Query("no_power_cycle") != "1" {
		powerCycleOutcome = TriggerPowerCycle(&node)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Node rebuild initiated. System will reinstall on next boot.",
		"status":      node.Status,
		"power_cycle": powerCycleOutcome,
	})
}

// RotatePassphrase generates a new encryption passphrase for a node,
// stores it in Vault (or DB fallback), and returns the new passphrase.
// @Summary      Rotate node encryption passphrase
// @Description  Generate a new passphrase, store it in the secret store, and return it
// @Tags         assets
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Node ID"
// @Success      200  {object} map[string]interface{}
// @Failure      400  {object} map[string]string
// @Failure      404  {object} map[string]string
// @Router       /nodes/{id}/rotate-passphrase [post]
func (h *NodeHandler) RotatePassphrase(c *gin.Context) {
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

	// Generate cryptographically secure passphrase (32 chars)
	newPassphrase, err := generateSecurePassphrase(32)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to generate passphrase")
		return
	}

	// Persist AES-GCM encrypted; preseed decrypts at provisioning time.
	if err := getDB().Model(&node).Update("encryption_passphrase", EncryptField(newPassphrase)).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to persist new passphrase")
		return
	}

	WriteAuditLog(c, "node.rotate_passphrase", "node", strconv.Itoa(id), "Passphrase rotated")

	c.JSON(http.StatusOK, gin.H{
		"passphrase": newPassphrase,
		"message":    "Passphrase rotated successfully",
	})
}

// hostnamePattern enforces an RFC1123 single-label hostname:
//   - 1 to 63 characters
//   - letters, digits, hyphen only
//   - cannot start or end with a hyphen
//
// Dots are deliberately NOT allowed. OS-Baka treats Node.Hostname as the
// short host name; the domain part comes from preseed (currently fixed
// at os-baka.local). Allowing dots would let an operator type
// "node.bad.example" which then gets jammed verbatim into
// `d-i netcfg/hostname string ...`, producing a fqdn-shaped value where
// Debian wants the short name. Reject upfront — less surprising.
//
// We reject upfront rather than escape downstream because hostname flows
// to many sinks (iPXE imgargs, preseed debconf lines, postinstall
// `hostnamectl set-hostname`, comments, eventually systemd / DNS / SSH
// on the installed OS). Each sink has different escape requirements;
// the union of "safe in all of them" is RFC1123 anyway.
var hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

// isValidHostname reports whether s is an acceptable Node hostname.
func isValidHostname(s string) bool {
	return hostnamePattern.MatchString(s)
}

// validatePassphrase rejects passphrases that can't safely round-trip
// through the preseed pipeline. Specifically:
//
//   - NUL byte: debconf truncates the value, so the stored "passphrase"
//     becomes a prefix of what the operator typed → LUKS unlock fails.
//   - Newline / CR: would break preseed line structure (it's one line per
//     directive).
//
// Other shell metacharacters (`$`, `'`, `"`, backtick) are NOT rejected
// here — the TPM config write now base64-encodes the value (see
// pxe.go), and the preseed `partman-crypto/passphrase password ...`
// directive consumes the value via debconf, not shell. The remaining
// risk is purely the two control characters above.
//
// Empty string is allowed (caller's responsibility to reject when
// encryption_enabled = true).
func validatePassphrase(pw string) error {
	for i := 0; i < len(pw); i++ {
		switch pw[i] {
		case 0:
			return fmt.Errorf("contains NUL byte at offset %d", i)
		case '\n', '\r':
			return fmt.Errorf("contains newline at offset %d", i)
		}
	}
	return nil
}

// generateSecurePassphrase creates a cryptographically secure random
// passphrase. Charset is intentionally restricted to URL-safe base64
// alphabet (A-Za-z0-9_-): 64 chars exactly. Two reasons:
//
//   1. Shell safety. The passphrase travels through preseed `late_command`
//      and a TPM-config write that uses `echo "..." > /etc/osbaka-tpm.conf`
//      inside the installed system. Inside double quotes, `$`, backtick,
//      and backslash trigger expansion; `!` triggers bash history; quotes
//      break tokenization. The earlier charset included `!@#$%^&*()` —
//      meaning real-world LUKS keys could differ from what got written to
//      the TPM config (e.g. `pass$word` -> `password`), bricking the disk
//      on first reboot.
//
//   2. No modulo bias. 256 / 64 = 4 exactly, so `byte % 64` is uniform.
//      The old 75-char set introduced a slight skew toward early chars.
//
// A 32-char string from this set holds 32*6 = 192 bits of entropy — far
// above any LUKS brute-force threat model. Longer doesn't help; shorter
// would be fine too.
func generateSecurePassphrase(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	if len(charset) != 64 {
		// Defensive — if someone edits the charset and breaks the
		// even-divides-256 property, fail loud rather than silently
		// regress to modulo bias.
		return "", fmt.Errorf("charset length must be a power-of-two divisor of 256, got %d", len(charset))
	}
	result := make([]byte, length)
	randomBytes := make([]byte, length)
	if _, err := cryptoRandRead(randomBytes); err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		result[i] = charset[int(randomBytes[i])%len(charset)]
	}
	return string(result), nil
}

// cryptoRandRead is a variable to allow testing. Default is crypto/rand.Read.
var cryptoRandRead = cryptoRandReadDefault

func cryptoRandReadDefault(b []byte) (int, error) {
	return cryptoRand.Read(b)
}
