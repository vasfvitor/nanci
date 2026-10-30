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

// updateManifestURL is the update manifest of the latest stable release.
// GitHub redirects releases/latest/download to the asset of the newest
// release that is neither a draft nor a prerelease. The release workflow
// writes the manifest with `wails3 updater manifest`, which signs the exe it
// lists; a release without the manifest answers 404, which the endpoint
// provider reads as "up to date".
//
// The github provider of Wails cannot be used with a pinned key: it fills
// only a digest from a checksums file, never a signature
// (pkg/updater/providers/github/github.go:167-179), and with Config.PublicKey
// set the updater rejects every release without one (pkg/updater/verify.go:87).
//
// It is a var, not a const, so a local test build can point at a test server
// with -ldflags "-X main.updateManifestURL=...".
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
	// errUpdateDuringSync keeps the restart from cutting a sync short.
	errUpdateDuringSync = errors.New("aguarde a sincronização terminar para atualizar")
	errNoUpdate         = errors.New("nenhuma atualização disponível; procure atualizações de novo")
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
// verifies its signature, stages it beside the exe and restarts into it. A
// helper copy of the exe waits for this process to exit, swaps the files and
// starts the new version; on any failure it puts the old exe back.
//
// It refuses while a sync runs, before the download and again before the
// restart: the restart goes through Options.ShouldQuit, and beforeClose would
// stop it to ask the user, leaving the helper waiting until it gives up.
func (a *App) InstallUpdate(ctx context.Context) error {
	if a.updates == nil {
		return errUpdatesDisabled
	}
	if a.guard.pulls.Load() > 0 {
		return errUpdateDuringSync
	}
	ctx, cancel := a.callContext(ctx)
	defer cancel()
	if err := a.updates.DownloadAndInstall(ctx); err != nil {
		if errors.Is(err, updater.ErrNoPendingRelease) {
			return errNoUpdate
		}
		return fmt.Errorf("baixar atualização: %w", err)
	}
	if a.guard.pulls.Load() > 0 {
		return errUpdateDuringSync
	}
	if err := a.updates.Restart(ctx); err != nil {
		return fmt.Errorf("reiniciar para atualizar: %w", err)
	}
	return nil
}
