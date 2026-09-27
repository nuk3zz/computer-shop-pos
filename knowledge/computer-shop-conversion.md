# Computer-shop conversion

Sources: [`../raw/2026-09-27-restaurant-ui-screenshot.png`](../raw/2026-09-27-restaurant-ui-screenshot.png) and [`../raw/2026-09-27-commerce-and-repair-workflow-request.md`](../raw/2026-09-27-commerce-and-repair-workflow-request.md)

## Confirmed requirements

- Remove restaurant-facing navigation, language, sample data, and workflows.
- Do not expose tables, seating, guests, servers, kitchen displays, dine-in, takeaway, or food preparation concepts.
- Replace the order-entry screen with a computer-shop sales and services workspace.
- Support both direct product sales and service/repair tickets.
- Show uploaded product or service images as thumbnails in the selling grid.
- Start the catalog with computer-focused categories and examples such as RAM, motherboards, processors, storage, accessories, software installation, cleaning, and diagnostics.
- Replace the kitchen workflow with a repair queue using business-facing labels such as Waiting, Diagnosing, Repairing, Ready for pickup, and Completed.
- Keep the catalog editable so the owner can remove examples and add real stock or services.
- Default money display to LKR, with optional USD, zero tax, and zero service charge.
- Store cost and selling price separately; services default to zero cost.
- Snapshot item cost on order lines so completed-sales reports can calculate gross profit.
- Make service catalog items automatically create repair tickets.
- Capture customer WhatsApp numbers and open status-specific pre-filled WhatsApp messages.
- Allow the owner to edit WhatsApp templates in Settings.
- Provide a compact sales grid with an adjustable tile-size slider.

## Implementation mapping

- Sales & Services replaces Server Interface and Counter/Checkout.
- Repair Tickets replaces Kitchen Display.
- Catalog & Inventory replaces Manage Menu.
- Dining-table management is removed from navigation and API registration.
- Stable existing status keys remain internal for compatibility; the repair queue supplies computer-shop display labels.
- The database reset migration removes restaurant transactions, tables, demo staff, categories, and food items before inserting a clean computer-shop starter catalog.
- Catalog type distinguishes physical products from services; selecting a service automatically changes checkout into a repair ticket.
- Repair status labels map stable internal keys to Waiting, Diagnosing, In progress, Waiting for customer, and Delivered / completed.
- Financial and WhatsApp preferences are saved locally in the browser. LKR and zero charges are the committed defaults.
- A Discord webhook URL can be saved as a placeholder, but automatic webhook delivery is intentionally deferred.

## Current scope boundary

This pass creates usable sales and repair-ticket workflows, basic cost/profit reporting, WhatsApp handoff links, and removes restaurant concepts from the active product. Supplier purchasing, stock-movement accounting, serial-number tracking, server-persisted settings, Discord delivery, and fully configurable workflow labels remain separate V1 milestones documented in the [V1 roadmap](v1-roadmap.md).
