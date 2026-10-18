//go:build !bindings

package main

import (
	"flag"
	"fmt"
	"os"
)

// parseArgs handles -h / -llm-help and returns the file path to open
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
	if err := preflight(path); err != nil {
		fmt.Fprintf(os.Stderr, "glimmer: %v\n", err)
		os.Exit(1)
	}

	return path
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "glimmer - a viewer for markdown, source code and text files\n\nUsage:\n  glimmer [flags] [file]\n\nFlags:\n")
	fs.PrintDefaults()
}

func printLLMHelp() {
	fmt.Print(`glimmer - viewer for markdown, source code and text files (GUI)

PURPOSE
  Opens a native window and shows a file read-only. Markdown is rendered as
  styled, responsive HTML (real tables, wrapped text, dark/light theme).
  Source and config files are shown with syntax highlighting. Other text
  files are shown as plain text. Built with Wails (Go backend + embedded
  webview frontend).

USAGE
  glimmer [flags] [file]
  glimmer                     (no args: prints this help, no window)

FLAGS
  -h          print human usage
  -llm-help   print this machine-readable help

BEHAVIOR
  - With a file argument: opens a window with that file shown.
  - Without a file argument: prints this help and exits (use the window's
    "Open" button to pick a file interactively instead).
  - The window has an "Open" button to choose another file. The dialog
    offers "All files" (the default) and "Markdown".
  - Three modes, chosen per file:
      markdown  files ending in ".md" or ".markdown" (case-insensitive).
                Rendered as GitHub-Flavored Markdown: tables, code blocks,
                lists, blockquotes and task lists. Fenced code blocks that
                name a language are syntax-highlighted. Code blocks scroll
                horizontally instead of wrapping.
      source    files whose name or content chroma recognises: Go, Python,
                Java, shell, YAML, JSON, XML, TOML, INI, Dockerfile,
                Makefile, ".env" and many more, including files that start
                with a shebang line. Highlighted, never wrapped; a
                horizontal scrollbar appears for long lines.
      text      ".txt" and ".log" files, and any other text file that is not
                recognised as source. Shown as plain text, wrapped.
  - Structured files (JSON, YAML, XML, TOML, INI) are shown exactly as they
    are on disk: never reformatted, so line numbers match the file. A
    malformed file still displays; the bad tokens are marked.
  - Files are limited to 5 MB. Binary files (a NUL byte or invalid UTF-8 in
    the first 8 KB) are refused. Markdown files skip the binary check.
  - Theme follows the OS light/dark preference by default; use the toolbar's
    sun/moon button to override, which is remembered between runs.
  - Source line numbers: the toolbar's #️⃣ button toggles a left-hand gutter.
    For markdown it shows the source line number of each top-level block
    (heading, paragraph, code fence, table, etc); numbers mark block
    starts, not every wrapped visual line. For source and text files every
    line is numbered, and a wrapped line has one number. Off by default;
    remembered between runs.
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
  - Binary file                 -> "<path>: binary file, not shown. glimmer shows text files only."
  - File over 5 MB              -> "<path>: file too large (<size> MB; limit is 5 MB)"
  - Missing/unreadable file     -> "cannot open <name>: <os error>"
  - Directory path              -> "<path> is a directory, not a file"
  A binary or oversized file given on the command line exits with status 1
  before any window opens. A missing file or a directory still opens the
  window, which shows the error.

EXIT CODES
  0  success (or help printed)
  1  any error; message is written to stderr prefixed with "glimmer: "

EXAMPLES
  glimmer README.md
  glimmer main.go
  glimmer config.yaml
  glimmer -llm-help
`)
}
