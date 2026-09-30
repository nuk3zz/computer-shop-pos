package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pos-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestWarrantyReplacementWorkflowCommitsStockOnce(t *testing.T) {
	db, err := database.Connect(database.Config{Driver: "sqlite", DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ownerID := uuid.New()
	categoryID := uuid.New()
	originalProductID := uuid.New()
	replacementProductID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	customerID := uuid.New()
	mustExecWarranty(t, db, `DELETE FROM users`)
	mustExecWarranty(t, db, `INSERT INTO users (id,username,email,password_hash,first_name,last_name,role,is_active) VALUES ($1,'owner','owner@example.invalid','!','Owner','','admin',true)`, ownerID)
	mustExecWarranty(t, db, `INSERT INTO categories (id,name) VALUES ($1,'Storage')`, categoryID)
	mustExecWarranty(t, db, `INSERT INTO products (id,category_id,name,price,cost_price,item_type) VALUES ($1,$2,'Original drive',1000,700,'product'),($3,$2,'Replacement drive',1000,700,'product')`, originalProductID, categoryID, replacementProductID)
	mustExecWarranty(t, db, `INSERT INTO inventory (id,product_id,current_stock) VALUES ($1,$2,0),($3,$4,1)`, uuid.New(), originalProductID, uuid.New(), replacementProductID)
	mustExecWarranty(t, db, `INSERT INTO customers (id,name,phone) VALUES ($1,'Example Customer','+10000000000')`, customerID)
	mustExecWarranty(t, db, `INSERT INTO orders (id,order_number,user_id,customer_id,customer_name,customer_phone,order_type,status,fulfillment_type,fulfillment_status,stock_committed,total_amount) VALUES ($1,'SALE-TEST',$2,$3,'Example Customer','+10000000000','sale','completed','in_store','completed',true,1000)`, orderID, ownerID, customerID)
	mustExecWarranty(t, db, `INSERT INTO order_items (id,order_id,product_id,quantity,unit_price,unit_cost,total_price) VALUES ($1,$2,$3,1,1000,700,1000)`, orderItemID, orderID, originalProductID)

	handler := NewWarrantyHandler(db)
	createBody := map[string]any{"order_item_id": orderItemID, "customer_name": "Example Customer", "customer_phone": "+10000000000", "issue_description": "Not detected"}
	createRecorder := warrantyRequest(t, http.MethodPost, "/warranty", createBody, ownerID, handler.CreateClaim)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var response struct {
		Data struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	claimID := response.Data.ID

	checking := warrantyRequest(t, http.MethodPatch, "/warranty/"+claimID.String(), map[string]any{"status": "checking"}, ownerID, handler.UpdateClaim)
	if checking.Code != http.StatusOK {
		t.Fatalf("checking returned %d: %s", checking.Code, checking.Body.String())
	}
	ready := warrantyRequest(t, http.MethodPatch, "/warranty/"+claimID.String(), map[string]any{"status": "ready_for_customer", "resolution": "replacement", "replacement_source": "shop_stock", "replacement_product_id": replacementProductID}, ownerID, handler.UpdateClaim)
	if ready.Code != http.StatusOK {
		t.Fatalf("ready returned %d: %s", ready.Code, ready.Body.String())
	}
	returned := warrantyRequest(t, http.MethodPatch, "/warranty/"+claimID.String(), map[string]any{"status": "returned"}, ownerID, handler.UpdateClaim)
	if returned.Code != http.StatusOK {
		t.Fatalf("returned returned %d: %s", returned.Code, returned.Body.String())
	}
	var stock int
	var status, resolution, replacementSource string
	var savedReplacement uuid.UUID
	if err := db.QueryRow(`SELECT current_stock FROM inventory WHERE product_id=$1`, replacementProductID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	var replacementCost float64
	if err := db.QueryRow(`SELECT status,resolution,replacement_source,replacement_product_id,replacement_cost FROM warranty_claims WHERE id=$1`, claimID).Scan(&status, &resolution, &replacementSource, &savedReplacement, &replacementCost); err != nil {
		t.Fatal(err)
	}
	if stock != 0 || status != "returned" || resolution != "replacement" || replacementSource != "shop_stock" || savedReplacement != replacementProductID || replacementCost != 700 {
		t.Fatalf("unexpected result stock=%d status=%s resolution=%s source=%s replacement=%s cost=%.2f", stock, status, resolution, replacementSource, savedReplacement, replacementCost)
	}
	var historyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM warranty_status_history WHERE claim_id=$1`, claimID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 4 {
		t.Fatalf("expected 4 history events, got %d", historyCount)
	}
	listRecorder := warrantyRequest(t, http.MethodGet, "/warranty", nil, ownerID, handler.GetClaims)
	if listRecorder.Code != http.StatusOK || !strings.Contains(listRecorder.Body.String(), "Replacement drive") || !strings.Contains(listRecorder.Body.String(), "returned") {
		t.Fatalf("list did not return the completed replacement claim: %d %s", listRecorder.Code, listRecorder.Body.String())
	}
}

func warrantyRequest(t *testing.T, method, path string, body any, userID uuid.UUID, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, bytes.NewReader(payload))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "id", Value: path[strings.LastIndex(path, "/")+1:]}}
	context.Set("user_id", userID)
	context.Set("username", "owner")
	context.Set("role", "admin")
	handler(context)
	return recorder
}

func mustExecWarranty(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
