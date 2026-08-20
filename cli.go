//go:build !bindings

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// parseArgs handles -h / -llm-help and returns the markdown file path to open
// on startup, if any. It exits the process directly for help output or
// argument errors, matching a CLI tool's expected behaviour.
//
// This file is excluded from Wails' "bindings" build (see cli_bindings.go) —
// that build runs the compiled binary with no arguments purely to dump the
// JS/TS API descriptors, and must reach wails.Run() rather than exit early.
func parseArgs(args []string) string {
	var help, llmHelp bool

	fs := flag.NewFlagSet("glimmer", flag.ContinueOnError)
	fs.BoolVar(&help, "h", false, "show usage")
	fs.BoolVar(&llmHelp, "llm-help", false, "print machine-readable help for LLM/agent consumption")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if help {
		usage(fs)
		os.Exit(0)
	}
	if llmHelp {
		printLLMHelp()
		os.Exit(0)
	}

	files := fs.Args()
	if len(files) == 0 {
		printLLMHelp()
		os.Exit(0)
	}

	path := files[0]
	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".markdown") {
		fmt.Fprintf(os.Stderr, "glimmer: %s: not a .md file\n", path)
		os.Exit(1)
	}

	return path
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "glimmer - a markdown file reader\n\nUsage:\n  glimmer [flags] [file.md]\n\nFlags:\n")
	fs.PrintDefaults()
}

func printLLMHelp() {
	fmt.Print(`glimmer - markdown file reader (GUI)

PURPOSE
  Opens a native window and renders a markdown file as styled, responsive
  HTML (real tables, wrapped text, dark/light theme). Built with Wails
  (Go backend + embedded webview frontend).

USAGE
  glimmer [flags] [file.md]
  glimmer                     (no args: prints this help, no window)

FLAGS
  -h          print human usage
  -llm-help   print this machine-readable help

BEHAVIOR
  - With a file argument: opens a window with that file rendered.
  - Without a file argument: prints this help and exits (use the window's
    "Open" button to pick a file interactively instead).
  - The window has an "Open" button to choose another .md/.markdown file.
  - Only files ending in ".md" or ".markdown" (case-insensitive) are accepted.
  - Tables, code blocks, lists, blockquotes and task lists render via GFM
    (GitHub-Flavored Markdown) rules. Tables reflow responsively with the
    window; code blocks scroll horizontally instead of wrapping.
  - Theme follows the OS light/dark preference by default; use the toolbar's
    sun/moon button to override, which is remembered between runs.
  - Source line numbers: the toolbar's #️⃣ button toggles a left-hand gutter
    showing the source .md file's line number for each top-level block
    (heading, paragraph, code fence, table, etc). Numbers mark block starts,
    not every wrapped visual line. Off by default; remembered between runs.
  - Copy on selection: the toolbar's 📋 button toggles automatic clipboard
    copy of any text selected in the reading area (mouse only; a brief
    toast confirms). On by default, since the window is read-only and
    selecting text has no other purpose; remembered between runs.
  - The open file is watched for external changes; editing it in another
    program (editor, script, etc.) re-renders the window automatically
    within ~200ms, preserving scroll position. No confirmation, no banner —
    the content simply updates in place. Opening a different file re-targets
    the watch to it.

ERRORS
  - Non-.md/.markdown path      -> "<path>: not a .md file"
  - Missing/unreadable file     -> "cannot open <name>: <os error>"
  - Directory path              -> "<path> is a directory, not a file"

EXIT CODES
  0  success (or help printed)
  1  any error; message is written to stderr prefixed with "glimmer: "

EXAMPLES
  glimmer README.md
  glimmer -llm-help
`)
}
