#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-dev}"
BINARY="${2:-$ROOT_DIR/dist-native/universal-repair-pos}"
OUTPUT_DIR="${3:-$ROOT_DIR/dist-native}"
PACKAGE_VERSION="${VERSION#v}"
PACKAGE_ROOT="$(mktemp -d)"

cleanup() {
  rm -rf "$PACKAGE_ROOT"
}
trap cleanup EXIT

if [[ ! -x "$BINARY" ]]; then
  echo "macOS universal binary not found: $BINARY" >&2
  exit 1
fi

install -d "$PACKAGE_ROOT/usr/local/bin" "$PACKAGE_ROOT/Library/LaunchAgents" "$OUTPUT_DIR"
install -m 0755 "$BINARY" "$PACKAGE_ROOT/usr/local/bin/universal-repair-pos"
install -m 0644 "$ROOT_DIR/installer/macos/io.github.universal-repair-pos.plist" "$PACKAGE_ROOT/Library/LaunchAgents/io.github.universal-repair-pos.plist"
chmod +x "$ROOT_DIR/installer/macos/scripts/postinstall"
xattr -cr "$PACKAGE_ROOT"
find "$PACKAGE_ROOT" -name '._*' -delete

COPYFILE_DISABLE=1 pkgbuild \
  --root "$PACKAGE_ROOT" \
  --scripts "$ROOT_DIR/installer/macos/scripts" \
  --identifier "io.github.universal-repair-pos" \
  --version "$PACKAGE_VERSION" \
  --install-location / \
  "$OUTPUT_DIR/Universal-Repair-POS-$VERSION-macOS-Universal.pkg"

echo "Built $OUTPUT_DIR/Universal-Repair-POS-$VERSION-macOS-Universal.pkg"
