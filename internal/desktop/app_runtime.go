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

	"github.com/wailsapp/wails/v2/pkg/runtime"

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
	ctx           context.Context
	events        eventEmitter
	passwordChans map[string]chan []byte
	mu            sync.Mutex
}

func newWailsCredentialProvider() *WailsCredentialProvider {
	return &WailsCredentialProvider{
		events:        wailsEvents{},
		passwordChans: make(map[string]chan []byte),
	}
}

func (p *WailsCredentialProvider) setCtx(ctx context.Context) { p.ctx = ctx }

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
	p.events.Emit(p.ctx, "request-cert-password", req) //nolint:contextcheck // Wails runtime calls need the app context from startup; ctx only bounds the wait.

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
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selecione o Certificado Digital",
		Filters: []runtime.FileFilter{
			{DisplayName: "Certificados (*.pfx; *.p12)", Pattern: "*.pfx;*.p12"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
}

// SelectXMLFiles opens a dialog to pick one or more XML files to import.
// It returns an empty list when the user cancels.
func (a *App) SelectXMLFiles() ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Selecione os XMLs",
		Filters: []runtime.FileFilter{
			{DisplayName: "XML (*.xml)", Pattern: "*.xml"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
}

// SelectSaveFile opens a dialog to select an output file path for exports
func (a *App) SelectSaveFile(title, defaultFilename, pattern string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: title, Pattern: pattern},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
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

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exportar Logs",
		DefaultFilename: "nanci_desktop_logs.zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "Arquivos ZIP (*.zip)", Pattern: "*.zip"},
			{DisplayName: "Todos os Arquivos", Pattern: "*.*"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}

	return savePath, exportRotatedLogs(savePath, a.logPath)
}

func exportRotatedLogs(savePath string, basePath string) error {
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

	if err := archive.Close(); err != nil {
		return fmt.Errorf("fechar zip: %w", err)
	}
	return nil
}

func (a *App) GetBuildInfo() desktopapi.BuildInfo {
	return desktopapi.BuildInfo{
		Version: buildinfo.Version,
		Commit:  buildinfo.Commit,
		Date:    buildinfo.Date,
	}
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
