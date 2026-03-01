package model

import (
	"log/slog"
	"os"
	"time"

	"github.com/os-baka/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	gorm.Model
	Email       string `gorm:"uniqueIndex:idx_user_email_active,where:deleted_at IS NULL"`
	Username    string `gorm:"uniqueIndex:idx_user_username_active,where:deleted_at IS NULL"`
	FullName    string
	Password    string
	IsSuperuser bool
}

type Node struct {
	gorm.Model
	Hostname             string
	IPAddress            string
	MACAddress           string `gorm:"uniqueIndex:idx_node_mac_active,where:deleted_at IS NULL"`
	AssetTag             string
	Status               string // active, inactive, installing, maintenance, error
	OSType               string // ubuntu, debian, centos, rhel, rocky, windows
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
}

type Notification struct {
	gorm.Model
	Title   string
	Message string
	Type    string // info, warning, error, success
	Read    bool
	UserID  uint
}

// DHCPConfig stores the DHCP server configuration
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
	MirrorURL    string         `json:"mirror_url"`                               // OS mirror URL (e.g., http://archive.ubuntu.com/ubuntu)
	KernelParams string         `json:"kernel_params"`                            // Kernel parameters for PXE boot
	IsActive     bool           `gorm:"default:false" json:"is_active"`           // Is this the active configuration
	EnablePXE    bool           `gorm:"default:true" json:"enable_pxe"`           // Enable PXE boot
}

// DHCPReservation stores MAC-to-IP static mappings
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

// BootAsset stores information about uploaded PXE assets (kernels, initrds, etc)
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

func InitDB(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return
	}

	slog.Info("Database connected successfully")

	// Auto Migrate
	err = DB.AutoMigrate(&User{}, &Node{}, &Notification{}, &DHCPConfig{}, &DHCPReservation{}, &BootAsset{})
	if err != nil {
		slog.Error("Failed to migrate database", "error", err)
	}

	// Seed default DHCP config if none exists
	var count int64
	DB.Model(&DHCPConfig{}).Count(&count)
	if count == 0 {
		defaultConfig := DHCPConfig{
			Name:       "default",
			Interface:  "eth0",
			RangeStart: "192.168.10.100",
			RangeEnd:   "192.168.10.200",
			SubnetMask: "255.255.255.0",
			Gateway:    "192.168.10.1",
			DNSServer:  "192.168.10.1",
			LeaseTime:  "12h",
			Domain:     "os-baka.local",
			BootFile:   "undionly.kpxe",
			IsActive:   true,
			EnablePXE:  true,
		}
		DB.Create(&defaultConfig)
		slog.Info("Created default DHCP configuration")
	}

	// Seed default admin user if none exists
	var userCount int64
	DB.Model(&User{}).Where("username = ?", "admin").Count(&userCount)
	if userCount == 0 {
		// Use ADMIN_PASSWORD env var, fallback to "admin" for initial setup
		adminPassword := os.Getenv("ADMIN_PASSWORD")
		if adminPassword == "" {
			adminPassword = "admin"
			slog.Warn("Using default admin password, set ADMIN_PASSWORD env var in production")
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 14)
		if err != nil {
			slog.Error("Failed to hash admin password", "error", err)
		} else {
			admin := User{
				Username:    "admin",
				Email:       "admin@os-baka.local",
				FullName:    "System Administrator",
				Password:    string(hashed),
				IsSuperuser: true,
			}
			DB.Create(&admin)
			slog.Info("Created default admin user", "username", "admin")
		}
	}
}
