# Warranty returns

Source: [`raw/2026-09-30-warranty-return-workflow-request.md`](../raw/2026-09-30-warranty-return-workflow-request.md)

## Purpose

Warranty Returns is a dedicated after-sale workflow for physical products returned by customers. It remains separate from paid repair tickets because the original sale and warranty resolution are the primary business context.

## Claim intake

- A claim links to one original sale line and snapshots the product name.
- Customer name and WhatsApp/contact number are required and default from the original sale when available.
- Serial number, received condition, and internal notes are optional.
- The reported problem is required.
- Each claim receives a short `WR-` reference.

## Stable workflow

1. `received`: item accepted from the customer.
2. `checking`: item is being inspected.
3. `awaiting_supplier`: the item has been sent to the supplier and the shop is waiting for an outcome.
4. `ready_for_customer`: inspection/supplier handling is complete and a resolution is recorded.
5. `returned`: the original or replacement item has been handed back, or the refund has been paid, to the customer.
6. `cancelled`: intake was cancelled before resolution.

The resolution is stored separately as `pending`, `no_fault_found`, `replacement`, or `refund`. Display labels may change later without changing these machine keys.

## Inventory and money boundaries

- A shop-stock replacement must be selected from positive catalog stock and is deducted exactly once when the claim becomes ready for the customer. Its saved cost is recorded as a warranty expense.
- A supplier-provided replacement does not consume shop inventory and creates no additional warranty expense.
- A returned defective item is not automatically added to sellable inventory.
- A customer refund becomes a negative sales adjustment when the claim is marked returned. It reduces Sales Collected and Gross Profit in the completion period.
- A supplier cash recovery increases Gross Profit as a cost recovery but does not increase Sales Collected.
- The original sold item's cost remains part of cost of goods sold. For example, a product sold for LKR 2,000 with LKR 4,000 saved cost starts at LKR -2,000 gross profit; refunding the LKR 2,000 without supplier recovery makes the final gross profit LKR -4,000.

## Customer contact

The claim card opens WhatsApp with an editable template for checking, ready-for-customer, or returned states. Templates use the same `{customer}` and `{job_number}` placeholders as repair messages; the warranty claim number is supplied as the job number.

Related: [product orders and supply chain](product-orders-and-supply-chain.md), [domain model](domain-model.md), and [repair-shop conversion](repair-shop-conversion.md).
