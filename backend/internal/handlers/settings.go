package handlers

import (
	"database/sql"
	"net"
	"net/http"
	"regexp"
	"strings"

	"pos-backend/internal/database"
	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type ShopProfileHandler struct {
	db *sql.DB
}

func NewShopProfileHandler(db *sql.DB) *ShopProfileHandler {
	return &ShopProfileHandler{db: db}
}

func (h *ShopProfileHandler) Get(c *gin.Context) {
	var profile models.ShopProfile
	err := h.db.QueryRow(`
		SELECT id, company_name, description, logo_url, setup_completed, network_mode, auto_backup, backup_time, created_at, updated_at
		FROM shop_profile WHERE id = 1
	`).Scan(&profile.ID, &profile.CompanyName, &profile.Description, &profile.LogoURL, &profile.SetupCompleted, &profile.NetworkMode, &profile.AutoBackup, &profile.BackupTime, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load shop profile", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Shop profile retrieved", Data: profile})
}

func (h *ShopProfileHandler) Update(c *gin.Context) {
	var req struct {
		CompanyName string  `json:"company_name" binding:"required"`
		Description string  `json:"description"`
		LogoURL     *string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Company name is required", Error: stringPtr(err.Error())})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.Description = strings.TrimSpace(req.Description)
	if req.CompanyName == "" || len(req.CompanyName) > 150 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Company name must be between 1 and 150 characters"})
		return
	}
	if len(req.Description) > 200 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Shop description cannot exceed 200 characters"})
		return
	}
	if req.LogoURL != nil && strings.TrimSpace(*req.LogoURL) == "" {
		req.LogoURL = nil
	}

	var profile models.ShopProfile
	err := h.db.QueryRow(`
		UPDATE shop_profile
		SET company_name = $1, description = $2, logo_url = $3, setup_completed = true, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
		RETURNING id, company_name, description, logo_url, setup_completed, network_mode, auto_backup, backup_time, created_at, updated_at
	`, req.CompanyName, req.Description, req.LogoURL).Scan(&profile.ID, &profile.CompanyName, &profile.Description, &profile.LogoURL, &profile.SetupCompleted, &profile.NetworkMode, &profile.AutoBackup, &profile.BackupTime, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save shop profile", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Shop setup completed", Data: profile})
}

var setupUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,50}$`)

type InitialSetupHandler struct {
	db              *sql.DB
	uploadHandler   *ImageUploadHandler
	allowPrivateLAN bool
}

func NewInitialSetupHandler(db *sql.DB, uploadHandler *ImageUploadHandler) *InitialSetupHandler {
	return &InitialSetupHandler{db: db, uploadHandler: uploadHandler, allowPrivateLAN: database.IsSQLite(db)}
}

func (h *InitialSetupHandler) Status(c *gin.Context) {
	var profile models.ShopProfile
	err := h.db.QueryRow(`
		SELECT id, company_name, description, logo_url, setup_completed, network_mode, auto_backup, backup_time, created_at, updated_at
		FROM shop_profile WHERE id = 1
	`).Scan(&profile.ID, &profile.CompanyName, &profile.Description, &profile.LogoURL, &profile.SetupCompleted, &profile.NetworkMode, &profile.AutoBackup, &profile.BackupTime, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load setup status", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Setup status retrieved", Data: profile})
}

func (h *InitialSetupHandler) UploadLogo(c *gin.Context) {
	if !h.isAllowedSetupRequest(c) || h.setupAlreadyCompleted() {
		c.JSON(http.StatusForbidden, models.APIResponse{Success: false, Message: "Initial setup is available only from this computer or its trusted private network before setup is completed"})
		return
	}
	h.uploadHandler.UploadProductImage(c)
}

func (h *InitialSetupHandler) Complete(c *gin.Context) {
	h.complete(c, true)
}

func (h *InitialSetupHandler) CompleteAuthenticated(c *gin.Context) {
	h.complete(c, false)
}

func (h *InitialSetupHandler) complete(c *gin.Context, requireLocal bool) {
	if requireLocal && !h.isAllowedSetupRequest(c) {
		c.JSON(http.StatusForbidden, models.APIResponse{Success: false, Message: "Complete first-time setup from the server computer or its trusted private network"})
		return
	}
	var req struct {
		CompanyName string  `json:"company_name" binding:"required"`
		LogoURL     *string `json:"logo_url"`
		FirstName   string  `json:"first_name" binding:"required"`
		LastName    string  `json:"last_name"`
		Username    string  `json:"username" binding:"required"`
		Email       string  `json:"email"`
		Password    string  `json:"password" binding:"required"`
		NetworkMode string  `json:"network_mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Complete all required setup fields", Error: stringPtr(err.Error())})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.CompanyName == "" || len(req.CompanyName) > 150 || req.FirstName == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Company name and owner name are required"})
		return
	}
	if !setupUsernamePattern.MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Username must be 3-50 characters using letters, numbers, dot, dash, or underscore"})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Password must contain at least 8 characters"})
		return
	}
	if req.NetworkMode != "local" && req.NetworkMode != "lan" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Network mode must be local or lan"})
		return
	}
	if req.Email == "" {
		req.Email = req.Username + "@local.invalid"
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to secure owner password"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to start setup"})
		return
	}
	defer tx.Rollback()
	var completed bool
	if err := tx.QueryRow(`SELECT setup_completed FROM shop_profile WHERE id = 1`).Scan(&completed); err != nil || completed {
		c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "Initial setup has already been completed"})
		return
	}
	result, err := tx.Exec(`
		UPDATE users SET username = $1, email = $2, password_hash = $3, first_name = $4, last_name = $5,
			role = 'admin', is_active = true, updated_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT id FROM users WHERE role = 'admin' ORDER BY created_at LIMIT 1)
	`, req.Username, req.Email, string(passwordHash), req.FirstName, req.LastName)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Username or email is already in use", Error: stringPtr(err.Error())})
		return
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Bootstrap owner account is missing"})
		return
	}
	if _, err := tx.Exec(`
		UPDATE shop_profile SET company_name = $1, logo_url = $2, network_mode = $3,
			setup_completed = true, updated_at = CURRENT_TIMESTAMP WHERE id = 1
	`, req.CompanyName, req.LogoURL, req.NetworkMode); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save shop setup", Error: stringPtr(err.Error())})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to finish setup", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Setup complete. Sign in with your new owner account."})
}

func (h *InitialSetupHandler) setupAlreadyCompleted() bool {
	var completed bool
	return h.db.QueryRow(`SELECT setup_completed FROM shop_profile WHERE id = 1`).Scan(&completed) != nil || completed
}

func (h *InitialSetupHandler) isAllowedSetupRequest(c *gin.Context) bool {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && (ip.IsLoopback() || (h.allowPrivateLAN && ip.IsPrivate()))
}
