# V1 roadmap

## Phase 0 - Foundation

- Fork and configure upstream remote.
- Add the standard knowledge architecture.
- Record the repair-and-retail requirements and conversion risks.
- Verify the inherited project builds before relying on it.

## Phase 1 - Repair-shop catalog

- Completed: rebrand primary UI to products and services.
- Completed: add item kind, cost price, selling price, and flexible estimated duration.
- Completed: add local image upload, validation, preview, persistent storage, and selling-grid thumbnails.
- Completed: replace food seed data with computer parts and services.
- Remaining: expose complete stock quantity, reorder level, and restocking controls in the admin UI.

Acceptance: an owner can create a cable with stock/cost/sale price and an uploaded thumbnail, plus a Windows installation service with a five-hour estimate and optional thumbnail but no stock.

## Phase 2 - Sales, repairs, and workflow

- Partially completed: customer name, WhatsApp phone, and device/fault notes are captured on service intake.
- Completed: create work orders containing services and parts.
- Add stable workflow keys with editable labels/colors.
- Completed: replace the kitchen display with Repair Tickets and repair-workflow status labels.
- Completed: add editable WhatsApp templates and pre-filled customer contact links.

Acceptance: a repair can move through owner-labelled stages with a full status history.

## Phase 3 - Inventory, profit, and reporting

- Record supplier receipts and all stock movements.
- Completed: snapshot cost on each order line.
- Completed: add revenue and gross-profit totals to the sales report.
- Remaining: deduct stock atomically and add margin, low-stock, supplier, and item/service performance reports.

Acceptance: stock and gross profit reconcile to completed transaction lines and inventory movements.

## Phase 4 - Invoice and operational hardening

- Add configurable business/invoice settings and a clean print layout.
- Add backups, restore verification, audit logs, production secrets, and permission hardening.
- Test desktop, tablet, receipt printer, and A4 printing.

Acceptance: a completed sale or repair generates a printable invoice and survives backup/restore testing.
