#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:?usage: build-linux-deb.sh VERSION BINARY [OUTPUT_DIR]}"
BINARY="${2:?usage: build-linux-deb.sh VERSION BINARY [OUTPUT_DIR]}"
OUTPUT_DIR="${3:-$ROOT_DIR/dist-native}"
PACKAGE_VERSION="${VERSION#v}"
PACKAGE_ROOT="$(mktemp -d)"

cleanup() {
  rm -rf "$PACKAGE_ROOT"
}
trap cleanup EXIT

mkdir -p "$PACKAGE_ROOT/DEBIAN" "$PACKAGE_ROOT/usr/bin" "$PACKAGE_ROOT/usr/share/computer-shop-pos" "$OUTPUT_DIR"
install -m 0755 "$BINARY" "$PACKAGE_ROOT/usr/bin/computer-shop-pos"
install -m 0755 "$ROOT_DIR/installer/linux/computer-shop-pos-status" "$PACKAGE_ROOT/usr/bin/computer-shop-pos-status"
install -m 0644 "$ROOT_DIR/installer/linux/computer-shop-pos.service.in" "$PACKAGE_ROOT/usr/share/computer-shop-pos/computer-shop-pos.service.in"
install -m 0755 "$ROOT_DIR/installer/linux/postinst" "$PACKAGE_ROOT/DEBIAN/postinst"
install -m 0755 "$ROOT_DIR/installer/linux/prerm" "$PACKAGE_ROOT/DEBIAN/prerm"
install -m 0755 "$ROOT_DIR/installer/linux/postrm" "$PACKAGE_ROOT/DEBIAN/postrm"

INSTALLED_SIZE="$(du -sk "$PACKAGE_ROOT/usr" | awk '{print $1}')"
sed -e "s/@VERSION@/$PACKAGE_VERSION/g" -e "s/@INSTALLED_SIZE@/$INSTALLED_SIZE/g" \
  "$ROOT_DIR/installer/linux/control.in" > "$PACKAGE_ROOT/DEBIAN/control"

dpkg-deb --build --root-owner-group "$PACKAGE_ROOT" "$OUTPUT_DIR/computer-shop-pos_${PACKAGE_VERSION}_amd64.deb"
echo "Built $OUTPUT_DIR/computer-shop-pos_${PACKAGE_VERSION}_amd64.deb"
