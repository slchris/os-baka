package vault

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// DBStore implements SecretStore using the application database.
// This is the legacy fallback when Vault is not available.
// Passphrases are stored as plaintext in the Node model's EncryptionPassphrase field.
//
// WARNING: This is NOT recommended for production use.
// Passphrases are stored unencrypted in the database.
type DBStore struct {
	mu         sync.RWMutex
	warnedOnce bool
}

func (d *DBStore) Type() string {
	return "database"
}

func (d *DBStore) warn() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.warnedOnce {
		slog.Warn("Using database for passphrase storage — passphrases are NOT encrypted at rest. " +
			"Configure Vault for production environments.")
		d.warnedOnce = true
	}
}

// StorePassphrase is a no-op for DBStore because the passphrase is already
// stored in the Node model by the handler. This method exists to satisfy
// the interface.
func (d *DBStore) StorePassphrase(_ context.Context, nodeID uint, _ string) error {
	d.warn()
	slog.Debug("Passphrase stored in database (plaintext)", "nodeID", nodeID)
	return nil
}

// GetPassphrase returns an error indicating the caller should read from the DB directly.
// This signals to the handler to fall back to the Node model field.
func (d *DBStore) GetPassphrase(_ context.Context, nodeID uint) (string, error) {
	d.warn()
	// Return a sentinel error so the caller knows to read from DB
	return "", fmt.Errorf("%w: node %d", ErrUseDBFallback, nodeID)
}

// DeletePassphrase is a no-op for DBStore — the passphrase is deleted
// when the node record is deleted (via CASCADE or soft delete).
func (d *DBStore) DeletePassphrase(_ context.Context, nodeID uint) error {
	slog.Debug("Passphrase removed from database via node deletion", "nodeID", nodeID)
	return nil
}

// ErrUseDBFallback is returned by DBStore.GetPassphrase to signal
// that the caller should read the passphrase from the database directly.
var ErrUseDBFallback = fmt.Errorf("vault not configured, use database fallback")
