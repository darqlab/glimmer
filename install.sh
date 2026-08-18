#!/usr/bin/env bash
# Downloads and installs the latest (or a specific) glimmer release for
# macOS or Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/darqlab/glimmer/main/install.sh | sh
#   GLIMMER_VERSION=v0.2.0 DEST=/usr/local/bin ./install.sh
#
# Env vars:
#   GLIMMER_VERSION  release tag to install (default: latest)
#   DEST             directory for the `glimmer` command (default: ~/.local/bin)
#   APPDIR           macOS only: where to place glimmer.app (default: ~/Applications)
set -eu

REPO="darqlab/glimmer"
VERSION="${GLIMMER_VERSION:-latest}"
DEST="${DEST:-$HOME/.local/bin}"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
    Linux)
        if [ "$arch" != "x86_64" ]; then
            echo "glimmer: unsupported architecture for Linux: $arch (amd64 only)" >&2
            exit 1
        fi
        asset="glimmer-linux-amd64.tar.gz"
        ;;
    Darwin)
        asset="glimmer-macos-universal.zip"
        ;;
    *)
        echo "glimmer: unsupported OS: $os (use install.ps1 on Windows)" >&2
        exit 1
        ;;
esac

if [ "$VERSION" = "latest" ]; then
    api_url="https://api.github.com/repos/$REPO/releases/latest"
else
    api_url="https://api.github.com/repos/$REPO/releases/tags/$VERSION"
fi

echo "==> resolving $VERSION release for $asset"
download_url="$(
    curl -fsSL "$api_url" \
        | grep '"browser_download_url"' \
        | grep "$asset" \
        | sed -E 's/.*"browser_download_url": *"([^"]+)".*/\1/'
)"

if [ -z "$download_url" ]; then
    echo "glimmer: could not find asset '$asset' in release '$VERSION' of $REPO" >&2
    exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "==> downloading $download_url"
curl -fsSL "$download_url" -o "$tmpdir/$asset"

echo "==> extracting"
case "$asset" in
    *.tar.gz) tar -xzf "$tmpdir/$asset" -C "$tmpdir" ;;
    *.zip)    unzip -q "$tmpdir/$asset" -d "$tmpdir" ;;
esac

mkdir -p "$DEST"

webkit_missing=0
if [ "$os" = "Linux" ]; then
    if ! ldconfig -p 2>/dev/null | grep -q 'libwebkit2gtk-4\.1\.so\.0'; then
        echo "==> libwebkit2gtk-4.1 not found (glimmer's runtime dependency)"
        if command -v apt-get >/dev/null 2>&1; then
            echo "==> installing libwebkit2gtk-4.1-0 via apt (may prompt for sudo password)"
            sudo apt-get update && sudo apt-get install -y libwebkit2gtk-4.1-0 || true
        fi
        if ! ldconfig -p 2>/dev/null | grep -q 'libwebkit2gtk-4\.1\.so\.0'; then
            webkit_missing=1
        fi
    fi
fi

if [ "$os" = "Darwin" ]; then
    app_dest="${APPDIR:-$HOME/Applications}"
    mkdir -p "$app_dest"
    rm -rf "${app_dest:?}/glimmer.app"
    mv "$tmpdir/glimmer.app" "$app_dest/glimmer.app"
    xattr -dr com.apple.quarantine "$app_dest/glimmer.app" 2>/dev/null || true
    ln -sf "$app_dest/glimmer.app/Contents/MacOS/glimmer" "$DEST/glimmer"
    echo "==> installed $app_dest/glimmer.app (command linked at $DEST/glimmer)"
else
    mv "$tmpdir/glimmer" "$DEST/glimmer"
    chmod +x "$DEST/glimmer"
    echo "==> installed $DEST/glimmer"
fi

case ":$PATH:" in
    *":$DEST:"*) ;;
    *) echo "==> note: $DEST is not on your PATH — add it to use the 'glimmer' command directly" ;;
esac

if [ "$webkit_missing" -eq 1 ]; then
    echo "glimmer: WARNING — libwebkit2gtk-4.1 is still missing. glimmer is installed but will fail to launch with:" >&2
    echo "  error while loading shared libraries: libwebkit2gtk-4.1.so.0: cannot open shared object file: No such file or directory" >&2
    echo "  Install it manually, e.g.: sudo apt-get install -y libwebkit2gtk-4.1-0" >&2
    exit 1
fi
