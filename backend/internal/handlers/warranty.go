package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"pos-backend/internal/middleware"
	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WarrantyHandler struct{ db *sql.DB }

func NewWarrantyHandler(db *sql.DB) *WarrantyHandler { return &WarrantyHandler{db: db} }

type createWarrantyClaimRequest struct {
	OrderItemID       uuid.UUID `json:"order_item_id" binding:"required"`
	CustomerName      string    `json:"customer_name" binding:"required"`
	CustomerPhone     string    `json:"customer_phone" binding:"required"`
	SerialNumber      *string   `json:"serial_number"`
	IssueDescription  string    `json:"issue_description" binding:"required"`
	ReceivedCondition *string   `json:"received_condition"`
	Notes             *string   `json:"notes"`
}

type updateWarrantyClaimRequest struct {
	Status               string     `json:"status" binding:"required"`
	Resolution           string     `json:"resolution"`
	SupplierStatus       string     `json:"supplier_status"`
	SupplierRecovery     float64    `json:"supplier_recovery_amount"`
	ReplacementSource    string     `json:"replacement_source"`
	ReplacementProductID *uuid.UUID `json:"replacement_product_id"`
	RefundAmount         float64    `json:"refund_amount"`
	Notes                *string    `json:"notes"`
}

var warrantyTransitions = map[string]map[string]bool{
	"received":           {"checking": true, "cancelled": true},
	"checking":           {"awaiting_supplier": true, "ready_for_customer": true, "cancelled": true},
	"awaiting_supplier":  {"ready_for_customer": true, "cancelled": true},
	"ready_for_customer": {"returned": true},
	"returned":           {},
	"cancelled":          {},
}

func (h *WarrantyHandler) GetClaims(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT w.id, w.claim_number, w.order_id, o.order_number, w.order_item_id, w.customer_id,
		       w.customer_name, w.customer_phone, w.product_id, w.product_name, w.serial_number,
		       w.issue_description, w.received_condition, w.status, w.resolution,
		       w.supplier_status, w.supplier_recovery_amount, w.replacement_source,
		       w.replacement_product_id, rp.name, w.replacement_stock_committed, w.replacement_cost, w.refund_amount,
		       w.notes, w.received_at, w.completed_at, w.created_at, w.updated_at
		FROM warranty_claims w
		JOIN orders o ON o.id = w.order_id
		LEFT JOIN products rp ON rp.id = w.replacement_product_id
		ORDER BY CASE WHEN w.status IN ('returned', 'cancelled') THEN 1 ELSE 0 END, w.received_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load warranty returns", Error: stringPtr(err.Error())})
		return
	}
	defer rows.Close()
	claims := make([]gin.H, 0)
	for rows.Next() {
		var id, orderID, orderItemID, productID uuid.UUID
		var customerID, replacementProductID uuid.NullUUID
		var claimNumber, orderNumber, customerName, customerPhone, productName, issue, status, resolution, supplierStatus, replacementSource string
		var serial, condition, replacementName, notes sql.NullString
		var stockCommitted bool
		var supplierRecovery, replacementCost, refund float64
		var receivedAt, completedAt, createdAt, updatedAt any
		if err := rows.Scan(&id, &claimNumber, &orderID, &orderNumber, &orderItemID, &customerID,
			&customerName, &customerPhone, &productID, &productName, &serial, &issue, &condition,
			&status, &resolution, &supplierStatus, &supplierRecovery, &replacementSource,
			&replacementProductID, &replacementName, &stockCommitted, &replacementCost, &refund,
			&notes, &receivedAt, &completedAt, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read warranty return", Error: stringPtr(err.Error())})
			return
		}
		claim := gin.H{"id": id, "claim_number": claimNumber, "order_id": orderID, "order_number": orderNumber,
			"order_item_id": orderItemID, "customer_name": customerName, "customer_phone": customerPhone,
			"product_id": productID, "product_name": productName, "serial_number": nullableString(serial),
			"issue_description": issue, "received_condition": nullableString(condition), "status": status,
			"resolution": resolution, "supplier_status": supplierStatus, "supplier_recovery_amount": supplierRecovery,
			"replacement_source": replacementSource, "replacement_stock_committed": stockCommitted,
			"replacement_cost": replacementCost, "refund_amount": refund,
			"notes": nullableString(notes), "received_at": receivedAt, "completed_at": completedAt,
			"created_at": createdAt, "updated_at": updatedAt}
		if customerID.Valid {
			claim["customer_id"] = customerID.UUID
		}
		if replacementProductID.Valid {
			claim["replacement_product_id"] = replacementProductID.UUID
			claim["replacement_product_name"] = replacementName.String
		}
		claims = append(claims, claim)
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Warranty returns retrieved", Data: claims})
}

