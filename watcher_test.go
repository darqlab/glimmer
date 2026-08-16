package main

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

// AT-08: an event for a differently-named file in the same directory (e.g.
// an editor's temp file "foo.md.tmp") must be filtered out.
func TestFileWatcherMatchesFiltersByBasename(t *testing.T) {
	w := &fileWatcher{path: "/x/y/foo.md"}

	if !w.matches(fsnotify.Event{Name: "/x/y/foo.md", Op: fsnotify.Write}) {
		t.Error("expected match for the watched file itself")
	}
	if w.matches(fsnotify.Event{Name: "/x/y/foo.md.tmp", Op: fsnotify.Write}) {
		t.Error("expected no match for a differently-named temp file")
	}
	if w.matches(fsnotify.Event{Name: "/x/y/bar.md", Op: fsnotify.Write}) {
		t.Error("expected no match for an unrelated file")
	}

	w2 := &fileWatcher{}
	if w2.matches(fsnotify.Event{Name: "/x/y/foo.md", Op: fsnotify.Write}) {
		t.Error("expected no match when no file is being watched")
	}
}
