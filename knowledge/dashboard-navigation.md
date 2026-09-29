# Dashboard navigation and financial presentation

Source: [`raw/2026-09-29-dashboard-navigation-and-financial-colors.md`](../raw/2026-09-29-dashboard-navigation-and-financial-colors.md)

## Navigation contract

Dashboard controls that visually present as actions must be real keyboard-accessible links:

- Settings -> `/admin/settings`
- Reports and View Reports -> `/admin/reports`
- Catalog & Inventory -> `/admin/catalog`
- Repair Tickets -> `/admin/repairs`
- Manage Staff -> `/admin/staff`

Do not rely on `cursor-pointer` alone. It changes appearance without providing navigation, keyboard behavior, or link semantics.

## Financial color language

- Sales Collected is purple. It represents all customer money received.
- Product / Service Cost is red. It represents the business's cost or expense.
- Gross Profit is green. It represents sales collected minus recorded cost.

These labels remain mathematically distinct; color does not replace the label.
