package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"pos-backend/internal/middleware"
	"pos-backend/internal/models"
	possystem "pos-backend/internal/system"

	"github.com/gin-gonic/gin"
)

type MaintenanceHandler struct {
	db        *sql.DB
	manager   *possystem.Manager
	version   string
	uploadDir string
	updateMu  sync.Mutex
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Digest      string `json:"digest"`
		Size        int64  `json:"size"`
	} `json:"assets"`
}

type updateAsset struct {
	Name        string
	DownloadURL string
	Digest      string
	Size        int64
}

func NewMaintenanceHandler(db *sql.DB, dataDir, uploadDir, version string) *MaintenanceHandler {
	return &MaintenanceHandler{db: db, manager: possystem.NewManager(db, dataDir), version: version, uploadDir: uploadDir}
}

func (h *MaintenanceHandler) Info(c *gin.Context) {
	port := "3000"
	if _, requestPort, err := net.SplitHostPort(c.Request.Host); err == nil && requestPort != "" {
		port = requestPort
	}
	addresses := []string{fmt.Sprintf("http://localhost:%s", port)}
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
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose a .urposbackup file"})
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
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Restore verified and staged. Restart Universal Repair POS to apply it.", Data: gin.H{"restart_required": true}})
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
	release, err := h.latestRelease(c.Request.Context())
	if err != nil {
		if err == errNoPublishedRelease {
			c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "No published installer update is available yet", Data: gin.H{"current_version": h.version, "update_available": false}})
			return
		}
		c.JSON(http.StatusBadGateway, models.APIResponse{Success: false, Message: "Could not contact the update server", Error: stringPtr(err.Error())})
		return
	}
	available := h.version != "dev" && isVersionNewer(release.TagName, h.version)
	asset, assetOK := selectUpdateAsset(release, runtime.GOOS, runtime.GOARCH)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Update check complete", Data: gin.H{
		"current_version": h.version, "latest_version": release.TagName, "update_available": available,
		"release_name": release.Name, "release_url": release.HTMLURL, "install_supported": h.manager.Supported() && assetOK,
		"asset_name": asset.Name, "asset_size": asset.Size,
	}})
}

var errNoPublishedRelease = fmt.Errorf("no published release")

func (h *MaintenanceHandler) latestRelease(ctx context.Context) (githubRelease, error) {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/nuk3zz/universal-repair-pos/releases/latest", nil)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Universal-Repair-POS/"+h.version)
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	if err != nil {
		return githubRelease{}, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return githubRelease{}, errNoPublishedRelease
	}
	if response.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("update server returned HTTP %d", response.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("read update information: %w", err)
	}
	return release, nil
}

func (h *MaintenanceHandler) DownloadUpdate(c *gin.Context) {
	if !h.manager.Supported() {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Built-in installer downloads are available in the standalone edition"})
		return
	}
	if !h.updateMu.TryLock() {
		c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "An update download is already running"})
		return
	}
	defer h.updateMu.Unlock()

	release, err := h.latestRelease(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIResponse{Success: false, Message: "Could not retrieve the latest release", Error: stringPtr(err.Error())})
		return
	}
	if h.version == "dev" || !isVersionNewer(release.TagName, h.version) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "This installation is already up to date"})
		return
	}
	asset, ok := selectUpdateAsset(release, runtime.GOOS, runtime.GOARCH)
	if !ok {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "No compatible installer is attached to this release"})
		return
	}
	if _, err := h.manager.CreateBackup("manual", h.version); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "A safety backup could not be created, so the update was not downloaded", Error: stringPtr(err.Error())})
		return
	}
	installerPath, err := h.downloadAndVerify(c.Request.Context(), asset)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIResponse{Success: false, Message: "Update download or verification failed", Error: stringPtr(err.Error())})
		return
	}
	launched, installCommand, err := launchUpdateInstaller(installerPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "The update was downloaded but the installer could not be opened", Error: stringPtr(err.Error()), Data: gin.H{"installer_path": installerPath}})
		return
	}
	message := "Update verified and ready"
	if launched {
		message = "Update verified. Complete the operating-system installer that just opened."
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: message, Data: gin.H{
		"latest_version": release.TagName, "installer_path": installerPath, "installer_launched": launched, "install_command": installCommand,
	}})
}

func (h *MaintenanceHandler) downloadAndVerify(ctx context.Context, asset updateAsset) (string, error) {
	parsed, err := url.Parse(asset.DownloadURL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "github.com" {
		return "", fmt.Errorf("release asset URL is not trusted")
	}
	expectedDigest := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asset.Digest)), "sha256:")
	if len(expectedDigest) != 64 {
		return "", fmt.Errorf("release asset has no valid SHA-256 digest")
	}
	if _, err := hex.DecodeString(expectedDigest); err != nil {
		return "", fmt.Errorf("release asset digest is invalid")
	}
	if asset.Size <= 0 || asset.Size > 512<<20 {
		return "", fmt.Errorf("release asset size is invalid")
	}
	updatesDir := filepath.Join(h.manager.DataDir(), "updates")
	if err := os.MkdirAll(updatesDir, 0o700); err != nil {
		return "", err
	}
	finalPath := filepath.Join(updatesDir, filepath.Base(asset.Name))
	// Installer.app validates that a package has not changed between opening it
	// and receiving administrator approval. Reuse an already verified download
	// so a repeated click cannot replace the package while Installer is reading
	// it and trigger "Opened package is not the same at install time".
	if installerMatchesAsset(finalPath, asset) {
		return finalPath, nil
	}
	tempPath := finalPath + ".download"
	_ = os.Remove(tempPath)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	request.Header.Set("User-Agent", "Universal-Repair-POS/"+h.version)
	response, err := (&http.Client{Timeout: 15 * time.Minute}).Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("installer server returned HTTP %d", response.StatusCode)
	}
	output, err := os.OpenFile(tempPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hash), io.LimitReader(response.Body, asset.Size+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil || written != asset.Size {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("incomplete installer download")
	}
	actualDigest := hex.EncodeToString(hash.Sum(nil))
	if actualDigest != expectedDigest {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("installer SHA-256 verification failed")
	}
	_ = os.Remove(finalPath)
	if err := os.Rename(tempPath, finalPath); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return finalPath, nil
}

