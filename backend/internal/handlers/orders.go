package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pos-backend/internal/middleware"
	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	db *sql.DB
}

func NewOrderHandler(db *sql.DB) *OrderHandler {
	return &OrderHandler{db: db}
}

// GetOrders retrieves all orders with pagination and filtering
func (h *OrderHandler) GetOrders(c *gin.Context) {
	// Parse query parameters
	page := 1
	perPage := 20
	status := c.Query("status")
	orderType := c.Query("order_type")
	customerID := c.Query("customer_id")

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
		SELECT DISTINCT o.id, o.order_number, o.table_id, o.user_id, o.customer_id, o.customer_name, o.customer_phone,
		       o.order_type, o.status, o.fulfillment_type, o.fulfillment_status, o.stock_committed,
		       o.subtotal, o.tax_amount, o.discount_amount,
		       o.total_amount, o.notes, o.created_at, o.updated_at, o.served_at, o.completed_at,
		       t.table_number, t.location,
		       u.username, u.first_name, u.last_name
		FROM orders o
		LEFT JOIN dining_tables t ON o.table_id = t.id
		LEFT JOIN users u ON o.user_id = u.id
		WHERE 1=1
	`

	var args []interface{}
	argIndex := 0

	if status != "" {
		argIndex++
		queryBuilder += fmt.Sprintf(" AND o.status = $%d", argIndex)
		args = append(args, status)
	}

	if orderType != "" {
		argIndex++
		queryBuilder += fmt.Sprintf(" AND o.order_type = $%d", argIndex)
		args = append(args, orderType)
	}

	if customerID != "" {
		parsedCustomerID, err := uuid.Parse(customerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid client ID", Error: stringPtr("invalid_uuid")})
			return
		}
		argIndex++
		queryBuilder += fmt.Sprintf(" AND o.customer_id = $%d", argIndex)
		args = append(args, parsedCustomerID)
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM (" + queryBuilder + ") as count_query"
	var total int
	if err := h.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to count orders",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Add ordering and pagination
	argIndex++
	queryBuilder += fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d", argIndex)
	args = append(args, perPage)

	argIndex++
	queryBuilder += fmt.Sprintf(" OFFSET $%d", argIndex)
	args = append(args, offset)

	rows, err := h.db.Query(queryBuilder, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to fetch orders",
			Error:   stringPtr(err.Error()),
		})
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var tableNumber, tableLocation sql.NullString
		var username, firstName, lastName sql.NullString

		err := rows.Scan(
			&order.ID, &order.OrderNumber, &order.TableID, &order.UserID, &order.CustomerID, &order.CustomerName, &order.CustomerPhone,
			&order.OrderType, &order.Status, &order.FulfillmentType, &order.FulfillmentStatus, &order.StockCommitted,
			&order.Subtotal, &order.TaxAmount, &order.DiscountAmount,
			&order.TotalAmount, &order.Notes, &order.CreatedAt, &order.UpdatedAt, &order.ServedAt, &order.CompletedAt,
			&tableNumber, &tableLocation,
			&username, &firstName, &lastName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to scan order",
				Error:   stringPtr(err.Error()),
			})
			return
		}

		// Add table info if available
		if tableNumber.Valid {
			order.Table = &models.DiningTable{
				TableNumber: tableNumber.String,
				Location:    &tableLocation.String,
			}
		}

		// Add user info if available
		if username.Valid {
			order.User = &models.User{
				Username:  username.String,
				FirstName: firstName.String,
				LastName:  lastName.String,
			}
		}

		// Load order items
		if err := h.loadOrderItems(&order); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to load order items",
				Error:   stringPtr(err.Error()),
			})
			return
		}
		if err := h.loadOrderPayments(&order); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load order payments", Error: stringPtr(err.Error())})
			return
		}

		orders = append(orders, order)
	}

	totalPages := (total + perPage - 1) / perPage

	c.JSON(http.StatusOK, models.PaginatedResponse{
		Success: true,
		Message: "Orders retrieved successfully",
		Data:    orders,
		Meta: models.MetaData{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			TotalPages:  totalPages,
		},
	})
}

// GetOrder retrieves a specific order by ID
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid order ID",
			Error:   stringPtr("invalid_uuid"),
		})
		return
	}

	order, err := h.getOrderByID(orderID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Order not found",
			Error:   stringPtr("order_not_found"),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to fetch order",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Order retrieved successfully",
		Data:    order,
	})
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Authentication required",
			Error:   stringPtr("auth_required"),
		})
		return
	}

	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Validate request
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Order must contain at least one item",
			Error:   stringPtr("empty_order"),
		})
		return
	}

	if req.OrderType != "sale" && req.OrderType != "service" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Transaction type must be sale or service",
			Error:   stringPtr("invalid_order_type"),
		})
		return
	}

	fulfillmentType := req.FulfillmentType
	if req.OrderType == "service" {
		fulfillmentType = "service"
	} else if fulfillmentType == "" {
		fulfillmentType = "in_store"
	}
	validFulfillmentTypes := map[string]bool{"in_store": true, "pickup": true, "delivery": true, "cash_on_delivery": true, "service": true}
	if !validFulfillmentTypes[fulfillmentType] || (req.OrderType == "sale" && fulfillmentType == "service") {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose a valid sale method", Error: stringPtr("invalid_fulfillment_type")})
		return
	}
	if req.OrderType == "sale" && (fulfillmentType == "delivery" || fulfillmentType == "cash_on_delivery") {
		if req.CustomerName == nil || req.CustomerPhone == nil || strings.TrimSpace(*req.CustomerName) == "" || strings.TrimSpace(*req.CustomerPhone) == "" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Customer name and phone are required for delivery orders", Error: stringPtr("delivery_contact_required")})
			return
		}
	}
	paymentMethod := "cash"
	if req.PaymentMethod != nil && strings.TrimSpace(*req.PaymentMethod) != "" {
		paymentMethod = strings.TrimSpace(*req.PaymentMethod)
	}
	validPaymentMethods := map[string]bool{"cash": true, "credit_card": true, "debit_card": true, "digital_wallet": true}
	if req.OrderType == "sale" && fulfillmentType != "cash_on_delivery" && !validPaymentMethods[paymentMethod] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose a valid payment method", Error: stringPtr("invalid_payment_method")})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to start transaction",
			Error:   stringPtr(err.Error()),
		})
		return
	}
	defer tx.Rollback()

	// Link the transaction to an existing client, or save a new/re-entered client
	// automatically when both a name and phone number are provided.
	if req.CustomerID != nil {
		var name, phone string
		if err := tx.QueryRow("SELECT name, phone FROM customers WHERE id = $1", *req.CustomerID).Scan(&name, &phone); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Selected client was not found", Error: stringPtr("client_not_found")})
				return
			}
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read selected client", Error: stringPtr(err.Error())})
			return
		}
		req.CustomerName = &name
		req.CustomerPhone = &phone
	} else if req.CustomerName != nil && req.CustomerPhone != nil {
		name := strings.TrimSpace(*req.CustomerName)
		phone := strings.TrimSpace(*req.CustomerPhone)
		if name != "" && phone != "" {
			var customerID uuid.UUID
			err := tx.QueryRow(`
				INSERT INTO customers (name, phone)
				VALUES ($1, $2)
				ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name, updated_at = CURRENT_TIMESTAMP
				RETURNING id`, name, phone).Scan(&customerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save client", Error: stringPtr(err.Error())})
				return
			}
			req.CustomerID = &customerID
			req.CustomerName = &name
			req.CustomerPhone = &phone
		}
	}

	// Generate order number
	orderNumber := h.generateOrderNumber(req.OrderType)

	type pricedItem struct {
		request  models.CreateOrderItem
		price    float64
		cost     float64
		itemType string
	}
	pricedItems := make([]pricedItem, 0, len(req.Items))

	// Calculate totals and lock inventory rows before any stock is committed.
	var subtotalCents int64
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Item quantity must be at least one", Error: stringPtr("invalid_quantity")})
			return
		}
		var price, cost float64
		var itemType string
		err := tx.QueryRow("SELECT price, cost_price, item_type FROM products WHERE id = $1 AND is_available = true", item.ProductID).Scan(&price, &cost, &itemType)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "Product not found or not available",
				Error:   stringPtr("product_not_found"),
			})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to fetch product price",
				Error:   stringPtr(err.Error()),
			})
			return
		}
		if item.SellingPrice != nil {
			value := *item.SellingPrice
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > price || math.Abs(value*100-math.Round(value*100)) > 0.000001 {
				c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Selling price must be between zero and the catalog price, with at most two decimal places", Error: stringPtr("invalid_selling_price")})
				return
			}
			price = float64(int64(math.Round(value*100))) / 100
		}
		if itemType == "product" {
			var stock int
			if err := tx.QueryRow("SELECT current_stock FROM inventory WHERE product_id = $1", item.ProductID).Scan(&stock); err != nil {
				c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Product inventory is missing", Error: stringPtr("inventory_missing")})
				return
			}
			if stock < item.Quantity {
				c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: fmt.Sprintf("Only %d item(s) remain in stock", stock), Error: stringPtr("insufficient_stock")})
				return
			}
		}
		pricedItems = append(pricedItems, pricedItem{request: item, price: price, cost: cost, itemType: itemType})
		subtotalCents += int64(math.Round(price*100)) * int64(item.Quantity)
	}
	subtotal := float64(subtotalCents) / 100

	// Tax and service charge default to zero for the new shop.
	taxRate := 0.0
	taxAmount := subtotal * taxRate
	totalAmount := subtotal + taxAmount

	orderStatus := "pending"
	fulfillmentStatus := "service"
	if req.OrderType == "sale" {
		switch fulfillmentType {
		case "in_store", "pickup":
			orderStatus = "completed"
			fulfillmentStatus = "completed"
		case "delivery":
			orderStatus = "confirmed"
			fulfillmentStatus = "pending_packing"
		case "cash_on_delivery":
			orderStatus = "pending"
			fulfillmentStatus = "pending_packing"
		}
	}

	// Create order
	orderID := uuid.New()
	orderQuery := `
		INSERT INTO orders (id, order_number, table_id, user_id, customer_id, customer_name, customer_phone, order_type, status,
		                   fulfillment_type, fulfillment_status, stock_committed, subtotal, tax_amount, discount_amount, total_amount, notes, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
		        CASE WHEN $9 = 'completed' THEN CURRENT_TIMESTAMP ELSE NULL END)
	`

	_, err = tx.Exec(orderQuery, orderID, orderNumber, req.TableID, userID, req.CustomerID, req.CustomerName, req.CustomerPhone,
		req.OrderType, orderStatus, fulfillmentType, fulfillmentStatus, true, subtotal, taxAmount, 0, totalAmount, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to create order",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Create order items
	for _, priced := range pricedItems {
		item := priced.request
		totalPrice := priced.price * float64(item.Quantity)
		itemID := uuid.New()

		itemQuery := `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, unit_cost, total_price, special_instructions)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		_, err = tx.Exec(itemQuery, itemID, orderID, item.ProductID, item.Quantity, priced.price, priced.cost, totalPrice, item.SpecialInstructions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to create order item",
				Error:   stringPtr(err.Error()),
			})
			return
		}
		if priced.itemType == "product" {
			result, updateErr := tx.Exec(`UPDATE inventory SET current_stock = current_stock - $1, updated_at = CURRENT_TIMESTAMP WHERE product_id = $2 AND current_stock >= $1`, item.Quantity, item.ProductID)
			if updateErr != nil {
				c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to update stock", Error: stringPtr(updateErr.Error())})
				return
			}
			if affected, _ := result.RowsAffected(); affected != 1 {
				c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "Stock changed while the sale was being saved. Please try again.", Error: stringPtr("stock_conflict")})
				return
			}
		}
	}

	// Immediate purchases and prepaid deliveries record the money in the same transaction.
	if req.OrderType == "sale" && fulfillmentType != "cash_on_delivery" {
		paymentID := uuid.New()
		if _, err := tx.Exec(`
			INSERT INTO payments (id, order_id, payment_method, amount, status, processed_by, processed_at)
			VALUES ($1, $2, $3, $4, 'completed', $5, CURRENT_TIMESTAMP)
		`, paymentID, orderID, paymentMethod, totalAmount, userID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to record payment", Error: stringPtr(err.Error())})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to commit transaction",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Fetch and return the created order
	order, err := h.getOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Order created but failed to fetch details",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Order created successfully",
		Data:    order,
	})
}

