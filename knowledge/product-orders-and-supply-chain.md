# Product orders and supply chain

Sources: [`raw/2026-09-29-product-sales-and-supply-chain-request.md`](../raw/2026-09-29-product-sales-and-supply-chain-request.md), [`raw/2026-09-29-catalog-supplier-link-and-payment-attachments.md`](../raw/2026-09-29-catalog-supplier-link-and-payment-attachments.md), [`raw/2026-09-29-supplier-generated-reference-request.md`](../raw/2026-09-29-supplier-generated-reference-request.md)

## Product-sale transaction rule

A product checkout is atomic: it validates available stock, saves or links the client, snapshots selling price and cost, decrements inventory, creates the fulfillment state, and records payment when money has already been received. If any operation fails, none of them are committed.

Cart lines accept an optional `selling_price` per unit. Omitted/blank uses the catalog price; a value from zero to catalog price with at most two decimal places is snapshotted on the sale. Catalog price and saved acquisition cost are unchanged. Checkout subtotal and payments use the selected price, so existing reports and invoices inherit the actual sale value. Subtotal accumulation uses integer cents.

## Fulfillment and payment

- **Instant sale** and **customer pickup** are paid and completed immediately.
- **Prepaid delivery** records payment immediately and begins at Pending packing. It progresses through Packed, Handed to courier, and Delivered.
- **Cash on delivery** begins unpaid at Pending packing. Fulfillment and payment are independent; the order becomes financially complete only after delivery and payment receipt are both recorded.
- Stock is committed once, when the order is created. Status changes never deduct it a second time.

## Accounting language

- Sales revenue / sales collected: the full amount paid by customers.
- Cost of goods sold: the saved unit-cost snapshots for paid sales.
- Gross profit: sales revenue minus cost of goods sold and tax.
- A LKR 15,000 sale with LKR 10,000 saved cost therefore reports LKR 15,000 sales collected and LKR 5,000 gross profit.

## Suppliers and purchasing

- Suppliers store name, phone, location, notes, and whether purchases on credit are allowed.
- A stock purchase belongs to one supplier and records one or more catalog products, quantities, and unit costs.
- Recording a purchase increases inventory and updates the catalog item's current acquisition cost in one transaction.
- Payments to suppliers are an append-only ledger. An optional amount paid during purchase creation becomes the first ledger payment.
- Outstanding supplier debt equals total purchases minus supplier payments. A purchase with an unpaid balance is rejected unless that supplier permits credit.
- Supplier available-time fields are intentionally excluded.
- Physical catalog creation can optionally create the opening-stock purchase against a selected supplier. In that path the catalog row starts at zero and the supplier purchase adds the requested quantity, preventing double-counted stock.
- Non-credit suppliers force the initial purchase to be paid in full. Credit-enabled suppliers permit full, partial, or zero payment and carry the difference as outstanding debt.
- Purchase and payment records are separate append-only timeline events. Every later debt payment remains visible with its own date and time rather than only changing an aggregate paid total.
- Purchases and payments may each reference an optional PDF or image receipt. These files are stored with the installation uploads and therefore participate in native backup/restore.
- Every purchase or payment without an uploaded document exposes a compact, generated PDF reference from transaction history. It is generated on demand from the ledger rather than stored, so old transactions gain the feature and backups do not accumulate duplicate PDFs.
- A generated reference is explicitly labeled as an internal transaction reference, not a supplier-issued invoice or tax document. Uploaded evidence remains the primary document when present.

## Catalog stock labels

- Physical-product cards use compact text only: green **In stock**, red **Out of stock**, or amber **Pre-order**.
- Positive stock always takes precedence and displays the exact available quantity.
- At zero stock, an explicit `preorder_enabled` catalog setting selects Pre-order; otherwise the item is Out of stock.
- Pre-order is currently a merchandising label. Zero-stock checkout remains disabled so the system cannot record a paid sale for inventory that has not yet arrived.

Related: [domain model](domain-model.md), [clients and fresh-start behavior](clients-and-fresh-start.md), and [repair-shop conversion](repair-shop-conversion.md).
