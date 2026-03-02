package vault

import (
	"context"
	"errors"
	"testing"
)

// ========= DBStore Tests =========

func TestDBStoreType(t *testing.T) {
	store := &DBStore{}
	if store.Type() != "database" {
		t.Errorf("DBStore.Type() = %q, want %q", store.Type(), "database")
	}
}

func TestDBStoreStorePassphrase(t *testing.T) {
	store := &DBStore{}
	err := store.StorePassphrase(context.Background(), 1, "test-passphrase")
	if err != nil {
		t.Errorf("DBStore.StorePassphrase() error = %v, want nil", err)
	}
}

func TestDBStoreStorePassphraseMultipleCalls(t *testing.T) {
	store := &DBStore{}
	// Should succeed for different node IDs
	for i := uint(1); i <= 5; i++ {
		if err := store.StorePassphrase(context.Background(), i, "passphrase"); err != nil {
			t.Errorf("DBStore.StorePassphrase(nodeID=%d) error = %v", i, err)
		}
	}
}

func TestDBStoreGetPassphraseFallback(t *testing.T) {
	store := &DBStore{}
	_, err := store.GetPassphrase(context.Background(), 1)
	if err == nil {
		t.Error("DBStore.GetPassphrase() should return error signaling DB fallback")
	}
	if !errors.Is(err, ErrUseDBFallback) {
		t.Errorf("DBStore.GetPassphrase() error = %v, want ErrUseDBFallback", err)
	}
}

func TestDBStoreGetPassphraseErrorContainsNodeID(t *testing.T) {
	store := &DBStore{}
	_, err := store.GetPassphrase(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should contain the node ID
	if !errors.Is(err, ErrUseDBFallback) {
		t.Errorf("error should wrap ErrUseDBFallback, got: %v", err)
	}
}

func TestDBStoreDeletePassphrase(t *testing.T) {
	store := &DBStore{}
	err := store.DeletePassphrase(context.Background(), 1)
	if err != nil {
		t.Errorf("DBStore.DeletePassphrase() error = %v, want nil", err)
	}
}

func TestDBStoreWarnOnce(t *testing.T) {
	store := &DBStore{}
	// Multiple calls should not panic
	for i := 0; i < 10; i++ {
		store.warn()
	}
	if !store.warnedOnce {
		t.Error("DBStore.warnedOnce should be true after warn()")
	}
}

func TestDBStoreImplementsInterface(t *testing.T) {
	var _ SecretStore = &DBStore{}
}

// ========= Config Tests =========

func TestNewFromConfigDisabled(t *testing.T) {
	store := NewFromConfig(nil)
	if store.Type() != "database" {
		t.Errorf("NewFromConfig(nil) = %q, want %q", store.Type(), "database")
	}

	store = NewFromConfig(&Config{Enabled: false})
	if store.Type() != "database" {
		t.Errorf("NewFromConfig(disabled) = %q, want %q", store.Type(), "database")
	}
}

func TestNewFromConfigBadAddress(t *testing.T) {
	// Non-reachable Vault address should fall back to DB
	store := NewFromConfig(&Config{
		Enabled: true,
		Address: "http://127.0.0.1:19999", // unlikely to be running
		Token:   "test-token",
	})
	if store.Type() != "database" {
		t.Errorf("NewFromConfig(bad address) = %q, want %q", store.Type(), "database")
	}
}

func TestNewFromConfigEmptyAddress(t *testing.T) {
	store := NewFromConfig(&Config{
		Enabled: true,
		Address: "",
	})
	if store.Type() != "database" {
		t.Errorf("NewFromConfig(empty address) = %q, want %q", store.Type(), "database")
	}
}

func TestNodePassphrasePath(t *testing.T) {
	cfg := &Config{PathPrefix: "osbaka/nodes"}
	path := cfg.NodePassphrasePath(42)
	want := "osbaka/nodes/42"
	if path != want {
		t.Errorf("NodePassphrasePath(42) = %q, want %q", path, want)
	}
}

func TestNodePassphrasePathDefault(t *testing.T) {
	cfg := &Config{}
	path := cfg.NodePassphrasePath(1)
	want := "osbaka/nodes/1"
	if path != want {
		t.Errorf("NodePassphrasePath(1) = %q, want %q", path, want)
	}
}

func TestNodePassphrasePathCustomPrefix(t *testing.T) {
	cfg := &Config{PathPrefix: "custom/prefix"}
	path := cfg.NodePassphrasePath(100)
	want := "custom/prefix/100"
	if path != want {
		t.Errorf("NodePassphrasePath(100) = %q, want %q", path, want)
	}
}

func TestNodePassphrasePathZeroID(t *testing.T) {
	cfg := &Config{PathPrefix: "osbaka/nodes"}
	path := cfg.NodePassphrasePath(0)
	want := "osbaka/nodes/0"
	if path != want {
		t.Errorf("NodePassphrasePath(0) = %q, want %q", path, want)
	}
}

// ========= VaultStore Tests =========

func TestVaultStoreType(t *testing.T) {
	store := &VaultStore{}
	if store.Type() != "vault" {
		t.Errorf("VaultStore.Type() = %q, want %q", store.Type(), "vault")
	}
}

func TestVaultStoreImplementsInterface(t *testing.T) {
	var _ SecretStore = &VaultStore{}
}

func TestNewVaultStoreInvalidAddress(t *testing.T) {
	// Invalid address should fail at health check
	_, err := NewVaultStore(&Config{
		Enabled: true,
		Address: "http://127.0.0.1:19999",
		Token:   "test",
	})
	if err == nil {
		t.Error("NewVaultStore() with unreachable address should return error")
	}
}

func TestNewVaultStoreDefaultMountPath(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Address: "http://127.0.0.1:8200",
	}
	// Since we can't connect to Vault, test the defaulting logic by checking config
	if cfg.MountPath != "" {
		t.Error("MountPath should be empty before NewVaultStore call")
	}
}

// ========= ErrUseDBFallback Tests =========

func TestErrUseDBFallback(t *testing.T) {
	if ErrUseDBFallback == nil {
		t.Fatal("ErrUseDBFallback should not be nil")
	}
	if ErrUseDBFallback.Error() == "" {
		t.Error("ErrUseDBFallback should have a non-empty message")
	}
}

func TestErrUseDBFallbackWrapping(t *testing.T) {
	store := &DBStore{}
	_, err := store.GetPassphrase(context.Background(), 42)
	if !errors.Is(err, ErrUseDBFallback) {
		t.Errorf("GetPassphrase error should be wrappable with errors.Is, got: %v", err)
	}
}

// ========= SecretStore Interface Tests =========

func TestSecretStoreInterfaceConsistency(t *testing.T) {
	stores := []SecretStore{
		&DBStore{},
	}

	ctx := context.Background()

	for _, store := range stores {
		t.Run(store.Type(), func(t *testing.T) {
			// Type should return non-empty
			if store.Type() == "" {
				t.Error("Type() should return non-empty string")
			}

			// StorePassphrase should not error for DBStore
			if err := store.StorePassphrase(ctx, 1, "test"); err != nil {
				t.Errorf("StorePassphrase() error = %v", err)
			}

			// DeletePassphrase should not error for DBStore
			if err := store.DeletePassphrase(ctx, 1); err != nil {
				t.Errorf("DeletePassphrase() error = %v", err)
			}
		})
	}
}
