package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pos-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-pdf/fpdf"
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

// GetTransactionReferencePDF returns a compact, generated internal reference
// for a supplier purchase or payment. It is generated from the immutable ledger
// entry on demand, so existing transactions receive the feature without adding
// duplicate files to backups.
func (h *SupplyChainHandler) GetTransactionReferencePDF(c *gin.Context) {
	eventType := strings.ToLower(strings.TrimSpace(c.Param("type")))
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil || (eventType != "purchase" && eventType != "payment") {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid supplier transaction"})
		return
	}

	document, err := h.supplierReferenceDocument(eventType, eventID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Supplier transaction not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to generate supplier reference", Error: stringPtr(err.Error())})
		return
	}

	filename := fmt.Sprintf("supplier-%s-%s.pdf", eventType, strings.ToLower(eventID.String()[:8]))
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, "application/pdf", document)
}

type supplierReferenceData struct {
	ShopName         string
	SupplierName     string
	SupplierPhone    string
	SupplierLocation string
	EventType        string
	EventID          string
	OccurredAt       string
	ReferenceNumber  string
	Notes            string
	Amount           float64
	AmountPaid       float64
	Balance          float64
	Items            []supplierReferenceItem
}

type supplierReferenceItem struct {
	Name      string
	Quantity  int
	UnitCost  float64
	LineTotal float64
}

