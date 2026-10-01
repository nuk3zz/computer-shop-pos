package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a system user/staff member
type User struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"` // Don't expose password hash in JSON
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	ProfileImageURL *string   `json:"profile_image_url"`
	Role            string    `json:"role"` // admin, manager, sales, technician
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Category represents a product category
type Category struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Color       *string   `json:"color"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Product represents a menu item/product
type Product struct {
	ID              uuid.UUID  `json:"id"`
	CategoryID      *uuid.UUID `json:"category_id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description"`
	Price           float64    `json:"price"`
	CostPrice       float64    `json:"cost_price"`
	ItemType        string     `json:"item_type"` // product, service
	PreorderEnabled bool       `json:"preorder_enabled"`
	ImageURL        *string    `json:"image_url"`
	Images          []string   `json:"images"`
	Barcode         *string    `json:"barcode"`
	SKU             *string    `json:"sku"`
	IsAvailable     bool       `json:"is_available"`
	StockQuantity   int        `json:"stock_quantity"`
	PreparationTime int        `json:"preparation_time"` // optional service duration in minutes
	SortOrder       int        `json:"sort_order"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Category        *Category  `json:"category,omitempty"`
}

// ShopProfile is the persistent identity configured during first-run setup.
type ShopProfile struct {
	ID             int16     `json:"id"`
	CompanyName    string    `json:"company_name"`
	Description    string    `json:"description"`
	LogoURL        *string   `json:"logo_url"`
	SetupCompleted bool      `json:"setup_completed"`
	NetworkMode    string    `json:"network_mode"`
	AutoBackup     bool      `json:"auto_backup"`
	BackupTime     string    `json:"backup_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Customer represents a reusable client record and its sales summary.
type Customer struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Phone      string     `json:"phone"`
	ImageURL   *string    `json:"image_url"`
	Email      *string    `json:"email"`
	Notes      *string    `json:"notes"`
	OrderCount int        `json:"order_count"`
	TotalSpent float64    `json:"total_spent"`
	LastVisit  *time.Time `json:"last_visit"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// DiningTable represents a table or dining area
type DiningTable struct {
	ID              uuid.UUID `json:"id"`
	TableNumber     string    `json:"table_number"`
	SeatingCapacity int       `json:"seating_capacity"`
	Location        *string   `json:"location"`
	IsOccupied      bool      `json:"is_occupied"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Order represents a customer order
type Order struct {
	ID                uuid.UUID    `json:"id"`
	OrderNumber       string       `json:"order_number"`
	TableID           *uuid.UUID   `json:"table_id"`
	UserID            *uuid.UUID   `json:"user_id"`
	CustomerID        *uuid.UUID   `json:"customer_id"`
	CustomerName      *string      `json:"customer_name"`
	CustomerPhone     *string      `json:"customer_phone"`
	OrderType         string       `json:"order_type"` // sale, service
	Status            string       `json:"status"`     // pending, confirmed, preparing, ready, served, completed, cancelled
	FulfillmentType   string       `json:"fulfillment_type"`
	FulfillmentStatus string       `json:"fulfillment_status"`
	StockCommitted    bool         `json:"stock_committed"`
	Subtotal          float64      `json:"subtotal"`
	TaxAmount         float64      `json:"tax_amount"`
	DiscountAmount    float64      `json:"discount_amount"`
	TotalAmount       float64      `json:"total_amount"`
	Notes             *string      `json:"notes"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	ServedAt          *time.Time   `json:"served_at"`
	CompletedAt       *time.Time   `json:"completed_at"`
	Table             *DiningTable `json:"table,omitempty"`
	User              *User        `json:"user,omitempty"`
	Items             []OrderItem  `json:"items,omitempty"`
	Payments          []Payment    `json:"payments,omitempty"`
}

// OrderItem represents an item within an order
type OrderItem struct {
	ID                  uuid.UUID `json:"id"`
	OrderID             uuid.UUID `json:"order_id"`
	ProductID           uuid.UUID `json:"product_id"`
	Quantity            int       `json:"quantity"`
	UnitPrice           float64   `json:"unit_price"`
	UnitCost            float64   `json:"unit_cost"`
	TotalPrice          float64   `json:"total_price"`
	SpecialInstructions *string   `json:"special_instructions"`
	Status              string    `json:"status"` // pending, preparing, ready, served
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Product             *Product  `json:"product,omitempty"`
}

// Payment represents a payment transaction
type Payment struct {
	ID              uuid.UUID  `json:"id"`
	OrderID         uuid.UUID  `json:"order_id"`
	PaymentMethod   string     `json:"payment_method"` // cash, credit_card, debit_card, digital_wallet
	Amount          float64    `json:"amount"`
	ReferenceNumber *string    `json:"reference_number"`
	Status          string     `json:"status"` // pending, completed, failed, refunded
	ProcessedBy     *uuid.UUID `json:"processed_by"`
	ProcessedAt     *time.Time `json:"processed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	ProcessedByUser *User      `json:"processed_by_user,omitempty"`
}

// Inventory represents product inventory
type Inventory struct {
	ID              uuid.UUID  `json:"id"`
	ProductID       uuid.UUID  `json:"product_id"`
	CurrentStock    int        `json:"current_stock"`
	MinimumStock    int        `json:"minimum_stock"`
	MaximumStock    int        `json:"maximum_stock"`
	UnitCost        *float64   `json:"unit_cost"`
	LastRestockedAt *time.Time `json:"last_restocked_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Product         *Product   `json:"product,omitempty"`
}

// OrderStatusHistory tracks order status changes
type OrderStatusHistory struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"order_id"`
	PreviousStatus *string    `json:"previous_status"`
	NewStatus      string     `json:"new_status"`
	ChangedBy      *uuid.UUID `json:"changed_by"`
	Notes          *string    `json:"notes"`
	CreatedAt      time.Time  `json:"created_at"`
	ChangedByUser  *User      `json:"changed_by_user,omitempty"`
}

// Request/Response DTOs

// CreateOrderRequest represents the request to create a new order
type CreateOrderRequest struct {
	TableID         *uuid.UUID        `json:"table_id"`
	CustomerID      *uuid.UUID        `json:"customer_id"`
	CustomerName    *string           `json:"customer_name"`
	CustomerPhone   *string           `json:"customer_phone"`
	OrderType       string            `json:"order_type"`
	FulfillmentType string            `json:"fulfillment_type"`
	PaymentMethod   *string           `json:"payment_method"`
	Items           []CreateOrderItem `json:"items"`
	Notes           *string           `json:"notes"`
}

// UpdateFulfillmentRequest moves a physical-product order through delivery.
type UpdateFulfillmentRequest struct {
	Status string `json:"status"`
}

// CreateOrderItem represents an item in the order creation request
type CreateOrderItem struct {
	ProductID           uuid.UUID `json:"product_id"`
	Quantity            int       `json:"quantity"`
	SellingPrice        *float64  `json:"selling_price"`
	SpecialInstructions *string   `json:"special_instructions"`
}

// UpdateOrderStatusRequest represents the request to update order status
type UpdateOrderStatusRequest struct {
	Status string  `json:"status"`
	Notes  *string `json:"notes"`
}

// ProcessPaymentRequest represents the request to process a payment
type ProcessPaymentRequest struct {
	PaymentMethod   string  `json:"payment_method"`
	Amount          float64 `json:"amount"`
	ReferenceNumber *string `json:"reference_number"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *string     `json:"error,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    MetaData    `json:"meta"`
}

// MetaData represents pagination metadata
type MetaData struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	TotalPages  int `json:"total_pages"`
}
