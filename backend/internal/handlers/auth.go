package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"pos-backend/internal/middleware"
	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Username and password are required",
			Error:   stringPtr("missing_credentials"),
		})
		return
	}

	// Get user from database
	var user models.User
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, profile_image_url, role, is_active, created_at, updated_at
		FROM users 
		WHERE username = $1 AND is_active = true
	`

	err := h.db.QueryRow(query, req.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.ProfileImageURL, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Invalid username or password",
			Error:   stringPtr("invalid_credentials"),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Database error",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Invalid username or password",
			Error:   stringPtr("invalid_credentials"),
		})
		return
	}

	// Generate JWT token
	tokenLifetime := 24 * time.Hour
	if req.RememberMe {
		tokenLifetime = 30 * 24 * time.Hour
	}
	token, err := middleware.GenerateToken(&user, tokenLifetime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to generate token",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Return successful login response
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Login successful",
		Data: models.LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Authentication required",
			Error:   stringPtr("auth_required"),
		})
		return
	}

	// Get user from database
	var user models.User
	query := `
		SELECT id, username, email, first_name, last_name, profile_image_url, role, is_active, created_at, updated_at
		FROM users 
		WHERE id = $1
	`

	err := h.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.FirstName, &user.LastName, &user.ProfileImageURL, &user.Role, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "User not found",
			Error:   stringPtr("user_not_found"),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Database error",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "User retrieved successfully",
		Data:    user,
	})
}

// UpdateProfile updates the signed-in user's own public account details.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Authentication required"})
		return
	}
	var req struct {
		FirstName       string  `json:"first_name" binding:"required"`
		LastName        string  `json:"last_name"`
		Username        string  `json:"username" binding:"required"`
		Email           string  `json:"email" binding:"required"`
		ProfileImageURL *string `json:"profile_image_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Name, username, and email are required", Error: stringPtr(err.Error())})
		return
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.FirstName == "" || len(req.FirstName) > 50 || len(req.LastName) > 50 || !setupUsernamePattern.MatchString(req.Username) || req.Email == "" || len(req.Email) > 100 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Enter a valid name, username, and email"})
		return
	}
	if req.ProfileImageURL != nil && strings.TrimSpace(*req.ProfileImageURL) == "" {
		req.ProfileImageURL = nil
	}
	var user models.User
	err := h.db.QueryRow(`
		UPDATE users SET first_name = $1, last_name = $2, username = $3, email = $4,
			profile_image_url = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
		RETURNING id, username, email, first_name, last_name, profile_image_url, role, is_active, created_at, updated_at
	`, req.FirstName, req.LastName, req.Username, req.Email, req.ProfileImageURL, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.FirstName, &user.LastName, &user.ProfileImageURL,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Username or email is already in use", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Profile updated", Data: user})
}

// Logout handles user logout (in a stateless JWT system, this is mainly client-side)
func (h *AuthHandler) Logout(c *gin.Context) {
	// In a stateless JWT system, logout is handled client-side by removing the token
	// For additional security, you could implement a token blacklist here
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
