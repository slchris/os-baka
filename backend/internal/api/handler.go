package api

import (
	"github.com/os-baka/backend/internal/vault"
	"gorm.io/gorm"
)

// DB holds the database connection injected into handlers.
// This replaces direct usage of the model.DB global variable in handlers,
// enabling dependency injection and testability.
var handlerDB *gorm.DB

// secretStore holds the secret storage backend (Vault or DB fallback).
var secretStore vault.SecretStore

// InitHandlers sets the database connection and secret store used by all API handlers.
// This should be called once during application startup after InitDB.
func InitHandlers(db *gorm.DB, store vault.SecretStore) {
	handlerDB = db
	secretStore = store
}

// getDB returns the handler-level DB instance.
// Handlers should use this instead of model.DB directly.
func getDB() *gorm.DB {
	return handlerDB
}

// getSecretStore returns the configured secret store.
func getSecretStore() vault.SecretStore {
	return secretStore
}
