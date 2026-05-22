package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// safeFilenamePattern restricts uploaded asset filenames to a conservative
// character set so they're safe to:
//   - serve over TFTP (some clients trip on unicode / control chars)
//   - reference from preseed and iPXE scripts (no shell metas)
//   - look up in the local FS without surprises
//
// Length is capped at 200 chars — well below the 255-byte limit on common
// filesystems but plenty for human-readable names like
// "ubuntu-22.04-server-amd64-installer.iso".
var safeFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,200}$`)

// isSafeAssetFilename rejects names that would let an attacker escape
// the BaseDir (path traversal), overwrite hidden files (.htaccess-style),
// or trip TFTP / shell parsing downstream.
//
// Specifically rejects:
//   - empty, "." (CWD), ".." (parent)
//   - anything starting with a dot (avoid hidden / .htaccess type names)
//   - anything containing path separators
//   - characters outside [A-Za-z0-9._-]
func isSafeAssetFilename(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	return safeFilenamePattern.MatchString(name)
}

// resolveSafeAssetPath joins name into baseDir AND verifies the resulting
// absolute path stays under baseDir. Belt-and-suspenders against any
// future caller forgetting the filename validation above.
func resolveSafeAssetPath(baseDir, name string) (string, error) {
	if !isSafeAssetFilename(name) {
		return "", fmt.Errorf("invalid asset filename %q", name)
	}
	dest := filepath.Join(baseDir, name)
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve baseDir: %w", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("resolve dest: %w", err)
	}
	if absDest != absBase && !strings.HasPrefix(absDest, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes baseDir", name)
	}
	return dest, nil
}

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
	getDB().Find(&assets)
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
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Printf("Warning: Failed to close uploaded file: %v\n", cerr)
		}
	}()

	assetType := c.PostForm("type")
	if assetType == "" {
		assetType = "other"
	}

	displayName := c.PostForm("name")
	if displayName == "" {
		displayName = header.Filename
	}

	// 2. Prepare destination.
	//
	// filepath.Base alone is NOT sufficient against path traversal:
	// header.Filename = ".." survives Base unchanged, then Join into
	// BaseDir produces the parent dir. Validate the cleaned name and
	// verify the resolved absolute path stays under BaseDir.
	cleanName := filepath.Base(header.Filename)
	destPath, err := resolveSafeAssetPath(h.BaseDir, cleanName)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid filename: "+err.Error())
		return
	}

	// Ensure directory exists. 0755 is required for cross-container
	// reads by pxe-services (which serves these files over TFTP).
	if err := os.MkdirAll(h.BaseDir, 0755); err != nil { // #nosec G301 — cross-container read required
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create storage directory")
		return
	}

	// Check if file exists, maybe suffix if duplicate?
	// For now, overwrite or error? Let's error to be safe
	if _, err := os.Stat(destPath); err == nil {
		ErrorResponse(c, http.StatusConflict, "File with this name already exists")
		return
	}

	// 3. Save file and calculate hash. destPath is validated by
	// resolveSafeAssetPath above; #nosec G304 silences the false
	// positive about variable file paths.
	out, err := os.Create(destPath) // #nosec G304 — destPath validated
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create file: "+err.Error())
		return
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			fmt.Printf("Warning: Failed to close output file: %v\n", cerr)
		}
	}()

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

	if result := getDB().Create(&asset); result.Error != nil {
		// Cleanup file if DB insert fails
		if removeErr := os.Remove(destPath); removeErr != nil {
			fmt.Printf("Warning: Failed to cleanup file %s: %v\n", destPath, removeErr)
		}
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
	if result := getDB().First(&asset, id); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "Asset not found")
		return
	}

	// Delete file. asset.Path is constrained by isSafeAssetFilename at
	// upload time, but validate again defensively — if a legacy row
	// from before the validation lands here, we'd rather refuse than
	// rm something outside BaseDir.
	fullPath, err := resolveSafeAssetPath(h.BaseDir, asset.Path)
	if err != nil {
		fmt.Printf("Warning: refusing to delete unsafe asset path %q: %v\n", asset.Path, err)
	} else if err := os.Remove(fullPath); err != nil {
		fmt.Printf("Warning: Failed to delete file %s: %v\n", fullPath, err)
	}

	getDB().Delete(&model.BootAsset{}, id)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
