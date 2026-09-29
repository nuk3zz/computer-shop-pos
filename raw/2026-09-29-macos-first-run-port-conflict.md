# macOS first-run wizard did not appear

Observed after installing the macOS package on a development Mac that already had the Docker edition running:

- Opening `localhost:3000` showed the existing installation's login page instead of the fresh native installation's setup wizard.
- The native SQLite database was new and still had `setup_completed = false`.
- Docker and the native LaunchAgent were both listening on port 3000, so browser requests could reach different editions depending on address resolution.

Expected behavior:

- Only one Universal Repair POS server should own port 3000 on a host.
- After disabling the older Docker stack and restarting the native LaunchAgent, `/setup` should display the first-run wizard.
- Installing a newer macOS package over an existing native installation must preserve the Documents data directory.
