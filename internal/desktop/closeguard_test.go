package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

// blockingSync is a fakeSync whose Pull signals started and then waits for
// release, so a test can look at the App while a sync runs.
type blockingSync struct {
	*fakeSync
	started chan struct{}
	release chan struct{}
}

func (b *blockingSync) Pull(ctx context.Context, in nsync.PullInput) (nsync.PullResult, error) {
	close(b.started)
	<-b.release
	return b.fakeSync.Pull(ctx, in)
}

// guardedApp returns a fake App whose events and quit are recorded.
func guardedApp(t *testing.T) (fakeSet, *App, *fakeEvents, *int) {
	t.Helper()
	f, a := newFakeApp(nil)
	events := &fakeEvents{emitted: make(chan call, 1)}
	a.events = events
	quits := 0
	a.quit = func() { quits++ }
	return f, a, events, &quits
}

func TestBeforeCloseAllowsWhenIdle(t *testing.T) {
	_, a, events, _ := guardedApp(t)

	if a.beforeClose() {
		t.Fatal("beforeClose = true with no sync running, want false")
	}
	select {
	case got := <-events.emitted:
		t.Errorf("emitted %#v, want nothing", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBeforeClosePreventsAndAsksDuringPull(t *testing.T) {
	f, a, events, _ := guardedApp(t)
	sync := &blockingSync{fakeSync: f.sync, started: make(chan struct{}), release: make(chan struct{})}
	a.svc.sync = sync

	done := make(chan error, 1)
	go func() {
		_, err := a.Pull(context.Background(), desktopapi.PullInput{CNPJ: testCNPJ})
		done <- err
	}()
	<-sync.started

	if !a.beforeClose() {
		t.Fatal("beforeClose = false during a sync, want true")
	}
	select {
	case got := <-events.emitted:
		want := call{"app:before-close", []any{1}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("emitted %#v, want %#v", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("beforeClose did not ask the frontend")
	}

	close(sync.release)
	if err := <-done; err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if a.beforeClose() {
		t.Error("beforeClose = true after the sync finished, want false")
	}
}

func TestPullReleasesGuardOnError(t *testing.T) {
	_, a := newFakeApp(errCore)
	runs := map[string]func() error{
		"Pull": func() error {
			_, err := a.Pull(context.Background(), desktopapi.PullInput{CNPJ: testCNPJ})
			return err
		},
		"PullNFe": func() error {
			_, err := a.PullNFe(context.Background(), desktopapi.PullNFeInput{CNPJ: testCNPJ})
			return err
		},
		"PullCTe": func() error {
			_, err := a.PullCTe(context.Background(), desktopapi.PullCTeInput{CNPJ: testCNPJ})
			return err
		},
	}
	for name, run := range runs {
		if err := run(); err == nil {
			t.Errorf("%s: err = nil, want %v", name, errCore)
		}
		if n := a.guard.pulls.Load(); n != 0 {
			t.Errorf("%s: pulls = %d after the error, want 0", name, n)
		}
	}
	if a.beforeClose() {
		t.Error("beforeClose = true after failed syncs, want false")
	}
}

func TestConfirmQuitAllowsClose(t *testing.T) {
	_, a, events, quits := guardedApp(t)
	done, err := a.trackPull()
	if err != nil {
		t.Fatal(err)
	}
	defer done()

	a.ConfirmQuit()

	if *quits != 1 {
		t.Fatalf("quit called %d times, want 1", *quits)
	}
	if a.beforeClose() {
		t.Error("beforeClose = true after ConfirmQuit, want false")
	}
	select {
	case got := <-events.emitted:
		t.Errorf("emitted %#v, want nothing", got)
	case <-time.After(50 * time.Millisecond):
	}
}

// TestLongOperationsAreRefusedWhileAnUpdateInstalls checks that every long
// method is tracked: once an update is installing, each one is refused
// before it reaches the core.
func TestLongOperationsAreRefusedWhileAnUpdateInstalls(t *testing.T) {
	f, a := newFakeApp(nil)
	if err := a.guard.beginInstall(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	files := []string{"nota.xml"}
	runs := map[string]func() error{
		"Pull": func() error {
			_, err := a.Pull(ctx, desktopapi.PullInput{CNPJ: testCNPJ})
			return err
		},
		"PullNFe": func() error {
			_, err := a.PullNFe(ctx, desktopapi.PullNFeInput{CNPJ: testCNPJ})
			return err
		},
		"PullCTe": func() error {
			_, err := a.PullCTe(ctx, desktopapi.PullCTeInput{CNPJ: testCNPJ})
			return err
		},
		"ImportNFeXML": func() error {
			_, err := a.ImportNFeXML(ctx, desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: files})
			return err
		},
		"ImportCTeXML": func() error {
			_, err := a.ImportCTeXML(ctx, desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: files})
			return err
		},
		"FetchNFeByChave": func() error {
			_, err := a.FetchNFeByChave(ctx, desktopapi.FetchByChaveInput{CNPJ: testCNPJ, ChaveAcesso: "chave"})
			return err
		},
		"QueryNFSeEvents": func() error {
			_, err := a.QueryNFSeEvents(ctx, desktopapi.QueryNFSeInput{CompanyCNPJ: testCNPJ})
			return err
		},
		"TestConnection": func() error {
			_, err := a.TestConnection(ctx, testCNPJ)
			return err
		},
		"TestCTeConnection": func() error {
			_, err := a.TestCTeConnection(ctx, testCNPJ)
			return err
		},
		"PlanNFeCiencia": func() error {
			_, err := a.PlanNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ})
			return err
		},
		"RegisterNFeCiencia": func() error {
			_, err := a.RegisterNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ})
			return err
		},
		"RegisterNFeManifestacao": func() error {
			_, err := a.RegisterNFeManifestacao(desktopapi.RegisterNFeManifestacaoInput{CNPJ: testCNPJ})
			return err
		},
		"ResetSyncState": func() error {
			return a.ResetSyncState(desktopapi.ResetSyncInput{CompanyCNPJ: testCNPJ})
		},
		"ResetNFe": func() error {
			_, err := a.ResetNFe(testCNPJ)
			return err
		},
		"PreviewResetCTe": func() error {
			_, err := a.PreviewResetCTe(testCNPJ)
			return err
		},
		"ResetCTe": func() error {
			_, err := a.ResetCTe(testCNPJ)
			return err
		},
		"ExportDANFSe": func() error {
			_, err := a.ExportDANFSe(ctx, desktopapi.ExportDANFSeInput{CNPJ: testCNPJ, OutPath: "n.pdf"})
			return err
		},
		"ExportXML": func() error {
			_, err := a.ExportXML(ctx, desktopapi.ExportXMLInput{CNPJ: testCNPJ, OutPath: "n.xml"})
			return err
		},
		"ExportDocuments": func() error {
			_, err := a.ExportDocuments(ctx, desktopapi.ExportDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: "n.csv"})
			return err
		},
		"ExportNFeXML": func() error {
			_, err := a.ExportNFeXML(ctx, desktopapi.ExportNFeXMLInput{CNPJ: testCNPJ, OutPath: "n.xml"})
			return err
		},
		"ExportDANFE": func() error {
			_, err := a.ExportDANFE(ctx, desktopapi.ExportDANFEInput{CNPJ: testCNPJ, OutPath: "n.pdf"})
			return err
		},
		"ExportNFeDocuments": func() error {
			_, err := a.ExportNFeDocuments(ctx, desktopapi.ExportNFeDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: "n.csv"})
			return err
		},
		"ExportCTeXML": func() error {
			_, err := a.ExportCTeXML(ctx, desktopapi.ExportCTeXMLInput{CNPJ: testCNPJ, OutPath: "n.xml"})
			return err
		},
		"ExportDACTE": func() error {
			_, err := a.ExportDACTE(ctx, desktopapi.ExportDACTEInput{CNPJ: testCNPJ, OutPath: "n.pdf"})
			return err
		},
		"ExportCTeDocuments": func() error {
			_, err := a.ExportCTeDocuments(ctx, desktopapi.ExportCTeDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: "n.csv"})
			return err
		},
	}
	for name, run := range runs {
		if err := run(); !errors.Is(err, errUpdateInstalling) {
			t.Errorf("%s: err = %v, want errUpdateInstalling", name, err)
		}
	}
	f.assertNoCalls(t)
}
