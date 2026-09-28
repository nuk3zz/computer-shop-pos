# Product identity

Source: [`raw/2026-09-28-universal-repair-pos-rename-request.md`](../raw/2026-09-28-universal-repair-pos-rename-request.md)

## Canonical identity

- Product name: **Universal Repair POS**
- Repository slug: `universal-repair-pos`
- Purpose: reusable POS, inventory, customer, sales, service, and repair-ticket management for repair-and-retail businesses.
- Version introducing the identity: `v0.3.1`.

The operator's company name and logo remain installation settings. They must not be replaced by the product name after setup.

## Upgrade compatibility

- New standalone data defaults to `Documents/Universal Repair POS`.
- Installers migrate the former data directory when possible and never deliberately delete business data.
- The native runtime recognizes the former database filename, environment override, and backup format so existing installations remain recoverable.
- Windows retains its installer application identity so the renamed package upgrades the existing installation.
- Linux and macOS packages retire the former service/launcher while preserving and migrating the data directory.

Related: [standalone distribution](standalone-distribution.md), [product brief](product-brief.md), and [architecture](architecture.md).
