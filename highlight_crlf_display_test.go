package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
)

const crlfSample = "# Service configuration (glimmer fixture)\r\n" +
	"server:\r\n  host: 0.0.0.0\r\n  port: 8080\r\n  timeouts:\r\n    read: 5s\r\n    write: 10s\r\n\r\n" +
	"database:\r\n  url: postgres://app:changeme@localhost:5432/app\r\n  pool:\r\n    min: 2\r\n    max: 10\r\n\r\n" +
	"features:\r\n  - search\r\n  - export\r\n  - dark-mode\r\n\r\n" +
	"logging:\r\n  level: info   # debug | info | warn | error\r\n  format: json\r\n"

var crlfLineRe = regexp.MustCompile(`<span class="line">`)

func TestRenderSourceCRLFDisplay(t *testing.T) {
	html := renderSource([]byte(crlfSample), lexers.Get("yaml"), "source")
	want := strings.Count(crlfSample, "\n")

	if strings.Contains(html, "\r") {
		t.Errorf("rendered HTML contains carriage return (\\r): %d occurrences", strings.Count(html, "\r"))
	}
	if got := len(crlfLineRe.FindAllString(html, -1)); got != want {
		t.Errorf("line elements = %d, want %d", got, want)
	}
	// Equivalent LF input must render the same number of lines.
	lf := renderSource([]byte(strings.ReplaceAll(crlfSample, "\r\n", "\n")), lexers.Get("yaml"), "source")
	if g, w := len(crlfLineRe.FindAllString(html, -1)), len(crlfLineRe.FindAllString(lf, -1)); g != w {
		t.Errorf("CRLF renders %d line elements, LF renders %d", g, w)
	}
}
