# glimmer

A terminal markdown file reader written in Go. Renders `.md` files with styling,
or as raw text / HTML.

## Usage

```
glimmer [flags] <file.md> [file2.md ...]
```

| Flag | Description |
|------|-------------|
| `-t` | Render as plain text (no styling) |
| `-html` | Render as HTML |
| `-p` | Page output through `$PAGER` (default `less -R`) |
| `-w N` | Terminal wrap width (default 80) |
| `-h` | Show usage |

## Build

Requires Go 1.24+ (glamour dependency).

```bash
go build ./...
```

## Development

```bash
go vet ./...
go build ./...
```
