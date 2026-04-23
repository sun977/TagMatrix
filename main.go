package main

import (
	"embed"
	"log"

	"TagMatrix/internal/model"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Initialize local SQLite database
	// "data.db" 会在应用程序当前目录生成，也可以根据系统平台设置特定的 AppData 目录
	err := model.InitDB("data.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v\n", err)
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "TagMatrix 数据打标系统",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
