package main

import (
	"context"
	"embed"
	"log"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	paths, err := storage.DefaultPaths()
	if err != nil {
		log.Printf("resolve application data paths: %v", err)
		return
	}
	services, err := appservices.Open(paths)
	if err != nil {
		log.Printf("open application services: %v", err)
		return
	}
	defer services.Close()
	app := NewApp(services)

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
			if err := services.Close(); err != nil {
				log.Printf("close application services: %v", err)
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
