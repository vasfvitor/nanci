// Desktop runtime methods of App: the certificate password prompt, the file
// dialogs, build info, the data and log folders, the log level and the log
// export.

package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	logpkg "github.com/vasfvitor/nanci/internal/foundation/logger"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
)

// WailsCredentialProvider implements app.CredentialProvider using Wails frontend interaction.
// It owns the passwordChans map and its mutex; the App delegates Submit/Cancel calls to it.
type WailsCredentialProvider struct {
	events        eventEmitter // NewApp sets it to the App's emitter
	passwordChans map[string]chan []byte
	mu            sync.Mutex
}

func newWailsCredentialProvider() *WailsCredentialProvider {
	return &WailsCredentialProvider{
		passwordChans: make(map[string]chan []byte),
	}
}

// GetCertPassword asks the frontend for the certificate password and blocks until one is provided.
// The caller owns the returned slice and should zero it with cert.ZeroBytes once done.
func (p *WailsCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) ([]byte, error) {
	ch := make(chan []byte, 1)

	p.mu.Lock()
	p.passwordChans[req.RequestID] = ch
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.passwordChans, req.RequestID)
		p.mu.Unlock()
	}()

	// Notify the frontend to show the password dialog
	p.events.Emit(eventRequestCertPassword, req)

	// Block until the password is submitted by the frontend
	select {
	case pass := <-ch:
		// CancelPassword sends nil; an empty submission means the same thing.
		if len(pass) == 0 {
			return nil, app.ErrOperationCanceled
		}
		return pass, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SubmitPassword receives the password from the frontend dialog and unblocks GetCertPassword.
// Wails marshals the password from JS as a string, so the copy taken here is the
// first one that can be zeroed; the string itself cannot be.
func (p *WailsCredentialProvider) SubmitPassword(reqID, password string) {
	p.mu.Lock()
	ch, ok := p.passwordChans[reqID]
	p.mu.Unlock()

	if !ok {
		return
	}

	pass := []byte(password)
	select {
	case ch <- pass:
	default:
		// Nobody is waiting any more; do not leave the password in memory.
		cert.ZeroBytes(pass)
	}
}

// CancelPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (p *WailsCredentialProvider) CancelPassword(reqID string) {
	p.mu.Lock()
	ch, ok := p.passwordChans[reqID]
	p.mu.Unlock()

	if !ok {
		return
	}

	select {
	case ch <- nil:
	default:
	}
}

// SubmitCertPassword receives the password from the frontend dialog and unblocks GetCertPassword
func (a *App) SubmitCertPassword(reqID string, password string) {
	a.cred.SubmitPassword(reqID, password)
}

// CancelCertPassword receives a cancellation from the frontend and unblocks GetCertPassword
func (a *App) CancelCertPassword(reqID string) {
	a.cred.CancelPassword(reqID)
}

// SelectCertificate opens a file dialog to select a .pfx or .p12 file
func (a *App) SelectCertificate() (string, error) {
	return dialogResult(a.wails.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:          "Selecione o Certificado Digital",
		CanChooseFiles: true,
		Window:         a.window,
		Filters: []application.FileFilter{
			{DisplayName: "Certificados (*.pfx; *.p12)", Pattern: "*.pfx;*.p12"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	}).PromptForSingleSelection())
}

// SelectXMLFiles opens a dialog to pick one or more XML files to import.
// It returns an empty list when the user cancels.
func (a *App) SelectXMLFiles() ([]string, error) {
	return dialogResult(a.wails.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                   "Selecione os XMLs",
		CanChooseFiles:          true,
		AllowsMultipleSelection: true,
		Window:                  a.window,
		Filters: []application.FileFilter{
			{DisplayName: "XML (*.xml)", Pattern: "*.xml"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	}).PromptForMultipleSelection())
}

// SelectSaveFile opens a dialog to select an output file path for exports
func (a *App) SelectSaveFile(title, defaultFilename, pattern string) (string, error) {
	return dialogResult(a.saveFileDialog(&application.SaveFileDialogOptions{
		Title:    title,
		Filename: defaultFilename,
		Filters: []application.FileFilter{
			{DisplayName: title, Pattern: pattern},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	}))
}

// saveFileDialog shows a save dialog attached to the main window.
func (a *App) saveFileDialog(options *application.SaveFileDialogOptions) (string, error) {
	options.CanCreateDirectories = true
	options.Window = a.window
	return a.wails.Dialog.SaveFileWithOptions(options).PromptForSingleSelection()
}

// SetWindowTheme applies the theme picked in Configurações ("dark", "light" or
// "system") to the native window. Wails v3 only sets the window theme when the
// window is created, and its JS runtime has no theme calls.
func (a *App) SetWindowTheme(theme string) {
	dark, fixed := fixedWindowTheme(theme)
	a.themeFixed.Store(fixed)
	if !fixed {
		dark = systemPrefersDark()
	}
	setNativeDarkMode(a.window, dark)
}

// keepFixedTheme is a hook on events.Windows.SystemThemeChanged, which Wails
// handles by setting the window to the new system theme. Hooks run
// synchronously before the listeners and a cancelled event reaches none, so
// cancelling is the only deterministic way to keep a dark or light theme
// picked in Configurações.
func (a *App) keepFixedTheme(event *application.ApplicationEvent) {
	if a.themeFixed.Load() {
		event.Cancel()
	}
}

func (a *App) SetLogLevel(level string) {
	a.logLevel.Set(parseDesktopLogLevel(level))
}

func parseDesktopLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return logpkg.LevelTrace
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}

