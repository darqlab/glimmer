package main

import (
	stdhtml "html"
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
)

var (
	preRe = regexp.MustCompile(`(?s)<pre[^>]*>(.*)</pre>`)
	tagRe = regexp.MustCompile(`<[^>]*>`)
)

// verbatim extracts the text shown inside the <pre> of rendered source html.
func verbatim(t *testing.T, rendered string) string {
	t.Helper()
	m := preRe.FindStringSubmatch(rendered)
	if m == nil {
		t.Fatalf("no <pre> in output:\n%s", rendered)
	}
	return stdhtml.UnescapeString(tagRe.ReplaceAllString(m[1], ""))
}

// checkVerbatim asserts the displayed text equals src (one added trailing
// newline tolerated) and that class="line" count equals the line count.
func checkVerbatim(t *testing.T, name, src, rendered string) {
	t.Helper()
	got := verbatim(t, rendered)
	if got != src && got != src+"\n" {
		t.Errorf("%s: displayed text differs from source\n got: %q\nwant: %q", name, got, src)
	}
	wantLines := strings.Count(src, "\n")
	if src != "" && !strings.HasSuffix(src, "\n") {
		wantLines++
	}
	if n := strings.Count(rendered, `class="line"`); n != wantLines {
		t.Errorf("%s: class=\"line\" count = %d, want %d", name, n, wantLines)
	}
}

// AT-S08: config formats are shown byte for byte.
func TestRenderSourceVerbatimAT_S08(t *testing.T) {
	fixtures := []struct{ name, src string }{
		{"app.yaml", "# service config\nserver:\n  port: 8080\n  hosts:\n    - a.example.com\n    - b.example.com\nlogging:\n  level: debug # inline\n"},
		{"data.json", "{\n  \"name\": \"glimmer\",\n  \"tags\": [\"a\", \"b\"],\n  \"nested\": {\n    \"k\": 1.5,\n    \"ok\": true\n  }\n}\n"},
		{"pom.xml", "<?xml version=\"1.0\"?>\n<!-- build -->\n<project>\n  <modelVersion>4.0.0</modelVersion>\n  <dependencies>\n    <dependency>\n      <artifactId>x &amp; y</artifactId>\n    </dependency>\n  </dependencies>\n</project>\n"},
		{"Cargo.toml", "# crate\n[package]\nname = \"demo\"\nversion = \"0.1.0\"\n\n[dependencies]\nserde = { version = \"1\", features = [\"derive\"] }\n"},
		{"app.ini", "; comment\n[core]\nname = demo\n\n[paths]\nroot = /var/www\n"},
		{"crlf.yaml", "a: 1\r\nb:\r\n  - x\r\n  - y\r\n"},
		{"tabs.json", "{\n\t\"a\": 1,\n\t\"b\": [\n\t\t2,\n\t\t3\n\t]\n}\n"},
		{"nonl.yaml", "a: 1\nb: 2"},
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			res, err := renderFile(writeFixture(t, f.name, []byte(f.src)))
			if err != nil {
				t.Fatalf("renderFile: %v", err)
			}
			if res.Mode != "source" {
				t.Errorf("Mode = %q, want source", res.Mode)
			}
			checkVerbatim(t, f.name, f.src, res.HTML)
		})
	}
}

// AT-S09: malformed JSON still displays.
func TestRenderMalformedJSONAT_S09(t *testing.T) {
	src := "{\n  \"a\": 1,,\n  \"b\": [\n}\n"
	res, err := renderFile(writeFixture(t, "bad.json", []byte(src)))
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}
	if res.Mode != "source" {
		t.Errorf("Mode = %q, want source", res.Mode)
	}
	checkVerbatim(t, "bad.json", src, res.HTML)
}

// AT-S10: markup in source is escaped, never emitted.
func TestRenderSourceEscapesAT_S10(t *testing.T) {
	src := "<script>alert(1)</script>\n"
	for _, name := range []string{"x.txt", "x.html"} {
		t.Run(name, func(t *testing.T) {
			res, err := renderFile(writeFixture(t, name, []byte(src)))
			if err != nil {
				t.Fatalf("renderFile: %v", err)
			}
			// The lexer may split "&lt;" and "script" into separate token
			// spans (x.html), so the literal "&lt;script&gt;" is only
			// guaranteed for plain text; the displayed text must match
			// the source either way.
			if name == "x.txt" && !strings.Contains(res.HTML, "&lt;script&gt;") {
				t.Errorf("missing escaped script tag:\n%s", res.HTML)
			}
			if !strings.Contains(res.HTML, "&lt;") {
				t.Errorf("no escaped angle bracket:\n%s", res.HTML)
			}
			checkVerbatim(t, name, src, res.HTML)
			if strings.Contains(res.HTML, "<script") {
				t.Errorf("raw <script present:\n%s", res.HTML)
			}
		})
	}
}

// AT-S13: highlight stylesheet structure.
func TestHighlightCSSStructureAT_S13(t *testing.T) {
	css := highlightCSS()
	for _, want := range []string{
		".chroma",
		`html[data-theme="dark"] .chroma`,
		"@media (prefers-color-scheme: dark)",
		`html:not([data-theme="light"]) .chroma`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("css missing %q", want)
		}
	}
	// No background-color inside any rule whose selector is the bare
	// PreWrapper .chroma { ... } (or its dark-scoped forms).
	ruleRe := regexp.MustCompile(`(?m)^\s*(?:html[^{]*)?\.chroma\s*\{[^}]*\}`)
	for _, r := range ruleRe.FindAllString(css, -1) {
		if strings.Contains(r, "background-color") {
			t.Errorf("PreWrapper rule has background-color: %s", r)
		}
	}
	if strings.Contains(css, "background-color") {
		t.Errorf("css contains background-color somewhere")
	}
}

// AT-S14: no chroma line-number markup.
func TestRenderSourceNoChromaLineNumbersAT_S14(t *testing.T) {
	out := renderSource([]byte("package main\n\nfunc main() {}\n"), lexers.Get("go"), "source")
	for _, bad := range []string{`class="ln"`, `class="lnt"`, `class="lntd"`, `class="lntable"`} {
		if strings.Contains(out, bad) {
			t.Errorf("output contains %s", bad)
		}
	}
	if !strings.Contains(out, `class="line"`) {
		t.Error("output lacks class=\"line\" spans")
	}
}

// AT-S29: contrast overrides are present and the originals are gone.
func TestHighlightCSSContrastAT_S29(t *testing.T) {
	css := strings.ToLower(highlightCSS())
	for _, want := range []string{"#59636e", "#82071e", "#9198a1", "#ff8a80"} {
		if !strings.Contains(css, want) {
			t.Errorf("css missing override %s", want)
		}
	}
	for _, bad := range []string{"#f6f8fa", "#6a737d", "#8b949e", "#f85149"} {
		if strings.Contains(css, bad) {
			t.Errorf("css still contains %s", bad)
		}
	}
	// Dark comment rule keeps its italic.
	darkComment := regexp.MustCompile(`html\[data-theme="dark"\] \.chroma \.c \{[^}]*\}`).FindString(css)
	if darkComment == "" {
		t.Fatal("dark .c rule not found")
	}
	if !strings.Contains(darkComment, "font-style: italic") || !strings.Contains(darkComment, "#9198a1") {
		t.Errorf("dark comment rule = %s", darkComment)
	}
}
