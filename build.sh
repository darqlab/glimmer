#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

GOBIN_DIR="${GOBIN_DIR:-/usr/local/go/bin}"
GOPATH_BIN="${GOPATH_BIN:-$HOME/go/bin}"
export PATH="$GOBIN_DIR:$GOPATH_BIN:$PATH"

command -v wails >/dev/null 2>&1 || {
    echo "==> wails CLI not found; installing (go install github.com/wailsapp/wails/v2/cmd/wails@latest)"
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
}

echo "==> building glimmer ($(go version))"
wails build -tags webkit2_41

echo "==> built ./build/bin/glimmer"
