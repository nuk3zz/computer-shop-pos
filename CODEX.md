# Universal Repair POS - AI Development Instructions

## Product direction

This fork is a brand-neutral point-of-sale and work-order system for repair-and-retail businesses. Computer sales and repair remain the first fully tested workflow, while the domain model and product language must stay reusable for phone, electronics, appliance, craft, and other service businesses.

The product name is **Universal Repair POS**. Do not use the name of one shop category as product branding. Each installation's business identity, currency, tax, invoice text, and workflow labels must be configurable rather than hard-coded.

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
- Database: use additive, rerunnable migrations for existing installations. New installations must start without sample business data; seed only the secured bootstrap administrator.
- Money: use fixed-precision database values. Do not use floating-point arithmetic for new financial calculations when correctness matters.
- Inventory: distinguish stock-tracked products from non-stock services. Record cost snapshots on sale lines so historical profit does not change when supplier costs change.
- Workflows: store stable status keys separately from editable display labels. Business owners may rename labels, but code must not depend on display text.
- Invoices: keep presentation configurable and separate from transaction records.
- Security: never commit secrets. Demo credentials must be clearly marked and unsuitable for production.
- Privacy: treat the repository as public. Never publish owner-specific device names, personal example usernames, absolute user paths, private network addresses, or other deployment details; use generic product terminology.

## Verification protocol

- Run formatting, type checks, builds, and focused tests for touched areas.
- For database changes, test a clean initialization and document migration implications.
- Inspect `git diff --check`, `git diff --stat`, and the final diff before committing.
- Do not commit or push unless the user requests it or the active task explicitly includes it.

<!-- maintenance-policy:start -->
## Maintenance cadence (owner approved 2026-10-01)

This section overrides earlier per-session documentation and automatic commit/push/deploy cadence. Preserve project safety, ownership, and verification rules.

- Before coding, read CODEX.md and relevant/recent learnings; search older entries only when needed. Read only knowledge pages relevant to the task.
- Small cosmetic, wording, and routine fixes: implement and verify; do not update every Markdown file or add routine session-log entries.
- Immediately update only affected docs for security, outages, data integrity, breaking contracts, migrations, configuration/deployment changes, or facts needed to operate safely. Keep ownership handoffs accurate immediately.
- Save substantial approved plans in a dedicated knowledge plan page before implementation. Process raw sources needed for the current task immediately; defer unrelated sources to the weekly batch and preserve originals.
- Consolidate noncritical documentation and significant reusable learnings on Friday at 21:00 Asia/Colombo. Use the Git diff and existing task evidence; add a short pending note only when a decision or verification result would otherwise be lost. Skip unchanged projects.
- Routine commits and GitHub pushes wait for that batch. Explicit owner requests to commit, push, release, or deploy override the cadence. Critical fixes go immediately after appropriate validation within existing release authorization; criticality alone does not authorize production access.
- Weekly GitHub sync is owner-authorized for completed, verified, non-secret work. Inspect the full diff and both `git diff --cached --stat` and `git diff --cached --name-status`; isolate intended paths. Never sweep in unfinished, unrelated, or another agent's work. Do not force-push or overwrite changes.
- A push may trigger deployment. Check CI/release rules first; defer pushes that would deploy unfinished work. Routine deployment waits for the batch and still requires project-specific authorization and validation. Leave uncertain changes pending and report the precise blocker.
<!-- maintenance-policy:end -->
