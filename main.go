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
	application, err := newApplication()
	if err != nil {
		log.Printf("initialize application: %v", err)
		return
	}
	defer application.Close()

	err = wails.Run(&options.App{
		Title:  "Chess Trainer",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        application.startup,
		OnShutdown: func(context.Context) {
			if err := application.Close(); err != nil {
				log.Printf("close application services: %v", err)
			}
		},
		Bind: application.Bindings(),
	})

	if err != nil {
		log.Printf("run application: %v", err)
	}
}
