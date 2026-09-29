package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SupplyChainHandler struct{ db *sql.DB }

func NewSupplyChainHandler(db *sql.DB) *SupplyChainHandler { return &SupplyChainHandler{db: db} }

type supplierInput struct {
	Name          string  `json:"name"`
	Phone         *string `json:"phone"`
	Location      *string `json:"location"`
	Notes         *string `json:"notes"`
	CreditAllowed bool    `json:"credit_allowed"`
}

type supplierPurchaseInput struct {
	SupplierID      uuid.UUID `json:"supplier_id"`
	ReferenceNumber *string   `json:"reference_number"`
	AmountPaid      float64   `json:"amount_paid"`
	Notes           *string   `json:"notes"`
	AttachmentURL   *string   `json:"attachment_url"`
	Items           []struct {
		ProductID uuid.UUID `json:"product_id"`
		Quantity  int       `json:"quantity"`
		UnitCost  float64   `json:"unit_cost"`
	} `json:"items"`
}

func (h *SupplyChainHandler) GetSuppliers(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT s.id, s.name, s.phone, s.location, s.notes, s.credit_allowed, s.created_at, s.updated_at,
		       COALESCE(p.total_purchases, 0), COALESCE(pay.total_paid, 0),
		       COALESCE(p.total_purchases, 0) - COALESCE(pay.total_paid, 0) outstanding_debt
		FROM suppliers s
		LEFT JOIN (SELECT supplier_id, SUM(total_amount) total_purchases FROM supplier_purchases GROUP BY supplier_id) p ON p.supplier_id = s.id
		LEFT JOIN (SELECT supplier_id, SUM(amount) total_paid FROM supplier_payments GROUP BY supplier_id) pay ON pay.supplier_id = s.id
		ORDER BY s.name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load suppliers", Error: stringPtr(err.Error())})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var id, name string
		var phone, location, notes sql.NullString
		var creditAllowed bool
		var createdAt, updatedAt any
		var totalPurchases, totalPaid, debt float64
		if err := rows.Scan(&id, &name, &phone, &location, &notes, &creditAllowed, &createdAt, &updatedAt, &totalPurchases, &totalPaid, &debt); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read supplier", Error: stringPtr(err.Error())})
			return
		}
		result = append(result, gin.H{"id": id, "name": name, "phone": nullableString(phone), "location": nullableString(location), "notes": nullableString(notes), "credit_allowed": creditAllowed, "total_purchases": totalPurchases, "total_paid": totalPaid, "outstanding_debt": debt, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Suppliers retrieved", Data: result})
}

func (h *SupplyChainHandler) CreateSupplier(c *gin.Context) { h.saveSupplier(c, "") }
func (h *SupplyChainHandler) UpdateSupplier(c *gin.Context) { h.saveSupplier(c, c.Param("id")) }

func (h *SupplyChainHandler) saveSupplier(c *gin.Context, id string) {
	var req supplierInput
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Supplier name is required"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) > 150 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Supplier name is too long"})
		return
	}
	if id == "" {
		id = uuid.NewString()
		_, err := h.db.Exec(`INSERT INTO suppliers (id, name, phone, location, notes, credit_allowed) VALUES ($1,$2,$3,$4,$5,$6)`, id, req.Name, trimOptional(req.Phone), trimOptional(req.Location), trimOptional(req.Notes), req.CreditAllowed)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to create supplier", Error: stringPtr(err.Error())})
			return
		}
		c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Supplier created", Data: gin.H{"id": id}})
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid supplier ID"})
		return
	}
	result, err := h.db.Exec(`UPDATE suppliers SET name=$1, phone=$2, location=$3, notes=$4, credit_allowed=$5, updated_at=CURRENT_TIMESTAMP WHERE id=$6`, req.Name, trimOptional(req.Phone), trimOptional(req.Location), trimOptional(req.Notes), req.CreditAllowed, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to update supplier", Error: stringPtr(err.Error())})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Supplier not found"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Supplier updated"})
}

