package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lexerName returns the display name of the source lexer for path, or "".
func lexerName(path string, head []byte) string {
	if l := sourceLexer(path, head); l != nil {
		return l.Config().Name
	}
	return ""
}

// AT-S05: well-known names select the source format and the right lexer.
func TestSelectFormatSourceNamesAT_S05(t *testing.T) {
	cases := []struct{ name, lexer string }{
		{"main.go", "Go"},
		{"app.py", "Python"},
		{"Foo.java", "Java"},
		{"script.sh", "Bash"},
		{"config.yaml", "YAML"},
		{"data.json", "JSON"},
		{"pom.xml", "XML"},
		{"Cargo.toml", "TOML"},
		{"app.ini", "INI"},
		{"Dockerfile", "Docker"},
		{"Makefile", "Makefile"},
		{".env", "Bash"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := selectFormat(c.name, nil)
			if f.name != "source" {
				t.Errorf("selectFormat(%q).name = %q, want source", c.name, f.name)
			}
			if got := lexerName(c.name, nil); got != c.lexer {
				t.Errorf("lexer for %q = %q, want %q", c.name, got, c.lexer)
			}
		})
	}
}

// AT-S06: text files, by name or by fallback.
func TestSelectFormatTextAT_S06(t *testing.T) {
	logLine := []byte("2026-10-18 10:00:00 INFO server started on port 8080\n")
	cases := []struct {
		name    string
		content []byte
	}{
		{"notes.txt", []byte("some notes\n")},
		{"app.log", logLine},
		{"README", []byte("This is just prose without any code in it.\n")},
		{".gitignore", []byte("build/\n*.o\n")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if f := selectFormat(c.name, c.content); f.name != "text" {
				t.Errorf("selectFormat(%q).name = %q, want text", c.name, f.name)
			}
			res, err := renderFile(writeFixture(t, c.name, c.content))
			if err != nil {
				t.Fatalf("renderFile: %v", err)
			}
			if res.Mode != "text" || res.Language != "Plain text" {
				t.Errorf("Mode=%q Language=%q, want text / Plain text", res.Mode, res.Language)
			}
			if !res.EveryLine {
				t.Error("EveryLine = false, want true")
			}
		})
	}
}

// AT-S07: case-insensitivity, overrides and shebang detection.
func TestSelectFormatCaseOverrideShebangAT_S07(t *testing.T) {
	t.Run("DATA.JSON", func(t *testing.T) {
		if f := selectFormat("DATA.JSON", nil); f.name != "source" {
			t.Errorf("format = %q, want source", f.name)
		}
		if got := lexerName("DATA.JSON", nil); got != "JSON" {
			t.Errorf("lexer = %q, want JSON", got)
		}
	})
	t.Run("go.mod", func(t *testing.T) {
		content := []byte("module example.com/x\n\ngo 1.25\n")
		path := writeFixture(t, "go.mod", content)
		res, err := renderFile(path)
		if err != nil {
			t.Fatalf("renderFile: %v", err)
		}
		if res.Mode != "source" || res.Language != "Source" {
			t.Errorf("Mode=%q Language=%q, want source / Source", res.Mode, res.Language)
		}
	})
	t.Run("shebang", func(t *testing.T) {
		content := []byte("#!/bin/bash\necho hello\n")
		path := writeFixture(t, "runme", content)
		res, err := renderFile(path)
		if err != nil {
			t.Fatalf("renderFile: %v", err)
		}
		if res.Mode != "source" || res.Language != "Bash" {
			t.Errorf("Mode=%q Language=%q, want source / Bash", res.Mode, res.Language)
		}
	})
}

// AT-S11: binary sniff.
func TestBinarySniffAT_S11(t *testing.T) {
	pngHeader := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R'}
	textThenNUL := append(bytes.Repeat([]byte("x"), 100), 0)
	invalid := []byte{0xff, 0xfe, 'a', 'b'}

	refused := []struct {
		name    string
		content []byte
	}{
		{"image.png", pngHeader},
		{"nul.txt", textThenNUL},
		{"bad.txt", invalid},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			path := writeFixture(t, c.name, c.content)
			if _, _, err := loadSource(path); err == nil || !strings.Contains(err.Error(), "binary file") {
				t.Errorf("loadSource err = %v, want binary file", err)
			}
			if err := preflight(path); err == nil || !strings.Contains(err.Error(), "binary file") {
				t.Errorf("preflight err = %v, want binary file", err)
			}
		})
	}

	t.Run("split rune at sniff boundary", func(t *testing.T) {
		// '€' (3 bytes) starts at offset sniffLen-1, so the sniff window
		// ends in the middle of the rune.
		content := append(bytes.Repeat([]byte("a"), sniffLen-1), []byte("€ and more text\n")...)
		if content[sniffLen-1] != 0xe2 {
			t.Fatalf("fixture wrong: byte at sniffLen-1 = %#x", content[sniffLen-1])
		}
		path := writeFixture(t, "split.txt", content)
		if _, _, err := loadSource(path); err != nil {
			t.Errorf("loadSource: %v", err)
		}
		if err := preflight(path); err != nil {
			t.Errorf("preflight: %v", err)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		path := writeFixture(t, "empty.txt", nil)
		if _, _, err := loadSource(path); err != nil {
			t.Errorf("loadSource: %v", err)
		}
		if err := preflight(path); err != nil {
			t.Errorf("preflight: %v", err)
		}
	})

	t.Run("isBinary direct", func(t *testing.T) {
		if isBinary([]byte("plain\n")) {
			t.Error("plain text reported binary")
		}
		if !isBinary([]byte{'a', 0, 'b'}) {
			t.Error("NUL not reported binary")
		}
		// A truncated rune in a SHORT chunk is genuinely invalid.
		if !isBinary([]byte{'a', 0xe2, 0x82}) {
			t.Error("truncated rune in a short chunk should be binary")
		}
	})
}