// UpdateFulfillmentStatus advances a physical-product order independently of payment.
func (h *OrderHandler) UpdateFulfillmentStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid order ID", Error: stringPtr("invalid_uuid")})
		return
	}
	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Authentication required", Error: stringPtr("auth_required")})
		return
	}
	var req models.UpdateFulfillmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid request body", Error: stringPtr(err.Error())})
		return
	}
	validTransitions := map[string]map[string]bool{
		"pending_packing": {"packed": true, "with_courier": true},
		"packed":          {"with_courier": true},
		"with_courier":    {"delivered": true},
	}
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to start transaction", Error: stringPtr(err.Error())})
		return
	}
	defer tx.Rollback()

	var orderType, currentFulfillment string
	var totalAmount float64
	if err := tx.QueryRow(`SELECT order_type, fulfillment_status, total_amount FROM orders WHERE id = $1`, orderID).Scan(&orderType, &currentFulfillment, &totalAmount); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Order not found", Error: stringPtr("order_not_found")})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read product order", Error: stringPtr(err.Error())})
		return
	}
	if orderType != "sale" || !validTransitions[currentFulfillment][req.Status] {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "That product-order status change is not allowed", Error: stringPtr("invalid_fulfillment_transition")})
		return
	}

	nextOrderStatus := "confirmed"
	completedAtSQL := "NULL"
	if req.Status == "delivered" {
		var totalPaid float64
		if err := tx.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM payments WHERE order_id = $1 AND status = 'completed'`, orderID).Scan(&totalPaid); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read payment status", Error: stringPtr(err.Error())})
			return
		}
		if totalPaid >= totalAmount {
			nextOrderStatus = "completed"
			completedAtSQL = "CURRENT_TIMESTAMP"
		}
	}
	if _, err := tx.Exec(`UPDATE orders SET fulfillment_status = $1, status = $2, completed_at = `+completedAtSQL+`, updated_at = CURRENT_TIMESTAMP WHERE id = $3`, req.Status, nextOrderStatus, orderID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to update delivery status", Error: stringPtr(err.Error())})
		return
	}
	if _, err := tx.Exec(`INSERT INTO order_status_history (id, order_id, previous_status, new_status, changed_by, notes) VALUES ($1, $2, $3, $4, $5, $6)`, uuid.New(), orderID, currentFulfillment, req.Status, userID, "Product order fulfillment updated"); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to record delivery history", Error: stringPtr(err.Error())})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save delivery status", Error: stringPtr(err.Error())})
		return
	}
	order, err := h.getOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Delivery status saved but could not be reloaded", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Product order updated", Data: order})
}

// UpdateOrderStatus updates the status of an order
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid order ID",
			Error:   stringPtr("invalid_uuid"),
		})
		return
	}

	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Authentication required",
			Error:   stringPtr("auth_required"),
		})
		return
	}

	var req models.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Validate status
	validStatuses := []string{"pending", "confirmed", "preparing", "ready", "served", "completed", "cancelled"}
	isValidStatus := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid order status",
			Error:   stringPtr("invalid_status"),
		})
		return
	}

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to start transaction",
			Error:   stringPtr(err.Error()),
		})
		return
	}
	defer tx.Rollback()

	// Get current order status
	var currentStatus string
	err = tx.QueryRow("SELECT status FROM orders WHERE id = $1", orderID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Order not found",
			Error:   stringPtr("order_not_found"),
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to fetch current order status",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Update order status
	updateQuery := "UPDATE orders SET status = $1, updated_at = CURRENT_TIMESTAMP"
	args := []interface{}{req.Status, orderID}

	// Set served_at or completed_at timestamps
	if req.Status == "served" {
		updateQuery += ", served_at = CURRENT_TIMESTAMP"
	} else if req.Status == "completed" {
		updateQuery += ", completed_at = CURRENT_TIMESTAMP"
	}

	updateQuery += " WHERE id = $2"

	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to update order status",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Log status change in history
	historyQuery := `
		INSERT INTO order_status_history (order_id, previous_status, new_status, changed_by, notes)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(historyQuery, orderID, currentStatus, req.Status, userID, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to log status change",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to commit transaction",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	// Fetch and return the updated order
	order, err := h.getOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Order updated but failed to fetch details",
			Error:   stringPtr(err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Order status updated successfully",
		Data:    order,
	})
}

