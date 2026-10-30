package main

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"

	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/foundation/buildinfo"
)

// fakeUpdates records the updater calls and runs onDownload during
// DownloadAndInstall, so a test can start a sync in the middle.
type fakeUpdates struct {
	calls      []string
	release    *updater.Release
	checkErr   error
	installErr error
	restartErr error
	onDownload func()
}

func (f *fakeUpdates) Check(context.Context) (*updater.Release, error) {
	f.calls = append(f.calls, "Check")
	return f.release, f.checkErr
}

func (f *fakeUpdates) DownloadAndInstall(context.Context) error {
	f.calls = append(f.calls, "DownloadAndInstall")
	if f.onDownload != nil {
		f.onDownload()
	}
	return f.installErr
}

func (f *fakeUpdates) Restart(context.Context) error {
	f.calls = append(f.calls, "Restart")
	return f.restartErr
}

// nopHost is the smallest updater.Host: it drops the events, opens no window
// and never quits.
type nopHost struct{}

func (nopHost) Emit(string, ...any) bool                              { return true }
func (nopHost) OnEvent(string, func(any)) func()                      { return func() {} }
func (nopHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (nopHost) Quit()                                                 {}

func testKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv, base64.StdEncoding.EncodeToString(pub)
}

func TestUpdatableVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"0.6.0", "0.6.0", true},
		{"v1.2.3", "1.2.3", true},
		{"0.0.0-dev", "0.0.0-dev", false},
		{"1.2.3-rc1", "1.2.3-rc1", false},
		{"1.2", "1.2", false},
		{"", "", false},
		{"dev", "dev", false},
	}
	for _, tt := range tests {
		got, ok := updatableVersion(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("updatableVersion(%q) = %q, %v; want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSetupUpdaterIsOffForDevBuildsAndWithoutKey(t *testing.T) {
	_, _, key := testKey(t)
	for _, tt := range []struct{ version, key string }{
		{"0.0.0-dev", key},
		{"1.2.3-beta", key},
		{"1.2.3", ""},
	} {
		u := updater.New(nopHost{})
		engine, err := setupUpdater(u, tt.version, tt.key, "http://127.0.0.1/m.json")
		if !errors.Is(err, errUpdatesDisabled) || engine != nil {
			t.Errorf("setupUpdater(%q, key %q) = %v, %v; want nil, errUpdatesDisabled", tt.version, tt.key, engine, err)
		}
		if u.State() != updater.StateUnconfigured {
			t.Errorf("version %q: updater state %q, want it left unconfigured", tt.version, u.State())
		}
	}
}

func TestSetupUpdaterRejectsAMalformedKey(t *testing.T) {
	for _, key := range []string{"não é base64", base64.StdEncoding.EncodeToString([]byte("curta"))} {
		engine, err := setupUpdater(updater.New(nopHost{}), "1.2.3", key, "http://127.0.0.1/m.json")
		if err == nil || errors.Is(err, errUpdatesDisabled) || engine != nil {
			t.Errorf("key %q: setupUpdater = %v, %v; want an error", key, engine, err)
		}
	}
}

func TestSetupUpdaterConfiguresReleaseBuilds(t *testing.T) {
	_, _, key := testKey(t)
	u := updater.New(nopHost{})
	engine, err := setupUpdater(u, "v1.2.3", key, "http://127.0.0.1/m.json")
	if err != nil || engine == nil {
		t.Fatalf("setupUpdater = %v, %v; want the updater", engine, err)
	}
	if got := u.CurrentVersion(); got != "1.2.3" {
		t.Errorf("CurrentVersion = %q, want 1.2.3 (no v)", got)
	}
}

// TestCommittedUpdaterKeyIsValid keeps a committed key in the format
// setupUpdater reads; an empty key only turns updates off.
func TestCommittedUpdaterKeyIsValid(t *testing.T) {
	if updaterPublicKey == "" {
		t.Skip("updaterPublicKey is empty: updates are off")
	}
	key, err := base64.StdEncoding.DecodeString(updaterPublicKey)
	if err != nil || len(key) != ed25519.PublicKeySize {
		t.Fatalf("updaterPublicKey must be standard base64 of a raw %d-byte key", ed25519.PublicKeySize)
	}
}

func TestCheckForUpdateWhenUpdatesAreOff(t *testing.T) {
	_, a := newFakeApp(nil)

	got, err := a.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := desktopapi.UpdateInfo{CurrentVersion: buildinfo.Version}
	if got != want {
		t.Errorf("CheckForUpdate = %#v, want %#v", got, want)
	}
}

func TestCheckForUpdateMapsTheRelease(t *testing.T) {
	_, a := newFakeApp(nil)
	a.updates = &fakeUpdates{release: &updater.Release{Version: "0.7.0", Notes: "Novidades"}}

	got, err := a.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := desktopapi.UpdateInfo{Enabled: true, CurrentVersion: buildinfo.Version, Available: true, Version: "0.7.0", Notes: "Novidades"}
	if got != want {
		t.Errorf("CheckForUpdate = %#v, want %#v", got, want)
	}
}

func TestCheckForUpdateUpToDate(t *testing.T) {
	_, a := newFakeApp(nil)
	a.updates = &fakeUpdates{}

	got, err := a.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.Available || got.Version != "" {
		t.Errorf("CheckForUpdate = %#v, want enabled and nothing available", got)
	}
}

func TestCheckForUpdatePassesErrors(t *testing.T) {
	_, a := newFakeApp(nil)
	a.updates = &fakeUpdates{checkErr: errCore}

	got, err := a.CheckForUpdate(context.Background())
	if !errors.Is(err, errCore) {
		t.Fatalf("err = %v, want it to wrap errCore", err)
	}
	if got != (desktopapi.UpdateInfo{}) {
		t.Errorf("result = %#v, want the zero DTO", got)
	}
}

func TestInstallUpdateWhenUpdatesAreOff(t *testing.T) {
	_, a := newFakeApp(nil)

	err := a.InstallUpdate(context.Background())
	if !errors.Is(err, errUpdatesDisabled) {
		t.Fatalf("err = %v, want errUpdatesDisabled", err)
	}
}

func TestInstallUpdateDownloadsThenRestarts(t *testing.T) {
	_, a := newFakeApp(nil)
	updates := &fakeUpdates{}
	a.updates = updates

	if err := a.InstallUpdate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"DownloadAndInstall", "Restart"}; !reflect.DeepEqual(updates.calls, want) {
		t.Errorf("calls = %v, want %v", updates.calls, want)
	}
}

func TestInstallUpdateRefusesDuringSync(t *testing.T) {
	_, a := newFakeApp(nil)
	updates := &fakeUpdates{}
	a.updates = updates
	done, err := a.guard.start(true)
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	err = a.InstallUpdate(context.Background())
	if !errors.Is(err, errUpdateDuringSync) {
		t.Fatalf("err = %v, want errUpdateDuringSync", err)
	}
	if code := formatError(err).Code; code != "update_busy" {
		t.Errorf("code = %q, want update_busy", code)
	}
	if len(updates.calls) != 0 {
		t.Errorf("calls = %v, want none", updates.calls)
	}
}

func TestInstallUpdateRefusesDuringOtherOperations(t *testing.T) {
	_, a := newFakeApp(nil)
	updates := &fakeUpdates{}
	a.updates = updates
	done, err := a.guard.start(false)
	if err != nil {
		t.Fatal(err)
	}

	err = a.InstallUpdate(context.Background())
	if !errors.Is(err, errUpdateBusy) {
		t.Fatalf("err = %v, want errUpdateBusy", err)
	}
	if code := formatError(err).Code; code != "update_busy" {
		t.Errorf("code = %q, want update_busy", code)
	}
	if len(updates.calls) != 0 {
		t.Errorf("calls = %v, want none", updates.calls)
	}

	done()
	if err := a.InstallUpdate(context.Background()); err != nil {
		t.Fatalf("InstallUpdate after the operation ended: %v", err)
	}
}

// TestInstallUpdateRefusesOperationsWhileItRuns starts a sync and an export
// during the download: both are refused, so nothing can stop the restart.
func TestInstallUpdateRefusesOperationsWhileItRuns(t *testing.T) {
	f, a := newFakeApp(nil)
	var syncErr, exportErr error
	updates := &fakeUpdates{onDownload: func() {
		_, syncErr = a.Pull(context.Background(), desktopapi.PullInput{CNPJ: testCNPJ})
		_, exportErr = a.ExportXML(context.Background(), desktopapi.ExportXMLInput{CNPJ: testCNPJ, OutPath: "nota.xml"})
	}}
	a.updates = updates

	if err := a.InstallUpdate(context.Background()); err != nil {
		t.Fatal(err)
	}
	for name, err := range map[string]error{"Pull": syncErr, "ExportXML": exportErr} {
		if !errors.Is(err, errUpdateInstalling) {
			t.Errorf("%s err = %v, want errUpdateInstalling", name, err)
		}
	}
	f.assertNoCalls(t)
	if want := []string{"DownloadAndInstall", "Restart"}; !reflect.DeepEqual(updates.calls, want) {
		t.Errorf("calls = %v, want %v", updates.calls, want)
	}
	if err := a.InstallUpdate(context.Background()); !errors.Is(err, errUpdateInstalling) {
		t.Errorf("second InstallUpdate err = %v, want errUpdateInstalling", err)
	}
}

// TestInstallUpdateAndOperationNeverBothStart races an operation against an
// update. Exactly one of them may start: an update that starts while the
// operation runs would restart over it.
func TestInstallUpdateAndOperationNeverBothStart(t *testing.T) {
	for range 200 {
		_, a := newFakeApp(nil)
		a.updates = &fakeUpdates{}
		start := make(chan struct{})
		installed := make(chan error, 1)
		go func() {
			<-start
			installed <- a.InstallUpdate(context.Background())
		}()
		close(start)
		done, trackErr := a.guard.start(false)
		installErr := <-installed
		if done != nil {
			done()
		}
		if (trackErr == nil) == (installErr == nil) {
			t.Fatalf("track err = %v, InstallUpdate err = %v; want exactly one to succeed", trackErr, installErr)
		}
	}
}
func TestInstallUpdateWithoutACheck(t *testing.T) {
	_, a := newFakeApp(nil)
	a.updates = &fakeUpdates{installErr: updater.ErrNoPendingRelease}

	if err := a.InstallUpdate(context.Background()); !errors.Is(err, errNoUpdate) {
		t.Fatalf("err = %v, want errNoUpdate", err)
	}
}

func TestInstallUpdatePassesErrors(t *testing.T) {
	for _, tt := range []struct {
		name    string
		updates *fakeUpdates
		calls   []string
	}{
		{"download", &fakeUpdates{installErr: errCore}, []string{"DownloadAndInstall"}},
		{"restart", &fakeUpdates{restartErr: errCore}, []string{"DownloadAndInstall", "Restart"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, a := newFakeApp(nil)
			a.updates = tt.updates
			if err := a.InstallUpdate(context.Background()); !errors.Is(err, errCore) {
				t.Fatalf("err = %v, want it to wrap errCore", err)
			}
			if !reflect.DeepEqual(tt.updates.calls, tt.calls) {
				t.Errorf("calls = %v, want %v", tt.updates.calls, tt.calls)
			}
			// A failed install lets operations, and a new attempt, start again.
			done, err := a.guard.start(false)
			if err != nil {
				t.Fatalf("track after a failed install: %v", err)
			}
			done()
			tt.updates.installErr, tt.updates.restartErr = nil, nil
			if err := a.InstallUpdate(context.Background()); err != nil {
				t.Errorf("InstallUpdate after a failed one: %v", err)
			}
		})
	}
}

// manifestArtifact is one artifact of the manifest `wails3 updater manifest`
// writes (internal/commands/updater_tool.go of Wails v3.0.0,
// updaterManifestArtifact).
type manifestArtifact struct {
	URL           string `json:"url"`
	Platform      string `json:"platform"`
	Arch          string `json:"arch"`
	Size          int64  `json:"size"`
	DigestAlgo    string `json:"digestAlgo"`
	Digest        string `json:"digest"`
	SignatureAlgo string `json:"signatureAlgo"`
	Signature     string `json:"signature"`
}

// signLikeWails signs payload the way `wails3 updater sign` and `manifest`
// do (signUpdaterArtifact in the same file): Ed25519ph over the SHA-512
// digest, both base64.
func signLikeWails(t *testing.T, priv ed25519.PrivateKey, payload []byte) (digest, signature string) {
	t.Helper()
	sum := sha512.Sum512(payload)
	sig, err := priv.Sign(nil, sum[:], &ed25519.Options{Hash: crypto.SHA512})
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(sum[:]), base64.StdEncoding.EncodeToString(sig)
}

// serveRelease serves a manifest for version 9.9.9 listing payload, signed
// with priv, and the payload itself.
func serveRelease(t *testing.T, priv ed25519.PrivateKey, payload, served []byte) string {
	t.Helper()
	digest, signature := signLikeWails(t, priv, payload)
	mux := http.NewServeMux()
	var base string
	mux.HandleFunc("/nanci-update.json", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schemaVersion": 1,
			"version":       "9.9.9",
			"notes":         "Notas",
			"artifacts": []manifestArtifact{{
				// The updater picks the artifact of the platform it runs on, and
				// CI runs these tests on Linux.
				URL: base + "/nanci-desktop-windows-amd64-v9.9.9.exe", Platform: runtime.GOOS, Arch: runtime.GOARCH,
				Size: int64(len(served)), DigestAlgo: "sha512", Digest: digest,
				SignatureAlgo: "ed25519ph", Signature: signature,
			}},
		})
	})
	mux.HandleFunc("/nanci-desktop-windows-amd64-v9.9.9.exe", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(served)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	base = srv.URL
	return srv.URL + "/nanci-update.json"
}

