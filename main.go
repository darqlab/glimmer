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
		// Floor only. No MaxWidth/MaxHeight: the ceiling on an interactive
		// resize comes from the window manager's per-monitor work area, and
		// setting a max here would only tighten it further.
		MinWidth:  480,
		MinHeight: 360,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "glimmer:", err)
		os.Exit(1)
	}
}
