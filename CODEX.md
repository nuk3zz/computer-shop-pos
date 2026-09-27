# Computer Shop POS - AI Development Instructions

## Product direction

This fork is a brand-neutral point-of-sale and work-order system. The first supported business is a computer sales and repair shop. Keep the domain model reusable enough for another repair or craft business later, but do not add those business-specific workflows until requested.

The temporary product name is **Computer Shop POS**. Do not invent a shop or customer-facing brand name. Business identity, currency, tax, invoice text, and workflow labels must be configurable rather than hard-coded.

## Required knowledge architecture

- `raw/` contains untouched source material, logs, API documentation, screenshots, and feature drops. Do not edit raw inputs in place.
- `knowledge/` contains concise Markdown pages derived from confirmed requirements and inspected code.
- `knowledge/index.md` is the navigation hub. Add or update its links whenever a knowledge page is added.
- `learnings.md` is an append-only session log. Add a dated entry after every coding, debugging, or architecture session.
- Cross-reference related knowledge pages with relative Markdown links.

## Processing new raw inputs

1. Inventory new files in `raw/` and preserve them unchanged.
2. Extract facts, decisions, unknowns, and contradictions into the appropriate `knowledge/` pages.
3. Link the source filename from the derived page.
4. Mark assumptions explicitly. Never turn an unconfirmed idea into a requirement.
5. Update `knowledge/index.md` and append the result to `learnings.md`.

## Engineering conventions

- Read this file and `learnings.md` before editing code.
- Read the relevant `knowledge/` pages before changing business behavior.
- Preserve the upstream MIT license and fork attribution.
- Backend: Go, Gin, PostgreSQL, raw parameterized SQL. Keep handlers thin and transactions atomic.
- Frontend: React and TypeScript. Prefer shared domain types and reusable components; avoid `any` in new code.
- Database: use additive, rerunnable migrations for existing installations. Seed data must demonstrate the computer-shop domain and must not contain real customer data.
- Money: use fixed-precision database values. Do not use floating-point arithmetic for new financial calculations when correctness matters.
- Inventory: distinguish stock-tracked products from non-stock services. Record cost snapshots on sale lines so historical profit does not change when supplier costs change.
- Workflows: store stable status keys separately from editable display labels. Business owners may rename labels, but code must not depend on display text.
- Invoices: keep presentation configurable and separate from transaction records.
- Security: never commit secrets. Demo credentials must be clearly marked and unsuitable for production.

## Verification protocol

- Run formatting, type checks, builds, and focused tests for touched areas.
- For database changes, test a clean initialization and document migration implications.
- Inspect `git diff --check`, `git diff --stat`, and the final diff before committing.
- Do not commit or push unless the user requests it or the active task explicitly includes it.
