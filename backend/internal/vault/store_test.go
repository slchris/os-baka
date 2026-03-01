package vault

import (
	"context"
	"errors"
	"testing"
)

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

func TestDBStoreDeletePassphrase(t *testing.T) {
	store := &DBStore{}
	err := store.DeletePassphrase(context.Background(), 1)
	if err != nil {
		t.Errorf("DBStore.DeletePassphrase() error = %v, want nil", err)
	}
}

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
