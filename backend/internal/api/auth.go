package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/config"
	"github.com/os-baka/backend/internal/model"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthHandler struct {
	Config *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{Config: cfg}
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user and return access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials body LoginRequest true "Login credentials"
// @Success      200  {object} map[string]string
// @Failure      400  {object} map[string]string
// @Failure      401  {object} map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request")
		return
	}

	var user model.User
	if result := getDB().Where("username = ?", req.Username).First(&user); result.Error != nil {
		ErrorResponse(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	if !CheckPasswordHash(req.Password, user.Password) {
		ErrorResponse(c, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := GenerateToken(user.ID, user.Username, h.Config)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, gin.H{"access_token": token, "token_type": "bearer"})
}

// Me godoc
// @Summary      Get current user
// @Description  Get details of the currently logged-in user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object} map[string]interface{}
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok {
		ErrorResponse(c, http.StatusUnauthorized, "Could not identify user from token")
		return
	}

	var user model.User
	if result := getDB().First(&user, userID); result.Error != nil {
		ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"full_name":    user.FullName,
		"is_superuser": user.IsSuperuser,
	})
}
