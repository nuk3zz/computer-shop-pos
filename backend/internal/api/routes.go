package api

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pos-backend/internal/database"
	"pos-backend/internal/handlers"
	"pos-backend/internal/middleware"
	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.RouterGroup, db *sql.DB, authMiddleware gin.HandlerFunc, uploadDir, dataDir, appVersion string) {
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	orderHandler := handlers.NewOrderHandler(db)
	productHandler := handlers.NewProductHandler(db)
	customerHandler := handlers.NewCustomerHandler(db)
	paymentHandler := handlers.NewPaymentHandler(db)
	imageUploadHandler := handlers.NewImageUploadHandler(uploadDir)
	shopProfileHandler := handlers.NewShopProfileHandler(db)
	initialSetupHandler := handlers.NewInitialSetupHandler(db, imageUploadHandler)
	maintenanceHandler := handlers.NewMaintenanceHandler(db, dataDir, uploadDir, appVersion)
	supplyChainHandler := handlers.NewSupplyChainHandler(db)

	// Public routes (no authentication required)
	public := router.Group("/")
	{
		// Authentication routes
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/logout", authHandler.Logout)
		public.GET("/setup/status", initialSetupHandler.Status)
		public.POST("/setup/upload", initialSetupHandler.UploadLogo)
		public.POST("/setup/complete", initialSetupHandler.Complete)
	}

	// Protected routes (authentication required)
	protected := router.Group("/")
	protected.Use(authMiddleware)
	{
		// Authentication routes
		protected.GET("/auth/me", authHandler.GetCurrentUser)
		protected.PUT("/auth/profile", authHandler.UpdateProfile)

		// Product routes
		protected.GET("/products", productHandler.GetProducts)
		protected.GET("/products/:id", productHandler.GetProduct)
		protected.GET("/categories", productHandler.GetCategories)
		protected.GET("/categories/:id/products", productHandler.GetProductsByCategory)
		protected.GET("/customers", customerHandler.GetCustomers)
		protected.GET("/customers/:id", customerHandler.GetCustomer)
		protected.GET("/shop-profile", shopProfileHandler.Get)
		protected.GET("/suppliers", supplyChainHandler.GetSuppliers)
		protected.GET("/supplier-purchases", supplyChainHandler.GetPurchases)

		// Sales and repair-ticket routes
		protected.GET("/orders", orderHandler.GetOrders)
		protected.POST("/orders", middleware.RequireRoles([]string{"admin", "manager", "sales", "technician"}), orderHandler.CreateOrder)
		protected.GET("/orders/:id", orderHandler.GetOrder)
		protected.PATCH("/orders/:id/status", orderHandler.UpdateOrderStatus)
		protected.PATCH("/orders/:id/fulfillment", middleware.RequireRoles([]string{"admin", "manager", "sales"}), orderHandler.UpdateFulfillmentStatus)

		// Payment routes (counter/admin only)
		protected.GET("/orders/:id/payments", paymentHandler.GetPayments)
		protected.GET("/orders/:id/payment-summary", paymentHandler.GetPaymentSummary)
	}

	// Admin routes (admin/manager only)
	admin := router.Group("/admin")
	admin.Use(authMiddleware)
	admin.Use(middleware.RequireRoles([]string{"admin", "manager"}))
	{
		// Dashboard and monitoring
		admin.GET("/dashboard/stats", getDashboardStats(db))
		admin.GET("/reports/sales", getSalesReport(db))
		admin.GET("/reports/orders", getOrdersReport(db))
		admin.GET("/reports/income", getIncomeReport(db))

		// Catalog management with pagination
		admin.GET("/products", productHandler.GetProducts) // Use existing paginated handler
		admin.GET("/categories", getAdminCategories(db))   // Add pagination
		admin.POST("/categories", createCategory(db))
		admin.PUT("/categories/:id", updateCategory(db))
		admin.DELETE("/categories/:id", deleteCategory(db))
		admin.POST("/products", createProduct(db))
		admin.PUT("/products/:id", updateProduct(db))
		admin.DELETE("/products/:id", deleteProduct(db))
		admin.POST("/uploads/images", imageUploadHandler.UploadProductImage)
		admin.PUT("/shop-profile", shopProfileHandler.Update)
		admin.POST("/setup/complete", initialSetupHandler.CompleteAuthenticated)
		admin.GET("/system/info", maintenanceHandler.Info)
		admin.PUT("/system/preferences", maintenanceHandler.UpdatePreferences)
		admin.GET("/system/backups", maintenanceHandler.ListBackups)
		admin.POST("/system/backups", maintenanceHandler.CreateBackup)
		admin.GET("/system/backups/:name/download", maintenanceHandler.DownloadBackup)
		admin.POST("/system/backups/upload", maintenanceHandler.UploadBackup)
		admin.POST("/system/restore", maintenanceHandler.StageRestore)
		admin.GET("/system/updates", maintenanceHandler.CheckUpdates)
		admin.POST("/system/start-fresh", maintenanceHandler.StartFresh)
		admin.POST("/customers", customerHandler.CreateCustomer)
		admin.PUT("/customers/:id", customerHandler.UpdateCustomer)
		admin.POST("/suppliers", supplyChainHandler.CreateSupplier)
		admin.PUT("/suppliers/:id", supplyChainHandler.UpdateSupplier)
		admin.POST("/supplier-purchases", supplyChainHandler.CreatePurchase)
		admin.POST("/suppliers/:id/payments", supplyChainHandler.CreatePayment)

		// User management with pagination
		admin.GET("/users", getAdminUsers(db)) // Update with pagination
		admin.POST("/users", createUser(db))
		admin.PUT("/users/:id", updateUser(db))
		admin.DELETE("/users/:id", deleteUser(db))

		// Advanced order management
		admin.POST("/orders", orderHandler.CreateOrder)                   // Admins can create any type of order
		admin.POST("/orders/:id/payments", paymentHandler.ProcessPayment) // Admins can process payments
	}

}

