#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_OS="${1:-$(go env GOOS)}"
TARGET_ARCH="${2:-amd64}"
VERSION="${3:-dev}"
OUTPUT_DIR="${4:-$ROOT_DIR/dist-native}"
WEB_DIR="$ROOT_DIR/backend/web"
PLACEHOLDER="$(mktemp)"

cleanup() {
  find "$WEB_DIR" -mindepth 1 -maxdepth 1 ! -name index.html -exec rm -rf -- {} +
  cp "$PLACEHOLDER" "$WEB_DIR/index.html"
  rm -f "$PLACEHOLDER"
}
trap cleanup EXIT

cp "$WEB_DIR/index.html" "$PLACEHOLDER"
mkdir -p "$OUTPUT_DIR" "$WEB_DIR"

(
  cd "$ROOT_DIR/frontend"
  npm ci
  npm run build
)

find "$WEB_DIR" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
cp -R "$ROOT_DIR/frontend/dist/." "$WEB_DIR/"
find "$WEB_DIR" -type f -name '*.map' -delete

EXTENSION=""
GO_LDFLAGS="-s -w -X main.version=$VERSION -X main.commit=$(git -C "$ROOT_DIR" rev-parse --short HEAD) -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if [[ "$TARGET_OS" == "windows" ]]; then
  EXTENSION=".exe"
  GO_LDFLAGS="$GO_LDFLAGS -H=windowsgui"
fi

(
  cd "$ROOT_DIR/backend"
  CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build -trimpath -ldflags "$GO_LDFLAGS" -o "$OUTPUT_DIR/universal-repair-pos$EXTENSION" .
)

echo "Built $OUTPUT_DIR/universal-repair-pos$EXTENSION"
