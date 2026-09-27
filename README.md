# Computer Shop POS

A computer sales, inventory, repair-workflow, and invoicing system built from the open-source [`madebyaris/poinf-of-sales`](https://github.com/madebyaris/poinf-of-sales) project.

> This fork is under active conversion from a restaurant POS. The current development focus is a computer parts and repair shop; the final shop brand has not been chosen yet.

## V1 direction

- Products and services in one visual catalog.
- Uploaded thumbnails for fast item recognition.
- Supplier cost, selling price, stock, and low-stock visibility.
- Sales and repair/work orders containing products, services, or both.
- Owner-editable workflow labels backed by stable internal status keys.
- Revenue, cost, gross-profit, and margin reports.
- Basic printable invoices, with a custom template planned later.

The confirmed scope and staged implementation plan are in the [product brief](knowledge/product-brief.md) and [V1 roadmap](knowledge/v1-roadmap.md).

## Current foundation

- React 18, TypeScript, Vite, Tailwind CSS, and shadcn/ui.
- Go and Gin REST API.
- PostgreSQL.
- JWT authentication and role-based access.
- Docker Compose development and production definitions.

The inherited catalog, order, payment, report, and receipt components are being converted incrementally. Restaurant-specific screens may remain during this transition and should not be treated as final product behavior.

## Mac Mini self-hosting

The production stack is available on the Mac Mini at `http://localhost:3000` and to trusted local-network devices at `http://<mac-mini-ip>:3000`. Only the web entry point is exposed; PostgreSQL and the backend API stay private inside Docker.

See the [self-hosting guide](knowledge/self-hosting.md) for startup, backup, and safety details.

## Development setup

Prerequisites: Docker Desktop with Docker Compose and Make.

```bash
make dev
```

Then open:

- Frontend: `http://localhost:3000`
- Backend health check: `http://localhost:8080/health`
- API base: `http://localhost:8080/api/v1`

Uploaded product images are stored in the ignored local `uploads/` directory and mounted into the backend container. Back up this directory together with PostgreSQL data.

## Project knowledge

- [`CODEX.md`](CODEX.md) - development and knowledge-maintenance rules.
- [`knowledge/`](knowledge/) - product, domain, architecture, and roadmap documentation.
- [`learnings.md`](learnings.md) - dated engineering findings and decisions.
- [`raw/`](raw/) - untouched source material awaiting processing.

## Upstream and license

This repository is a GitHub fork and keeps `madebyaris/poinf-of-sales` configured as the `upstream` Git remote. The original work and this fork are distributed under the [MIT License](LICENSE).
