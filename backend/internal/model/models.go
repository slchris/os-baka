package model

import (
	"time"

	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────
// Identity
// ──────────────────────────────────────────────────

// User represents a platform user with authentication credentials and role.
type User struct {
	gorm.Model
	Email       string `gorm:"uniqueIndex:idx_user_email_active,where:deleted_at IS NULL"`
	Username    string `gorm:"uniqueIndex:idx_user_username_active,where:deleted_at IS NULL"`
	FullName    string
	Password    string
	IsSuperuser bool
}

// ──────────────────────────────────────────────────
// Node / Asset
// ──────────────────────────────────────────────────

// Node represents a bare-metal server managed by OS-Baka.
type Node struct {
	gorm.Model
	Hostname             string
	IPAddress            string
	MACAddress           string `gorm:"uniqueIndex:idx_node_mac_active,where:deleted_at IS NULL"`
	AssetTag             string
	Status               string // active, inactive, installing, maintenance, error, offline
	OSType               string // ubuntu, debian
	OSVersion            string `json:"os_version"`
	MirrorURL            string `json:"mirror_url"`     // Custom mirror URL for this node
	Timezone             string `json:"timezone"`       // Timezone for the node (e.g., UTC, Asia/Shanghai)
	RootPassword         string `json:"-"`              // Root password (stored hashed in preseed)
	SSHEnabled           bool   `json:"ssh_enabled"`    // Enable SSH server
	SSHRootLogin         bool   `json:"ssh_root_login"` // Allow root SSH login
	EncryptionEnabled    bool
	EncryptionPassphrase string `json:"-"`
	TPMEnabled           bool   `json:"tpm_enabled"`
	USBKeyRequired       bool   `json:"usb_key_required"`
	PCRBinding           string `json:"pcr_binding"` // comma-separated PCR ids, optional
	// IPMI / BMC
	IPMIAddress        string `json:"ipmi_address"`
	IPMIUsername       string `json:"ipmi_username"`
	IPMIPassword       string `json:"-"` // stored encrypted (AES-256-GCM)
	IPMIAllowUntrusted bool   `json:"ipmi_allow_untrusted"`
	// Heartbeat / Health
	LastHeartbeat *time.Time `json:"last_heartbeat"`
	CPUUsage      float64    `json:"cpu_usage"`    // percentage 0-100
	MemoryUsage   float64    `json:"memory_usage"` // percentage 0-100
	DiskUsage     float64    `json:"disk_usage"`   // percentage 0-100
	Uptime        int64      `json:"uptime"`       // seconds
	// Grouping
	GroupID *uint `json:"group_id"`
}

// NodeGroup allows organizing nodes into logical groups.
type NodeGroup struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `gorm:"uniqueIndex" json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"` // hex color for UI display
}

// NodeTag provides key-value labels for nodes.
type NodeTag struct {
	ID     uint   `gorm:"primarykey" json:"id"`
	NodeID uint   `gorm:"index" json:"node_id"`
	Key    string `gorm:"index" json:"key"`
	Value  string `json:"value"`
}

// ──────────────────────────────────────────────────
// DHCP
// ──────────────────────────────────────────────────

