package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pos-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestCreateProductSaleCommitsStockClientPaymentAndWorkflowAtomically(t *testing.T) {
	tests := []struct {
		name                string
		fulfillmentType     string
		expectedOrderStatus string
		expectedFulfillment string
		expectedPayments    int
	}{
		{name: "instant sale", fulfillmentType: "in_store", expectedOrderStatus: "completed", expectedFulfillment: "completed", expectedPayments: 1},
		{name: "prepaid delivery", fulfillmentType: "delivery", expectedOrderStatus: "confirmed", expectedFulfillment: "pending_packing", expectedPayments: 1},
		{name: "cash on delivery", fulfillmentType: "cash_on_delivery", expectedOrderStatus: "pending", expectedFulfillment: "pending_packing", expectedPayments: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, userID, productID := productSaleTestDatabase(t, 1)
			defer db.Close()
			body := fmt.Sprintf(`{"order_type":"sale","fulfillment_type":%q,"payment_method":"cash","customer_name":"Test Client","customer_phone":"0770000000","items":[{"product_id":%q,"quantity":1}]}`, test.fulfillmentType, productID)

			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
			context.Request.Header.Set("Content-Type", "application/json")
			context.Set("user_id", userID)
			context.Set("username", "owner")
			context.Set("role", "admin")

			NewOrderHandler(db).CreateOrder(context)
			if recorder.Code != http.StatusCreated {
				t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
			}
			var stock, clients, payments int
			var orderStatus, fulfillmentStatus string
			if err := db.QueryRow(`SELECT current_stock FROM inventory WHERE product_id = $1`, productID).Scan(&stock); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM customers`).Scan(&clients); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM payments`).Scan(&payments); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRow(`SELECT status, fulfillment_status FROM orders LIMIT 1`).Scan(&orderStatus, &fulfillmentStatus); err != nil {
				t.Fatal(err)
			}
			if stock != 0 || clients != 1 || payments != test.expectedPayments || orderStatus != test.expectedOrderStatus || fulfillmentStatus != test.expectedFulfillment {
				t.Fatalf("unexpected sale state: stock=%d clients=%d payments=%d status=%s fulfillment=%s", stock, clients, payments, orderStatus, fulfillmentStatus)
			}
		})
	}
}

func TestSaleSellingPriceOverride(t *testing.T) {
	for _, test := range []struct {
		name, price string
		status      int
		expected    float64
	}{
		{"blank uses catalog", "", 201, 15000},
		{"discount", `,"selling_price":12000`, 201, 12000},
		{"negative rejected", `,"selling_price":-1`, 400, 0},
		{"above catalog rejected", `,"selling_price":16000`, 400, 0},
		{"fractional cents rejected", `,"selling_price":12000.001`, 400, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, userID, productID := productSaleTestDatabase(t, 1)
			defer db.Close()
			body := fmt.Sprintf(`{"order_type":"sale","items":[{"product_id":%q,"quantity":1%s}]}`, productID, test.price)
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
			context.Request.Header.Set("Content-Type", "application/json")
			context.Set("user_id", userID)
			context.Set("role", "admin")
			context.Set("username", "owner")
			NewOrderHandler(db).CreateOrder(context)
			if recorder.Code != test.status {
				t.Fatalf("got %d: %s", recorder.Code, recorder.Body.String())
			}
			if test.status == 201 {
				var amount, price, cost, payment, catalog float64
				if err := db.QueryRow(`SELECT o.total_amount, oi.unit_price, oi.unit_cost, p.amount, pr.price FROM orders o JOIN order_items oi ON oi.order_id=o.id JOIN payments p ON p.order_id=o.id JOIN products pr ON pr.id=oi.product_id`).Scan(&amount, &price, &cost, &payment, &catalog); err != nil {
					t.Fatal(err)
				}
				if amount != test.expected || price != test.expected || payment != test.expected || cost != 10000 || catalog != 15000 {
					t.Fatalf("incorrect discounted sale: amount=%v price=%v cost=%v payment=%v catalog=%v", amount, price, cost, payment, catalog)
				}
			} else {
				var count, stock int
				_ = db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count)
				_ = db.QueryRow(`SELECT current_stock FROM inventory WHERE product_id=$1`, productID).Scan(&stock)
				if count != 0 || stock != 1 {
					t.Fatal("invalid price mutated sale or inventory")
				}
			}
		})
	}
}

