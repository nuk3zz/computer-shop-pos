package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CustomerHandler struct {
	db *sql.DB
}

type customerRequest struct {
	Name     string  `json:"name"`
	Phone    string  `json:"phone"`
	ImageURL *string `json:"image_url"`
	Email    *string `json:"email"`
	Notes    *string `json:"notes"`
}

func NewCustomerHandler(db *sql.DB) *CustomerHandler {
	return &CustomerHandler{db: db}
}

func (h *CustomerHandler) GetCustomers(c *gin.Context) {
	page := positiveInt(c.Query("page"), 1)
	perPage := positiveInt(c.Query("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	search := strings.TrimSpace(c.Query("search"))
	pattern := "%" + search + "%"

	var total int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM customers WHERE $1 = '' OR name ILIKE $2 OR phone ILIKE $2`, search, pattern).Scan(&total); err != nil {
		customerError(c, "Failed to count clients", err)
		return
	}

	rows, err := h.db.Query(`
		SELECT c.id, c.name, c.phone, c.image_url, c.email, c.notes, c.created_at, c.updated_at,
		       COUNT(o.id),
		       COALESCE(SUM(CASE WHEN o.status <> 'cancelled' THEN o.total_amount ELSE 0 END), 0),
		       MAX(o.created_at)
		FROM customers c
		LEFT JOIN orders o ON o.customer_id = c.id
		WHERE $1 = '' OR c.name ILIKE $2 OR c.phone ILIKE $2
		GROUP BY c.id
		ORDER BY MAX(o.created_at) DESC NULLS LAST, c.name ASC
		LIMIT $3 OFFSET $4`, search, pattern, perPage, (page-1)*perPage)
	if err != nil {
		customerError(c, "Failed to fetch clients", err)
		return
	}
	defer rows.Close()

	customers := make([]models.Customer, 0)
	for rows.Next() {
		customer, err := scanCustomer(rows)
		if err != nil {
			customerError(c, "Failed to read client", err)
			return
		}
		customers = append(customers, customer)
	}

	c.JSON(http.StatusOK, models.PaginatedResponse{
		Success: true,
		Message: "Clients retrieved successfully",
		Data:    customers,
		Meta:    models.MetaData{CurrentPage: page, PerPage: perPage, Total: total, TotalPages: (total + perPage - 1) / perPage},
	})
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid client ID", Error: stringPtr("invalid_uuid")})
		return
	}

	customer, err := h.fetchCustomer(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Client not found", Error: stringPtr("client_not_found")})
			return
		}
		customerError(c, "Failed to fetch client", err)
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Client retrieved successfully", Data: customer})
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid client details", Error: stringPtr(err.Error())})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Name == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Client name and phone are required", Error: stringPtr("missing_client_details")})
		return
	}

	var exists bool
	if err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM customers WHERE phone = $1)", req.Phone).Scan(&exists); err != nil {
		customerError(c, "Failed to validate client phone", err)
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "A client with this phone already exists", Error: stringPtr("phone_exists")})
		return
	}

	var id uuid.UUID
	if err := h.db.QueryRow(`
		INSERT INTO customers (name, phone, image_url, email, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, req.Name, req.Phone, req.ImageURL, req.Email, req.Notes).Scan(&id); err != nil {
		customerError(c, "Failed to create client", err)
		return
	}

	customer, err := h.fetchCustomer(id)
	if err != nil {
		customerError(c, "Client created but could not be reloaded", err)
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Client created successfully", Data: customer})
}

func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid client ID", Error: stringPtr("invalid_uuid")})
		return
	}

	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid client details", Error: stringPtr(err.Error())})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Name == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Client name and phone are required", Error: stringPtr("missing_client_details")})
		return
	}

	var exists bool
	if err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM customers WHERE phone = $1 AND id <> $2)", req.Phone, id).Scan(&exists); err != nil {
		customerError(c, "Failed to validate client phone", err)
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "Another client already uses this phone", Error: stringPtr("phone_exists")})
		return
	}

	result, err := h.db.Exec(`
		UPDATE customers
		SET name = $1, phone = $2, image_url = $3, email = $4, notes = $5
		WHERE id = $6`, req.Name, req.Phone, req.ImageURL, req.Email, req.Notes, id)
	if err != nil {
		customerError(c, "Failed to update client", err)
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Client not found", Error: stringPtr("client_not_found")})
		return
	}

	customer, err := h.fetchCustomer(id)
	if err != nil {
		customerError(c, "Client updated but could not be reloaded", err)
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Client updated successfully", Data: customer})
}

func (h *CustomerHandler) fetchCustomer(id uuid.UUID) (models.Customer, error) {
	row := h.db.QueryRow(`
		SELECT c.id, c.name, c.phone, c.image_url, c.email, c.notes, c.created_at, c.updated_at,
		       COUNT(o.id),
		       COALESCE(SUM(CASE WHEN o.status <> 'cancelled' THEN o.total_amount ELSE 0 END), 0),
		       MAX(o.created_at)
		FROM customers c
		LEFT JOIN orders o ON o.customer_id = c.id
		WHERE c.id = $1
		GROUP BY c.id`, id)

	var customer models.Customer
	err := row.Scan(&customer.ID, &customer.Name, &customer.Phone, &customer.ImageURL, &customer.Email, &customer.Notes,
		&customer.CreatedAt, &customer.UpdatedAt, &customer.OrderCount, &customer.TotalSpent, &customer.LastVisit)
	return customer, err
}

type customerScanner interface {
	Scan(dest ...interface{}) error
}

func scanCustomer(scanner customerScanner) (models.Customer, error) {
	var customer models.Customer
	err := scanner.Scan(&customer.ID, &customer.Name, &customer.Phone, &customer.ImageURL, &customer.Email, &customer.Notes,
		&customer.CreatedAt, &customer.UpdatedAt, &customer.OrderCount, &customer.TotalSpent, &customer.LastVisit)
	return customer, err
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func customerError(c *gin.Context, message string, err error) {
	detail := fmt.Sprintf("%v", err)
	c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: message, Error: &detail})
}