// Dashboard stats handler
func getDashboardStats(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get basic stats for dashboard
		stats := make(map[string]interface{})

		// Today's orders
		var todayOrders int
		todayOrdersQuery := `SELECT COUNT(*) FROM orders WHERE DATE(created_at) = CURRENT_DATE`
		if database.IsSQLite(db) {
			todayOrdersQuery = `SELECT COUNT(*) FROM orders WHERE date(created_at, 'localtime') = date('now', 'localtime')`
		}
		db.QueryRow(todayOrdersQuery).Scan(&todayOrders)

		// Revenue and profit are recognized when an order is fully paid, even if a
		// prepaid delivery is still moving through fulfillment.
		var todayRevenue, todayProfit float64
		paidToday := `DATE(paid.paid_at) = CURRENT_DATE`
		if database.IsSQLite(db) {
			paidToday = `date(paid.paid_at, 'localtime') = date('now', 'localtime')`
		}
		db.QueryRow(fmt.Sprintf(`
			WITH paid AS (
				SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at
				FROM payments WHERE status = 'completed' GROUP BY order_id
			), costs AS (
				SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id
			)
			SELECT COALESCE(SUM(o.total_amount), 0),
			       COALESCE(SUM(o.total_amount - o.tax_amount - COALESCE(costs.total_cost, 0)), 0)
			FROM orders o JOIN paid ON paid.order_id = o.id
			LEFT JOIN costs ON costs.order_id = o.id
			WHERE paid.total_paid >= o.total_amount AND %s
		`, paidToday)).Scan(&todayRevenue, &todayProfit)

		// Active orders
		var activeOrders int
		db.QueryRow(`
			SELECT COUNT(*) 
			FROM orders 
			WHERE status NOT IN ('completed', 'cancelled')
		`).Scan(&activeOrders)

		// Open service/repair tickets
		var openRepairs int
		db.QueryRow(`
			SELECT COUNT(*) 
			FROM orders
			WHERE order_type = 'service' AND status NOT IN ('completed', 'cancelled')
		`).Scan(&openRepairs)

		stats["today_orders"] = todayOrders
		stats["today_revenue"] = todayRevenue
		stats["today_profit"] = todayProfit
		stats["active_orders"] = activeOrders
		stats["open_repairs"] = openRepairs

		c.JSON(200, gin.H{
			"success": true,
			"message": "Dashboard stats retrieved successfully",
			"data":    stats,
		})
	}
}

// Sales report handler
func getSalesReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "today") // today, week, month

		var query string
		if database.IsSQLite(db) {
			groupExpression := "date(paid.paid_at)"
			whereExpression := "date(paid.paid_at, 'localtime') = date('now', 'localtime')"
			switch period {
			case "week":
				whereExpression = "paid.paid_at >= datetime('now', '-7 days')"
			case "month":
				whereExpression = "paid.paid_at >= datetime('now', '-30 days')"
			default:
				groupExpression = "strftime('%Y-%m-%dT%H:00:00', paid.paid_at, 'localtime')"
			}
			query = fmt.Sprintf(`
				WITH paid AS (SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at FROM payments WHERE status = 'completed' GROUP BY order_id),
				cost AS (SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id)
				SELECT %s as date, COUNT(*) as order_count, SUM(o.total_amount) as revenue,
				       SUM(o.total_amount - o.tax_amount - COALESCE(cost.total_cost, 0)) as profit
				FROM orders o JOIN paid ON paid.order_id = o.id LEFT JOIN cost ON cost.order_id = o.id
				WHERE %s AND paid.total_paid >= o.total_amount AND o.status != 'cancelled'
				GROUP BY %s ORDER BY date DESC`, groupExpression, whereExpression, groupExpression)
		} else {
			switch period {
			case "week":
				query = `
				WITH paid AS (SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at FROM payments WHERE status = 'completed' GROUP BY order_id),
				cost AS (SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id)
				SELECT DATE(paid.paid_at) as date, COUNT(*) as order_count, SUM(o.total_amount) as revenue,
				       SUM(o.total_amount - o.tax_amount - COALESCE(cost.total_cost, 0)) as profit
				FROM orders o JOIN paid ON paid.order_id = o.id LEFT JOIN cost ON cost.order_id = o.id
				WHERE paid.paid_at >= CURRENT_DATE - INTERVAL '7 days' AND paid.total_paid >= o.total_amount AND o.status != 'cancelled'
				GROUP BY DATE(paid.paid_at)
				ORDER BY date DESC
			`
			case "month":
				query = `
				WITH paid AS (SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at FROM payments WHERE status = 'completed' GROUP BY order_id),
				cost AS (SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id)
				SELECT DATE(paid.paid_at) as date, COUNT(*) as order_count, SUM(o.total_amount) as revenue,
				       SUM(o.total_amount - o.tax_amount - COALESCE(cost.total_cost, 0)) as profit
				FROM orders o JOIN paid ON paid.order_id = o.id LEFT JOIN cost ON cost.order_id = o.id
				WHERE paid.paid_at >= CURRENT_DATE - INTERVAL '30 days' AND paid.total_paid >= o.total_amount AND o.status != 'cancelled'
				GROUP BY DATE(paid.paid_at)
				ORDER BY date DESC
			`
			default: // today
				query = `
				WITH paid AS (SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at FROM payments WHERE status = 'completed' GROUP BY order_id),
				cost AS (SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id)
				SELECT DATE_TRUNC('hour', paid.paid_at) as hour, COUNT(*) as order_count, SUM(o.total_amount) as revenue,
				       SUM(o.total_amount - o.tax_amount - COALESCE(cost.total_cost, 0)) as profit
				FROM orders o JOIN paid ON paid.order_id = o.id LEFT JOIN cost ON cost.order_id = o.id
				WHERE DATE(paid.paid_at) = CURRENT_DATE AND paid.total_paid >= o.total_amount AND o.status != 'cancelled'
				GROUP BY DATE_TRUNC('hour', paid.paid_at)
				ORDER BY hour DESC
			`
			}
		}

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch sales report",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var report []map[string]interface{}
		for rows.Next() {
			var date interface{}
			var orderCount int
			var revenue float64
			var profit float64

			err := rows.Scan(&date, &orderCount, &revenue, &profit)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan sales data",
					"error":   err.Error(),
				})
				return
			}

			report = append(report, map[string]interface{}{
				"date":        date,
				"order_count": orderCount,
				"revenue":     revenue,
				"profit":      profit,
			})
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Sales report retrieved successfully",
			"data":    report,
		})
	}
}

