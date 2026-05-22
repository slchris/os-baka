package api

import (
	"gorm.io/gorm"
)

// handlerDB holds the database connection injected into API handlers.
// Replaces direct usage of model.DB so handlers can be exercised against
// an injected DB in tests.
var handlerDB *gorm.DB

// InitHandlers sets the database connection used by all API handlers.
// Call once during application startup, after InitDB.
func InitHandlers(db *gorm.DB) {
	handlerDB = db
}

// getDB returns the handler-level DB instance.
func getDB() *gorm.DB {
	return handlerDB
}
