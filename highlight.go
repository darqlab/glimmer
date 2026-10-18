package main

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"regexp"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// sourceFormatter emits CSS classes only. No chroma line numbers: the gutter
// is a CSS counter on .line::before (DEC-S3).
var sourceFormatter = chromahtml.New(chromahtml.WithClasses(true), chromahtml.TabWidth(4))

// renderSource returns the HTML for a source or text view. It never
// reformats src: the only transformation is HTML escaping. mode is
// format.name ("source" or "text").
func renderSource(src []byte, lexer chroma.Lexer, mode string) string {
	if lexer == nil {
		lexer = lexers.Get("plaintext")
	}
	body, err := formatTokens(src, lexer)
	if err != nil {
		body, err = formatTokens(src, lexers.Get("plaintext"))
	}
	if err != nil {
		body = escapedLines(src)
	}

	attrs := ""
	if len(src) == 0 {
		attrs = ` data-empty="true"`
	}
	return fmt.Sprintf(`<div class="src-view" data-mode="%s"%s style="--gutter-ch:%d">%s</div>`,
		mode, attrs, len(fmt.Sprint(lineCount(src)))+1, body)
}

func formatTokens(src []byte, lexer chroma.Lexer) (string, error) {
	it, err := chroma.Coalesce(lexer).Tokenise(&chroma.TokeniseOptions{State: "root", EnsureLF: false}, string(src))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := sourceFormatter.Format(&buf, styles.Get("github"), it); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// escapedLines is the last-resort rendering: the escaped source, one line
// span per source line, in the same markup chroma produces.
func escapedLines(src []byte) string {
	var b strings.Builder
	b.WriteString(`<pre class="chroma"><code>`)
	rest := string(src)
	for rest != "" {
		line := rest
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			line, rest = rest[:i+1], rest[i+1:]
		} else {
			rest = ""
		}
		b.WriteString(`<span class="line"><span class="cl">` + stdhtml.EscapeString(line) + `</span></span>`)
	}
	b.WriteString(`</code></pre>`)
	return b.String()
}

// lineCount is the number of source lines: the count of '\n' bytes, plus one
// when the file is non-empty and does not end in '\n'.
func lineCount(src []byte) int {
	n := bytes.Count(src, []byte{'\n'})
	if len(src) > 0 && src[len(src)-1] != '\n' {
		n++
	}
	return n
}

var (
	highlightOnce sync.Once
	highlightBlob string
)

// highlightCSS returns the chroma stylesheet: light rules, plus dark rules
// scoped twice (explicit data-theme and OS preference). Built once.
func highlightCSS() string {
	highlightOnce.Do(func() {
		light := styleCSS("github", map[chroma.TokenType]string{
			chroma.NameBuiltinPseudo: "#59636e",
		}, func(b *chroma.StyleBuilder) {
			e := b.Get(chroma.Error)
			e.Colour = chroma.MustParseColour("#82071e")
			e.Background = chroma.Colour(0)
			b.AddEntry(chroma.Error, e)
		})

		dark := styleCSS("github-dark", map[chroma.TokenType]string{
			chroma.Comment:        "#9198a1",
			chroma.CommentSpecial: "#9198a1",
			chroma.CommentPreproc: "#9198a1",
			chroma.GenericOutput:  "#9198a1",
			chroma.GenericPrompt:  "#9198a1",
			chroma.Error:          "#ff8a80",
		}, nil)

		var b strings.Builder
		b.WriteString(light)
		b.WriteString(strings.ReplaceAll(dark, ".chroma", `html[data-theme="dark"] .chroma`))
		b.WriteString("@media (prefers-color-scheme: dark) {\n")
		b.WriteString(strings.ReplaceAll(dark, ".chroma", `html:not([data-theme="light"]) .chroma`))
		b.WriteString("}\n")
		highlightBlob = b.String()
	})
	return highlightBlob
}

// backgroundDecl matches a CSS background-color declaration. chroma's own
// backgrounds are stripped so glimmer's tokens own every background.
var backgroundDecl = regexp.MustCompile(`\s*background-color:\s*[^;}]*;?`)

// styleCSS writes the CSS for a named chroma style with its background
// stripped, the given colour
// overrides applied (Get, set colour, AddEntry, so italic/bold survive), and
// chroma's .bg rule dropped.
func styleCSS(name string, colours map[chroma.TokenType]string, extra func(*chroma.StyleBuilder)) string {
	b := styles.Get(name).Builder()
	for t, hex := range colours {
		e := b.Get(t)
		e.Colour = chroma.MustParseColour(hex)
		b.AddEntry(t, e)
	}
	if extra != nil {
		extra(b)
	}
	st, err := b.Build()
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	if err := sourceFormatter.WriteCSS(&buf, st); err != nil {
		return ""
	}
	var out strings.Builder
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "/* Background */") || strings.HasPrefix(line, ".bg ") {
			continue
		}
		line = backgroundDecl.ReplaceAllString(line, "")
		if strings.TrimSpace(line) == "" || strings.HasSuffix(line, "{  }") || strings.HasSuffix(line, "{ }") {
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.String()
}
