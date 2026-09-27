# V1 roadmap

## Phase 0 - Foundation

- Fork and configure upstream remote.
- Add the standard knowledge architecture.
- Record the computer-shop requirements and conversion risks.
- Verify the inherited project builds before relying on it.

## Phase 1 - Computer-shop catalog

- Rebrand primary UI from restaurant/menu to products and services.
- Add item kind, cost price, stock tracking, current stock, reorder level, and flexible estimated duration.
- Add local product/service image upload, validation, preview, persistent storage, and visual-grid thumbnails.
- Replace food seed data with computer parts and services.
- Provide catalog and stock management in the admin UI.

Acceptance: an owner can create a cable with stock/cost/sale price and an uploaded thumbnail, plus a Windows installation service with a five-hour estimate and optional thumbnail but no stock.

## Phase 2 - Sales, repairs, and workflow

- Add customers and device intake.
- Create work orders containing service and part lines.
- Add stable workflow keys with editable labels/colors.
- Rework the kitchen display into a technician job board.

Acceptance: a repair can move through owner-labelled stages with a full status history.

## Phase 3 - Inventory, profit, and reporting

- Record supplier receipts and all stock movements.
- Snapshot cost on each order line and deduct stock atomically.
- Add revenue, cost, gross profit, margin, low-stock, and item/service performance reports.

Acceptance: stock and gross profit reconcile to completed transaction lines and inventory movements.

## Phase 4 - Invoice and operational hardening

- Add configurable business/invoice settings and a clean print layout.
- Add backups, restore verification, audit logs, production secrets, and permission hardening.
- Test desktop, tablet, receipt printer, and A4 printing.

Acceptance: a completed sale or repair generates a printable invoice and survives backup/restore testing.