func (a *App) ExportLogs() (string, error) {
	if a.logPath == "" {
		return "", fmt.Errorf("logger de desktop não configurado")
	}

	savePath, err := dialogResult(a.saveFileDialog(&application.SaveFileDialogOptions{
		Title:    "Exportar Logs",
		Filename: "nanci_desktop_logs.zip",
		Filters: []application.FileFilter{
			{DisplayName: "Arquivos ZIP (*.zip)", Pattern: "*.zip"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	}))
	if err != nil || savePath == "" {
		return "", err
	}

	return savePath, exportRotatedLogs(savePath, a.logPath, diagnosticSummary(a.GetBuildInfo()))
}

// diagnosticSummaryName is the ZIP entry that opens the diagnostic package
// with the build and the runtime the logs came from.
const diagnosticSummaryName = "diagnostico.txt"

func diagnosticSummary(info desktopapi.BuildInfo) []byte {
	webView2 := info.WebView2Version
	if webView2 == "" {
		webView2 = "não encontrado"
	}
	return fmt.Appendf(nil, "Nanci %s\nCommit: %s\nBuild: %s\nSistema: %s/%s\nWebView2: %s\n",
		info.Version, info.Commit, info.Date, goruntime.GOOS, goruntime.GOARCH, webView2)
}

// exportRotatedLogs writes the rotated logs, masked, to a ZIP at savePath,
// followed by the summary as diagnostico.txt.
func exportRotatedLogs(savePath string, basePath string, summary []byte) error {
	file, err := os.Create(savePath) // #nosec G304 -- the user picks savePath in the save dialog.
	if err != nil {
		return fmt.Errorf("criar arquivo de exportação: %w", err)
	}
	defer func() { _ = file.Close() }()

	archive := zip.NewWriter(file)

	added := 0
	for _, path := range collectRotatedLogPaths(basePath, logFileMaxBackups) {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat log %s: %w", path, err)
		}
		if info.Size() == 0 {
			continue
		}

		content, err := os.ReadFile(path) // #nosec G304 -- path is one of the app's own rotated log files.
		if err != nil {
			return fmt.Errorf("ler log %s: %w", path, err)
		}

		entry, err := archive.Create(filepath.Base(path))
		if err != nil {
			return fmt.Errorf("criar entrada zip %s: %w", path, err)
		}
		// Logs on disk keep CNPJs in clear; the exported copy is masked.
		if _, err := entry.Write(sanitizeLogContent(content)); err != nil {
			return fmt.Errorf("escrever entrada zip %s: %w", path, err)
		}
		added++
	}

	if added == 0 {
		return fmt.Errorf("arquivo de log não encontrado")
	}

	entry, err := archive.Create(diagnosticSummaryName)
	if err != nil {
		return fmt.Errorf("criar entrada zip %s: %w", diagnosticSummaryName, err)
	}
	if _, err := entry.Write(summary); err != nil {
		return fmt.Errorf("escrever entrada zip %s: %w", diagnosticSummaryName, err)
	}

	if err := archive.Close(); err != nil {
		return fmt.Errorf("fechar zip: %w", err)
	}
	return nil
}

func (a *App) GetBuildInfo() desktopapi.BuildInfo {
	return desktopapi.BuildInfo{
		Version:         buildinfo.Version,
		Commit:          buildinfo.Commit,
		Date:            buildinfo.Date,
		WebView2Version: a.webView2Version(),
	}
}

// webView2Version returns the version of the installed WebView2 runtime, as
// Wails finds it when it starts the webview ("141.0.3537.71", plus the
// channel name for Beta, Dev or Canary), or "" when it cannot be read, off
// Windows (macOS and Linux use WebKit) or before main sets up Wails. The app
// ships no fixed runtime.
func (a *App) webView2Version() string {
	if a.wails == nil {
		return ""
	}
	version, _ := a.wails.Env.Info().PlatformInfo["WebView2"].(string)
	return version
}

func (a *App) GetDataDirectory() (string, error) {
	return paths.DataDir()
}

func openDir(dir string) error {
	opener := "xdg-open" // linux, freebsd, etc.
	switch goruntime.GOOS {
	case "windows":
		opener = "explorer"
	case "darwin":
		opener = "open"
	}
	// #nosec G204 -- fixed opener per OS; dir is the app's data or log directory, passed without a shell.
	cmd := exec.Command(opener, dir) //nolint:noctx // fire-and-forget launch of the file browser; nothing to cancel.
	return cmd.Start()
}

// ShowInFolder opens the file browser on the folder of path, with the file
// selected where the OS allows it. The export toast calls it with the
// exported file.
func (a *App) ShowInFolder(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("arquivo não encontrado: %s", path)
	}
	if err != nil {
		return fmt.Errorf("verificar arquivo %s: %w", path, err)
	}
	name, args := showInFolderCommand(goruntime.GOOS, path, info.IsDir())
	// #nosec G204 -- fixed opener per OS; path is a file the app exported, passed without a shell.
	cmd := exec.Command(name, args...) //nolint:noctx // fire-and-forget launch of the file browser; nothing to cancel.
	return cmd.Start()
}

// showInFolderCommand returns the command that shows path in the file
// browser of goos: Explorer and Finder select the file, other systems open
// its folder.
func showInFolderCommand(goos, path string, isDir bool) (name string, args []string) {
	dir := path
	if !isDir {
		dir = filepath.Dir(path)
	}
	switch goos {
	case "windows":
		if isDir {
			return "explorer", []string{path}
		}
		return "explorer", []string{"/select," + path}
	case "darwin":
		return "open", []string{"-R", path}
	default:
		return "xdg-open", []string{dir}
	}
}

func (a *App) OpenDataDirectory() error {
	dir, err := paths.DataDir()
	if err != nil {
		return err
	}
	return openDir(dir)
}

func (a *App) OpenLogsDirectory() error {
	dir, err := desktopLogDir()
	if err != nil {
		return err
	}
	return openDir(dir)
}