// Orders report handler
func getOrdersReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get order statistics
		query := `
			SELECT 
				status,
				COUNT(*) as count,
				AVG(total_amount) as avg_amount
			FROM orders 
			WHERE DATE(created_at) = CURRENT_DATE
			GROUP BY status
		`

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch orders report",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var report []map[string]interface{}
		for rows.Next() {
			var status string
			var count int
			var avgAmount float64

			err := rows.Scan(&status, &count, &avgAmount)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan orders data",
					"error":   err.Error(),
				})
				return
			}

			report = append(report, map[string]interface{}{
				"status":     status,
				"count":      count,
				"avg_amount": avgAmount,
			})
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Orders report retrieved successfully",
			"data":    report,
		})
	}
}

// Kitchen orders handler
func getKitchenOrders(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.DefaultQuery("status", "all")

		query := `
			SELECT DISTINCT o.id, o.order_number, o.table_id, o.order_type, o.status, 
			       o.created_at, o.customer_name,
			       t.table_number
			FROM orders o
			LEFT JOIN dining_tables t ON o.table_id = t.id
			WHERE o.status IN ('confirmed', 'preparing', 'ready')
		`

		if status != "all" {
			query += ` AND o.status = '` + status + `'`
		}

		query += ` ORDER BY o.created_at ASC`

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch kitchen orders",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var orders []map[string]interface{}
		for rows.Next() {
			var orderID, tableID interface{}
			var orderNumber, orderType, orderStatus, customerName, tableNumber sql.NullString
			var createdAt interface{}

			err := rows.Scan(&orderID, &orderNumber, &tableID, &orderType, &orderStatus,
				&createdAt, &customerName, &tableNumber)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan kitchen order",
					"error":   err.Error(),
				})
				return
			}

			order := map[string]interface{}{
				"id":            orderID,
				"order_number":  orderNumber.String,
				"table_id":      tableID,
				"table_number":  tableNumber.String,
				"order_type":    orderType.String,
				"status":        orderStatus.String,
				"customer_name": customerName.String,
				"created_at":    createdAt,
			}

			orders = append(orders, order)
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Kitchen orders retrieved successfully",
			"data":    orders,
		})
	}
}

// Update order item status handler
func updateOrderItemStatus(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("id")
		itemID := c.Param("item_id")

		var req struct {
			Status string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		// Update order item status
		_, err := db.Exec(`
			UPDATE order_items 
			SET status = $1, updated_at = CURRENT_TIMESTAMP 
			WHERE id = $2 AND order_id = $3
		`, req.Status, itemID, orderID)

		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to update order item status",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Order item status updated successfully",
		})
	}
}

// Admin handler - Income report
func getIncomeReport(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		period := c.DefaultQuery("period", "today") // today, week, month, year

		var query string
		if database.IsSQLite(db) {
			groupExpression := "strftime('%Y-%m-%dT%H:00:00', paid.paid_at, 'localtime')"
			whereExpression := "date(paid.paid_at, 'localtime') = date('now', 'localtime')"
			switch period {
			case "week":
				groupExpression = "date(paid.paid_at)"
				whereExpression = "paid.paid_at >= datetime('now', '-7 days')"
			case "month":
				groupExpression = "date(paid.paid_at)"
				whereExpression = "paid.paid_at >= datetime('now', '-30 days')"
			case "year":
				groupExpression = "strftime('%Y-%m-01', paid.paid_at)"
				whereExpression = "paid.paid_at >= datetime('now', '-1 year')"
			}
			query = fmt.Sprintf(`
				WITH paid AS (SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at FROM payments WHERE status = 'completed' GROUP BY order_id),
				costs AS (SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id)
				SELECT %s as period, COUNT(*) as total_orders, SUM(o.total_amount) as gross_income,
				       SUM(o.tax_amount) as tax_collected, SUM(o.total_amount - o.tax_amount) as net_income,
				       SUM(o.total_amount - o.tax_amount - COALESCE(costs.total_cost, 0)) as profit
				FROM orders o JOIN paid ON paid.order_id = o.id LEFT JOIN costs ON costs.order_id = o.id
				WHERE %s AND paid.total_paid >= o.total_amount AND o.status != 'cancelled'
				GROUP BY %s ORDER BY period DESC`, groupExpression, whereExpression, groupExpression)
		} else {
			groupExpression := "DATE_TRUNC('hour', paid.paid_at)"
			whereExpression := "DATE(paid.paid_at) = CURRENT_DATE"
			switch period {
			case "week":
				groupExpression = "DATE_TRUNC('day', paid.paid_at)"
				whereExpression = "paid.paid_at >= CURRENT_DATE - INTERVAL '7 days'"
			case "month":
				groupExpression = "DATE_TRUNC('day', paid.paid_at)"
				whereExpression = "paid.paid_at >= CURRENT_DATE - INTERVAL '30 days'"
			case "year":
				groupExpression = "DATE_TRUNC('month', paid.paid_at)"
				whereExpression = "paid.paid_at >= CURRENT_DATE - INTERVAL '1 year'"
			}
			query = fmt.Sprintf(`
				WITH paid AS (SELECT order_id, SUM(amount) total_paid, MAX(processed_at) paid_at FROM payments WHERE status = 'completed' GROUP BY order_id),
				costs AS (SELECT order_id, SUM(unit_cost * quantity) total_cost FROM order_items GROUP BY order_id)
				SELECT %s as period, COUNT(*) as total_orders, SUM(o.total_amount) as gross_income,
				       SUM(o.tax_amount) as tax_collected, SUM(o.total_amount - o.tax_amount) as net_income,
				       SUM(o.total_amount - o.tax_amount - COALESCE(costs.total_cost, 0)) as profit
				FROM orders o JOIN paid ON paid.order_id = o.id LEFT JOIN costs ON costs.order_id = o.id
				WHERE %s AND paid.total_paid >= o.total_amount AND o.status != 'cancelled'
				GROUP BY %s ORDER BY period DESC`, groupExpression, whereExpression, groupExpression)
		}

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch income report",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var report []map[string]interface{}
		var totalGross, totalTax, totalNet, totalProfit float64
		var totalOrders int

		for rows.Next() {
			var period interface{}
			var orders int
			var gross, tax, net, profit float64

			err := rows.Scan(&period, &orders, &gross, &tax, &net, &profit)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan income data",
					"error":   err.Error(),
				})
				return
			}

			totalOrders += orders
			totalGross += gross
			totalTax += tax
			totalNet += net
			totalProfit += profit

			report = append(report, map[string]interface{}{
				"period": period,
				"orders": orders,
				"gross":  gross,
				"tax":    tax,
				"net":    net,
				"profit": profit,
			})
		}

		result := map[string]interface{}{
			"summary": map[string]interface{}{
				"total_orders":  totalOrders,
				"gross_income":  totalGross,
				"tax_collected": totalTax,
				"net_income":    totalNet,
				"gross_profit":  totalProfit,
				"cost_of_goods": totalNet - totalProfit,
			},
			"breakdown": report,
			"period":    period,
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Income report retrieved successfully",
			"data":    result,
		})
	}
}

