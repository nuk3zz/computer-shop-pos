# Product brief

## Vision

Create a simple, owner-friendly system for selling computer parts and recording computer repair/service work. It should combine POS checkout, work-order tracking, inventory, cost and profit visibility, reporting, and printable invoices.

The architecture should later support other small businesses without making the computer-shop V1 vague or over-generalized.

## Confirmed V1 users

- Owner/admin: configures the business, catalog, users, workflow labels, reports, and invoice settings.
- Sales/counter staff: sells stocked products and takes payment.
- Technician: sees assigned work, moves jobs through the workflow, and records notes.

## Confirmed V1 capabilities

- Maintain products such as motherboards, processors, cables, and accessories.
- Maintain services such as PC cleaning, Windows installation, driver installation, and game/software setup.
- Upload a small thumbnail/photo for every product or service so staff can identify items quickly in the visual POS grid.
- Record supplier cost, selling price, stock quantity, and profit.
- Create a transaction containing products, services, or both.
- Track repair/service work from intake through completion.
- Allow the owner to edit customer-facing workflow labels without changing stable internal logic.
- View sales and profit reports.
- Print a basic invoice/receipt now; support a custom template later.

## Product principles

- No restaurant terminology in the primary computer-shop experience.
- Products may track stock; services normally do not.
- Duration is optional and supports minutes, hours, or days in the interface.
- Inventory movements must be auditable rather than silently overwriting stock.
- Profit uses the cost captured at the time of sale.
- Business name, currency, tax, invoice text, and workflow labels are settings.
- Item images are optional, show a preview before saving, and fall back to a clear placeholder when absent.

## Open decisions

- Final shop/product brand name.
- Default currency and tax rules.
- Whether customer devices need serial number, password/PIN handling, accessories received, and condition photos in the first repair release.
- Whether one installation will host multiple businesses or each business will run a separate installation. V1 assumes a single business per installation while keeping future multi-business support possible.
