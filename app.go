package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// App holds the running Wails application state.
type App struct {
	ctx     context.Context
	initial string // file path passed on the command line, if any
	watcher *fileWatcher
}

// NewApp creates a new App application struct.
func NewApp(initial string) *App {
	return &App{initial: initial}
}

// startup is called at application startup.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.watcher = newFileWatcher(ctx)
}

// shutdown is called when the application is closing.
func (a *App) shutdown(ctx context.Context) {
	a.watcher.close()
}

// FileResult is returned to the frontend after loading and rendering a
// markdown file.
type FileResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	HTML string `json:"html"`
}

var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

func renderFile(path string) (FileResult, error) {
	if !strings.HasSuffix(strings.ToLower(path), ".md") &&
		!strings.HasSuffix(strings.ToLower(path), ".markdown") {
		return FileResult{}, fmt.Errorf("%s: not a .md file", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return FileResult{}, fmt.Errorf("cannot open %s: %w", filepath.Base(path), err)
	}
	if info.IsDir() {
		return FileResult{}, fmt.Errorf("%s is a directory, not a file", path)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, err
	}

	var buf strings.Builder
	doc := markdown.Parser().Parse(text.NewReader(src))
	lineStarts := buildLineIndex(src)
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		ln := lineOf(lineStarts, firstOffset(n))
		fmt.Fprintf(&buf, `<div class="md-block" data-line="%d">`, ln)
		if err := markdown.Renderer().Render(&buf, src, n); err != nil {
			return FileResult{}, err
		}
		buf.WriteString("</div>")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	return FileResult{
		Path: abs,
		Name: filepath.Base(abs),
		HTML: buf.String(),
	}, nil
}

// buildLineIndex returns the byte offset of the start of each line in src.
// Index 0 holds the offset of line 1, index 1 the offset of line 2, etc.
func buildLineIndex(src []byte) []int {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// lineOf returns the 1-based source line containing byte offset off, given
// the line-start index built by buildLineIndex.
func lineOf(starts []int, off int) int {
	if off < 0 {
		return 1
	}
	// sort.SearchInts finds the first index i such that starts[i] > off,
	// which is the count of line starts at or before off — i.e. the
	// 1-based line number itself.
	return sort.SearchInts(starts, off+1)
}

// firstOffset returns the byte offset of the first source rune covered by
// node n. Leaf-ish blocks (paragraphs, headings, fenced code) carry their
// own Lines(); container blocks (List, Blockquote, Table) do not and must
// be resolved by recursing into their first child. Returns -1 if no offset
// can be found anywhere in the subtree.
func firstOffset(n ast.Node) int {
	if n == nil {
		return -1
	}
	if lines := n.Lines(); lines != nil && lines.Len() > 0 {
		return lines.At(0).Start
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if off := firstOffset(c); off >= 0 {
			return off
		}
	}
	return -1
}

// GetInitialFile returns the rendered file passed on the command line, if
// any. The frontend calls this once on startup.
func (a *App) GetInitialFile() (FileResult, error) {
	if a.initial == "" {
		return FileResult{}, nil
	}
	result, err := renderFile(a.initial)
	if err == nil {
		a.watcher.watch(result.Path)
	}
	return result, err
}

// OpenFile shows a native file-open dialog filtered to markdown files, then
// loads and renders the chosen file. Returns a zero-value FileResult (no
// error) if the user cancels the dialog.
func (a *App) OpenFile() (FileResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Markdown File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown (*.md, *.markdown)", Pattern: "*.md;*.markdown"},
		},
	})
	if err != nil {
		return FileResult{}, err
	}
	if path == "" {
		return FileResult{}, nil
	}
	result, err := renderFile(path)
	if err == nil {
		a.watcher.watch(result.Path)
	}
	return result, err
}

// LoadFile reads and renders a specific path, for future drag-and-drop use.
func (a *App) LoadFile(path string) (FileResult, error) {
	result, err := renderFile(path)
	if err == nil {
		a.watcher.watch(result.Path)
	}
	return result, err
}
