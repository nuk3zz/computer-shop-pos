# Built-in updater and macOS port conflict

User request:

- Replace the GitHub-release redirect with an in-app update flow.
- Download the correct operating-system installer in the background.
- Preserve database, uploads, settings, and backups during every update.
- Avoid making the owner manually locate and download release files.

Observed macOS v0.3.3 failure:

- The package installed successfully and the existing Documents data remained intact.
- The package's LaunchAgent was not loaded after installation.
- An Adobe After Effects ProductionCrate/LaForge CEP extension was listening on IPv4 port 3000.
- The POS could simultaneously listen on IPv6 port 3000, causing `localhost` and `127.0.0.1` to reach different applications.
- Reloading the LaunchAgent and stopping only the conflicting extension host restored the POS; database `PRAGMA quick_check` returned `ok`.

Confirmed updater boundary:

- The POS may download and checksum-verify the release installer and open it locally.
- macOS and Windows still require their normal operating-system authorization/UAC step to replace installed program files.
- Linux headless installations download the verified `.deb` and display the exact `sudo apt install` command because the web service must not silently elevate itself.
