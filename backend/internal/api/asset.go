package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

type AssetHandler struct {
	BaseDir string
}

func NewAssetHandler() *AssetHandler {
	// Default to local directory if env not set
	baseDir := os.Getenv("PXE_ASSETS_DIR")
	if baseDir == "" {
		baseDir = "/tftpboot"
	}
	return &AssetHandler{BaseDir: baseDir}
}

// ListAssets godoc
// @Summary      List all boot assets
// @Tags         assets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /assets/boot [get]
func (h *AssetHandler) ListAssets(c *gin.Context) {
	var assets []model.BootAsset
	model.DB.Find(&assets)
	c.JSON(http.StatusOK, gin.H{"items": assets, "total": len(assets)})
}

// UploadAsset godoc
// @Summary      Upload a new boot asset
// @Tags         assets
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "File to upload"
// @Param        type formData string true "Asset type (kernel, initrd, etc)"
// @Param        name formData string false "Display name"
// @Success      201  {object} model.BootAsset
// @Router       /assets/boot [post]
func (h *AssetHandler) UploadAsset(c *gin.Context) {
	// 1. Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	assetType := c.PostForm("type")
	if assetType == "" {
		assetType = "other"
	}

	displayName := c.PostForm("name")
	if displayName == "" {
		displayName = header.Filename
	}

	// 2. Prepare destination
	// We flatly store files in the base dir or subdirs by type?
	// Let's use clean filenames
	cleanName := filepath.Base(header.Filename)
	destPath := filepath.Join(h.BaseDir, cleanName)

	// Ensure directory exists
	if err := os.MkdirAll(h.BaseDir, 0755); err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create storage directory")
		return
	}

	// Check if file exists, maybe suffix if duplicate?
	// For now, overwrite or error? Let's error to be safe
	if _, err := os.Stat(destPath); err == nil {
		ErrorResponse(c, http.StatusConflict, "File with this name already exists")
		return
	}

	// 3. Save file and calculate hash
	out, err := os.Create(destPath)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create file: "+err.Error())
		return
	}
	defer out.Close()

	hash := sha256.New()
	writer := io.MultiWriter(out, hash)

	size, err := io.Copy(writer, file)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to display file: "+err.Error())
		return
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	// 4. Create DB record
	asset := model.BootAsset{
		Name:     displayName,
		FileName: cleanName,
		Type:     assetType,
		Size:     size,
		Path:     cleanName, // Relative path
		CheckSum: checksum,
	}

	if result := model.DB.Create(&asset); result.Error != nil {
		// Cleanup file if DB insert fails
		os.Remove(destPath)
		ErrorResponse(c, http.StatusInternalServerError, result.Error.Error())
		return
	}

	c.JSON(http.StatusCreated, asset)
}

// DeleteAsset godoc
// @Summary      Delete a boot asset
// @Tags         assets
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Asset ID"
// @Success      200  {object} map[string]interface{}
// @Router       /assets/boot/{id} [delete]
func (h *AssetHandler) DeleteAsset(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	var asset model.BootAsset
	if result := model.DB.First(&asset, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Asset not found")
		return
	}

	// Delete file
	fullPath := filepath.Join(h.BaseDir, asset.Path)
	if err := os.Remove(fullPath); err != nil {
		// Log error but continue to delete from DB?
		fmt.Printf("Warning: Failed to delete file %s: %v\n", fullPath, err)
	}

	model.DB.Delete(&model.BootAsset{}, id)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
