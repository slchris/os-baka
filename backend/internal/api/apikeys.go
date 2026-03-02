package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// APIKeyHandler handles API key CRUD operations.
type APIKeyHandler struct{}

func NewAPIKeyHandler() *APIKeyHandler {
	return &APIKeyHandler{}
}

const apiKeyPrefix = "osbaka_"

// generateAPIKey creates a cryptographically secure API key with prefix.
// Returns the full key (only shown once) and its SHA-256 hash for storage.
func generateAPIKey() (fullKey string, keyHash string, prefix string, err error) {
	// Generate 32 random bytes = 64 hex chars
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomHex := hex.EncodeToString(randomBytes)
	fullKey = apiKeyPrefix + randomHex

	// Hash the full key for storage
	hash := sha256.Sum256([]byte(fullKey))
	keyHash = hex.EncodeToString(hash[:])

	// Prefix for display identification (first 8 chars after "osbaka_")
	prefix = fullKey[:len(apiKeyPrefix)+8]

	return fullKey, keyHash, prefix, nil
}

// hashAPIKey computes the SHA-256 hash of an API key for lookup.
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// ListAPIKeys godoc
// @Summary      List API keys
// @Description  List all API keys (hashes are never exposed)
// @Tags         apikeys
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /api-keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	var keys []model.APIKey
	getDB().Order("created_at DESC").Find(&keys)

	type KeyView struct {
		ID         uint       `json:"id"`
		Name       string     `json:"name"`
		Prefix     string     `json:"prefix"`
		Role       string     `json:"role"`
		CreatedBy  uint       `json:"created_by"`
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at"`
		ExpiresAt  *time.Time `json:"expires_at"`
		IsActive   bool       `json:"is_active"`
	}

	views := make([]KeyView, len(keys))
	for i, k := range keys {
		views[i] = KeyView{
			ID:         k.ID,
			Name:       k.Name,
			Prefix:     k.Prefix,
			Role:       k.Role,
			CreatedBy:  k.CreatedBy,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
			ExpiresAt:  k.ExpiresAt,
			IsActive:   k.IsActive,
		}
	}

	c.JSON(http.StatusOK, gin.H{"items": views, "total": len(views)})
}

// CreateAPIKey godoc
// @Summary      Create a new API key
// @Description  Create a new API key. The full key is only returned once.
// @Tags         apikeys
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body object true "API key details"
// @Success      201 {object} map[string]interface{}
// @Router       /api-keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Role      string `json:"role" binding:"required"`
		ExpiresIn *int   `json:"expires_in_days"` // Optional: days until expiration
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate role
	if req.Role != "admin" && req.Role != "operator" {
		ErrorResponse(c, http.StatusBadRequest, "Role must be 'admin' or 'operator'")
		return
	}

	fullKey, keyHash, prefix, err := generateAPIKey()
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	userID, _ := GetAuthUserID(c)

	apiKey := model.APIKey{
		Name:      req.Name,
		Prefix:    prefix,
		KeyHash:   keyHash,
		Role:      req.Role,
		CreatedBy: userID,
		IsActive:  true,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		apiKey.ExpiresAt = &exp
	}

	if err := getDB().Create(&apiKey).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create API key")
		return
	}

	WriteAuditLog(c, "apikey.create", "apikey", fmt.Sprintf("%d", apiKey.ID),
		fmt.Sprintf("Created API key '%s' with role '%s'", req.Name, req.Role))

	// Return the full key ONLY on creation
	c.JSON(http.StatusCreated, gin.H{
		"id":      apiKey.ID,
		"name":    apiKey.Name,
		"key":     fullKey, // Only returned once!
		"prefix":  apiKey.Prefix,
		"role":    apiKey.Role,
		"message": "Store this key securely. It will not be shown again.",
	})
}

// RevokeAPIKey godoc
// @Summary      Revoke an API key
// @Description  Deactivate an API key
// @Tags         apikeys
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "API Key ID"
// @Success      200 {object} map[string]string
// @Router       /api-keys/{id} [delete]
func (h *APIKeyHandler) RevokeAPIKey(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	result := getDB().Model(&model.APIKey{}).Where("id = ?", id).Update("is_active", false)
	if result.RowsAffected == 0 {
		ErrorResponse(c, http.StatusNotFound, "API key not found")
		return
	}

	WriteAuditLog(c, "apikey.revoke", "apikey", fmt.Sprintf("%d", id), "API key revoked")

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}

// RotateAPIKey godoc
// @Summary      Rotate an API key
// @Description  Generate a new key value for an existing API key. Old key is invalidated.
// @Tags         apikeys
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "API Key ID"
// @Success      200 {object} map[string]interface{}
// @Router       /api-keys/{id}/rotate [post]
func (h *APIKeyHandler) RotateAPIKey(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var apiKey model.APIKey
	if err := getDB().First(&apiKey, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "API key not found")
		return
	}

	if !apiKey.IsActive {
		ErrorResponse(c, http.StatusBadRequest, "Cannot rotate a revoked key")
		return
	}

	fullKey, keyHash, prefix, err := generateAPIKey()
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to generate new key")
		return
	}

	apiKey.KeyHash = keyHash
	apiKey.Prefix = prefix
	if err := getDB().Save(&apiKey).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to rotate key")
		return
	}

	WriteAuditLog(c, "apikey.rotate", "apikey", fmt.Sprintf("%d", id), "API key rotated")

	c.JSON(http.StatusOK, gin.H{
		"id":      apiKey.ID,
		"name":    apiKey.Name,
		"key":     fullKey,
		"prefix":  prefix,
		"message": "Key rotated. Store the new key securely.",
	})
}
