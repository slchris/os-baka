// Package vault provides an abstraction for secret storage.
// It supports HashiCorp Vault as the primary backend with a database fallback
// for environments where Vault is not available.
//
// Architecture:
//
//	SecretStore (interface)
//	 ├── VaultStore   — encrypts secrets in HashiCorp Vault KV v2
//	 └── DBStore      — stores secrets as plaintext in the database (legacy fallback)
//
// Usage:
//
//	store := vault.NewFromConfig(cfg)
//	store.StorePassphrase(ctx, nodeID, passphrase)
//	passphrase, _ := store.GetPassphrase(ctx, nodeID)
package vault

import (
	"context"
	"fmt"
	"log/slog"
)

// SecretStore abstracts secret storage backends.
// Implementations must be safe for concurrent use.
type SecretStore interface {
	// StorePassphrase stores a LUKS encryption passphrase for a node.
	StorePassphrase(ctx context.Context, nodeID uint, passphrase string) error

	// GetPassphrase retrieves the LUKS encryption passphrase for a node.
	GetPassphrase(ctx context.Context, nodeID uint) (string, error)

	// DeletePassphrase removes the stored passphrase for a node.
	DeletePassphrase(ctx context.Context, nodeID uint) error

	// Type returns the backend type name (e.g., "vault", "database").
	Type() string
}

// Config holds Vault connection configuration.
type Config struct {
	// Enabled determines whether Vault integration is active.
	Enabled bool `yaml:"enabled"`

	// Address is the Vault server URL (e.g., "http://127.0.0.1:8200").
	Address string `yaml:"address"`

	// Token is the Vault authentication token.
	// In production, prefer VAULT_TOKEN env var or auto-auth methods.
	Token string `yaml:"token"`

	// MountPath is the KV v2 secrets engine mount path (default: "secret").
	MountPath string `yaml:"mount_path"`

	// PathPrefix is the prefix for node passphrase paths (default: "osbaka/nodes").
	PathPrefix string `yaml:"path_prefix"`
}

// NodePassphrasePath returns the Vault KV path for a node's passphrase.
func (c *Config) NodePassphrasePath(nodeID uint) string {
	prefix := c.PathPrefix
	if prefix == "" {
		prefix = "osbaka/nodes"
	}
	return fmt.Sprintf("%s/%d", prefix, nodeID)
}

// NewFromConfig creates the appropriate SecretStore based on configuration.
// If Vault is enabled and reachable, returns a VaultStore.
// Otherwise falls back to DBStore with a warning.
func NewFromConfig(cfg *Config) SecretStore {
	if cfg == nil || !cfg.Enabled {
		slog.Info("Vault not configured, using database for passphrase storage",
			"warning", "Passphrases stored in plaintext in DB. Enable Vault for production.")
		return &DBStore{}
	}

	store, err := NewVaultStore(cfg)
	if err != nil {
		slog.Error("Failed to initialize Vault, falling back to database storage",
			"error", err,
			"address", cfg.Address)
		return &DBStore{}
	}

	slog.Info("Vault secret store initialized",
		"address", cfg.Address,
		"mount", cfg.MountPath,
		"prefix", cfg.PathPrefix)
	return store
}
