package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/glamour"
	"github.com/yuin/goldmark"
)

func render(opts options, w io.Writer) error {
	for _, path := range opts.files {
		if err := validate(path); err != nil {
			return err
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var out string
		switch {
		case opts.html:
			var buf bytes.Buffer
			if err := goldmark.Convert(src, &buf); err != nil {
				return err
			}
			out = buf.String()
		case opts.raw:
			out = string(src)
		default:
			r, err := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(opts.width),
			)
			if err != nil {
				return err
			}
			out, err = r.Render(string(src))
			if err != nil {
				return err
			}
		}

		if _, err := io.WriteString(w, out); err != nil {
			return err
		}
	}
	return nil
}

func run(args []string) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}

	if opts.pager {
		pager := os.Getenv("PAGER")
		if pager == "" {
			pager = "less"
		}
		cmd := exec.Command(pager, "-R")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := render(opts, stdin); err != nil {
			return err
		}
		stdin.Close()
		return cmd.Wait()
	}

	return render(opts, os.Stdout)
}
