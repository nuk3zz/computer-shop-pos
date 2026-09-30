package api

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"pos-backend/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestReturnedWarrantyRefundAdjustsRevenueAndProfit(t *testing.T) {
	db, err := database.Connect(database.Config{Driver: "sqlite", DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ownerID, categoryID, productID := uuid.New(), uuid.New(), uuid.New()
	orderID, orderItemID, paymentID, claimID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mustExecFinancial(t, db, `DELETE FROM users`)
	mustExecFinancial(t, db, `INSERT INTO users (id,username,email,password_hash,first_name,last_name,role,is_active) VALUES ($1,'owner','owner@example.invalid','!','Owner','','admin',true)`, ownerID)
	mustExecFinancial(t, db, `INSERT INTO categories (id,name) VALUES ($1,'Storage')`, categoryID)
	mustExecFinancial(t, db, `INSERT INTO products (id,category_id,name,price,cost_price,item_type) VALUES ($1,$2,'Example drive',2000,4000,'product')`, productID, categoryID)
	mustExecFinancial(t, db, `INSERT INTO orders (id,order_number,user_id,customer_name,customer_phone,order_type,status,fulfillment_type,fulfillment_status,stock_committed,total_amount) VALUES ($1,'SALE-REFUND',$2,'Example Customer','+10000000000','sale','completed','in_store','completed',true,2000)`, orderID, ownerID)
	mustExecFinancial(t, db, `INSERT INTO order_items (id,order_id,product_id,quantity,unit_price,unit_cost,total_price) VALUES ($1,$2,$3,1,2000,4000,2000)`, orderItemID, orderID, productID)
	mustExecFinancial(t, db, `INSERT INTO payments (id,order_id,payment_method,amount,status,processed_by,processed_at) VALUES ($1,$2,'cash',2000,'completed',$3,CURRENT_TIMESTAMP)`, paymentID, orderID, ownerID)
	mustExecFinancial(t, db, `INSERT INTO warranty_claims (id,claim_number,order_id,order_item_id,customer_name,customer_phone,product_id,product_name,issue_description,status,resolution,refund_amount,completed_at,created_by) VALUES ($1,'WR-REFUND',$2,$3,'Example Customer','+10000000000',$4,'Example drive','Not detected','returned','refund',2000,CURRENT_TIMESTAMP,$5)`, claimID, orderID, orderItemID, productID, ownerID)

	dashboard := httptest.NewRecorder()
	dashboardContext, _ := gin.CreateTestContext(dashboard)
	getDashboardStats(db)(dashboardContext)
	var dashboardResponse struct {
		Data map[string]float64 `json:"data"`
	}
	if err := json.Unmarshal(dashboard.Body.Bytes(), &dashboardResponse); err != nil {
		t.Fatal(err)
	}
	if dashboardResponse.Data["today_revenue"] != 0 || dashboardResponse.Data["today_profit"] != -4000 {
		t.Fatalf("unexpected dashboard totals: %#v", dashboardResponse.Data)
	}

	income := httptest.NewRecorder()
	incomeContext, _ := gin.CreateTestContext(income)
	incomeContext.Request = httptest.NewRequest("GET", "/reports/income?period=today", nil)
	getIncomeReport(db)(incomeContext)
	var incomeResponse struct {
		Data struct {
			Summary map[string]float64 `json:"summary"`
		} `json:"data"`
	}
	if err := json.Unmarshal(income.Body.Bytes(), &incomeResponse); err != nil {
		t.Fatal(err)
	}
	if incomeResponse.Data.Summary["gross_income"] != 0 || incomeResponse.Data.Summary["gross_profit"] != -4000 || incomeResponse.Data.Summary["cost_of_goods"] != 4000 {
		t.Fatalf("unexpected income totals: %#v", incomeResponse.Data.Summary)
	}

	sales := httptest.NewRecorder()
	salesContext, _ := gin.CreateTestContext(sales)
	salesContext.Request = httptest.NewRequest("GET", "/reports/sales?period=today", nil)
	getSalesReport(db)(salesContext)
	var salesResponse struct {
		Data []struct {
			Revenue float64 `json:"revenue"`
			Profit  float64 `json:"profit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sales.Body.Bytes(), &salesResponse); err != nil {
		t.Fatal(err)
	}
	if len(salesResponse.Data) != 1 || salesResponse.Data[0].Revenue != 0 || salesResponse.Data[0].Profit != -4000 {
		t.Fatalf("unexpected sales report: %#v", salesResponse.Data)
	}
}

func mustExecFinancial(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
