# Domain model

See also the [product brief](product-brief.md) and [V1 roadmap](v1-roadmap.md).

## Core entities

### Catalog item

A sellable product or service.

- Stable identity, SKU/barcode, category, description, image, active state.
- `kind`: `product` or `service`.
- Selling price.
- Default supplier/unit cost.
- Optional estimated duration.
- `track_inventory`: normally true for products and false for services.

### Inventory movement

An auditable stock change: supplier receipt, sale, return, correction, or damage. Current stock is derived or transactionally maintained from these movements.

### Customer

Contact and billing details reusable across sales and repair jobs. V1 may allow guest checkout.

### Sale/order

A commercial transaction containing one or more line items. Each line captures selling price and cost snapshots, allowing stable historical gross-profit reporting.

### Work order

A service or repair job linked to a customer and optionally a device. It can contain service lines and product/part lines, estimates, technician notes, due dates, and status history.

### Workflow status

A stable machine key plus owner-editable label, color, order, and terminal-state behavior. Renaming “In progress” to “Repairing” changes presentation, not code or historical records.

### Payment and invoice

Payments settle a sale/work order. An invoice represents the printable commercial document and preserves its number and totals even if the display template changes later.

### Warranty claim

An after-sale product return linked to the original sale line and customer. It tracks intake details, inspection and supplier state, a separate resolution, replacement source/cost, customer refund, supplier recovery, and append-only status history.

## Financial definitions

- Revenue: completed customer sales minus completed customer warranty refunds.
- Cost of goods/services: sum of line-level cost snapshots.
- Gross profit: revenue minus saved sale cost and shop-stock warranty replacement cost, plus supplier cash recoveries.
- Gross margin: gross profit divided by revenue, when revenue is non-zero.
