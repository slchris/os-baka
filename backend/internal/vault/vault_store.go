package vault

import (
	"context"
	"fmt"
	"log/slog"

	vaultapi "github.com/hashicorp/vault/api"
)

// VaultStore implements SecretStore using HashiCorp Vault KV v2 secrets engine.
type VaultStore struct {
	client *vaultapi.Client
	config *Config
}

// NewVaultStore creates a new VaultStore connected to a Vault server.
func NewVaultStore(cfg *Config) (*VaultStore, error) {
	vaultCfg := vaultapi.DefaultConfig()
	vaultCfg.Address = cfg.Address

	client, err := vaultapi.NewClient(vaultCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Set token — supports env var VAULT_TOKEN as fallback (Vault SDK default)
	if cfg.Token != "" {
		client.SetToken(cfg.Token)
	}

	// Verify connectivity
	_, err = client.Sys().Health()
	if err != nil {
		return nil, fmt.Errorf("vault health check failed: %w", err)
	}

	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}
	if cfg.PathPrefix == "" {
		cfg.PathPrefix = "osbaka/nodes"
	}

	return &VaultStore{
		client: client,
		config: cfg,
	}, nil
}

func (v *VaultStore) Type() string {
	return "vault"
}

// StorePassphrase writes a node's LUKS passphrase to Vault KV v2.
func (v *VaultStore) StorePassphrase(ctx context.Context, nodeID uint, passphrase string) error {
	path := v.config.NodePassphrasePath(nodeID)
	mount := v.config.MountPath

	data := map[string]interface{}{
		"passphrase": passphrase,
	}

	_, err := v.client.KVv2(mount).Put(ctx, path, data)
	if err != nil {
		slog.Error("Failed to store passphrase in Vault",
			"nodeID", nodeID,
			"path", path,
			"error", err)
		return fmt.Errorf("vault store passphrase: %w", err)
	}

	slog.Info("Passphrase stored in Vault", "nodeID", nodeID, "path", path)
	return nil
}

// GetPassphrase reads a node's LUKS passphrase from Vault KV v2.
func (v *VaultStore) GetPassphrase(ctx context.Context, nodeID uint) (string, error) {
	path := v.config.NodePassphrasePath(nodeID)
	mount := v.config.MountPath

	secret, err := v.client.KVv2(mount).Get(ctx, path)
	if err != nil {
		return "", fmt.Errorf("vault get passphrase for node %d: %w", nodeID, err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no passphrase found in vault for node %d", nodeID)
	}

	passphrase, ok := secret.Data["passphrase"].(string)
	if !ok {
		return "", fmt.Errorf("invalid passphrase format in vault for node %d", nodeID)
	}

	return passphrase, nil
}

// DeletePassphrase removes a node's passphrase from Vault.
func (v *VaultStore) DeletePassphrase(ctx context.Context, nodeID uint) error {
	path := v.config.NodePassphrasePath(nodeID)
	mount := v.config.MountPath

	err := v.client.KVv2(mount).DeleteMetadata(ctx, path)
	if err != nil {
		slog.Error("Failed to delete passphrase from Vault",
			"nodeID", nodeID,
			"path", path,
			"error", err)
		return fmt.Errorf("vault delete passphrase: %w", err)
	}

	slog.Info("Passphrase deleted from Vault", "nodeID", nodeID)
	return nil
}
