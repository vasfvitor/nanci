package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

type callKey struct{}

func waitDone(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("context did not end")
	}
}

func TestCallContextEndsWithTheCall(t *testing.T) {
	a := newTestApp(services{})
	callCtx, cancelCall := context.WithCancel(context.WithValue(context.Background(), callKey{}, "janela"))
	ctx, cancel := a.callContext(callCtx)
	defer cancel()

	if got := ctx.Value(callKey{}); got != "janela" {
		t.Errorf("value = %v, want the call's value", got)
	}
	if ctx.Err() != nil {
		t.Fatalf("context ended before the call: %v", ctx.Err())
	}
	cancelCall()
	waitDone(t, ctx)
}

func TestCallContextEndsAtShutdown(t *testing.T) {
	a := newTestApp(services{})
	appCtx, shutdown := context.WithCancel(context.Background())
	a.ctx = appCtx
	ctx, cancel := a.callContext(context.Background())
	defer cancel()

	shutdown()
	waitDone(t, ctx)
}

func TestCallContextCancelLeavesTheAppContext(t *testing.T) {
	a := newTestApp(services{})
	appCtx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	a.ctx = appCtx
	ctx, cancel := a.callContext(context.Background())

	cancel()
	waitDone(t, ctx)
	if appCtx.Err() != nil {
		t.Errorf("app context ended with the call: %v", appCtx.Err())
	}
}

// waitingSync is a fakeSync whose Pull waits for its context to end, as the
// sync loop does between batches.
type waitingSync struct {
	*fakeSync
	started chan struct{}
}

func (s *waitingSync) Pull(ctx context.Context, _ nsync.PullInput) (nsync.PullResult, error) {
	close(s.started)
	<-ctx.Done()
	return nsync.PullResult{}, fmt.Errorf("sincronização: %w", ctx.Err())
}

// TestPullStopsWithItsCall cancels a Pull through the call context, as the
// frontend does, and through the app context, as shutdown does. Either way
// the core sees its context end and the error reaches the frontend with the
// canceled code.
func TestPullStopsWithItsCall(t *testing.T) {
	tests := []struct {
		name   string
		cancel func(cancelCall, shutdown context.CancelFunc)
	}{
		{"call canceled", func(cancelCall, _ context.CancelFunc) { cancelCall() }},
		{"app shut down", func(_, shutdown context.CancelFunc) { shutdown() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)
			sync := &waitingSync{fakeSync: f.sync, started: make(chan struct{})}
			a.svc.sync = sync
			appCtx, shutdown := context.WithCancel(context.Background())
			defer shutdown()
			a.ctx = appCtx
			callCtx, cancelCall := context.WithCancel(context.Background())
			defer cancelCall()

			done := make(chan error, 1)
			go func() {
				_, err := a.Pull(callCtx, desktopapi.PullInput{CNPJ: testCNPJ})
				done <- err
			}()
			<-sync.started
			tt.cancel(cancelCall, shutdown)

			select {
			case err := <-done:
				if code := formatError(err).Code; code != "canceled" {
					t.Errorf("code = %q (err %v), want canceled", code, err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Pull did not return")
			}
			if n := a.guard.pulls.Load(); n != 0 {
				t.Errorf("%d syncs still tracked, want 0", n)
			}
		})
	}
}
