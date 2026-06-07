package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
)

//go:embed build/appicon.png
var icon []byte

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Setup file logger
	configDir, _ := os.UserConfigDir()
	logDir := filepath.Join(configDir, "Nanci")
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "app.log")
	fileLogger := logger.NewFileLogger(logFile)

	// Create an instance of the app structure
	app := NewApp()

	// Create application menu
	AppMenu := menu.NewMenu()
	FileMenu := AppMenu.AddSubmenu("Arquivo")
	FileMenu.AddText("Exportar Logs", nil, func(_ *menu.CallbackData) {
		path, err := app.ExportLogs()
		if err == nil && path != "" {
			runtime.EventsEmit(app.ctx, "notify-success", "Logs exportados com sucesso para: "+path)
		} else if err != nil {
			runtime.EventsEmit(app.ctx, "notify-error", "Erro ao exportar logs: "+err.Error())
		}
	})
	FileMenu.AddSeparator()
	FileMenu.AddText("Sair", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(app.ctx)
	})

	HelpMenu := AppMenu.AddSubmenu("Ajuda")
	HelpMenu.AddText("Sobre", nil, func(_ *menu.CallbackData) {
		runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "Sobre o Nanci Desktop",
			Message: fmt.Sprintf("Nanci Desktop %s\nCommit: %s\n\nSistema de sincronização de notas fiscais (NFSe).", buildinfo.Version, buildinfo.Commit),
		})
	})

	err := wails.Run(&options.App{
		Title:  "Nanci",
		Menu:   AppMenu,
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Logger:           fileLogger,
		LogLevel:         logger.DEBUG,
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "Nanci Desktop",
				Message: fmt.Sprintf("© 2026 Nanci. Todos os direitos reservados.\nVersão: %s\nSistema de sincronização de notas fiscais (NFSe).", buildinfo.Version),
				Icon:    icon,
			},
		},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
