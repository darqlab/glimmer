# glimmer

A markdown file reader. Opens a native window (built with [Wails](https://wails.io):
Go backend + embedded webview frontend) and renders `.md` files as styled,
responsive HTML — real tables that reflow with the window, dark/light theme,
GitHub-Flavored Markdown (tables, task lists, strikethrough).

## Install

**macOS / Linux:**

```bash
curl -sL https://raw.githubusercontent.com/darqlab/glimmer/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
iwr -useb https://raw.githubusercontent.com/darqlab/glimmer/main/install.ps1 | iex
```

Both scripts fetch the latest [release](https://github.com/darqlab/glimmer/releases)
for your platform and install the `glimmer` command:

- macOS: installs `glimmer.app` to `~/Applications` and links `glimmer` into
  `~/.local/bin` (override with `APPDIR=...` / `DEST=...`)
- Linux: installs the `glimmer` binary to `~/.local/bin` (override with `DEST=...`)
- Windows: installs `glimmer.exe` to `%LOCALAPPDATA%\Programs\glimmer` and adds
  it to your user `PATH` (override with `-Dest`). A double-click NSIS installer
  (`glimmer-amd64-installer.exe`) is also attached to each release if preferred.

Install a specific version instead of latest with `GLIMMER_VERSION=v0.2.0`
(Unix) or `-Version v0.2.0` (Windows).

## Usage

```
glimmer [flags] [file.md]
```

| Flag | Description |
|------|-------------|
| `-h` | Print human usage |
| `-llm-help` | Print machine-readable help for LLM/agent consumption |

With a file argument, opens a window with that file rendered. Without one,
prints `-llm-help` output and exits (use the window's **Open** button to pick
a file interactively). The toolbar's sun/moon button overrides the OS
light/dark preference; the choice is remembered between runs.

### Auto-reload

The open file is watched for changes on disk. Edit it in another program —
your editor, a script, `git checkout` of a different revision — and the
window updates automatically, usually within ~200ms, with no click and no
confirmation banner. Scroll position is preserved across the update. Opening
a different file re-targets the watch to it; the previous file stops
triggering updates. If the file is deleted, the last good render stays on
screen until a file with the same name reappears in that directory.

## Build from source (development)

Requires Go 1.21+, Node.js/npm, and on Linux the GTK/WebKitGTK dev packages:

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev   # Linux only, one-time
./build.sh               # builds ./build/bin/glimmer
./scripts/dev-install.sh # builds and installs to ~/.local/bin/glimmer (DEST=... to override)
```

```bash
wails dev      # live-reloading dev server
wails doctor   # check build dependencies
```

## Releases

Pushing a tag matching `v*.*.*` triggers `.github/workflows/release.yml`,
which builds Linux (amd64), Windows (amd64, plus an NSIS installer) and macOS
(universal) binaries in parallel and publishes them to a GitHub Release.
Every push/PR also runs `.github/workflows/ci.yml` (Linux build + `go vet`).

## Project layout

- `main.go` / `cli.go` / `cli_bindings.go` — CLI entry + arg parsing (split via
  a `bindings` build tag so Wails' internal bindings-generation step, which
  runs the binary with no arguments, isn't intercepted by the help/exit path)
- `app.go` — Go backend: renders markdown to HTML (goldmark + GFM extensions),
  exposes `OpenFile` / `GetInitialFile` / `LoadFile` to the frontend
- `watcher.go` — filesystem watch on the open file's parent directory
  (fsnotify + debounce); pushes a `file:changed` event to the frontend on
  external edits
- `frontend/` — vanilla HTML/CSS/JS UI (toolbar, theme toggle, markdown styles)
- `install.sh` / `install.ps1` — end-user installers (download latest release)
- `build.sh` / `scripts/dev-install.sh` — build-from-source for development
- `.github/workflows/` — CI (build+vet) and release (cross-platform) pipelines
