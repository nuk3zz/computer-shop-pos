package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pos-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestStartFreshClearsBusinessDataAndKeepsOwner(t *testing.T) {
	dataDir := t.TempDir()
	db, err := database.Connect(database.Config{Driver: "sqlite", DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ownerID := uuid.New()
	if _, err := db.Exec(`DELETE FROM users`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, first_name, last_name, role, is_active) VALUES ($1, 'owner', 'owner@example.com', '!', 'Owner', '', 'admin', true)`, ownerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO categories (id, name) VALUES ($1, 'Test')`, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE shop_profile SET company_name = 'test', setup_completed = true WHERE id = 1`); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/system/start-fresh", bytes.NewBufferString(`{"confirmation":"START FRESH"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("user_id", ownerID)
	context.Set("username", "owner")
	context.Set("role", "admin")

	NewMaintenanceHandler(db, dataDir, filepath.Join(dataDir, "uploads"), "test").StartFresh(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var categories, users int
	var companyName string
	var setupCompleted bool
	if err := db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&categories); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT company_name, setup_completed FROM shop_profile WHERE id = 1`).Scan(&companyName, &setupCompleted); err != nil {
		t.Fatal(err)
	}
	if categories != 0 || users != 1 || companyName != "Universal Repair POS" || setupCompleted {
		t.Fatalf("unexpected reset result: categories=%d users=%d company=%q setup=%v", categories, users, companyName, setupCompleted)
	}
}

func TestSelectUpdateAssetMatchesPlatformInstaller(t *testing.T) {
	release := githubRelease{TagName: "v0.3.4"}
	release.Assets = append(release.Assets,
		struct {
			Name        string `json:"name"`
			DownloadURL string `json:"browser_download_url"`
			Digest      string `json:"digest"`
			Size        int64  `json:"size"`
		}{Name: "Universal-Repair-POS-v0.3.4-macOS-Universal.pkg", DownloadURL: "https://github.com/example/mac", Digest: "sha256:abc", Size: 10},
		struct {
			Name        string `json:"name"`
			DownloadURL string `json:"browser_download_url"`
			Digest      string `json:"digest"`
			Size        int64  `json:"size"`
		}{Name: "Universal-Repair-POS-v0.3.4-Windows-x64-Setup.exe", DownloadURL: "https://github.com/example/windows", Digest: "sha256:def", Size: 20},
		struct {
			Name        string `json:"name"`
			DownloadURL string `json:"browser_download_url"`
			Digest      string `json:"digest"`
			Size        int64  `json:"size"`
		}{Name: "universal-repair-pos_0.3.4_amd64.deb", DownloadURL: "https://github.com/example/linux", Digest: "sha256:123", Size: 30},
	)

	tests := []struct {
		goos, goarch, name string
	}{
		{goos: "darwin", goarch: "arm64", name: "Universal-Repair-POS-v0.3.4-macOS-Universal.pkg"},
		{goos: "windows", goarch: "amd64", name: "Universal-Repair-POS-v0.3.4-Windows-x64-Setup.exe"},
		{goos: "linux", goarch: "amd64", name: "universal-repair-pos_0.3.4_amd64.deb"},
	}
	for _, test := range tests {
		asset, ok := selectUpdateAsset(release, test.goos, test.goarch)
		if !ok || asset.Name != test.name {
			t.Fatalf("%s/%s selected %#v, ok=%v", test.goos, test.goarch, asset, ok)
		}
	}
	if _, ok := selectUpdateAsset(release, "linux", "arm64"); ok {
		t.Fatal("linux arm64 should not claim an unavailable installer")
	}
}

func TestIsVersionNewerRejectsDowngrades(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{candidate: "v0.3.4", current: "v0.3.3", want: true},
		{candidate: "v0.4.0", current: "v0.3.9", want: true},
		{candidate: "v1.0.0", current: "v0.99.99", want: true},
		{candidate: "v0.3.3", current: "v0.3.3", want: false},
		{candidate: "v0.3.3", current: "v0.3.4", want: false},
		{candidate: "invalid", current: "v0.3.4", want: false},
	}
	for _, test := range tests {
		if got := isVersionNewer(test.candidate, test.current); got != test.want {
			t.Fatalf("isVersionNewer(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

func TestInstallerMatchesAssetReusesOnlyVerifiedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update.pkg")
	payload := []byte("verified installer payload")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	asset := updateAsset{Size: int64(len(payload)), Digest: "sha256:" + hex.EncodeToString(digest[:])}
	if !installerMatchesAsset(path, asset) {
		t.Fatal("expected verified installer to be reusable")
	}
	asset.Size++
	if installerMatchesAsset(path, asset) {
		t.Fatal("installer with a mismatched size must not be reused")
	}
	asset.Size--
	asset.Digest = "sha256:" + strings.Repeat("0", 64)
	if installerMatchesAsset(path, asset) {
		t.Fatal("installer with a mismatched digest must not be reused")
	}
}

func TestDownloadAndVerifyDoesNotReplaceMatchingInstaller(t *testing.T) {
	dataDir := t.TempDir()
	updatesDir := filepath.Join(dataDir, "updates")
	if err := os.MkdirAll(updatesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("installer already open in the operating-system installer")
	digest := sha256.Sum256(payload)
	asset := updateAsset{
		Name:        "Universal-Repair-POS-v0.3.6-macOS-Universal.pkg",
		DownloadURL: "https://github.com/not-called-because-the-file-is-verified",
		Digest:      "sha256:" + hex.EncodeToString(digest[:]),
		Size:        int64(len(payload)),
	}
	path := filepath.Join(updatesDir, asset.Name)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	handler := NewMaintenanceHandler(nil, dataDir, filepath.Join(dataDir, "uploads"), "v0.3.5")
	returnedPath, err := handler.downloadAndVerify(context.Background(), asset)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if returnedPath != path || !os.SameFile(before, after) {
		t.Fatal("matching installer was replaced instead of reused")
	}
}
