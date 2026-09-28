# iPad and macOS distribution

Source: [`raw/2026-09-28-ipad-web-app-macos-installer-request.md`](../raw/2026-09-28-ipad-web-app-macos-installer-request.md)

## iPad Home Screen app

- The iPad is a client, not the database server.
- Safari can add the POS to the Home Screen with **Open as Web App**, giving it a standalone app window and icon.
- The iPad must reach a running Universal Repair POS server through a trusted local network or an HTTPS deployment.
- Local-network use requires the server installation to use **Same Wi-Fi / LAN** and the iPad to open the server's LAN address. `localhost` on the iPad refers to the iPad itself and will not reach the server.
- The web app does not promise offline business operations because sales, inventory, images, and authentication depend on the server.

## macOS standalone installer

- One universal package contains Apple Silicon and Intel code.
- It installs the same native Go/SQLite server used by Windows and Linux, starts it as a per-user LaunchAgent, and opens the setup page.
- Persistent data remains in `Documents/Universal Repair POS`; installing a newer package does not own or delete that directory.
- Public unsigned builds can be produced without an Apple Developer membership, but Gatekeeper may require manual approval. Normal one-click public distribution requires Developer ID signing and Apple notarization.

## Native iPad boundary

A true iPadOS binary, TestFlight build, or App Store listing requires an Apple-native wrapper/project, signing, device testing, and Apple distribution credentials. It is intentionally deferred because the Home Screen web app covers the current connected POS workflow with far less maintenance.

Related: [standalone distribution](standalone-distribution.md) and [Docker self-hosting](self-hosting.md).
