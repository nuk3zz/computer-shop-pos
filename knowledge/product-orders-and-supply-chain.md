# Product orders and supply chain

Source: [`raw/2026-09-29-product-sales-and-supply-chain-request.md`](../raw/2026-09-29-product-sales-and-supply-chain-request.md)

## Product-sale transaction rule

A product checkout is atomic: it validates available stock, saves or links the client, snapshots selling price and cost, decrements inventory, creates the fulfillment state, and records payment when money has already been received. If any operation fails, none of them are committed.

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

## Catalog stock labels

- Physical-product cards use compact text only: green **In stock**, red **Out of stock**, or amber **Pre-order**.
- Positive stock always takes precedence and displays the exact available quantity.
- At zero stock, an explicit `preorder_enabled` catalog setting selects Pre-order; otherwise the item is Out of stock.
- Pre-order is currently a merchandising label. Zero-stock checkout remains disabled so the system cannot record a paid sale for inventory that has not yet arrived.

Related: [domain model](domain-model.md), [clients and fresh-start behavior](clients-and-fresh-start.md), and [repair-shop conversion](repair-shop-conversion.md).
