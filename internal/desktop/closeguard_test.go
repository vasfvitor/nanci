package main

import (
	"context"
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
	a.quit = func(context.Context) { quits++ }
	return f, a, events, &quits
}

func TestBeforeCloseAllowsWhenIdle(t *testing.T) {
	_, a, events, _ := guardedApp(t)

	if a.beforeClose(a.ctx) {
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
		_, err := a.Pull(desktopapi.PullInput{CNPJ: testCNPJ})
		done <- err
	}()
	<-sync.started

	if !a.beforeClose(a.ctx) {
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
	if a.beforeClose(a.ctx) {
		t.Error("beforeClose = true after the sync finished, want false")
	}
}

func TestPullReleasesGuardOnError(t *testing.T) {
	_, a := newFakeApp(errCore)
	runs := map[string]func() error{
		"Pull": func() error {
			_, err := a.Pull(desktopapi.PullInput{CNPJ: testCNPJ})
			return err
		},
		"PullNFe": func() error {
			_, err := a.PullNFe(desktopapi.PullNFeInput{CNPJ: testCNPJ})
			return err
		},
		"PullCTe": func() error {
			_, err := a.PullCTe(desktopapi.PullCTeInput{CNPJ: testCNPJ})
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
	if a.beforeClose(a.ctx) {
		t.Error("beforeClose = true after failed syncs, want false")
	}
}

func TestConfirmQuitAllowsClose(t *testing.T) {
	_, a, events, quits := guardedApp(t)
	defer a.trackPull()()

	a.ConfirmQuit()

	if *quits != 1 {
		t.Fatalf("quit called %d times, want 1", *quits)
	}
	if a.beforeClose(a.ctx) {
		t.Error("beforeClose = true after ConfirmQuit, want false")
	}
	select {
	case got := <-events.emitted:
		t.Errorf("emitted %#v, want nothing", got)
	case <-time.After(50 * time.Millisecond):
	}
}
