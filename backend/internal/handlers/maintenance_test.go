package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	if categories != 0 || users != 1 || companyName != "Computer Shop POS" || setupCompleted {
		t.Fatalf("unexpected reset result: categories=%d users=%d company=%q setup=%v", categories, users, companyName, setupCompleted)
	}
}