// Admin handler - Create category
func createCategory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string  `json:"name" binding:"required"`
			Description *string `json:"description"`
			Color       *string `json:"color"`
			SortOrder   int     `json:"sort_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		var categoryID string
		err := db.QueryRow(`
			INSERT INTO categories (name, description, color, sort_order)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, req.Name, req.Description, req.Color, req.SortOrder).Scan(&categoryID)

		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to create category",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(201, gin.H{
			"success": true,
			"message": "Category created successfully",
			"data":    map[string]interface{}{"id": categoryID},
		})
	}
}

// Admin handler - Update category
func updateCategory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Param("id")

		var req struct {
			Name        *string `json:"name"`
			Description *string `json:"description"`
			Color       *string `json:"color"`
			SortOrder   *int    `json:"sort_order"`
			IsActive    *bool   `json:"is_active"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argCount := 1

		if req.Name != nil {
			updates = append(updates, fmt.Sprintf("name = $%d", argCount))
			args = append(args, *req.Name)
			argCount++
		}
		if req.Description != nil {
			updates = append(updates, fmt.Sprintf("description = $%d", argCount))
			args = append(args, req.Description)
			argCount++
		}
		if req.Color != nil {
			updates = append(updates, fmt.Sprintf("color = $%d", argCount))
			args = append(args, req.Color)
			argCount++
		}
		if req.SortOrder != nil {
			updates = append(updates, fmt.Sprintf("sort_order = $%d", argCount))
			args = append(args, *req.SortOrder)
			argCount++
		}
		if req.IsActive != nil {
			updates = append(updates, fmt.Sprintf("is_active = $%d", argCount))
			args = append(args, *req.IsActive)
			argCount++
		}

		if len(updates) == 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "No fields to update",
			})
			return
		}

		updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, categoryID)

		query := fmt.Sprintf(`
			UPDATE categories 
			SET %s 
			WHERE id = $%d
		`, strings.Join(updates, ", "), argCount)

		result, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to update category",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "Category not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Category updated successfully",
		})
	}
}

// Admin handler - Delete category
func deleteCategory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Param("id")

		// Check if category has products
		var productCount int
		db.QueryRow("SELECT COUNT(*) FROM products WHERE category_id = $1", categoryID).Scan(&productCount)

		if productCount > 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Cannot delete category with existing products",
				"error":   "category_has_products",
			})
			return
		}

		result, err := db.Exec("DELETE FROM categories WHERE id = $1", categoryID)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to delete category",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "Category not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Category deleted successfully",
		})
	}
}

// Admin handler - Create product
func createProduct(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CategoryID      *string  `json:"category_id"`
			Name            string   `json:"name" binding:"required"`
			Description     *string  `json:"description"`
			Price           float64  `json:"price" binding:"required"`
			CostPrice       float64  `json:"cost_price"`
			ItemType        string   `json:"item_type" binding:"required"`
			PreorderEnabled bool     `json:"preorder_enabled"`
			ImageURL        *string  `json:"image_url"`
			ImageURLs       []string `json:"image_urls"`
			Barcode         *string  `json:"barcode"`
			SKU             *string  `json:"sku"`
			IsAvailable     *bool    `json:"is_available"`
			StockQuantity   int      `json:"stock_quantity"`
			PreparationTime int      `json:"preparation_time"`
			SortOrder       int      `json:"sort_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		var productID string
		if req.ItemType != "product" && req.ItemType != "service" {
			c.JSON(400, gin.H{"success": false, "message": "Item type must be product or service"})
			return
		}
		if req.StockQuantity < 0 {
			c.JSON(400, gin.H{"success": false, "message": "Stock quantity cannot be negative"})
			return
		}
		if len(req.ImageURLs) == 0 && req.ImageURL != nil && strings.TrimSpace(*req.ImageURL) != "" {
			req.ImageURLs = []string{*req.ImageURL}
		}
		if len(req.ImageURLs) > 10 {
			c.JSON(400, gin.H{"success": false, "message": "A catalog item can have at most 10 images"})
			return
		}
		var primaryImage *string
		if len(req.ImageURLs) > 0 {
			primaryImage = &req.ImageURLs[0]
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to start product transaction", "error": err.Error()})
			return
		}
		defer tx.Rollback()

		err = tx.QueryRow(`
			INSERT INTO products (category_id, name, description, price, cost_price, item_type, preorder_enabled, image_url, barcode, sku, is_available, preparation_time, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE($11, true), $12, $13)
			RETURNING id
		`, req.CategoryID, req.Name, req.Description, req.Price, req.CostPrice, req.ItemType, req.PreorderEnabled, primaryImage, req.Barcode, req.SKU, req.IsAvailable, req.PreparationTime, req.SortOrder).Scan(&productID)

		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to create product",
				"error":   err.Error(),
			})
			return
		}

		stock := req.StockQuantity
		if req.ItemType == "service" {
			stock = 0
		}
		if _, err := tx.Exec(`INSERT INTO inventory (product_id, current_stock, minimum_stock, maximum_stock, unit_cost)
			VALUES ($1, $2, 0, $3, $4) ON CONFLICT (product_id) DO UPDATE SET unit_cost = EXCLUDED.unit_cost`,
			productID, stock, stock, req.CostPrice); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to create product inventory", "error": err.Error()})
			return
		}
		if err := replaceProductImages(tx, productID, req.ImageURLs); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to save product images", "error": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to save product", "error": err.Error()})
			return
		}

		c.JSON(201, gin.H{
			"success": true,
			"message": "Product created successfully",
			"data":    map[string]interface{}{"id": productID},
		})
	}
}