// Helper functions

func (h *OrderHandler) getOrderByID(orderID uuid.UUID) (*models.Order, error) {
	var order models.Order
	var tableNumber, tableLocation sql.NullString
	var username, firstName, lastName sql.NullString

	query := `
		SELECT o.id, o.order_number, o.table_id, o.user_id, o.customer_id, o.customer_name, o.customer_phone,
		       o.order_type, o.status, o.fulfillment_type, o.fulfillment_status, o.stock_committed,
		       o.subtotal, o.tax_amount, o.discount_amount,
		       o.total_amount, o.notes, o.created_at, o.updated_at, o.served_at, o.completed_at,
		       t.table_number, t.location,
		       u.username, u.first_name, u.last_name
		FROM orders o
		LEFT JOIN dining_tables t ON o.table_id = t.id
		LEFT JOIN users u ON o.user_id = u.id
		WHERE o.id = $1
	`

	err := h.db.QueryRow(query, orderID).Scan(
		&order.ID, &order.OrderNumber, &order.TableID, &order.UserID, &order.CustomerID, &order.CustomerName, &order.CustomerPhone,
		&order.OrderType, &order.Status, &order.FulfillmentType, &order.FulfillmentStatus, &order.StockCommitted,
		&order.Subtotal, &order.TaxAmount, &order.DiscountAmount,
		&order.TotalAmount, &order.Notes, &order.CreatedAt, &order.UpdatedAt, &order.ServedAt, &order.CompletedAt,
		&tableNumber, &tableLocation,
		&username, &firstName, &lastName,
	)

	if err != nil {
		return nil, err
	}

	// Add table info if available
	if tableNumber.Valid {
		order.Table = &models.DiningTable{
			TableNumber: tableNumber.String,
			Location:    &tableLocation.String,
		}
	}

	// Add user info if available
	if username.Valid {
		order.User = &models.User{
			Username:  username.String,
			FirstName: firstName.String,
			LastName:  lastName.String,
		}
	}

	// Load order items
	if err := h.loadOrderItems(&order); err != nil {
		return nil, err
	}

	// Load payments
	if err := h.loadOrderPayments(&order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (h *OrderHandler) loadOrderItems(order *models.Order) error {
	query := `
		SELECT oi.id, oi.product_id, oi.quantity, oi.unit_price, oi.unit_cost, oi.total_price,
		       oi.special_instructions, oi.status, oi.created_at, oi.updated_at,
		       p.name, p.description, p.price, p.cost_price, p.item_type, p.preparation_time
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		WHERE oi.order_id = $1
		ORDER BY oi.created_at
	`

	rows, err := h.db.Query(query, order.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var productName string
		var productDescription sql.NullString
		var productPrice, productCost float64
		var itemType string
		var preparationTime int

		err := rows.Scan(
			&item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.UnitCost, &item.TotalPrice,
			&item.SpecialInstructions, &item.Status, &item.CreatedAt, &item.UpdatedAt,
			&productName, &productDescription, &productPrice, &productCost, &itemType, &preparationTime,
		)
		if err != nil {
			return err
		}

		item.OrderID = order.ID
		var description *string
		if productDescription.Valid {
			description = &productDescription.String
		}
		item.Product = &models.Product{
			ID:              item.ProductID,
			Name:            productName,
			Description:     description,
			Price:           productPrice,
			CostPrice:       productCost,
			ItemType:        itemType,
			PreparationTime: preparationTime,
		}

		items = append(items, item)
	}

	order.Items = items
	return nil
}

func (h *OrderHandler) loadOrderPayments(order *models.Order) error {
	query := `
		SELECT p.id, p.payment_method, p.amount, p.reference_number, p.status, 
		       p.processed_by, p.processed_at, p.created_at,
		       u.username, u.first_name, u.last_name
		FROM payments p
		LEFT JOIN users u ON p.processed_by = u.id
		WHERE p.order_id = $1
		ORDER BY p.created_at
	`

	rows, err := h.db.Query(query, order.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var payments []models.Payment
	for rows.Next() {
		var payment models.Payment
		var username, firstName, lastName sql.NullString

		err := rows.Scan(
			&payment.ID, &payment.PaymentMethod, &payment.Amount, &payment.ReferenceNumber,
			&payment.Status, &payment.ProcessedBy, &payment.ProcessedAt, &payment.CreatedAt,
			&username, &firstName, &lastName,
		)
		if err != nil {
			return err
		}

		payment.OrderID = order.ID

		// Add processed by user info if available
		if username.Valid {
			payment.ProcessedByUser = &models.User{
				Username:  username.String,
				FirstName: firstName.String,
				LastName:  lastName.String,
			}
		}

		payments = append(payments, payment)
	}

	order.Payments = payments
	return nil
}

func (h *OrderHandler) generateOrderNumber(orderType string) string {
	timestamp := time.Now().Format("20060102")
	prefix := "SAL"
	if orderType == "service" {
		prefix = "REP"
	}
	return fmt.Sprintf("%s%s%04d", prefix, timestamp, time.Now().UnixNano()%10000)
}
