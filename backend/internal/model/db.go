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

// DB is the global database connection. It is initialized by InitDB.
var DB *gorm.DB

// InitDB connects to PostgreSQL, verifies the connection, configures the
// connection pool, runs AutoMigrate for all models, and seeds default data.
// The process exits immediately if the database is unreachable.
func InitDB(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Verify connection is actually usable
	sqlDB, err := DB.DB()
	if err != nil {
		slog.Error("Failed to get underlying sql.DB", "error", err)
		os.Exit(1)
	}
	if err := sqlDB.Ping(); err != nil {
		slog.Error("Database ping failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Database connected successfully")

	// ── Connection pool ──
	maxIdle := cfg.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	maxOpen := cfg.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	slog.Info("Database pool configured", "max_idle", maxIdle, "max_open", maxOpen)

	// ── Auto Migrate ──
	if err := DB.AutoMigrate(AllModels()...); err != nil {
		slog.Error("Failed to migrate database", "error", err)
	}

	// ── Seed defaults ──
	seedDefaults()
}

// seedDefaults creates initial records if the database is empty.
func seedDefaults() {
	seedDHCPConfig()
	seedAdminUser()
}

func seedDHCPConfig() {
	var count int64
	DB.Model(&DHCPConfig{}).Count(&count)
	if count > 0 {
		return
	}
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

func seedAdminUser() {
	var count int64
	DB.Model(&User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin"
		slog.Warn("Using default admin password, set ADMIN_PASSWORD env var in production")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 14)
	if err != nil {
		slog.Error("Failed to hash admin password", "error", err)
		return
	}
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