// Admin handler - Update product
func updateProduct(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		productID := c.Param("id")

		var req struct {
			CategoryID      *string   `json:"category_id"`
			Name            *string   `json:"name"`
			Description     *string   `json:"description"`
			Price           *float64  `json:"price"`
			CostPrice       *float64  `json:"cost_price"`
			ItemType        *string   `json:"item_type"`
			PreorderEnabled *bool     `json:"preorder_enabled"`
			ImageURL        *string   `json:"image_url"`
			ImageURLs       *[]string `json:"image_urls"`
			Barcode         *string   `json:"barcode"`
			SKU             *string   `json:"sku"`
			IsAvailable     *bool     `json:"is_available"`
			StockQuantity   *int      `json:"stock_quantity"`
			PreparationTime *int      `json:"preparation_time"`
			SortOrder       *int      `json:"sort_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}
		if req.StockQuantity != nil && *req.StockQuantity < 0 {
			c.JSON(400, gin.H{"success": false, "message": "Stock quantity cannot be negative"})
			return
		}
		if req.ImageURLs != nil && len(*req.ImageURLs) > 10 {
			c.JSON(400, gin.H{"success": false, "message": "A catalog item can have at most 10 images"})
			return
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argCount := 1

		if req.CategoryID != nil {
			updates = append(updates, fmt.Sprintf("category_id = $%d", argCount))
			args = append(args, req.CategoryID)
			argCount++
		}
		if req.Name != nil {
			updates = append(updates, fmt.Sprintf("name = $%d", argCount))
			args = append(args, *req.Name)
			argCount++
		}
		if req.Description != nil {
			updates = append(updates, fmt.Sprintf("description = $%d", argCount))
			args = append(args, req.Description)
			argCount++
		}
		if req.Price != nil {
			updates = append(updates, fmt.Sprintf("price = $%d", argCount))
			args = append(args, *req.Price)
			argCount++
		}
		if req.CostPrice != nil {
			updates = append(updates, fmt.Sprintf("cost_price = $%d", argCount))
			args = append(args, *req.CostPrice)
			argCount++
		}
		if req.ItemType != nil {
			if *req.ItemType != "product" && *req.ItemType != "service" {
				c.JSON(400, gin.H{"success": false, "message": "Item type must be product or service"})
				return
			}
			updates = append(updates, fmt.Sprintf("item_type = $%d", argCount))
			args = append(args, *req.ItemType)
			argCount++
		}
		if req.PreorderEnabled != nil {
			updates = append(updates, fmt.Sprintf("preorder_enabled = $%d", argCount))
			args = append(args, *req.PreorderEnabled)
			argCount++
		}
		if req.ImageURLs != nil {
			var primaryImage *string
			if len(*req.ImageURLs) > 0 {
				primaryImage = &(*req.ImageURLs)[0]
			}
			updates = append(updates, fmt.Sprintf("image_url = $%d", argCount))
			args = append(args, primaryImage)
			argCount++
		} else if req.ImageURL != nil {
			updates = append(updates, fmt.Sprintf("image_url = $%d", argCount))
			args = append(args, req.ImageURL)
			argCount++
		}
		if req.Barcode != nil {
			updates = append(updates, fmt.Sprintf("barcode = $%d", argCount))
			args = append(args, req.Barcode)
			argCount++
		}
		if req.SKU != nil {
			updates = append(updates, fmt.Sprintf("sku = $%d", argCount))
			args = append(args, req.SKU)
			argCount++
		}
		if req.IsAvailable != nil {
			updates = append(updates, fmt.Sprintf("is_available = $%d", argCount))
			args = append(args, *req.IsAvailable)
			argCount++
		}
		if req.PreparationTime != nil {
			updates = append(updates, fmt.Sprintf("preparation_time = $%d", argCount))
			args = append(args, *req.PreparationTime)
			argCount++
		}
		if req.SortOrder != nil {
			updates = append(updates, fmt.Sprintf("sort_order = $%d", argCount))
			args = append(args, *req.SortOrder)
			argCount++
		}

		if len(updates) == 0 && req.StockQuantity == nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "No fields to update",
			})
			return
		}

		updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, productID)

		query := fmt.Sprintf(`
			UPDATE products 
			SET %s 
			WHERE id = $%d
		`, strings.Join(updates, ", "), argCount)

		tx, err := db.Begin()
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to start product transaction", "error": err.Error()})
			return
		}
		defer tx.Rollback()

		result, err := tx.Exec(query, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to update product",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "Product not found",
			})
			return
		}

		var itemType string
		var costPrice float64
		var currentStock int
		if err := tx.QueryRow(`
			SELECT p.item_type, p.cost_price, COALESCE(i.current_stock, 0)
			FROM products p
			LEFT JOIN inventory i ON i.product_id = p.id
			WHERE p.id = $1`, productID).Scan(&itemType, &costPrice, &currentStock); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to read product inventory", "error": err.Error()})
			return
		}

		if req.StockQuantity != nil {
			currentStock = *req.StockQuantity
		}
		if itemType == "service" {
			currentStock = 0
		}
		if _, err := tx.Exec(`
			INSERT INTO inventory (product_id, current_stock, minimum_stock, maximum_stock, unit_cost)
			VALUES ($1, $2, 0, $2, $3)
			ON CONFLICT (product_id) DO UPDATE
			SET current_stock = EXCLUDED.current_stock, unit_cost = EXCLUDED.unit_cost, updated_at = CURRENT_TIMESTAMP`,
			productID, currentStock, costPrice); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to update product inventory", "error": err.Error()})
			return
		}
		if req.ImageURLs != nil {
			if err := replaceProductImages(tx, productID, *req.ImageURLs); err != nil {
				c.JSON(500, gin.H{"success": false, "message": "Failed to update product images", "error": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(500, gin.H{"success": false, "message": "Failed to save product", "error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Product updated successfully",
		})
	}
}

func replaceProductImages(tx *sql.Tx, productID string, imageURLs []string) error {
	if _, err := tx.Exec(`DELETE FROM product_images WHERE product_id = $1`, productID); err != nil {
		return err
	}
	for index, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO product_images (product_id, image_url, sort_order)
			VALUES ($1, $2, $3)
		`, productID, imageURL, index); err != nil {
			return err
		}
	}
	return nil
}