func (h *SupplyChainHandler) GetPurchases(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT p.id, p.supplier_id, s.name, p.reference_number, p.total_amount, p.notes, p.attachment_url, p.purchased_at,
		       COALESCE((SELECT SUM(sp.amount) FROM supplier_payments sp WHERE sp.purchase_id=p.id), 0)
		FROM supplier_purchases p JOIN suppliers s ON s.id=p.supplier_id ORDER BY p.purchased_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load purchases", Error: stringPtr(err.Error())})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var id, supplierID, supplierName string
		var reference, notes, attachmentURL sql.NullString
		var total, paid float64
		var purchasedAt any
		if err := rows.Scan(&id, &supplierID, &supplierName, &reference, &total, &notes, &attachmentURL, &purchasedAt, &paid); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read purchase", Error: stringPtr(err.Error())})
			return
		}
		result = append(result, gin.H{"id": id, "supplier_id": supplierID, "supplier_name": supplierName, "reference_number": nullableString(reference), "total_amount": total, "amount_paid": paid, "balance": total - paid, "notes": nullableString(notes), "attachment_url": nullableString(attachmentURL), "purchased_at": purchasedAt})
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Purchases retrieved", Data: result})
}

func (h *SupplyChainHandler) GetTransactions(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT event_id, supplier_id, supplier_name, event_type, amount, notes, attachment_url, occurred_at
		FROM (
			SELECT p.id event_id, p.supplier_id, s.name supplier_name, 'purchase' event_type,
			       p.total_amount amount, p.notes, p.attachment_url, p.purchased_at occurred_at
			FROM supplier_purchases p JOIN suppliers s ON s.id=p.supplier_id
			UNION ALL
			SELECT pay.id event_id, pay.supplier_id, s.name supplier_name, 'payment' event_type,
			       pay.amount, pay.notes, pay.attachment_url, pay.paid_at occurred_at
			FROM supplier_payments pay JOIN suppliers s ON s.id=pay.supplier_id
		) history ORDER BY occurred_at DESC, event_id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to load supplier history", Error: stringPtr(err.Error())})
		return
	}
	defer rows.Close()
	result := make([]gin.H, 0)
	for rows.Next() {
		var id, supplierID, supplierName, eventType string
		var amount float64
		var notes, attachmentURL sql.NullString
		var occurredAt any
		if err := rows.Scan(&id, &supplierID, &supplierName, &eventType, &amount, &notes, &attachmentURL, &occurredAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to read supplier history", Error: stringPtr(err.Error())})
			return
		}
		result = append(result, gin.H{"id": id, "supplier_id": supplierID, "supplier_name": supplierName, "type": eventType, "amount": amount, "notes": nullableString(notes), "attachment_url": nullableString(attachmentURL), "occurred_at": occurredAt})
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Supplier transaction history retrieved", Data: result})
}