func TestCreateProductSaleRollsBackWhenStockIsUnavailable(t *testing.T) {
	db, userID, productID := productSaleTestDatabase(t, 0)
	defer db.Close()
	body := fmt.Sprintf(`{"order_type":"sale","fulfillment_type":"in_store","customer_name":"Should Roll Back","customer_phone":"0771111111","items":[{"product_id":%q,"quantity":1}]}`, productID)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("user_id", userID)
	context.Set("username", "owner")
	context.Set("role", "admin")

	NewOrderHandler(db).CreateOrder(context)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var orders, customers int
	_ = db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orders)
	_ = db.QueryRow(`SELECT COUNT(*) FROM customers`).Scan(&customers)
	if orders != 0 || customers != 0 {
		t.Fatalf("failed sale was not rolled back: orders=%d customers=%d", orders, customers)
	}
}

func TestCashOnDeliveryCompletesAfterDeliveryAndPayment(t *testing.T) {
	db, userID, productID := productSaleTestDatabase(t, 1)
	defer db.Close()
	body := fmt.Sprintf(`{"order_type":"sale","fulfillment_type":"cash_on_delivery","customer_name":"COD Client","customer_phone":"0772222222","items":[{"product_id":%q,"quantity":1}]}`, productID)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("user_id", userID)
	context.Set("username", "owner")
	context.Set("role", "admin")
	handler := NewOrderHandler(db)
	handler.CreateOrder(context)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create COD order: %d %s", recorder.Code, recorder.Body.String())
	}
	var orderID string
	if err := db.QueryRow(`SELECT id FROM orders LIMIT 1`).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"packed", "with_courier", "delivered"} {
		recorder = httptest.NewRecorder()
		context, _ = gin.CreateTestContext(recorder)
		context.Params = gin.Params{{Key: "id", Value: orderID}}
		context.Request = httptest.NewRequest(http.MethodPatch, "/orders/"+orderID+"/fulfillment", bytes.NewBufferString(fmt.Sprintf(`{"status":%q}`, status)))
		context.Request.Header.Set("Content-Type", "application/json")
		context.Set("user_id", userID)
		context.Set("username", "owner")
		context.Set("role", "admin")
		handler.UpdateFulfillmentStatus(context)
		if recorder.Code != http.StatusOK {
			t.Fatalf("set fulfillment %s: %d %s", status, recorder.Code, recorder.Body.String())
		}
	}
	var orderStatus string
	_ = db.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&orderStatus)
	if orderStatus == "completed" {
		t.Fatal("unpaid delivered COD order must not be complete")
	}

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: orderID}}
	context.Request = httptest.NewRequest(http.MethodPost, "/orders/"+orderID+"/payments", bytes.NewBufferString(`{"payment_method":"cash","amount":15000}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("user_id", userID)
	context.Set("username", "owner")
	context.Set("role", "admin")
	NewPaymentHandler(db).ProcessPayment(context)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("record COD payment: %d %s", recorder.Code, recorder.Body.String())
	}
	_ = db.QueryRow(`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&orderStatus)
	if orderStatus != "completed" {
		t.Fatalf("paid delivered COD order should be complete, got %s", orderStatus)
	}
}

func productSaleTestDatabase(t *testing.T, stock int) (*sql.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := database.Connect(database.Config{Driver: "sqlite", DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	userID, productID := uuid.New(), uuid.New()
	if _, err := db.Exec(`DELETE FROM users`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, username, email, password_hash, first_name, last_name, role, is_active) VALUES ($1, 'owner', 'owner@example.com', '!', 'Owner', '', 'admin', true)`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO products (id, name, price, cost_price, item_type, is_available) VALUES ($1, 'Dock', 15000, 10000, 'product', true)`, productID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inventory (product_id, current_stock, unit_cost) VALUES ($1, $2, 10000)`, productID, stock); err != nil {
		t.Fatal(err)
	}
	return db, userID, productID
}
