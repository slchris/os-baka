package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/os-baka/backend/internal/model"
)

// UserHandler handles user management endpoints.
type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// ---- Request / Response types ----

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role"` // admin, operator, auditor
}

type UpdateUserRequest struct {
	Email    *string `json:"email" binding:"omitempty,email"`
	FullName *string `json:"full_name"`
	Role     *string `json:"role"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type UserView struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

func userToView(u model.User) UserView {
	role := "operator"
	if u.IsSuperuser {
		role = "admin"
	}
	return UserView{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		FullName: u.FullName,
		Role:     role,
	}
}

// ListUsers godoc
// @Summary      List all users
// @Description  Returns all registered users (admin only)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Only superusers can list all users
	callerID, _ := GetAuthUserID(c)
	var caller model.User
	if err := getDB().First(&caller, callerID).Error; err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "User not found")
		return
	}
	if !caller.IsSuperuser {
		ErrorResponse(c, http.StatusForbidden, "Admin access required")
		return
	}

	var users []model.User
	getDB().Order("id ASC").Find(&users)

	views := make([]UserView, len(users))
	for i, u := range users {
		views[i] = userToView(u)
	}

	c.JSON(http.StatusOK, gin.H{"items": views, "total": len(views)})
}

// CreateUser godoc
// @Summary      Create a new user
// @Description  Create a new user with the given credentials (admin only)
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user body CreateUserRequest true "User details"
// @Success      201 {object} UserView
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	callerID, _ := GetAuthUserID(c)
	var caller model.User
	if err := getDB().First(&caller, callerID).Error; err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "User not found")
		return
	}
	if !caller.IsSuperuser {
		ErrorResponse(c, http.StatusForbidden, "Admin access required")
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Check for duplicate username or email
	var existing model.User
	if getDB().Where("username = ?", req.Username).First(&existing).Error == nil {
		ErrorResponse(c, http.StatusConflict, "Username already exists")
		return
	}
	if getDB().Where("email = ?", req.Email).First(&existing).Error == nil {
		ErrorResponse(c, http.StatusConflict, "Email already exists")
		return
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := model.User{
		Username:    req.Username,
		Email:       req.Email,
		FullName:    req.FullName,
		Password:    hashedPassword,
		IsSuperuser: req.Role == "admin",
	}

	if err := getDB().Create(&user).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	c.JSON(http.StatusCreated, userToView(user))
}

// UpdateUser godoc
// @Summary      Update a user
// @Description  Update user details (admin only, or self for non-privileged fields)
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "User ID"
// @Param        user body UpdateUserRequest true "Fields to update"
// @Success      200 {object} UserView
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	callerID, _ := GetAuthUserID(c)
	var caller model.User
	if err := getDB().First(&caller, callerID).Error; err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "Caller not found")
		return
	}

	// Non-admin can only update themselves (and cannot change role)
	isSelf := callerID == uint(id)
	if !caller.IsSuperuser && !isSelf {
		ErrorResponse(c, http.StatusForbidden, "Admin access required")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Non-admin cannot change role
	if !caller.IsSuperuser && req.Role != nil {
		ErrorResponse(c, http.StatusForbidden, "Only admins can change roles")
		return
	}

	var user model.User
	if err := getDB().First(&user, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Role != nil {
		user.IsSuperuser = *req.Role == "admin"
	}

	if err := getDB().Save(&user).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	c.JSON(http.StatusOK, userToView(user))
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Change password for a user. Admins can change any user's password without providing old password.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "User ID"
// @Param        passwords body ChangePasswordRequest true "Password change request"
// @Success      200 {object} map[string]string
// @Router       /users/{id}/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	callerID, _ := GetAuthUserID(c)
	var caller model.User
	if err := getDB().First(&caller, callerID).Error; err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "Caller not found")
		return
	}

	isSelf := callerID == uint(id)
	if !caller.IsSuperuser && !isSelf {
		ErrorResponse(c, http.StatusForbidden, "Admin access required")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	var user model.User
	if err := getDB().First(&user, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	// Non-admin (self) must provide old password
	if !caller.IsSuperuser {
		if req.OldPassword == "" {
			ErrorResponse(c, http.StatusBadRequest, "Old password is required")
			return
		}
		if !CheckPasswordHash(req.OldPassword, user.Password) {
			ErrorResponse(c, http.StatusUnauthorized, "Old password is incorrect")
			return
		}
	}

	hashedPassword, err := HashPassword(req.NewPassword)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user.Password = hashedPassword
	if err := getDB().Save(&user).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to update password")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// DeleteUser godoc
// @Summary      Delete a user
// @Description  Delete a user (admin only, cannot delete self)
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "User ID"
// @Success      200 {object} map[string]string
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, ok := ParseIDParam(c, "id")
	if !ok {
		return
	}

	callerID, _ := GetAuthUserID(c)
	var caller model.User
	if err := getDB().First(&caller, callerID).Error; err != nil {
		ErrorResponse(c, http.StatusUnauthorized, "Caller not found")
		return
	}
	if !caller.IsSuperuser {
		ErrorResponse(c, http.StatusForbidden, "Admin access required")
		return
	}
	if callerID == uint(id) {
		ErrorResponse(c, http.StatusBadRequest, "Cannot delete yourself")
		return
	}

	var user model.User
	if err := getDB().First(&user, id).Error; err != nil {
		ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	if err := getDB().Delete(&user).Error; err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