// Admin handler - Delete product
func deleteProduct(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		productID := c.Param("id")

		// Check if product is used in any active orders
		var orderCount int
		db.QueryRow(`
			SELECT COUNT(*) 
			FROM order_items oi 
			JOIN orders o ON oi.order_id = o.id 
			WHERE oi.product_id = $1 AND o.status NOT IN ('completed', 'cancelled')
		`, productID).Scan(&orderCount)

		if orderCount > 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Cannot delete product with active orders",
				"error":   "product_has_active_orders",
			})
			return
		}

		result, err := db.Exec("DELETE FROM products WHERE id = $1", productID)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to delete product",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "Product not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Product deleted successfully",
		})
	}
}

// Admin handler - Create table
func createTable(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TableNumber     string  `json:"table_number" binding:"required"`
			SeatingCapacity int     `json:"seating_capacity"`
			Location        *string `json:"location"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		var tableID string
		err := db.QueryRow(`
			INSERT INTO dining_tables (table_number, seating_capacity, location)
			VALUES ($1, $2, $3)
			RETURNING id
		`, req.TableNumber, req.SeatingCapacity, req.Location).Scan(&tableID)

		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to create table",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(201, gin.H{
			"success": true,
			"message": "Table created successfully",
			"data":    map[string]interface{}{"id": tableID},
		})
	}
}

// Admin handler - Update table
func updateTable(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("id")

		var req struct {
			TableNumber     *string `json:"table_number"`
			SeatingCapacity *int    `json:"seating_capacity"`
			Location        *string `json:"location"`
			IsOccupied      *bool   `json:"is_occupied"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argCount := 1

		if req.TableNumber != nil {
			updates = append(updates, fmt.Sprintf("table_number = $%d", argCount))
			args = append(args, *req.TableNumber)
			argCount++
		}
		if req.SeatingCapacity != nil {
			updates = append(updates, fmt.Sprintf("seating_capacity = $%d", argCount))
			args = append(args, *req.SeatingCapacity)
			argCount++
		}
		if req.Location != nil {
			updates = append(updates, fmt.Sprintf("location = $%d", argCount))
			args = append(args, req.Location)
			argCount++
		}
		if req.IsOccupied != nil {
			updates = append(updates, fmt.Sprintf("is_occupied = $%d", argCount))
			args = append(args, *req.IsOccupied)
			argCount++
		}

		if len(updates) == 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "No fields to update",
			})
			return
		}

		updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, tableID)

		query := fmt.Sprintf(`
			UPDATE dining_tables 
			SET %s 
			WHERE id = $%d
		`, strings.Join(updates, ", "), argCount)

		result, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to update table",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "Table not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Table updated successfully",
		})
	}
}

// Admin handler - Delete table
func deleteTable(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tableID := c.Param("id")

		// Check if table has active orders
		var orderCount int
		db.QueryRow(`
			SELECT COUNT(*) 
			FROM orders 
			WHERE table_id = $1 AND status NOT IN ('completed', 'cancelled')
		`, tableID).Scan(&orderCount)

		if orderCount > 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Cannot delete table with active orders",
				"error":   "table_has_active_orders",
			})
			return
		}

		result, err := db.Exec("DELETE FROM dining_tables WHERE id = $1", tableID)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to delete table",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "Table not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "Table deleted successfully",
		})
	}
}

// Admin handler - Create user
func createUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username  string `json:"username" binding:"required"`
			Email     string `json:"email" binding:"required"`
			Password  string `json:"password" binding:"required"`
			FirstName string `json:"first_name" binding:"required"`
			LastName  string `json:"last_name" binding:"required"`
			Role      string `json:"role" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to hash password",
				"error":   err.Error(),
			})
			return
		}

		var userID string
		err = db.QueryRow(`
			INSERT INTO users (username, email, password_hash, first_name, last_name, role)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, req.Username, req.Email, string(hashedPassword), req.FirstName, req.LastName, req.Role).Scan(&userID)

		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to create user",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(201, gin.H{
			"success": true,
			"message": "User created successfully",
			"data":    map[string]interface{}{"id": userID},
		})
	}
}

