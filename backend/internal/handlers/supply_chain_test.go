package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pos-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestSupplierHistoryKeepsEveryPayment(t *testing.T) {
	dataDir := t.TempDir()
	db, err := database.Connect(database.Config{Driver: "sqlite", DataDir: dataDir})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	supplierID := uuid.NewString()
	categoryID := uuid.NewString()
	productID := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO suppliers (id,name,credit_allowed) VALUES ($1,'Test Supplier',true)`, supplierID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO categories (id,name) VALUES ($1,'Storage')`, categoryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO products (id,category_id,name,price,cost_price,item_type) VALUES ($1,$2,'500 GB SSD',15000,10000,'product')`, productID, categoryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inventory (id,product_id,current_stock) VALUES ($1,$2,0)`, uuid.NewString(), productID); err != nil {
		t.Fatal(err)
	}

	handler := NewSupplyChainHandler(db)
	callJSONHandler(t, http.MethodPost, "/supplier-purchases", `{"supplier_id":"`+supplierID+`","amount_paid":2000,"attachment_url":"/uploads/supplier-documents/invoice.pdf","items":[{"product_id":"`+productID+`","quantity":1,"unit_cost":10000}]}`, handler.CreatePurchase, http.StatusCreated)
	callJSONHandler(t, http.MethodPost, "/suppliers/"+supplierID+"/payments", `{"amount":3000,"notes":"First debt payment"}`, func(c *gin.Context) { c.Params = gin.Params{{Key: "id", Value: supplierID}}; handler.CreatePayment(c) }, http.StatusCreated)
	callJSONHandler(t, http.MethodPost, "/suppliers/"+supplierID+"/payments", `{"amount":5000,"notes":"Final debt payment","attachment_url":"/uploads/supplier-documents/receipt.png"}`, func(c *gin.Context) { c.Params = gin.Params{{Key: "id", Value: supplierID}}; handler.CreatePayment(c) }, http.StatusCreated)

	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodGet, "/supplier-transactions", nil)
	handler.GetTransactions(context)
	if response.Code != http.StatusOK {
		t.Fatalf("history status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Data []struct {
			Type          string  `json:"type"`
			Amount        float64 `json:"amount"`
			AttachmentURL *string `json:"attachment_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 4 {
		t.Fatalf("expected purchase plus three payment events, got %d: %s", len(body.Data), response.Body.String())
	}
	var stock int
	if err := db.QueryRow(`SELECT current_stock FROM inventory WHERE product_id=$1`, productID).Scan(&stock); err != nil || stock != 1 {
		t.Fatalf("expected one item in stock, stock=%d err=%v", stock, err)
	}
	var debt float64
	if err := db.QueryRow(`SELECT COALESCE((SELECT SUM(total_amount) FROM supplier_purchases WHERE supplier_id=$1),0)-COALESCE((SELECT SUM(amount) FROM supplier_payments WHERE supplier_id=$1),0)`, supplierID).Scan(&debt); err != nil || debt != 0 {
		t.Fatalf("expected cleared debt, debt=%v err=%v", debt, err)
	}
}

func callJSONHandler(t *testing.T, method, path, payload string, handler gin.HandlerFunc, wantStatus int) {
	t.Helper()
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(method, path, bytes.NewBufferString(payload))
	context.Request.Header.Set("Content-Type", "application/json")
	handler(context)
	if response.Code != wantStatus {
		t.Fatalf("%s returned %d, want %d: %s", path, response.Code, wantStatus, response.Body.String())
	}
}
