# Clients and fresh-start behavior

Derived from [the owner request](../raw/2026-09-28-fresh-catalog-clients-layout-request.md) and its [sales-grid reference](../raw/2026-09-28-sales-grid-layout-reference.png).

Additional manual-client and conditional-catalog requirements come from [the follow-up request](../raw/2026-09-28-conditional-catalog-and-manual-clients-request.md) and its [form reference](../raw/2026-09-28-catalog-type-stock-reference.png).

## Fresh database rule

New installations contain only the secured bootstrap administrator. Categories, products, services, customers, transactions, payments, inventory rows, dining-table remnants, and non-admin demo users start empty. The live conversion migration applies the same reset after a verified backup.

## Client identity and history

- A client has a stable ID, name, unique phone number, optional email/notes, and timestamps.
- When checkout contains both a name and phone, the backend creates or updates the client by phone and links the transaction to that client.
- Selecting an existing client during checkout sends its stable ID; the backend uses the saved name and phone as the transaction snapshot.
- Product sales and repair/service tickets share the same client history.
- The client directory summarizes transaction count, non-cancelled value, and last visit. Selecting a client loads the linked orders and item details.
- Administrators can also create a client without a sale and edit the name, WhatsApp/contact number, optional email, notes, and optional photo.

## Service duration compatibility

The inherited database column remains `preparation_time` in minutes to avoid a destructive schema rewrite. The catalog form accepts whole days and converts each day to 1,440 minutes before saving. The sales grid converts stored minutes back to a rounded-up day count. Services require at least one day; physical products store zero.

## Type-specific catalog fields

- Actual Cost and Selling Price are shared by products and services. Service cost may represent consumed materials.
- Physical products show Quantity in Stock and hide service duration.
- Services show Estimated Service Duration and hide stock quantity. Service inventory is stored as zero rather than a fake unlimited quantity.
- Product list responses include the linked inventory quantity so create and edit forms show the persisted value.

## Sales-card layout

Cards use a compact 4:3 thumbnail and a fixed content hierarchy: two-line name, price, short description, category, type/duration, then an aligned action row. Card width remains adjustable so staff can choose density for each monitor.

Related: [computer-shop conversion](computer-shop-conversion.md), [domain model](domain-model.md), and [architecture](architecture.md).
