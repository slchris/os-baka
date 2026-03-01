package api

import "gorm.io/gorm"

// DB holds the database connection injected into handlers.
// This replaces direct usage of the model.DB global variable in handlers,
// enabling dependency injection and testability.
var handlerDB *gorm.DB

// InitHandlers sets the database connection used by all API handlers.
// This should be called once during application startup after InitDB.
func InitHandlers(db *gorm.DB) {
	handlerDB = db
}

// getDB returns the handler-level DB instance.
// Handlers should use this instead of model.DB directly.
func getDB() *gorm.DB {
	return handlerDB
}
