package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	raw    bool
	html   bool
	pager  bool
	width  int
	help   bool
	files  []string
}

func parseFlags(args []string) (options, error) {
	opts := options{width: 80}

	fs := flag.NewFlagSet("glimmer", flag.ContinueOnError)
	fs.BoolVar(&opts.raw, "t", false, "render as plain text (no styling)")
	fs.BoolVar(&opts.html, "html", false, "render as HTML")
	fs.BoolVar(&opts.pager, "p", false, "page output through $PAGER")
	fs.IntVar(&opts.width, "w", 80, "wrap width for terminal output")
	fs.BoolVar(&opts.help, "h", false, "show usage")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	opts.files = fs.Args()
	if opts.help {
		usage(fs)
		os.Exit(0)
	}

	if len(opts.files) == 0 {
		return opts, fmt.Errorf("no markdown file specified")
	}

	for _, f := range opts.files {
		if !strings.HasSuffix(strings.ToLower(f), ".md") {
			return opts, fmt.Errorf("%s: not a .md file", f)
		}
	}

	return opts, nil
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), "glimmer - a markdown file reader\n\nUsage:\n  glimmer [flags] <file.md> [file2.md ...]\n\nFlags:\n")
	fs.PrintDefaults()
}

func validate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", filepath.Base(path), err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", path)
	}
	return nil
}
