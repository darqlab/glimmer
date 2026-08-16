#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

DEST="${DEST:-$HOME/.local/bin}"
BIN="$DEST/glimmer"

./build.sh

mkdir -p "$DEST"
cp build/bin/glimmer "$BIN"
chmod +x "$BIN"

echo "==> installed $BIN"
