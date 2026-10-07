package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
)

// waitingProvider returns a provider with one pending request, plus the channel
// GetCertPassword would be blocked on.
func waitingProvider(reqID string) (*WailsCredentialProvider, chan []byte) {
	p := newWailsCredentialProvider()
	ch := make(chan []byte, 1)
	p.passwordChans[reqID] = ch
	return p, ch
}

func TestWailsCredentialProviderSubmitPassword(t *testing.T) {
	p, ch := waitingProvider("req-1")

	p.SubmitPassword("req-1", "mockdata")

	pass := <-ch
	if !bytes.Equal(pass, []byte("mockdata")) {
		t.Fatalf("password = %q, want %q", pass, "mockdata")
	}
}

// TestWailsCredentialProviderCancelPassword pins the cancellation sentinel:
// CancelPassword sends nil, which GetCertPassword reads as a cancellation.
func TestWailsCredentialProviderCancelPassword(t *testing.T) {
	p, ch := waitingProvider("req-1")

	p.CancelPassword("req-1")

	pass := <-ch
	if len(pass) != 0 {
		t.Fatalf("cancellation delivered %q, want an empty password", pass)
	}
}

func TestWailsCredentialProviderUnknownRequest(t *testing.T) {
	p, ch := waitingProvider("req-1")

	// No request under this ID: both calls must be no-ops rather than blocking.
	p.SubmitPassword("req-other", "mockdata")
	p.CancelPassword("req-other")

	select {
	case pass := <-ch:
		t.Fatalf("pending request received %q from an unrelated request ID", pass)
	default:
	}
}

// TestWailsCredentialProviderSubmitTwice covers the dropped-send path: the
// second password cannot be delivered, and the first one must survive intact.
func TestWailsCredentialProviderSubmitTwice(t *testing.T) {
	p, ch := waitingProvider("req-1")

	p.SubmitPassword("req-1", "mockdata")
	p.SubmitPassword("req-1", "segunda")

	pass := <-ch
	if !bytes.Equal(pass, []byte("mockdata")) {
		t.Fatalf("password = %q, want the first submission %q", pass, "mockdata")
	}
	select {
	case extra := <-ch:
		t.Fatalf("channel delivered a second password %q", extra)
	default:
	}
}

type passwordResult struct {
	pass []byte
	err  error
}

// askPassword starts GetCertPassword and waits until it asked the frontend,
// so the caller can answer through the App.
func askPassword(t *testing.T, ctx context.Context, a *App, req app.CertPasswordRequest) <-chan passwordResult {
	t.Helper()
	events := &fakeEvents{emitted: make(chan call, 1)}
	a.cred.events = events

	done := make(chan passwordResult, 1)
	go func() {
		pass, err := a.cred.GetCertPassword(ctx, req)
		done <- passwordResult{pass, err}
	}()

	select {
	case got := <-events.emitted:
		want := call{"request-cert-password", []any{req}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("emitted %#v, want %#v", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetCertPassword did not ask the frontend")
	}
	return done
}

func waitPassword(t *testing.T, done <-chan passwordResult) passwordResult {
	t.Helper()
	select {
	case res := <-done:
		return res
	case <-time.After(5 * time.Second):
		t.Fatal("GetCertPassword did not return")
		return passwordResult{}
	}
}

func TestGetCertPassword(t *testing.T) {
	req := app.CertPasswordRequest{RequestID: "req-1", CompanyName: "Empresa Teste", TargetCNPJ: testCNPJ}

	tests := []struct {
		name     string
		answer   func(a *App)
		wantPass string
		wantErr  error
	}{
		{"submit", func(a *App) { a.SubmitCertPassword("req-1", "mockdata") }, "mockdata", nil},
		{"cancel", func(a *App) { a.CancelCertPassword("req-1") }, "", app.ErrOperationCanceled},
		{"empty submission", func(a *App) { a.SubmitCertPassword("req-1", "") }, "", app.ErrOperationCanceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestApp(services{})
			done := askPassword(t, context.Background(), a, req)

			tt.answer(a)
			res := waitPassword(t, done)

			if !errors.Is(res.err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", res.err, tt.wantErr)
			}
			if string(res.pass) != tt.wantPass {
				t.Errorf("password = %q, want %q", res.pass, tt.wantPass)
			}
			if n := len(a.cred.passwordChans); n != 0 {
				t.Errorf("%d pending requests left behind, want 0", n)
			}
		})
	}
}

func TestGetCertPasswordStopsWithContext(t *testing.T) {
	a := newTestApp(services{})
	ctx, cancel := context.WithCancel(context.Background())
	done := askPassword(t, ctx, a, app.CertPasswordRequest{RequestID: "req-1"})

	cancel()
	res := waitPassword(t, done)

	if !errors.Is(res.err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", res.err)
	}
	if res.pass != nil {
		t.Errorf("password = %q, want nil", res.pass)
	}
	// A late answer finds nobody waiting and must not block.
	a.SubmitCertPassword("req-1", "mockdata")
}