func (h *WarrantyHandler) CreateClaim(c *gin.Context) {
	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Authentication required"})
		return
	}
	var req createWarrantyClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose a sold item and enter the customer, phone, and reported issue"})
		return
	}
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.IssueDescription = strings.TrimSpace(req.IssueDescription)
	if req.CustomerName == "" || req.CustomerPhone == "" || req.IssueDescription == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Customer name, contact number, and reported issue are required"})
		return
	}

	var orderID, productID uuid.UUID
	var customerID uuid.NullUUID
	var productName string
	err := h.db.QueryRow(`SELECT oi.order_id, oi.product_id, p.name, o.customer_id FROM order_items oi JOIN orders o ON o.id=oi.order_id JOIN products p ON p.id=oi.product_id WHERE oi.id=$1 AND o.order_type='sale'`, req.OrderItemID).Scan(&orderID, &productID, &productName, &customerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "The selected sold item could not be found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to verify the original sale", Error: stringPtr(err.Error())})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not start warranty return"})
		return
	}
	defer tx.Rollback()
	claimID := uuid.New()
	claimNumber := fmt.Sprintf("WR-%s", strings.ToUpper(strings.ReplaceAll(claimID.String()[:8], "-", "")))
	_, err = tx.Exec(`INSERT INTO warranty_claims (id, claim_number, order_id, order_item_id, customer_id, customer_name, customer_phone, product_id, product_name, serial_number, issue_description, received_condition, notes, created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, claimID, claimNumber, orderID, req.OrderItemID, nullableUUID(customerID), req.CustomerName, req.CustomerPhone, productID, productName, trimOptional(req.SerialNumber), req.IssueDescription, trimOptional(req.ReceivedCondition), trimOptional(req.Notes), userID)
	if err == nil {
		_, err = tx.Exec(`INSERT INTO warranty_status_history (id, claim_id, new_status, resolution, notes, changed_by) VALUES ($1,$2,'received','pending',$3,$4)`, uuid.New(), claimID, "Item received from customer", userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save warranty return", Error: stringPtr(err.Error())})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to finish warranty return"})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Warranty return created", Data: gin.H{"id": claimID, "claim_number": claimNumber}})
}

func (h *WarrantyHandler) UpdateClaim(c *gin.Context) {
	userID, _, _, ok := middleware.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Authentication required"})
		return
	}
	claimID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid warranty return"})
		return
	}
	var req updateWarrantyClaimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose the next warranty status"})
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not update warranty return"})
		return
	}
	defer tx.Rollback()
	var currentStatus, currentResolution, currentSupplierStatus, currentReplacementSource string
	var currentReplacementID uuid.NullUUID
	var stockCommitted bool
	var currentSupplierRecovery, currentReplacementCost, currentRefund float64
	if err := tx.QueryRow(`SELECT status, resolution, supplier_status, supplier_recovery_amount, replacement_source, replacement_product_id, replacement_stock_committed, replacement_cost, refund_amount FROM warranty_claims WHERE id=$1`, claimID).Scan(&currentStatus, &currentResolution, &currentSupplierStatus, &currentSupplierRecovery, &currentReplacementSource, &currentReplacementID, &stockCommitted, &currentReplacementCost, &currentRefund); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Warranty return not found"})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read warranty return", Error: stringPtr(err.Error())})
		}
		return
	}
	if !warrantyTransitions[currentStatus][req.Status] {
		c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "That warranty status change is not allowed"})
		return
	}
	resolution := currentResolution
	supplierStatus := currentSupplierStatus
	supplierRecovery := currentSupplierRecovery
	replacementSource := currentReplacementSource
	replacementProductID := nullableUUID(currentReplacementID)
	replacementCost := currentReplacementCost
	refundAmount := currentRefund
	if req.Status == "awaiting_supplier" {
		supplierStatus = "waiting"
	}
	if req.Status == "ready_for_customer" {
		resolution = strings.TrimSpace(req.Resolution)
		if !map[string]bool{"no_fault_found": true, "replacement": true, "refund": true}[resolution] {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose working/no fault, replacement, or refund before marking ready"})
			return
		}
		if req.SupplierStatus != "" {
			supplierStatus = strings.TrimSpace(req.SupplierStatus)
		}
		if !map[string]bool{"not_sent": true, "waiting": true, "replaced": true, "refunded": true, "rejected": true}[supplierStatus] {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose a valid supplier outcome"})
			return
		}
		if supplierStatus == "waiting" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Record the supplier outcome before marking the return ready"})
			return
		}
		supplierRecovery = req.SupplierRecovery
		if supplierRecovery < 0 || (supplierStatus == "refunded" && supplierRecovery <= 0) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Enter the amount recovered from the supplier"})
			return
		}
		if supplierStatus != "refunded" {
			supplierRecovery = 0
		}
		if resolution == "replacement" {
			replacementSource = strings.TrimSpace(req.ReplacementSource)
			if !map[string]bool{"shop_stock": true, "supplier": true}[replacementSource] {
				c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose whether the replacement comes from shop stock or the supplier"})
				return
			}
			if replacementSource == "supplier" {
				if supplierStatus != "replaced" {
					c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Mark the supplier outcome as replacement received"})
					return
				}
				replacementProductID = nil
				replacementCost = 0
			} else if req.ReplacementProductID == nil {
				c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Choose the replacement catalog item"})
				return
			} else if !stockCommitted {
				result, err := tx.Exec(`UPDATE inventory SET current_stock=current_stock-1, updated_at=CURRENT_TIMESTAMP WHERE product_id=$1 AND current_stock>=1`, *req.ReplacementProductID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Could not reserve replacement stock", Error: stringPtr(err.Error())})
					return
				}
				count, _ := result.RowsAffected()
				if count != 1 {
					c.JSON(http.StatusConflict, models.APIResponse{Success: false, Message: "The selected replacement is out of stock"})
					return
				}
				if err := tx.QueryRow(`SELECT cost_price FROM products WHERE id=$1 AND item_type='product'`, *req.ReplacementProductID).Scan(&replacementCost); err != nil {
					c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "The replacement catalog item could not be found"})
					return
				}
				stockCommitted = true
				replacementProductID = *req.ReplacementProductID
			}
			refundAmount = 0
		}
		if resolution == "refund" && req.RefundAmount <= 0 {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Enter the refund amount"})
			return
		}
		if resolution == "refund" {
			refundAmount = req.RefundAmount
			replacementSource = "none"
			replacementProductID = nil
			replacementCost = 0
		}
		if resolution == "no_fault_found" {
			refundAmount = 0
			replacementSource = "none"
			replacementProductID = nil
			replacementCost = 0
		}
	}
	completedSQL := "NULL"
	if req.Status == "returned" || req.Status == "cancelled" {
		completedSQL = "CURRENT_TIMESTAMP"
	}
	_, err = tx.Exec(`UPDATE warranty_claims SET status=$1, resolution=$2, supplier_status=$3, supplier_recovery_amount=$4, replacement_source=$5, replacement_product_id=$6, replacement_stock_committed=$7, replacement_cost=$8, refund_amount=$9, notes=COALESCE($10,notes), completed_at=`+completedSQL+`, updated_at=CURRENT_TIMESTAMP WHERE id=$11`, req.Status, resolution, supplierStatus, supplierRecovery, replacementSource, replacementProductID, stockCommitted, replacementCost, refundAmount, trimOptional(req.Notes), claimID)
	if err == nil {
		_, err = tx.Exec(`INSERT INTO warranty_status_history (id, claim_id, previous_status, new_status, resolution, notes, changed_by) VALUES ($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), claimID, currentStatus, req.Status, resolution, trimOptional(req.Notes), userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to update warranty return", Error: stringPtr(err.Error())})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to finish warranty update"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Warranty return updated"})
}

func nullableUUID(value uuid.NullUUID) any {
	if value.Valid {
		return value.UUID
	}
	return nil
}