// TestUpdaterVerifiesAManifestSignedLikeWails runs the real updater and
// endpoint provider against a manifest in the format of the release
// workflow, with the key in the format of updaterPublicKey. It stages the
// download beside the test binary and removes it afterwards.
func TestUpdaterVerifiesAManifestSignedLikeWails(t *testing.T) {
	_, priv, key := testKey(t)
	payload := []byte("MZ nova versão")
	url := serveRelease(t, priv, payload, payload)

	u := updater.New(nopHost{})
	engine, err := setupUpdater(u, "1.0.0", key, url)
	if err != nil {
		t.Fatal(err)
	}
	release, err := engine.Check(context.Background())
	if err != nil || release == nil || release.Version != "9.9.9" || release.Notes != "Notas" {
		t.Fatalf("Check = %#v, %v; want release 9.9.9", release, err)
	}
	if err := engine.DownloadAndInstall(context.Background()); err != nil {
		t.Fatalf("DownloadAndInstall: %v", err)
	}
	staged := u.DownloadedPath()
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(staged)) })
	got, err := os.ReadFile(staged) // #nosec G304 -- the updater's own staging path in this test.
	if err != nil || string(got) != string(payload) {
		t.Errorf("staged %q = %q, %v; want the payload", staged, got, err)
	}
}

func TestUpdaterRefusesATamperedRelease(t *testing.T) {
	_, priv, key := testKey(t)
	_, otherPriv, _ := testKey(t)
	payload := []byte("MZ nova versão")
	for _, tt := range []struct {
		name   string
		signer ed25519.PrivateKey
		served []byte
		want   string
	}{
		{"other key", otherPriv, payload, "signature did not verify"},
		{"altered payload", priv, []byte("MZ nova versãO"), "digest mismatch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			url := serveRelease(t, tt.signer, payload, tt.served)
			u := updater.New(nopHost{})
			engine, err := setupUpdater(u, "1.0.0", key, url)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := engine.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			err = engine.DownloadAndInstall(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("DownloadAndInstall = %v, want an error containing %q", err, tt.want)
			}
			if u.DownloadedPath() != "" {
				t.Errorf("staged %q, want nothing staged", u.DownloadedPath())
			}
		})
	}
}
