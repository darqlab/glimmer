package main

import (
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
)

// Dark-mode legibility of source view tokens. Background used: the dark
// --bg token in frontend/src/style.css (html[data-theme="dark"]), because the
// chroma CSS strips every background-color.

const javaSample = `package net.darqlab.sample;

import java.util.List;

public class Foo {
    private static final int LIMIT = 3;

    public static List<String> shout(List<String> words) {
        return words.stream()
                .limit(LIMIT)
                .map(String::toUpperCase)
                .collect(Collectors.toList());
    }
}
`

var (
	cssRuleRe  = regexp.MustCompile(`(?m)^(?:/\* [^*]* \*/ )?([^{]+?)\s*\{([^}]*)\}`)
	colourRe   = regexp.MustCompile(`(?:^|;|\s)color:\s*(#[0-9a-fA-F]{6})`)
	classAttrR = regexp.MustCompile(`<span class="([a-z0-9]+)">`)
	darkBgRe   = regexp.MustCompile(`(?s)html\[data-theme="dark"\]\s*\{[^}]*--bg:\s*(#[0-9a-fA-F]{6})`)
)

func luminance(hex string) float64 {
	ch := func(s string) float64 {
		v, _ := strconv.ParseUint(s, 16, 8)
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(hex[1:3]) + 0.7152*ch(hex[3:5]) + 0.0722*ch(hex[5:7])
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// colourFor returns the colour from the first rule in css whose selector is
// exactly sel, or "" if none sets a colour.
func colourFor(css, sel string) string {
	for _, m := range cssRuleRe.FindAllStringSubmatch(css, -1) {
		if strings.TrimSpace(m[1]) == sel {
			if c := colourRe.FindStringSubmatch(m[2]); c != nil {
				return strings.ToLower(c[1])
			}
		}
	}
	return ""
}

func TestDarkModeSourceTokensAreLegible(t *testing.T) {
	css := highlightCSS()
	// Only the explicit-dark block; the same rules are repeated in @media.
	darkPrefix := `html[data-theme="dark"] .chroma`

	styleSrc, err := os.ReadFile("frontend/src/style.css")
	if err != nil {
		t.Fatal(err)
	}
	m := darkBgRe.FindSubmatch(styleSrc)
	if m == nil {
		t.Fatal("dark --bg not found in frontend/src/style.css")
	}
	bg := strings.ToLower(string(m[1]))

	base := colourFor(css, darkPrefix)
	if base == "" {
		t.Fatalf("no dark base .chroma colour in generated CSS")
	}

	// Classes actually emitted for the sample.
	rendered := renderSource([]byte(javaSample), lexers.Get("java"), "source")
	seen := map[string]bool{}
	for _, c := range classAttrR.FindAllStringSubmatch(rendered, -1) {
		if c[1] != "line" && c[1] != "cl" {
			seen[c[1]] = true
		}
	}
	// Plain text and whitespace tokens carry no class or .w / .p / .n.
	for _, c := range []string{"p", "n", "w", "x", "nx", "na", "nb", "bp"} {
		seen[c] = true
	}

	var classes []string
	for c := range seen {
		classes = append(classes, c)
	}
	sort.Strings(classes)

	t.Logf("dark background %s (frontend/src/style.css --bg); dark base .chroma colour %s", bg, base)
	for _, c := range classes {
		// Cascade for <span class=c> in dark: a dark rule beats the light
		// `.chroma .c` rule (higher specificity); otherwise the light rule
		// still applies and sets the colour; otherwise it inherits the base.
		eff, src := colourFor(css, darkPrefix+" ."+c), "dark rule"
		if eff == "" {
			if eff = colourFor(css, ".chroma ."+c); eff != "" {
				src = "LIGHT rule leaking into dark (.chroma ." + c + ")"
			}
		}
		if eff == "" {
			eff, src = base, "inherited base"
		}
		if c == "w" {
			continue // whitespace has no glyphs to read
		}
		if r := contrast(eff, bg); r < 4.5 {
			t.Errorf("class .%s: colour %s on %s has contrast %.2f:1 (< 4.5:1) via %s",
				c, eff, bg, r, src)
		}
	}
}
