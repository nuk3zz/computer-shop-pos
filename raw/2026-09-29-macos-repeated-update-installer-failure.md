# macOS repeated update installer failure

## Observed behavior

- The built-in updater downloaded and verified the macOS v0.3.5 package and opened Installer.
- A second **Download & Install** request occurred while Installer was still waiting for administrator approval.
- The second request replaced the package at the same path.
- macOS rejected the now-changed package with “there was no software found to install.”

## Installer evidence

- Installer opened the package before the second updater request.
- The system log reported `Opened package is not the same at install time` immediately before the failure.

## Required behavior

- A repeated update request must reuse an existing package only after verifying its exact size and SHA-256 digest.
- It must never replace a verified package that the operating-system installer may already have open.
- After a successful launch, the interface should describe a later click as reopening the installer rather than downloading it again.
