// harden-gui est une GUI minimaliste Wails v2 pour le moteur harden-engine.
// Réutilise les packages pkg/engine/* (manifest, executor, journal, etc.).
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	initLogger()
	logf("main: starting GUI")
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "Win11 Hardening",
		Width:  1100,
		Height: 750,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		Bind: []any{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