func installerMatchesAsset(path string, asset updateAsset) bool {
	expectedDigest := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asset.Digest)), "sha256:")
	if len(expectedDigest) != 64 || asset.Size <= 0 {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != asset.Size {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return hex.EncodeToString(hash.Sum(nil)) == expectedDigest
}

func selectUpdateAsset(release githubRelease, goos, goarch string) (updateAsset, bool) {
	wanted := ""
	switch {
	case goos == "darwin":
		wanted = "-macOS-Universal.pkg"
	case goos == "windows" && goarch == "amd64":
		wanted = "-Windows-x64-Setup.exe"
	case goos == "linux" && goarch == "amd64":
		wanted = "_amd64.deb"
	default:
		return updateAsset{}, false
	}
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, wanted) {
			return updateAsset{Name: asset.Name, DownloadURL: asset.DownloadURL, Digest: asset.Digest, Size: asset.Size}, true
		}
	}
	return updateAsset{}, false
}

func launchUpdateInstaller(path string) (bool, string, error) {
	switch runtime.GOOS {
	case "darwin":
		return true, "", exec.Command("/usr/bin/open", path).Start()
	case "windows":
		return true, "", exec.Command("cmd", "/c", "start", "", path).Start()
	case "linux":
		return false, "sudo apt install " + shellQuote(path), nil
	default:
		return false, "", fmt.Errorf("automatic installer launch is not supported on %s", runtime.GOOS)
	}
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

// StartFresh clears business records and identity while preserving the current
// owner account, server preferences, and backup archives.
func (h *MaintenanceHandler) StartFresh(c *gin.Context) {
	userID, _, role, ok := middleware.GetUserFromContext(c)
	if !ok || role != "admin" {
		c.JSON(http.StatusForbidden, models.APIResponse{Success: false, Message: "Only the owner administrator can start fresh"})
		return
	}
	var req struct {
		Confirmation string `json:"confirmation" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Confirmation != "START FRESH" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Type START FRESH exactly to confirm"})
		return
	}
	if h.manager.Supported() {
		if _, err := h.manager.CreateBackup("manual", h.version); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "A safety backup could not be created, so no data was deleted", Error: stringPtr(err.Error())})
			return
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not begin fresh start"})
		return
	}
	defer tx.Rollback()
	statements := []string{
		"DELETE FROM payments", "DELETE FROM order_status_history", "DELETE FROM order_items", "DELETE FROM orders",
		"DELETE FROM supplier_payments", "DELETE FROM supplier_purchase_items", "DELETE FROM supplier_purchases", "DELETE FROM suppliers",
		"DELETE FROM inventory", "DELETE FROM product_images", "DELETE FROM products", "DELETE FROM customers",
		"DELETE FROM categories", "DELETE FROM dining_tables",
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Fresh start failed; no database changes were applied", Error: stringPtr(err.Error())})
			return
		}
	}
	if _, err := tx.Exec("DELETE FROM users WHERE id <> $1", userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not clear staff accounts", Error: stringPtr(err.Error())})
		return
	}
	if _, err := tx.Exec("UPDATE users SET role = 'admin', is_active = true, profile_image_url = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $1", userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not preserve the owner account", Error: stringPtr(err.Error())})
		return
	}
	if _, err := tx.Exec(`UPDATE shop_profile SET company_name = 'Universal Repair POS', description = 'Sales, service, and repair management',
		logo_url = NULL, setup_completed = false, updated_at = CURRENT_TIMESTAMP WHERE id = 1`); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not reset shop identity", Error: stringPtr(err.Error())})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not finish fresh start", Error: stringPtr(err.Error())})
		return
	}
	if h.uploadDir != "" {
		entries, _ := os.ReadDir(h.uploadDir)
		for _, entry := range entries {
			_ = os.RemoveAll(filepath.Join(h.uploadDir, entry.Name()))
		}
		_ = os.MkdirAll(h.uploadDir, 0o755)
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Business data cleared. Continue through setup to enter your real shop details."})
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

func isVersionNewer(candidate, current string) bool {
	candidateParts, candidateOK := numericVersion(candidate)
	currentParts, currentOK := numericVersion(current)
	if !candidateOK || !currentOK {
		return false
	}
	for index := range candidateParts {
		if candidateParts[index] != currentParts[index] {
			return candidateParts[index] > currentParts[index]
		}
	}
	return false
}

func numericVersion(value string) ([3]int, bool) {
	var result [3]int
	value = strings.SplitN(normalizedVersion(value), "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) != len(result) {
		return result, false
	}
	for index, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return result, false
		}
		result[index] = parsed
	}
	return result, true
}
