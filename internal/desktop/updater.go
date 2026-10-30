package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"

	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
)

// updateManifestURL is the signed update manifest of the latest stable
// release; GitHub redirects releases/latest/download to the newest release
// that is not a draft or a prerelease, and a release without the manifest
// answers 404, which the endpoint provider reads as "up to date". The github
// provider of Wails cannot be used: it never fills a signature
// (pkg/updater/providers/github/github.go:167-179), which a pinned
// Config.PublicKey requires. It is a var so a local test build can point at
// a test server with -ldflags "-X main.updateManifestURL=...".
var updateManifestURL = "https://github.com/vasfvitor/nanci/releases/latest/download/nanci-update.json"

// updateEngine is the part of the Wails updater App uses; tests replace it.
type updateEngine interface {
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
}

var (
	// errUpdatesDisabled: a dev build, a version that is not X.Y.Z or a
	// build without updaterPublicKey.
	errUpdatesDisabled = errors.New("atualização automática indisponível nesta versão")
	errNoUpdate        = errors.New("nenhuma atualização disponível; procure atualizações de novo")
)

var releaseVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// updatableVersion returns the version the updater compares against the
// manifest: buildinfo.Version without a leading "v". Only release builds
// update; a dev build ("0.0.0-dev") or any other version returns false.
func updatableVersion(version string) (string, bool) {
	v := strings.TrimPrefix(version, "v")
	return v, releaseVersion.MatchString(v)
}

// setupUpdater configures the Wails updater for a release build that has a
// public key, and returns errUpdatesDisabled for any other build. No update
// window is configured: Check and DownloadAndInstall open none, and the
// frontend shows its own notice.
func setupUpdater(u *updater.Updater, version, publicKey, manifestURL string) (updateEngine, error) {
	current, ok := updatableVersion(version)
	if !ok || publicKey == "" {
		return nil, errUpdatesDisabled
	}
	key, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("chave pública de atualização inválida: esperada base64 de %d bytes", ed25519.PublicKeySize)
	}
	provider, err := endpoint.New(endpoint.Config{URL: manifestURL})
	if err != nil {
		return nil, err
	}
	err = u.Init(updater.Config{
		CurrentVersion: current,
		Providers:      []updater.Provider{provider},
		PublicKey:      key,
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

// CheckForUpdate asks the manifest for a newer release. With updates off it
// returns Enabled false and makes no request.
func (a *App) CheckForUpdate(ctx context.Context) (desktopapi.UpdateInfo, error) {
	info := desktopapi.UpdateInfo{CurrentVersion: buildinfo.Version}
	if a.updates == nil {
		return info, nil
	}
	release, err := a.updates.Check(ctx)
	if err != nil {
		return desktopapi.UpdateInfo{}, fmt.Errorf("procurar atualização: %w", err)
	}
	info.Enabled = true
	if release != nil {
		info.Available = true
		info.Version = release.Version
		info.Notes = release.Notes
	}
	return info, nil
}

// InstallUpdate downloads the release the last CheckForUpdate found,
// verifies its signature and restarts into it. It refuses while a tracked
// operation runs (closeguard.go) and blocks new ones until the restart, which
// would cut them short and, during a sync, be stopped by beforeClose. A
// failed download or restart lets operations start again.
func (a *App) InstallUpdate(ctx context.Context) error {
	if a.updates == nil {
		return errUpdatesDisabled
	}
	if err := a.guard.beginInstall(); err != nil {
		return err
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	if err := a.updates.DownloadAndInstall(ctx); err != nil {
		a.guard.endInstall()
		if errors.Is(err, updater.ErrNoPendingRelease) {
			return errNoUpdate
		}
		return fmt.Errorf("baixar atualização: %w", err)
	}
	if err := a.updates.Restart(ctx); err != nil {
		a.guard.endInstall()
		return fmt.Errorf("reiniciar para atualizar: %w", err)
	}
	return nil
}
