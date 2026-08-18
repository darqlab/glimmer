package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	initial := parseArgs(os.Args[1:])

	app := NewApp(initial)

	err := wails.Run(&options.App{
		Title:  "glimmer",
		Width:  1024,
		Height: 768,
		// Leaving MaxWidth/MaxHeight at 0 makes Wails' Linux/GTK backend
		// substitute the current monitor's geometry as the resize ceiling
		// (internal/frontend/desktop/linux/window.c: SetMinMaxSize). That
		// query is not rotation-aware: on a monitor rotated to portrait via
		// xrandr (e.g. "1080x1920 right"), it reports the pre-rotation
		// landscape dimensions, so the vertical max gets clamped to the
		// portrait width (1080) instead of the true portrait height (1920)
		// — the window can't be resized taller than that on such a monitor.
		// Set explicit, generous maximums so resizing never depends on
		// monitor auto-detection.
		MinWidth:  480,
		MinHeight: 360,
		MaxWidth:  7680,
		MaxHeight: 4320,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "glimmer:", err)
		os.Exit(1)
	}
}
