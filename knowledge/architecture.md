# Architecture

## Inherited system

- React 18 + TypeScript + Vite frontend.
- Go + Gin REST API.
- PostgreSQL database initialized from SQL scripts.
- Docker Compose development and production configurations.
- JWT authentication and role-based routes.

## Reusable foundations

- Category and product CRUD.
- Fast product search and cart.
- Order, payment, reporting, and receipt components.
- Admin navigation and responsive data tables.
- Kitchen queue can evolve into a technician/work-order board.

## Restaurant coupling to replace

- Dining tables and dine-in order types.
- Server and kitchen role terminology.
- Menu and food preparation wording.
- Restaurant seed catalog and demo users.
- Fixed kitchen/order statuses and sounds.

## Migration approach

1. Introduce neutral domain fields and compatibility aliases where necessary.
2. Convert seed data and primary UI terminology.
3. Add inventory and profit APIs before presenting profit as authoritative.
4. Evolve the kitchen queue into a technician board backed by configurable workflow definitions.
5. Retire restaurant-only tables/routes after no active screen depends on them.

This incremental approach keeps the fork runnable while the domain is converted.