// AT-S12: size-cap boundaries.
func TestSizeCapAT_S12(t *testing.T) {
	t.Run("exactly at cap is accepted", func(t *testing.T) {
		path := writeFixture(t, "cap.txt", bytes.Repeat([]byte("a\n"), maxFileSize/2))
		src, _, err := loadSource(path)
		if err != nil {
			t.Fatalf("loadSource: %v", err)
		}
		if len(src) != maxFileSize {
			t.Errorf("len(src) = %d, want %d", len(src), maxFileSize)
		}
		if err := preflight(path); err != nil {
			t.Errorf("preflight: %v", err)
		}
	})
	over := []string{"over.txt", "over.md"}
	for _, name := range over {
		t.Run("cap+1 "+name, func(t *testing.T) {
			path := writeFixture(t, name, nil)
			if err := os.Truncate(path, maxFileSize+1); err != nil {
				t.Fatal(err)
			}
			_, _, err := loadSource(path)
			if err == nil || !strings.Contains(err.Error(), "file too large") {
				t.Errorf("loadSource err = %v, want file too large", err)
			}
			var ref *refusalError
			if !errors.As(err, &ref) {
				t.Errorf("error is not *refusalError: %T", err)
			}
			if err := preflight(path); err == nil || !strings.Contains(err.Error(), "file too large") {
				t.Errorf("preflight err = %v, want file too large", err)
			}
		})
	}
	t.Run("6 MB message", func(t *testing.T) {
		path := writeFixture(t, "six.txt", nil)
		if err := os.Truncate(path, 6<<20); err != nil {
			t.Fatal(err)
		}
		err := preflight(path)
		if err == nil || !strings.Contains(err.Error(), "file too large (6.0 MB; limit is 5 MB)") {
			t.Errorf("err = %v", err)
		}
		var ref *refusalError
		if errors.As(err, &ref) {
			want := "Can't show six.txt: file too large (6.0 MB; limit is 5 MB)."
			if ref.inApp() != want {
				t.Errorf("inApp = %q, want %q", ref.inApp(), want)
			}
		}
	})
}

// Preflight must leave missing files and directories to the window.
func TestPreflightMissingAndDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := preflight(filepath.Join(dir, "missing.md")); err != nil {
		t.Errorf("missing file: %v, want nil", err)
	}
	if err := preflight(dir); err != nil {
		t.Errorf("directory: %v, want nil", err)
	}
}

// Markdown skips the binary sniff.
func TestMarkdownSkipsSniff(t *testing.T) {
	path := writeFixture(t, "bin.md", []byte{'#', ' ', 'a', 0, 'b', '\n'})
	if _, f, err := loadSource(path); err != nil || f.name != "markdown" {
		t.Errorf("loadSource: format=%q err=%v, want markdown/nil", f.name, err)
	}
	if err := preflight(path); err != nil {
		t.Errorf("preflight: %v", err)
	}
}

// AT-S30: Latin-1 markdown still renders; an empty text file is flagged.
func TestLatin1MarkdownAndEmptyTextAT_S30(t *testing.T) {
	md := writeFixture(t, "latin.md", []byte("caf\xe9 au lait\n"))
	res, err := renderFile(md)
	if err != nil {
		t.Fatalf("renderFile latin1 md: %v", err)
	}
	if res.Mode != "markdown" {
		t.Errorf("Mode = %q, want markdown", res.Mode)
	}

	txt := writeFixture(t, "empty.txt", nil)
	res, err = renderFile(txt)
	if err != nil {
		t.Fatalf("renderFile empty: %v", err)
	}
	if res.Mode != "text" {
		t.Errorf("Mode = %q, want text", res.Mode)
	}
	if !strings.Contains(res.HTML, `data-empty="true"`) {
		t.Errorf("wrapper lacks data-empty: %s", res.HTML)
	}
}

// AT-S27: a new format registers by prepending a table entry only.
func TestRegisterFakeFormatAT_S27(t *testing.T) {
	const marker = "<p>GLIMFAKE-MARKER</p>"
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	fake := write("x.glimfake", "whatever\n")
	md := write("a.md", "# Hi\n")
	goSrc := write("main.go", "package main\n")

	beforeMD, err := renderFile(md)
	if err != nil {
		t.Fatal(err)
	}
	beforeGo, err := renderFile(goSrc)
	if err != nil {
		t.Fatal(err)
	}

	orig := formats
	t.Cleanup(func() { formats = orig })
	formats = append([]format{{
		name: "glimfake",
		match: func(path string, _ []byte) bool {
			return strings.HasSuffix(path, ".glimfake")
		},
		render: func(string, []byte) (string, string, error) {
			return marker, "Fake", nil
		},
	}}, orig...)

	got, err := renderFile(fake)
	if err != nil {
		t.Fatalf("renderFile fake: %v", err)
	}
	if got.Mode != "glimfake" || got.HTML != marker {
		t.Errorf("Mode=%q HTML=%q, want glimfake / marker", got.Mode, got.HTML)
	}
	afterMD, _ := renderFile(md)
	afterGo, _ := renderFile(goSrc)
	if afterMD != beforeMD {
		t.Error("markdown result changed after registering fake format")
	}
	if afterGo != beforeGo {
		t.Error("Go result changed after registering fake format")
	}
}
