package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
)

//go:embed build/appicon.png
var icon []byte

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Wails system messages go to their own file next to the desktop log.
	var wailsLog *slog.Logger
	if logDir, err := desktopLogDir(); err == nil {
		writer, err := newRotatingFileWriter(filepath.Join(logDir, wailsLogFileName), logFileMaxBytes, logFileMaxBackups)
		if err == nil {
			defer func() { _ = writer.Close() }()
			wailsLog = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelWarn}))
		}
	}

	app := NewApp()

	wailsApp := application.New(application.Options{
		Name:        "Nanci",
		Description: fmt.Sprintf("Nanci %s\nSistema de sincronização de documentos fiscais (NFS-e, NF-e e CT-e).", buildinfo.Version),
		Icon:        icon,
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Logger:       wailsLog,
		LogLevel:     slog.LevelWarn,
		MarshalError: marshalError,
		// Application.Quit (the title-bar X), Ctrl+Q and the menu "Sair"
		// end here; closing the window from the OS goes through the
		// WindowClosing hook below.
		ShouldQuit: func() bool { return !app.beforeClose() },
		// The frameless window has no menu bar on Windows, so the menu
		// accelerator never fires there. KeyBinding.Add would skip the
		// accelerator parsing these options get.
		KeyBindings: map[string]func(application.Window){
			"CmdOrCtrl+Q": func(application.Window) { app.quit() },
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	app.wails = wailsApp

	wailsApp.Menu.Set(newAppMenu(app))

	app.window = wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Nanci",
		Width:            1280,
		Height:           768,
		Frameless:        true,
		BackgroundType:   application.BackgroundTypeTranslucent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              "/",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent:   true,
				HideToolbarSeparator: true,
			},
		},
		Windows: application.WindowsWindow{
			Theme:        application.SystemDefault,
			BackdropType: application.Mica,
		},
	})
	app.window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if app.beforeClose() {
			event.Cancel()
		}
	})

	if err := wailsApp.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
	}
}

// newAppMenu builds the native menu. Windows does not draw a menu bar on a
// frameless window, so there it is only reachable through its accelerators.
func newAppMenu(app *App) *application.Menu {
	menu := application.NewMenu()

	fileMenu := menu.AddSubmenu("Arquivo")
	fileMenu.Add("Exportar Logs").OnClick(func(*application.Context) {
		path, err := app.ExportLogs()
		if err == nil && path != "" {
			app.events.Emit("notify-success", "Logs exportados com sucesso para: "+path)
		} else if err != nil {
			app.events.Emit("notify-error", "Erro ao exportar logs: "+err.Error())
		}
	})
	fileMenu.AddSeparator()
	fileMenu.Add("Sair").SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) {
		app.quit()
	})

	helpMenu := menu.AddSubmenu("Ajuda")
	helpMenu.Add("Sobre").OnClick(func(*application.Context) {
		app.wails.Dialog.Info().
			SetTitle("Sobre o Nanci").
			SetMessage(fmt.Sprintf("Nanci %s\nCommit: %s\n\nSistema de sincronização de documentos fiscais (NFS-e, NF-e e CT-e).", buildinfo.Version, buildinfo.Commit)).
			AttachToWindow(app.window).
			Show()
	})

	return menu
}
