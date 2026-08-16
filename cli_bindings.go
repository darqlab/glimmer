//go:build bindings

package main

// parseArgs is a no-op under Wails' "bindings" build tag: that build runs the
// compiled binary with no arguments to dump JS/TS API descriptors via
// wails.Run(), and must not exit early. See cli.go for the real CLI parsing.
func parseArgs(args []string) string {
	return ""
}