// Admin handler - Update user
func updateUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")

		var req struct {
			Username  *string `json:"username"`
			Email     *string `json:"email"`
			Password  *string `json:"password"`
			FirstName *string `json:"first_name"`
			LastName  *string `json:"last_name"`
			Role      *string `json:"role"`
			IsActive  *bool   `json:"is_active"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Invalid request body",
				"error":   err.Error(),
			})
			return
		}

		// Build dynamic update query
		updates := []string{}
		args := []interface{}{}
		argCount := 1

		if req.Username != nil {
			updates = append(updates, fmt.Sprintf("username = $%d", argCount))
			args = append(args, *req.Username)
			argCount++
		}
		if req.Email != nil {
			updates = append(updates, fmt.Sprintf("email = $%d", argCount))
			args = append(args, *req.Email)
			argCount++
		}
		if req.Password != nil {
			if len(*req.Password) < 6 {
				c.JSON(400, gin.H{
					"success": false,
					"message": "Password must be at least 6 characters",
				})
				return
			}
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to hash password",
					"error":   err.Error(),
				})
				return
			}
			updates = append(updates, fmt.Sprintf("password_hash = $%d", argCount))
			args = append(args, string(hashedPassword))
			argCount++
		}
		if req.FirstName != nil {
			updates = append(updates, fmt.Sprintf("first_name = $%d", argCount))
			args = append(args, *req.FirstName)
			argCount++
		}
		if req.LastName != nil {
			updates = append(updates, fmt.Sprintf("last_name = $%d", argCount))
			args = append(args, *req.LastName)
			argCount++
		}
		if req.Role != nil {
			updates = append(updates, fmt.Sprintf("role = $%d", argCount))
			args = append(args, *req.Role)
			argCount++
		}
		if req.IsActive != nil {
			updates = append(updates, fmt.Sprintf("is_active = $%d", argCount))
			args = append(args, *req.IsActive)
			argCount++
		}

		if len(updates) == 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "No fields to update",
			})
			return
		}

		updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
		args = append(args, userID)

		query := fmt.Sprintf(`
			UPDATE users 
			SET %s 
			WHERE id = $%d
		`, strings.Join(updates, ", "), argCount)

		result, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to update user",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "User not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "User updated successfully",
		})
	}
}

// Admin handler - Delete user
func deleteUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")

		// Prevent deletion if user has associated orders
		var orderCount int
		db.QueryRow("SELECT COUNT(*) FROM orders WHERE user_id = $1", userID).Scan(&orderCount)

		if orderCount > 0 {
			c.JSON(400, gin.H{
				"success": false,
				"message": "Cannot delete user with existing orders",
				"error":   "user_has_orders",
			})
			return
		}

		result, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to delete user",
				"error":   err.Error(),
			})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(404, gin.H{
				"success": false,
				"message": "User not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"message": "User deleted successfully",
		})
	}
}

// Admin handler - Get users with pagination
func getAdminUsers(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		page := 1
		perPage := 20
		role := c.Query("role")
		isActive := c.Query("active")
		search := c.Query("search")

		if pageStr := c.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if perPageStr := c.Query("per_page"); perPageStr != "" {
			if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
				perPage = pp
			}
		}

		offset := (page - 1) * perPage

		// Build query with filters
		queryBuilder := "SELECT id, username, email, first_name, last_name, role, is_active, created_at FROM users WHERE 1=1"
		args := []interface{}{}
		argCount := 0

		if role != "" {
			argCount++
			queryBuilder += fmt.Sprintf(" AND role = $%d", argCount)
			args = append(args, role)
		}

		if isActive != "" {
			argCount++
			queryBuilder += fmt.Sprintf(" AND is_active = $%d", argCount)
			args = append(args, isActive == "true")
		}

		if search != "" {
			argCount++
			queryBuilder += fmt.Sprintf(" AND (LOWER(first_name) LIKE LOWER($%d) OR LOWER(last_name) LIKE LOWER($%d) OR LOWER(username) LIKE LOWER($%d) OR LOWER(email) LIKE LOWER($%d))", argCount, argCount, argCount, argCount)
			args = append(args, "%"+search+"%")
		}

		// Count total records
		countQuery := "SELECT COUNT(*) FROM (" + queryBuilder + ") as count_query"
		var total int
		if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to count users",
				"error":   err.Error(),
			})
			return
		}

		// Add ordering and pagination
		queryBuilder += " ORDER BY created_at DESC"
		argCount++
		queryBuilder += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, perPage)

		argCount++
		queryBuilder += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)

		rows, err := db.Query(queryBuilder, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch users",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var users []map[string]interface{}
		for rows.Next() {
			var user map[string]interface{} = make(map[string]interface{})
			var id, username, email, firstName, lastName, userRole string
			var isActive bool
			var createdAt time.Time

			err := rows.Scan(&id, &username, &email, &firstName, &lastName, &userRole, &isActive, &createdAt)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan user data",
					"error":   err.Error(),
				})
				return
			}

			user["id"] = id
			user["username"] = username
			user["email"] = email
			user["first_name"] = firstName
			user["last_name"] = lastName
			user["role"] = userRole
			user["is_active"] = isActive
			user["created_at"] = createdAt

			users = append(users, user)
		}

		totalPages := (total + perPage - 1) / perPage

		c.JSON(200, gin.H{
			"success": true,
			"message": "Users retrieved successfully",
			"data":    users,
			"meta": gin.H{
				"current_page": page,
				"per_page":     perPage,
				"total":        total,
				"total_pages":  totalPages,
			},
		})
	}
}

// Admin handler - Get categories with pagination
func getAdminCategories(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		page := 1
		perPage := 20
		activeOnly := c.Query("active_only") == "true"
		search := c.Query("search")

		if pageStr := c.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if perPageStr := c.Query("per_page"); perPageStr != "" {
			if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
				perPage = pp
			}
		}

		offset := (page - 1) * perPage

		// Build query with filters
		queryBuilder := "SELECT id, name, description, color, sort_order, is_active, created_at, updated_at FROM categories WHERE 1=1"
		args := []interface{}{}
		argCount := 0

		if activeOnly {
			queryBuilder += " AND is_active = true"
		}

		if search != "" {
			argCount++
			queryBuilder += fmt.Sprintf(" AND (LOWER(name) LIKE LOWER($%d) OR LOWER(COALESCE(description, '')) LIKE LOWER($%d))", argCount, argCount)
			args = append(args, "%"+search+"%")
		}

		// Count total records
		countQuery := "SELECT COUNT(*) FROM (" + queryBuilder + ") as count_query"
		var total int
		if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to count categories",
				"error":   err.Error(),
			})
			return
		}

		// Add ordering and pagination
		queryBuilder += " ORDER BY sort_order ASC, name ASC"
		argCount++
		queryBuilder += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, perPage)

		argCount++
		queryBuilder += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)

		rows, err := db.Query(queryBuilder, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch categories",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var categories []models.Category
		for rows.Next() {
			var category models.Category

			err := rows.Scan(
				&category.ID, &category.Name, &category.Description, &category.Color,
				&category.SortOrder, &category.IsActive, &category.CreatedAt, &category.UpdatedAt,
			)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan category",
					"error":   err.Error(),
				})
				return
			}

			categories = append(categories, category)
		}

		totalPages := (total + perPage - 1) / perPage

		c.JSON(200, gin.H{
			"success": true,
			"message": "Categories retrieved successfully",
			"data":    categories,
			"meta": gin.H{
				"current_page": page,
				"per_page":     perPage,
				"total":        total,
				"total_pages":  totalPages,
			},
		})
	}
}

// Admin handler - Get tables with pagination
func getAdminTables(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse pagination parameters
		page := 1
		perPage := 20
		location := c.Query("location")
		status := c.Query("status") // "occupied", "available", or empty for all
		search := c.Query("search")

		if pageStr := c.Query("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if perPageStr := c.Query("per_page"); perPageStr != "" {
			if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
				perPage = pp
			}
		}

		offset := (page - 1) * perPage

		// Build query with filters
		queryBuilder := `
			SELECT t.id, t.table_number, t.seating_capacity, t.location, t.is_occupied, 
			       t.created_at, t.updated_at,
			       o.id as order_id, o.order_number, o.customer_name, o.status as order_status,
			       o.created_at as order_created_at, o.total_amount
			FROM dining_tables t
			LEFT JOIN orders o ON t.id = o.table_id AND o.status NOT IN ('completed', 'cancelled')
			WHERE 1=1
		`

		args := []interface{}{}
		argCount := 0

		if location != "" {
			argCount++
			queryBuilder += fmt.Sprintf(" AND LOWER(COALESCE(t.location, '')) LIKE LOWER($%d)", argCount)
			args = append(args, "%"+location+"%")
		}

		if status == "occupied" {
			queryBuilder += " AND t.is_occupied = true"
		} else if status == "available" {
			queryBuilder += " AND t.is_occupied = false"
		}

		if search != "" {
			argCount++
			queryBuilder += fmt.Sprintf(" AND (LOWER(t.table_number) LIKE LOWER($%d) OR LOWER(COALESCE(t.location, '')) LIKE LOWER($%d))", argCount, argCount)
			args = append(args, "%"+search+"%")
		}

		// Count total records
		countQuery := "SELECT COUNT(*) FROM (" + queryBuilder + ") as count_query"
		var total int
		if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to count tables",
				"error":   err.Error(),
			})
			return
		}

		// Add ordering and pagination
		queryBuilder += " ORDER BY t.table_number ASC"
		argCount++
		queryBuilder += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, perPage)

		argCount++
		queryBuilder += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)

		rows, err := db.Query(queryBuilder, args...)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"message": "Failed to fetch tables",
				"error":   err.Error(),
			})
			return
		}
		defer rows.Close()

		var tables []map[string]interface{}
		for rows.Next() {
			var table models.DiningTable
			var orderID, orderNumber, customerName, orderStatus sql.NullString
			var orderCreatedAt sql.NullTime
			var totalAmount sql.NullFloat64

			err := rows.Scan(
				&table.ID, &table.TableNumber, &table.SeatingCapacity, &table.Location, &table.IsOccupied,
				&table.CreatedAt, &table.UpdatedAt,
				&orderID, &orderNumber, &customerName, &orderStatus, &orderCreatedAt, &totalAmount,
			)
			if err != nil {
				c.JSON(500, gin.H{
					"success": false,
					"message": "Failed to scan table",
					"error":   err.Error(),
				})
				return
			}

			// Create table data with current order info
			tableData := map[string]interface{}{
				"id":               table.ID,
				"table_number":     table.TableNumber,
				"seating_capacity": table.SeatingCapacity,
				"location":         table.Location,
				"is_occupied":      table.IsOccupied,
				"created_at":       table.CreatedAt,
				"updated_at":       table.UpdatedAt,
				"current_order":    nil,
			}

			// Add current order info if available
			if orderID.Valid {
				tableData["current_order"] = map[string]interface{}{
					"id":            orderID.String,
					"order_number":  orderNumber.String,
					"customer_name": customerName.String,
					"status":        orderStatus.String,
					"created_at":    orderCreatedAt.Time,
					"total_amount":  totalAmount.Float64,
				}
			}

			tables = append(tables, tableData)
		}

		totalPages := (total + perPage - 1) / perPage

		c.JSON(200, gin.H{
			"success": true,
			"message": "Tables retrieved successfully",
			"data":    tables,
			"meta": gin.H{
				"current_page": page,
				"per_page":     perPage,
				"total":        total,
				"total_pages":  totalPages,
			},
		})
	}
}

// Helper function to convert string to pointer
func stringPtr(s string) *string {
	return &s
}
