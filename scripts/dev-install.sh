#!/usr/bin/env bash
# Build glimmer from source and install it for local development/testing.
# For a prebuilt release install, use install.sh (Unix) or install.ps1
# (Windows) from the repo root instead.
set -euo pipefail

cd "$(dirname "$0")/.."

DEST="${DEST:-$HOME/.local/bin}"
BIN="$DEST/glimmer"

./build.sh

mkdir -p "$DEST"
cp build/bin/glimmer "$BIN"
chmod +x "$BIN"

echo "==> installed $BIN"
