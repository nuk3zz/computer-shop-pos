# Warranty return workflow request

## Example

A customer returns a product such as a pen drive approximately one week after purchase because it appears not to work.

## Requested workflow

- Add the returned item to the system as a warranty return.
- Track when the item is received and being checked.
- Record whether the item works or is defective.
- If defective, record whether the business provides a replacement or refund.
- Track when the resolved item/replacement is waiting for the customer and when it is returned to the customer.
- Allow the customer to be notified during the workflow.

## Interpretation for the first implementation

- Link the claim to the original sale line so the sold product, sale reference, and client remain auditable.
- Keep stable status and resolution keys separate.
- Replacement inventory is deducted once when the replacement is committed.
- Refund value is recorded on the claim; automatic cash-ledger reversal is outside this first workflow and must not be implied.

## Financial and supplier follow-up

- A returned item may be sent back to its supplier and remain in a waiting state.
- Record whether the supplier supplied a replacement, refunded the shop, or rejected the claim.
- A customer refund must reduce sales collected and gross profit.
- A replacement supplied by the supplier must not consume shop inventory.
- A replacement taken from shop stock must reduce inventory and count its saved cost as a warranty expense.
- Money recovered from a supplier must offset the warranty loss without being labeled as customer sales revenue.
