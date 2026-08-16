package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// fileWatcher watches the parent directory of the currently-open markdown
// file and re-renders on change, pushing the result to the frontend via the
// "file:changed" event.
//
// It watches the DIRECTORY, not the file itself. Editors save atomically
// (write a temp file, then rename() it over the target), which replaces the
// file's inode; an inotify watch bound to the old inode goes permanently
// deaf after the first save. Watching the directory and filtering events by
// basename survives repeated atomic saves. See TDD §5.2.
type fileWatcher struct {
	ctx      context.Context
	fsw      *fsnotify.Watcher
	mu       sync.Mutex
	path     string // absolute path of the currently watched file
	dir      string // directory currently registered with fsw
	debounce func(func())
}

// newFileWatcher creates a fileWatcher. It never returns an error to the
// caller — if the underlying fsnotify.Watcher can't be created (inotify
// limits exhausted, sandboxed environment, etc.), it returns a fileWatcher
// with a nil fsw that no-ops on every call. Auto-reload is an enhancement;
// its absence must never block startup or file opening (TDD §5.4).
func newFileWatcher(ctx context.Context) *fileWatcher {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintln(os.Stderr, "glimmer: watcher unavailable:", err)
		return &fileWatcher{ctx: ctx, debounce: debounce.New(100 * time.Millisecond)}
	}

	w := &fileWatcher{
		ctx:      ctx,
		fsw:      fsw,
		debounce: debounce.New(100 * time.Millisecond),
	}
	go w.loop()
	return w
}

// watch re-targets the watcher at path, removing any previously watched
// directory first. At most one file is watched at a time. Safe to call with
// an empty path (no-op) or repeatedly with the same directory.
func (w *fileWatcher) watch(path string) {
	if w == nil || w.fsw == nil || path == "" {
		return
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	dir := filepath.Dir(abs)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dir != "" && w.dir != dir {
		if err := w.fsw.Remove(w.dir); err != nil {
			fmt.Fprintln(os.Stderr, "glimmer: watch remove failed:", err)
		}
	}
	if w.dir != dir {
		if err := w.fsw.Add(dir); err != nil {
			fmt.Fprintln(os.Stderr, "glimmer: watch add failed:", err)
			w.dir = ""
			w.path = ""
			return
		}
		w.dir = dir
	}
	w.path = abs
}

// currentPath returns the currently watched absolute path.
func (w *fileWatcher) currentPath() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.path
}

// matches reports whether ev refers to the currently watched file, compared
// by basename so a rename/replace of the same-named file still counts.
func (w *fileWatcher) matches(ev fsnotify.Event) bool {
	target := w.currentPath()
	if target == "" {
		return false
	}
	return filepath.Base(ev.Name) == filepath.Base(target)
}

const relevantOps = fsnotify.Write | fsnotify.Create | fsnotify.Rename | fsnotify.Chmod

// loop is the event goroutine. It exits when the underlying watcher's
// channels are closed (i.e. after close()).
func (w *fileWatcher) loop() {
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&relevantOps == 0 {
				continue
			}
			if !w.matches(ev) {
				continue
			}
			w.debounce(func() {
				path := w.currentPath()
				if path == "" {
					return
				}
				res, err := renderFile(path)
				if err != nil {
					// Render failed (deleted, truncated mid-write,
					// unreadable). Keep the last good render on screen;
					// emit nothing. TDD §5.4.
					return
				}
				runtime.EventsEmit(w.ctx, "file:changed", res)
			})
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// Drain and log; never let this block the loop.
			fmt.Fprintln(os.Stderr, "glimmer: watch error:", err)
		}
	}
}

// close shuts down the underlying fsnotify watcher, if any. Safe to call on
// a nil-fsw watcher or a nil *fileWatcher.
func (w *fileWatcher) close() {
	if w == nil || w.fsw == nil {
		return
	}
	_ = w.fsw.Close()
}
