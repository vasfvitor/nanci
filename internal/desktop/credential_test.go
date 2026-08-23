package main

import (
	"bytes"
	"testing"
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
