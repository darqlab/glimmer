# glimmer

A viewer for markdown, source code and text files. Opens a native window (built
with [Wails](https://wails.io): Go backend + embedded webview frontend) and
renders `.md` files as styled, responsive HTML — real tables that reflow with
the window, dark/light theme, GitHub-Flavored Markdown (tables, task lists,
strikethrough). Source and config files are shown with syntax highlighting;
other text files are shown as plain text.

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
glimmer [flags] [file]
```

| Flag | Description |
|------|-------------|
| `-h` | Print human usage |
| `-llm-help` | Print machine-readable help for LLM/agent consumption |

With a file argument, opens a window with that file shown. Without one,
prints `-llm-help` output and exits (use the window's **Open** button to pick
a file interactively). The toolbar's sun/moon button overrides the OS
light/dark preference; the choice is remembered between runs.

### File types

glimmer is read-only and picks a mode per file. The toolbar shows which one.

| Mode | Files | Shown as |
|------|-------|----------|
| Markdown | `.md`, `.markdown` | Rendered GitHub-Flavored Markdown. Fenced code blocks that name a language are syntax-highlighted. |
| Source | Files chroma recognises by name or content: Go, Python, Java, shell, YAML, JSON, XML, TOML, INI, `Dockerfile`, `Makefile`, `.env`, files with a shebang line, and many more | Syntax-highlighted, never wrapped; long lines scroll sideways. |
| Text | `.txt`, `.log`, and any other text file that is not recognised as source | Plain text, wrapped. |

Structured files (JSON, YAML, XML, TOML, INI) are shown exactly as they are on
disk and are never reformatted, so line numbers match the file. A malformed
file still displays, with the bad tokens marked.

Markdown files over 5 MB and source or text files over 256 KB are refused, and so are binary files (a NUL byte or invalid
UTF-8 in the first 8 KB; markdown files skip the binary check). On the command
line this exits with status 1 and a message on stderr before any window opens.
From the **Open** button, the message appears in a notice bar and the current
document stays on screen.

### Source line numbers

The toolbar's `#️⃣` button toggles a left-hand gutter. For markdown it shows
the source `.md` file's line number for each top-level block (heading,
paragraph, code fence, table, etc); numbers mark block starts, not every
wrapped visual line. For source and text files every line is numbered, and a
wrapped line has one number. Off by default; the choice is remembered between
runs.

### Copy on selection

The toolbar's `📋` button toggles automatic clipboard copy of any text you
select in the reading area with the mouse, with a brief toast confirming the
copy. Since the window is read-only, selecting text has no other purpose. On
by default; the choice is remembered between runs. (Keyboard selection still
works with Ctrl+C as usual — this toggle only affects mouse selection.)

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
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev   # Linux only, one-time
./build.sh               # builds ./build/bin/glimmer
./scripts/dev-install.sh # builds and installs to ~/.local/bin/glimmer (DEST=... to override)
```

```bash
wails dev -tags webkit2_41   # live-reloading dev server
wails doctor                 # check build dependencies
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
  exposes `OpenFile` / `GetInitialFile` / `GetHighlightCSS` to the frontend
- `filetype.go` — file loading and limits (5 MB markdown / 256 KB source and text caps, binary check), the ordered
  `formats` table that picks markdown, source or text for a file, and the CLI
  pre-flight check
- `highlight.go` — chroma rendering of source and text files, and the
  light/dark highlight stylesheet
- `watcher.go` — filesystem watch on the open file's parent directory
  (fsnotify + debounce); pushes a `file:changed` event to the frontend on
  external edits
- `frontend/` — vanilla HTML/CSS/JS UI (toolbar, theme toggle, markdown styles)
- `install.sh` / `install.ps1` — end-user installers (download latest release)
- `build.sh` / `scripts/dev-install.sh` — build-from-source for development
- `.github/workflows/` — CI (build+vet) and release (cross-platform) pipelines
