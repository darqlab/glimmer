//go:build !bindings

package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	func() {
		defer func() { os.Stdout = orig }()
		fn()
	}()
	w.Close()
	return <-done
}

// AT-S15: help text no longer claims markdown only and documents refusals.
func TestHelpTextAT_S15(t *testing.T) {
	llm := captureStdout(t, printLLMHelp)
	for _, want := range []string{"binary", "too large", "5 MB"} {
		if !strings.Contains(llm, want) {
			t.Errorf("llm help missing %q", want)
		}
	}
	if strings.Contains(llm, "not a .md file") {
		t.Error("llm help still contains \"not a .md file\"")
	}

	var buf bytes.Buffer
	fs := flag.NewFlagSet("glimmer", flag.ContinueOnError)
	fs.SetOutput(&buf)
	usage(fs)
	if !strings.Contains(buf.String(), "[file]") {
		t.Errorf("usage lacks [file]:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "not a .md file") {
		t.Error("usage contains \"not a .md file\"")
	}
}