func (h *SupplyChainHandler) supplierReferenceDocument(eventType string, eventID uuid.UUID) ([]byte, error) {
	data := supplierReferenceData{ShopName: "Universal Repair POS", EventType: eventType, EventID: eventID.String()}
	_ = h.db.QueryRow(`SELECT company_name FROM shop_profile WHERE id=1`).Scan(&data.ShopName)

	if eventType == "purchase" {
		var occurredAt any
		var phone, location, reference, notes sql.NullString
		err := h.db.QueryRow(`
			SELECT s.name, s.phone, s.location, p.reference_number, p.total_amount, p.notes, p.purchased_at,
			       COALESCE((SELECT SUM(pay.amount) FROM supplier_payments pay WHERE pay.purchase_id=p.id),0)
			FROM supplier_purchases p JOIN suppliers s ON s.id=p.supplier_id WHERE p.id=$1`, eventID).
			Scan(&data.SupplierName, &phone, &location, &reference, &data.Amount, &notes, &occurredAt, &data.AmountPaid)
		if err != nil {
			return nil, err
		}
		data.SupplierPhone = nullStringValue(phone)
		data.SupplierLocation = nullStringValue(location)
		data.ReferenceNumber = nullStringValue(reference)
		data.Notes = nullStringValue(notes)
		data.OccurredAt = referenceTimestamp(occurredAt)
		data.Balance = data.Amount - data.AmountPaid

		rows, err := h.db.Query(`
			SELECT p.name, i.quantity, i.unit_cost, i.total_cost
			FROM supplier_purchase_items i JOIN products p ON p.id=i.product_id
			WHERE i.purchase_id=$1 ORDER BY p.name`, eventID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var item supplierReferenceItem
			if err := rows.Scan(&item.Name, &item.Quantity, &item.UnitCost, &item.LineTotal); err != nil {
				return nil, err
			}
			data.Items = append(data.Items, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	} else {
		var occurredAt any
		var phone, location, notes sql.NullString
		err := h.db.QueryRow(`
			SELECT s.name, s.phone, s.location, pay.amount, pay.notes, pay.paid_at
			FROM supplier_payments pay JOIN suppliers s ON s.id=pay.supplier_id WHERE pay.id=$1`, eventID).
			Scan(&data.SupplierName, &phone, &location, &data.Amount, &notes, &occurredAt)
		if err != nil {
			return nil, err
		}
		data.SupplierPhone = nullStringValue(phone)
		data.SupplierLocation = nullStringValue(location)
		data.Notes = nullStringValue(notes)
		data.OccurredAt = referenceTimestamp(occurredAt)
	}

	return renderSupplierReferencePDF(data)
}

func renderSupplierReferencePDF(data supplierReferenceData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A5", "")
	pdf.SetCompression(true)
	pdf.SetMargins(13, 12, 13)
	pdf.SetAutoPageBreak(true, 12)
	pdf.AddPage()

	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(0, 8, pdfSafe(data.ShopName), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	title := "Supplier Payment Reference"
	if data.EventType == "purchase" {
		title = "Supplier Purchase Reference"
	}
	pdf.CellFormat(0, 7, title, "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(100, 116, 139)
	pdf.MultiCell(0, 4, "Internal transaction reference - not a supplier-issued invoice or tax document.", "", "L", false)
	pdf.Ln(3)

	pdf.SetDrawColor(203, 213, 225)
	pdf.Line(13, pdf.GetY(), 135, pdf.GetY())
	pdf.Ln(4)
	writeReferenceRow(pdf, "Reference", generatedReference(data.EventType, data.EventID))
	writeReferenceRow(pdf, "Transaction ID", data.EventID)
	writeReferenceRow(pdf, "Date / time", data.OccurredAt)
	writeReferenceRow(pdf, "Supplier", data.SupplierName)
	if data.SupplierPhone != "" {
		writeReferenceRow(pdf, "Contact", data.SupplierPhone)
	}
	if data.SupplierLocation != "" {
		writeReferenceRow(pdf, "Location", data.SupplierLocation)
	}
	if data.ReferenceNumber != "" {
		writeReferenceRow(pdf, "Supplier ref.", data.ReferenceNumber)
	}
	pdf.Ln(3)

	if data.EventType == "purchase" && len(data.Items) > 0 {
		pdf.SetFillColor(241, 245, 249)
		pdf.SetTextColor(15, 23, 42)
		pdf.SetFont("Helvetica", "B", 7.5)
		pdf.CellFormat(60, 7, "Item", "TB", 0, "L", true, 0, "")
		pdf.CellFormat(14, 7, "Qty", "TB", 0, "C", true, 0, "")
		pdf.CellFormat(24, 7, "Unit cost", "TB", 0, "R", true, 0, "")
		pdf.CellFormat(24, 7, "Total", "TB", 1, "R", true, 0, "")
		pdf.SetFont("Helvetica", "", 7.5)
		for _, item := range data.Items {
			pdf.CellFormat(60, 7, truncatePDFText(item.Name, 38), "B", 0, "L", false, 0, "")
			pdf.CellFormat(14, 7, fmt.Sprintf("%d", item.Quantity), "B", 0, "C", false, 0, "")
			pdf.CellFormat(24, 7, formatLKR(item.UnitCost), "B", 0, "R", false, 0, "")
			pdf.CellFormat(24, 7, formatLKR(item.LineTotal), "B", 1, "R", false, 0, "")
		}
		pdf.Ln(3)
	}

	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "B", 9)
	if data.EventType == "purchase" {
		writeMoneyRow(pdf, "Purchase total", data.Amount)
		writeMoneyRow(pdf, "Payments linked to purchase", data.AmountPaid)
		writeMoneyRow(pdf, "Purchase balance", data.Balance)
	} else {
		writeMoneyRow(pdf, "Payment amount", data.Amount)
	}
	if data.Notes != "" {
		pdf.Ln(3)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(0, 5, "Notes", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(0, 4.5, pdfSafe(data.Notes), "", "L", false)
	}

	pdf.SetY(-18)
	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(0, 4, "Generated by Universal Repair POS from the saved supplier ledger.", "", 0, "C", false, 0, "")

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func writeReferenceRow(pdf *fpdf.Fpdf, label, value string) {
	pdf.SetTextColor(100, 116, 139)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(28, 5, label, "", 0, "L", false, 0, "")
	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(0, 5, truncatePDFText(value, 78), "", 1, "L", false, 0, "")
}

func writeMoneyRow(pdf *fpdf.Fpdf, label string, amount float64) {
	pdf.CellFormat(88, 6, label, "", 0, "R", false, 0, "")
	pdf.CellFormat(34, 6, formatLKR(amount), "", 1, "R", false, 0, "")
}

func generatedReference(eventType, id string) string {
	prefix := "SUP-PAY"
	if eventType == "purchase" {
		prefix = "SUP-PUR"
	}
	compact := strings.ToUpper(strings.ReplaceAll(id, "-", ""))
	if len(compact) > 10 {
		compact = compact[:10]
	}
	return prefix + "-" + compact
}

func formatLKR(amount float64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	parts := strings.Split(fmt.Sprintf("%.2f", amount), ".")
	whole := parts[0]
	for index := len(whole) - 3; index > 0; index -= 3 {
		whole = whole[:index] + "," + whole[index:]
	}
	return "LKR " + sign + whole + "." + parts[1]
}

func referenceTimestamp(value any) string {
	switch timestamp := value.(type) {
	case time.Time:
		return timestamp.Local().Format("2006-01-02 03:04 PM")
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, timestamp); err == nil {
				return parsed.Local().Format("2006-01-02 03:04 PM")
			}
		}
		return timestamp
	case []byte:
		return referenceTimestamp(string(timestamp))
	default:
		return fmt.Sprint(value)
	}
}

func nullStringValue(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func pdfSafe(value string) string {
	var result strings.Builder
	for _, character := range strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ") {
		if character >= 32 && character <= 126 {
			result.WriteRune(character)
		} else {
			result.WriteRune('?')
		}
	}
	return result.String()
}

func truncatePDFText(value string, maximum int) string {
	value = pdfSafe(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum-3] + "..."
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