// DHCPConfig stores the DHCP server configuration.
type DHCPConfig struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"uniqueIndex" json:"name"`         // Configuration profile name
	Interface    string         `gorm:"default:'eth0'" json:"interface"` // Network interface
	RangeStart   string         `json:"range_start"`                     // DHCP range start IP
	RangeEnd     string         `json:"range_end"`                       // DHCP range end IP
	SubnetMask   string         `gorm:"default:'255.255.255.0'" json:"subnet_mask"`
	Gateway      string         `json:"gateway"`                                  // Default gateway
	DNSServer    string         `json:"dns_server"`                               // DNS server
	LeaseTime    string         `gorm:"default:'12h'" json:"lease_time"`          // DHCP lease time
	Domain       string         `gorm:"default:'os-baka.local'" json:"domain"`    // Domain name
	TFTPServer   string         `json:"tftp_server"`                              // TFTP server IP (for PXE)
	BootFile     string         `gorm:"default:'undionly.kpxe'" json:"boot_file"` // PXE boot file
	NextServer   string         `json:"next_server"`                              // Next server IP (usually same as TFTP)
	BootServerIP string         `json:"boot_server_ip"`                           // IP address for HTTP boot (API/Nginx)
	MirrorURL    string         `json:"mirror_url"`                               // OS mirror URL
	KernelParams string         `json:"kernel_params"`                            // Kernel parameters for PXE boot
	IsActive     bool           `gorm:"default:false" json:"is_active"`           // Is this the active configuration
	EnablePXE    bool           `gorm:"default:true" json:"enable_pxe"`           // Enable PXE boot
}

// DHCPReservation stores MAC-to-IP static mappings.
type DHCPReservation struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	MACAddress  string         `gorm:"uniqueIndex" json:"mac_address"` // Client MAC address
	IPAddress   string         `json:"ip_address"`                     // Reserved IP address
	Hostname    string         `json:"hostname"`                       // Hostname to assign
	Description string         `json:"description"`                    // Optional description
	IsActive    bool           `gorm:"default:true" json:"is_active"`  // Is reservation active
}

// ──────────────────────────────────────────────────
// PXE / Boot Assets
// ──────────────────────────────────────────────────

// BootAsset stores information about uploaded PXE assets (kernels, initrds, etc).
type BootAsset struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `json:"name"`      // Display name
	FileName  string         `json:"file_name"` // Actual file name on disk
	Type      string         `json:"type"`      // kernel, initrd, bootloader, other
	Size      int64          `json:"size"`
	Path      string         `json:"path"`     // Relative path in tftpboot
	CheckSum  string         `json:"checksum"` // SHA256
}

// ──────────────────────────────────────────────────
// Audit / Notifications
// ──────────────────────────────────────────────────

// AuditLog records security-relevant actions for compliance and troubleshooting.
type AuditLog struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time `json:"timestamp"`
	Action     string    `gorm:"index" json:"action"`      // e.g. node.create, user.login, dhcp.update
	UserID     uint      `json:"user_id"`                  // 0 for system/anonymous
	Username   string    `json:"user"`                     // denormalized for display
	Resource   string    `json:"resource"`                 // e.g. node, user, dhcp_config
	ResourceID string    `json:"resource_id"`              // ID of the affected resource
	Details    string    `gorm:"type:text" json:"details"` // JSON or free-form description
	IPAddress  string    `json:"ip_address"`               // Client IP
}

// Notification represents a user-facing notification.
type Notification struct {
	gorm.Model
	Title   string
	Message string
	Type    string // info, warning, error, success
	Read    bool
	UserID  uint
}

// ──────────────────────────────────────────────────
// API Keys
// ──────────────────────────────────────────────────

// APIKey enables programmatic access (CI/CD, Terraform, scripts).
type APIKey struct {
	ID         uint       `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Name       string     `json:"name"`                          // Human-readable description
	Prefix     string     `gorm:"uniqueIndex" json:"prefix"`     // First 8 chars for identification
	KeyHash    string     `gorm:"index" json:"-"`                // SHA-256 hash of full key
	Role       string     `json:"role"`                          // admin or operator
	CreatedBy  uint       `json:"created_by"`                    // User ID who created the key
	LastUsedAt *time.Time `json:"last_used_at"`                  // Last time the key was used
	ExpiresAt  *time.Time `json:"expires_at"`                    // Optional expiration
	IsActive   bool       `gorm:"default:true" json:"is_active"` // Soft disable
}

// AllModels returns all GORM model types for AutoMigrate.
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Node{},
		&Notification{},
		&DHCPConfig{},
		&DHCPReservation{},
		&BootAsset{},
		&AuditLog{},
		&NodeGroup{},
		&NodeTag{},
		&APIKey{},
	}
}
