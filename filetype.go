package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

const (
	maxFileSize = 5 << 20 // 5 MiB = 5,242,880 bytes
	sniffLen    = 8 << 10 // first 8 KiB inspected for binary content
)

// format is one entry in the selection table. Plain struct, no interface
// (ADR-003 Option B).
type format struct {
	name          string // FileResult.Mode: "markdown" | "source" | "text"
	everyLine     bool   // gutter: false = block starts (markdown), true = every source line
	skipSniff     bool   // true only for markdown (DEC-S7)
	filterName    string // Open-dialog filter label; "" = no per-format filter
	filterPattern string // e.g. "*.md;*.markdown"
	match         func(path string, head []byte) bool
	render        func(path string, src []byte) (html, language string, err error)
}

// formats is ordered; first match wins. The plain-text fallback (textFormat)
// is deliberately NOT in the slice, so a format prepended or appended by a
// test is reachable.
var formats = []format{markdownFormat, textByNameFormat, sourceFormat}

var markdownFormat = format{
	name:          "markdown",
	skipSniff:     true,
	filterName:    "Markdown (*.md, *.markdown)",
	filterPattern: "*.md;*.markdown",
	match: func(path string, _ []byte) bool {
		lower := strings.ToLower(path)
		return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
	},
	render: func(_ string, src []byte) (string, string, error) {
		out, err := renderMarkdown(src)
		return out, "Markdown", err
	},
}

// textByNameFormat forces .txt and .log to plain text unless chroma has a
// more specific filename match (for example CMakeLists.txt).
var textByNameFormat = format{
	name:      "text",
	everyLine: true,
	match: func(path string, _ []byte) bool {
		lower := strings.ToLower(filepath.Base(path))
		if !strings.HasSuffix(lower, ".txt") && !strings.HasSuffix(lower, ".log") {
			return false
		}
		return matchByName(path) == nil
	},
	render: renderText,
}

var sourceFormat = format{
	name:      "source",
	everyLine: true,
	match: func(path string, head []byte) bool {
		return sourceLexer(path, head) != nil
	},
	render: func(path string, src []byte) (string, string, error) {
		lexer := sourceLexer(path, src[:min(len(src), sniffLen)])
		if lexer == nil {
			lexer = lexers.Fallback
		}
		language := "Source"
		if lexer != lexers.Fallback {
			language = lexer.Config().Name
		}
		return renderSource(src, lexer, "source"), language, nil
	},
}

// textFormat is the fallback when nothing in formats matches.
var textFormat = format{
	name:      "text",
	everyLine: true,
	render:    renderText,
}

func renderText(_ string, src []byte) (string, string, error) {
	return renderSource(src, lexers.Get("plaintext"), "text"), "Plain text", nil
}

// lexerOverrides corrects observed chroma misdetections. An empty value
// means "unhighlighted source" (lexers.Fallback). Add an entry only with a
// probe showing the misdetection.
var lexerOverrides = map[string]string{
	"go.mod":  "",
	"go.sum":  "",
	"go.work": "",
}

// selectFormat walks formats (first match wins), else textFormat.
func selectFormat(path string, head []byte) format {
	for _, f := range formats {
		if f.match(path, head) {
			return f
		}
	}
	return textFormat
}

// matchByName returns chroma's filename match for path, retrying with the
// lowercased base name. A plaintext result counts as no match.
func matchByName(path string) chroma.Lexer {
	base := filepath.Base(path)
	l := lexers.Match(base)
	if l == nil {
		l = lexers.Match(strings.ToLower(base))
	}
	if l == nil || l.Config().Name == "plaintext" {
		return nil
	}
	return l
}

// sourceLexer returns the lexer for a source file: overrides, then filename
// match, then content analysis. nil when none applies or the result is
// plaintext.
func sourceLexer(path string, head []byte) chroma.Lexer {
	lower := strings.ToLower(filepath.Base(path))
	if name, ok := lexerOverrides[lower]; ok {
		if name == "" {
			return lexers.Fallback
		}
		if l := lexers.Get(name); l != nil {
			return l
		}
	}
	if l := matchByName(path); l != nil {
		return l
	}
	if l := lexers.Analyse(string(head)); l != nil && l.Config().Name != "plaintext" {
		return l
	}
	return nil
}

// refusalError is returned for binary and over-size files. Error() is the
// CLI form; inApp() is the text shown in the window.
type refusalError struct {
	path   string
	binary bool
	size   int64
}

func (e *refusalError) reason() string {
	if e.binary {
		return "binary file"
	}
	return fmt.Sprintf("file too large (%.1f MB; limit is 5 MB)", float64(e.size)/(1<<20))
}

func (e *refusalError) Error() string {
	if e.binary {
		return e.path + ": binary file, not shown. glimmer shows text files only."
	}
	return e.path + ": " + e.reason()
}

func (e *refusalError) inApp() string {
	base := filepath.Base(e.path)
	if e.binary {
		return "Can't show " + base + ": binary file. glimmer shows text files only."
	}
	return "Can't show " + base + ": " + e.reason() + "."
}

// loadSource is the only reader of user files: stat, directory and size
// checks, bounded read, format selection, then the binary sniff unless the
// chosen format skips it.
func loadSource(path string) ([]byte, format, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, format{}, fmt.Errorf("cannot open %s: %w", filepath.Base(path), err)
	}
	if info.IsDir() {
		return nil, format{}, fmt.Errorf("%s is a directory, not a file", path)
	}
	if info.Size() > maxFileSize {
		return nil, format{}, &refusalError{path: path, size: info.Size()}
	}

	fh, err := os.Open(path)
	if err != nil {
		return nil, format{}, fmt.Errorf("cannot open %s: %w", filepath.Base(path), err)
	}
	defer fh.Close()
	src, err := io.ReadAll(io.LimitReader(fh, maxFileSize+1))
	if err != nil {
		return nil, format{}, err
	}
	if len(src) > maxFileSize {
		return nil, format{}, &refusalError{path: path, size: int64(len(src))}
	}

	head := src[:min(len(src), sniffLen)]
	f := selectFormat(path, head)
	if !f.skipSniff && isBinary(head) {
		return nil, format{}, &refusalError{path: path, binary: true}
	}
	return src, f, nil
}

// isBinary reports whether chunk (the first sniffLen bytes of a file, or the
// whole file if shorter) looks binary: a NUL byte, or invalid UTF-8. A rune
// cut by the end of a full-length chunk is not counted as invalid.
func isBinary(chunk []byte) bool {
	for _, b := range chunk {
		if b == 0 {
			return true
		}
	}
	if len(chunk) >= sniffLen {
		for i := 1; i <= utf8.UTFMax-1 && i <= len(chunk); i++ {
			if utf8.RuneStart(chunk[len(chunk)-i]) {
				if !utf8.FullRune(chunk[len(chunk)-i:]) {
					chunk = chunk[:len(chunk)-i]
				}
				break
			}
		}
	}
	return !utf8.Valid(chunk)
}

// preflight is the CLI check run before a window opens. It returns nil when
// the path cannot be stat'ed or is a directory (the window shows that error,
// as before), and otherwise applies the same size and binary checks as
// loadSource, reading at most sniffLen bytes.
func preflight(path string) error {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil
	}
	if info.Size() > maxFileSize {
		return &refusalError{path: path, size: info.Size()}
	}
	fh, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer fh.Close()
	head, err := io.ReadAll(io.LimitReader(fh, sniffLen))
	if err != nil {
		return nil
	}
	if f := selectFormat(path, head); !f.skipSniff && isBinary(head) {
		return &refusalError{path: path, binary: true}
	}
	return nil
}
