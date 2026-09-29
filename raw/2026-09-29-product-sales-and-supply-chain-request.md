# Product sales, accounting, and supply-chain request

Observed critical product-sale failure on the installed macOS standalone edition:

- A physical-product order was created, but its stock was not reduced and no payment was recorded.
- Because the order stayed pending, dashboard revenue and profit reports remained zero.
- The entered customer was persisted in the database, but the client screen did not refresh visibly after checkout.
- Product deliveries need separate fulfillment tracking for instant counter sale, pickup, prepaid courier delivery, and cash on delivery.

Requested supply-chain behavior:

- Add a Supply Chain area with multiple suppliers.
- Store supplier name, contact number, location, optional notes, and whether purchasing on credit is permitted.
- Record stock purchases from a supplier, including item quantity, unit cost, total purchase value, and amount paid.
- Increase inventory when a supplier purchase is recorded and retain an auditable purchase record.
- For credit-enabled suppliers, show the outstanding debt and allow later repayments.
- Do not add supplier available-time fields.

Accounting clarification:

- A product bought for LKR 10,000 and sold for LKR 15,000 produces LKR 15,000 sales revenue and LKR 5,000 gross profit.
- The dashboard must show both figures clearly rather than calling gross profit “revenue.”

Catalog-card follow-up:

- Show stock state as small plain text without bordered badges or pills.
- Use green “In stock,” red “Out of stock,” and amber “Pre-order.”
- Pre-order must be an explicit catalog choice rather than inferred from a zero stock count.
