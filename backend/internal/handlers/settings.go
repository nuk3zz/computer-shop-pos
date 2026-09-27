package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
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
		SELECT id, company_name, logo_url, setup_completed, created_at, updated_at
		FROM shop_profile WHERE id = 1
	`).Scan(&profile.ID, &profile.CompanyName, &profile.LogoURL, &profile.SetupCompleted, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load shop profile", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Shop profile retrieved", Data: profile})
}

func (h *ShopProfileHandler) Update(c *gin.Context) {
	var req struct {
		CompanyName string  `json:"company_name" binding:"required"`
		LogoURL     *string `json:"logo_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Company name is required", Error: stringPtr(err.Error())})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.CompanyName == "" || len(req.CompanyName) > 150 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Company name must be between 1 and 150 characters"})
		return
	}
	if req.LogoURL != nil && strings.TrimSpace(*req.LogoURL) == "" {
		req.LogoURL = nil
	}

	var profile models.ShopProfile
	err := h.db.QueryRow(`
		UPDATE shop_profile
		SET company_name = $1, logo_url = $2, setup_completed = true, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
		RETURNING id, company_name, logo_url, setup_completed, created_at, updated_at
	`, req.CompanyName, req.LogoURL).Scan(&profile.ID, &profile.CompanyName, &profile.LogoURL, &profile.SetupCompleted, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save shop profile", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Shop setup completed", Data: profile})
}
