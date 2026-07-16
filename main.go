package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, services, err := newApplication()
	if err != nil {
		log.Printf("initialize application: %v", err)
		return
	}
	if services != nil {
		defer services.Close()
	}

	err = wails.Run(&options.App{
		Title:  "Chess Trainer",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown: func(context.Context) {
			if services != nil {
				if err := services.Close(); err != nil {
					log.Printf("close application services: %v", err)
				}
			}
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Printf("run application: %v", err)
	}
}
