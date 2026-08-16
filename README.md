# glimmer

A markdown file reader. Opens a native window (built with [Wails](https://wails.io):
Go backend + embedded webview frontend) and renders `.md` files as styled,
responsive HTML — real tables that reflow with the window, dark/light theme,
GitHub-Flavored Markdown (tables, task lists, strikethrough).

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

## Build

Requires Go 1.21+, Node.js/npm, and on Linux the GTK/WebKitGTK dev packages:

```bash
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev   # Linux only, one-time
./build.sh     # builds ./build/bin/glimmer
./install.sh   # builds and installs to ~/.local/bin/glimmer (DEST=... to override)
```

## Development

```bash
wails dev      # live-reloading dev server
wails doctor   # check build dependencies
```

## Project layout

- `main.go` / `cli.go` / `cli_bindings.go` — CLI entry + arg parsing (split via
  a `bindings` build tag so Wails' internal bindings-generation step, which
  runs the binary with no arguments, isn't intercepted by the help/exit path)
- `app.go` — Go backend: renders markdown to HTML (goldmark + GFM extensions),
  exposes `OpenFile` / `GetInitialFile` / `LoadFile` to the frontend
- `frontend/` — vanilla HTML/CSS/JS UI (toolbar, theme toggle, markdown styles)
