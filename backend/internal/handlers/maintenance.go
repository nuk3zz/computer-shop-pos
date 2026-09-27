package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"pos-backend/internal/models"
	possystem "pos-backend/internal/system"

	"github.com/gin-gonic/gin"
)

type MaintenanceHandler struct {
	db      *sql.DB
	manager *possystem.Manager
	version string
}

func NewMaintenanceHandler(db *sql.DB, dataDir, version string) *MaintenanceHandler {
	return &MaintenanceHandler{db: db, manager: possystem.NewManager(db, dataDir), version: version}
}

func (h *MaintenanceHandler) Info(c *gin.Context) {
	addresses := []string{"http://localhost:3000"}
	port := "3000"
	if _, requestPort, err := net.SplitHostPort(c.Request.Host); err == nil && requestPort != "" {
		port = requestPort
	}
	for _, address := range localIPv4Addresses() {
		addresses = append(addresses, fmt.Sprintf("http://%s:%s", address, port))
	}
	var networkMode string
	var autoBackup bool
	var backupTime string
	_ = h.db.QueryRow(`SELECT network_mode, auto_backup, backup_time FROM shop_profile WHERE id = 1`).Scan(&networkMode, &autoBackup, &backupTime)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "System information retrieved", Data: gin.H{
		"version": h.version, "os": runtime.GOOS, "arch": runtime.GOARCH,
		"native": h.manager.Supported(), "data_dir": h.manager.DataDir(), "backup_dir": h.manager.BackupDir(),
		"network_mode": networkMode, "addresses": addresses, "auto_backup": autoBackup, "backup_time": backupTime,
	}})
}

func (h *MaintenanceHandler) ListBackups(c *gin.Context) {
	backups, err := h.manager.ListBackups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to list backups", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Backups retrieved", Data: backups})
}

func (h *MaintenanceHandler) CreateBackup(c *gin.Context) {
	backup, err := h.manager.CreateBackup("manual", h.version)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Failed to create backup", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Backup created and verified", Data: backup})
}

func (h *MaintenanceHandler) DownloadBackup(c *gin.Context) {
	path, err := h.manager.BackupPath(c.Param("name"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Backup not found"})
		return
	}
	c.FileAttachment(path, filepath.Base(path))
}

func (h *MaintenanceHandler) UploadBackup(c *gin.Context) {
	file, err := c.FormFile("backup")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose a .cspbackup file"})
		return
	}
	input, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Could not read uploaded backup", Error: stringPtr(err.Error())})
		return
	}
	defer input.Close()
	backup, err := h.manager.SaveUploadedBackup(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Backup validation failed", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Backup uploaded and verified", Data: backup})
}

func (h *MaintenanceHandler) StageRestore(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Select a backup to restore"})
		return
	}
	if err := h.manager.StageRestore(req.Name, h.version); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Restore could not be staged", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Restore verified and staged. Restart the Computer Shop POS service to apply it.", Data: gin.H{"restart_required": true}})
}

func (h *MaintenanceHandler) UpdatePreferences(c *gin.Context) {
	var req struct {
		NetworkMode string `json:"network_mode" binding:"required"`
		AutoBackup  bool   `json:"auto_backup"`
		BackupTime  string `json:"backup_time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.NetworkMode != "local" && req.NetworkMode != "lan") || !validTime(req.BackupTime) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Use a valid network mode and 24-hour backup time"})
		return
	}
	_, err := h.db.Exec(`UPDATE shop_profile SET network_mode = $1, auto_backup = $2, backup_time = $3, updated_at = CURRENT_TIMESTAMP WHERE id = 1`, req.NetworkMode, req.AutoBackup, req.BackupTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save system preferences", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "System preferences saved"})
}

func (h *MaintenanceHandler) CheckUpdates(c *gin.Context) {
	request, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, "https://api.github.com/repos/nuk3zz/computer-shop-pos/releases/latest", nil)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Computer-Shop-POS/"+h.version)
	response, err := (&http.Client{Timeout: 12 * time.Second}).Do(request)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIResponse{Success: false, Message: "Could not contact the update server", Error: stringPtr(err.Error())})
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "No published installer update is available yet", Data: gin.H{"current_version": h.version, "update_available": false}})
		return
	}
	if response.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, models.APIResponse{Success: false, Message: "Update server returned an unexpected response"})
		return
	}
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Name    string `json:"name"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		c.JSON(http.StatusBadGateway, models.APIResponse{Success: false, Message: "Could not read update information"})
		return
	}
	available := normalizedVersion(release.TagName) != normalizedVersion(h.version) && h.version != "dev"
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Update check complete", Data: gin.H{
		"current_version": h.version, "latest_version": release.TagName, "update_available": available,
		"release_name": release.Name, "release_url": release.HTMLURL,
	}})
}

func localIPv4Addresses() []string {
	var result []string
	interfaces, _ := net.Interfaces()
	for _, networkInterface := range interfaces {
		addresses, _ := networkInterface.Addrs()
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil && !ip.IsLoopback() {
				result = append(result, ip.String())
			}
		}
	}
	return result
}

func validTime(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}

func normalizedVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}
