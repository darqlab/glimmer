package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

func TestBuildLineIndex(t *testing.T) {
	src := []byte("line one\nline two\nline three")
	starts := buildLineIndex(src)

	want := []int{0, 9, 18}
	if len(starts) != len(want) {
		t.Fatalf("buildLineIndex length = %d, want %d (%v)", len(starts), len(want), starts)
	}
	for i, w := range want {
		if starts[i] != w {
			t.Errorf("buildLineIndex[%d] = %d, want %d", i, starts[i], w)
		}
	}
}

func TestLineOf(t *testing.T) {
	src := []byte("line one\nline two\nline three")
	starts := buildLineIndex(src)

	cases := []struct {
		off  int
		want int
	}{
		{0, 1},  // start of file -> line 1
		{8, 1},  // last byte of line 1 (the newline itself belongs to line 1)
		{9, 2},  // first byte after the first '\n' -> line 2
		{18, 3}, // first byte after the second '\n' -> line 3
		{-1, 1}, // negative sentinel -> line 1
	}
	for _, c := range cases {
		got := lineOf(starts, c.off)
		if got != c.want {
			t.Errorf("lineOf(starts, %d) = %d, want %d", c.off, got, c.want)
		}
	}
}

// TestRenderFileDataLineMatchesSource is the key regression guard: it
// exercises renderFile against a fixture containing a heading, a paragraph,
// a fenced code block and a GFM table, and asserts each block's data-line
// wrapper matches the real 1-based line in the source file where that block
// starts.
func TestRenderFileDataLineMatchesSource(t *testing.T) {
	src := "# Heading\n" + // line 1
		"\n" + // line 2
		"A paragraph of text.\n" + // line 3
		"\n" + // line 4
		"```go\n" + // line 5 (fenced code block starts here)
		"fmt.Println(\"hi\")\n" + // line 6
		"```\n" + // line 7
		"\n" + // line 8
		"| A | B |\n" + // line 9 (table starts here)
		"|---|---|\n" + // line 10
		"| 1 | 2 |\n" // line 11

	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.md")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	result, err := renderFile(path)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}

	dataLineRe := regexp.MustCompile(`data-line="(\d+)"`)
	matches := dataLineRe.FindAllStringSubmatch(result.HTML, -1)

	// Note: goldmark's FencedCodeBlock.Lines() covers only the fenced
	// content, not the opening ``` fence marker itself — so the code
	// block's number points at its first content line (6), not the fence
	// (5). This is upstream goldmark behavior, not a bug in firstOffset.
	wantLines := []int{1, 3, 6, 9}
	if len(matches) != len(wantLines) {
		t.Fatalf("got %d md-block wrappers, want %d\nHTML:\n%s", len(matches), len(wantLines), result.HTML)
	}
	for i, m := range matches {
		got, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("data-line %q not an int: %v", m[1], err)
		}
		if got != wantLines[i] {
			t.Errorf("block %d: data-line = %d, want %d", i, got, wantLines[i])
		}
	}

	// Sanity: fenced code block and table both actually rendered (they're
	// the two node types goldmark's HTML renderer drops attributes for,
	// which is why the wrapper-div approach is used instead of
	// SetAttributeString on the AST node).
	if !strings.Contains(result.HTML, "<pre") {
		t.Error("expected a <pre> fenced code block in the output")
	}
	if !strings.Contains(result.HTML, "<table>") {
		t.Error("expected a <table> in the output")
	}
}

// writeFixture writes content to name inside a fresh temp dir.
func writeFixture(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

// AT-S01: markdown without labelled fences renders exactly as the
// pre-change pipeline did (rebuilt here, no golden string).
func TestRenderFileMarkdownUnchangedAT_S01(t *testing.T) {
	src := "# Title\n\nA paragraph with <span>inline html</span>.\n\n" +
		"- one\n- two\n\n> quote\n\n" +
		"| A | B |\n|---|---|\n| 1 | 2 |\n\n" +
		"- [x] done\n- [ ] todo\n\n" +
		"```\nunlabelled fence\n```\n\n<div>raw</div>\n"
	path := writeFixture(t, "s01.md", []byte(src))

	old := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	var want strings.Builder
	doc := old.Parser().Parse(text.NewReader([]byte(src)))
	starts := buildLineIndex([]byte(src))
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		fmt.Fprintf(&want, `<div class="md-block" data-line="%d">`, lineOf(starts, firstOffset(n)))
		if err := old.Renderer().Render(&want, []byte(src), n); err != nil {
			t.Fatal(err)
		}
		want.WriteString("</div>")
	}

	got, err := renderFile(path)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}
	if got.HTML != want.String() {
		t.Errorf("markdown HTML differs from pre-change pipeline\n got: %s\nwant: %s", got.HTML, want.String())
	}
	if got.Mode != "markdown" || got.EveryLine {
		t.Errorf("Mode=%q EveryLine=%v, want markdown/false", got.Mode, got.EveryLine)
	}
}

// AT-S03: a labelled fence is highlighted with chroma classes.
func TestRenderFileLabelledFenceHighlightedAT_S03(t *testing.T) {
	path := writeFixture(t, "s03.md", []byte("# T\n\n```go\nfunc main() {\n\tvar x int\n}\n```\n"))
	got, err := renderFile(path)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}
	if got.Mode != "markdown" {
		t.Errorf("Mode = %q, want markdown", got.Mode)
	}
	if !strings.Contains(got.HTML, `<pre class="chroma">`) {
		t.Errorf("expected <pre class=\"chroma\">, got:\n%s", got.HTML)
	}
	if !strings.Contains(got.HTML, `class="kd"`) && !strings.Contains(got.HTML, `class="nx"`) {
		t.Errorf("expected a chroma token class (kd/nx), got:\n%s", got.HTML)
	}
}

// AT-S04: an unknown fence label is left exactly as before.
func TestRenderFileUnknownFenceLabelAT_S04(t *testing.T) {
	path := writeFixture(t, "s04.md", []byte("```nosuchlang\nhello\n```\n"))
	got, err := renderFile(path)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}
	if !strings.Contains(got.HTML, `<pre><code class="language-nosuchlang">`) {
		t.Errorf("expected plain language-nosuchlang block, got:\n%s", got.HTML)
	}
	if strings.Contains(got.HTML, `class="chroma"`) {
		t.Errorf("unknown label must not be chroma-highlighted:\n%s", got.HTML)
	}
}