func (h *SupplyChainHandler) CreatePurchase(c *gin.Context) {
	var req supplierPurchaseInput
	if err := c.ShouldBindJSON(&req); err != nil || req.SupplierID == uuid.Nil || len(req.Items) == 0 || req.AmountPaid < 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Supplier, purchase items, and a valid paid amount are required"})
		return
	}
	if !validSupplierAttachment(req.AttachmentURL) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Supplier attachment URL is invalid"})
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to start stock purchase"})
		return
	}
	defer tx.Rollback()
	var creditAllowed bool
	if err := tx.QueryRow(`SELECT credit_allowed FROM suppliers WHERE id=$1`, req.SupplierID).Scan(&creditAllowed); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Supplier not found"})
		return
	}
	var total float64
	for _, item := range req.Items {
		if item.Quantity <= 0 || item.UnitCost < 0 {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Purchase quantities and costs are invalid"})
			return
		}
		var itemType string
		if err := tx.QueryRow(`SELECT item_type FROM products WHERE id=$1`, item.ProductID).Scan(&itemType); err != nil || itemType != "product" {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Every purchase item must be a physical catalog product"})
			return
		}
		total += float64(item.Quantity) * item.UnitCost
	}
	if req.AmountPaid > total {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Amount paid cannot exceed the purchase total"})
		return
	}
	if req.AmountPaid < total && !creditAllowed {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "This supplier requires full payment and does not allow credit"})
		return
	}
	purchaseID := uuid.New()
	if _, err := tx.Exec(`INSERT INTO supplier_purchases (id,supplier_id,reference_number,total_amount,notes,attachment_url) VALUES ($1,$2,$3,$4,$5,$6)`, purchaseID, req.SupplierID, trimOptional(req.ReferenceNumber), total, trimOptional(req.Notes), trimOptional(req.AttachmentURL)); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save purchase", Error: stringPtr(err.Error())})
		return
	}
	for _, item := range req.Items {
		lineTotal := float64(item.Quantity) * item.UnitCost
		if _, err := tx.Exec(`INSERT INTO supplier_purchase_items (id,purchase_id,product_id,quantity,unit_cost,total_cost) VALUES ($1,$2,$3,$4,$5,$6)`, uuid.New(), purchaseID, item.ProductID, item.Quantity, item.UnitCost, lineTotal); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to save purchase item", Error: stringPtr(err.Error())})
			return
		}
		if _, err := tx.Exec(`UPDATE inventory SET current_stock=current_stock+$1, unit_cost=$2, last_restocked_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE product_id=$3`, item.Quantity, item.UnitCost, item.ProductID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to increase inventory", Error: stringPtr(err.Error())})
			return
		}
		if _, err := tx.Exec(`UPDATE products SET cost_price=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`, item.UnitCost, item.ProductID); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to update product cost", Error: stringPtr(err.Error())})
			return
		}
	}
	if req.AmountPaid > 0 {
		if _, err := tx.Exec(`INSERT INTO supplier_payments (id,supplier_id,purchase_id,amount,notes,attachment_url) VALUES ($1,$2,$3,$4,'Payment recorded with stock purchase',$5)`, uuid.New(), req.SupplierID, purchaseID, req.AmountPaid, trimOptional(req.AttachmentURL)); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to record supplier payment", Error: stringPtr(err.Error())})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to commit stock purchase", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Stock purchase recorded", Data: gin.H{"id": purchaseID, "total_amount": total, "amount_paid": req.AmountPaid, "balance": total - req.AmountPaid}})
}

func (h *SupplyChainHandler) CreatePayment(c *gin.Context) {
	supplierID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid supplier ID"})
		return
	}
	var req struct {
		Amount        float64 `json:"amount"`
		Notes         *string `json:"notes"`
		AttachmentURL *string `json:"attachment_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Payment amount must be greater than zero"})
		return
	}
	if !validSupplierAttachment(req.AttachmentURL) {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Supplier attachment URL is invalid"})
		return
	}
	var debt float64
	if err := h.db.QueryRow(`SELECT COALESCE((SELECT SUM(total_amount) FROM supplier_purchases WHERE supplier_id=$1),0)-COALESCE((SELECT SUM(amount) FROM supplier_payments WHERE supplier_id=$1),0)`, supplierID).Scan(&debt); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to calculate supplier debt", Error: stringPtr(err.Error())})
		return
	}
	if debt <= 0 || req.Amount > debt {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Payment cannot exceed the supplier's outstanding debt"})
		return
	}
	if _, err := h.db.Exec(`INSERT INTO supplier_payments (id,supplier_id,amount,notes,attachment_url) VALUES ($1,$2,$3,$4,$5)`, uuid.New(), supplierID, req.Amount, trimOptional(req.Notes), trimOptional(req.AttachmentURL)); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to record supplier payment", Error: stringPtr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Message: "Supplier payment recorded"})
}

func nullableString(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

func trimOptional(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func validSupplierAttachment(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == "" || strings.HasPrefix(strings.TrimSpace(*value), "/uploads/supplier-documents/")
}
