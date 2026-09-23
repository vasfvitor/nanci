package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

func newTestClient(t *testing.T, cfg Config) *Client {
	t.Helper()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func newLogger(level slog.Level) (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})), &buf
}

func TestNew_RejectsNegativeRetries(t *testing.T) {
	if _, err := New(Config{Retry: RetryConfig{MaxRetries: -1}}); err == nil {
		t.Fatal("expected error for negative MaxRetries")
	}
}

func TestDo_MaxRetriesZeroMakesOneAttempt(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(t, Config{Retry: RetryConfig{MaxRetries: 0, Initial: time.Millisecond}})
	_, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL})

	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected *StatusError, got %v", err)
	}
	if statusErr.StatusCode != http.StatusServiceUnavailable || !statusErr.Retryable {
		t.Errorf("unexpected StatusError: %+v", statusErr)
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected exactly 1 attempt, got %d", got)
	}
}

func TestDo_RetriesAndResendsBody(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != "<payload/>" {
			t.Errorf("attempt %d: body = %q, want the request body", n, body)
		}
		if got := r.Header.Get("Content-Type"); got != "application/soap+xml" {
			t.Errorf("attempt %d: Content-Type = %q", n, got)
		}
		if n < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := newTestClient(t, Config{Retry: RetryConfig{MaxRetries: 3, Initial: time.Millisecond, MaxDelay: 5 * time.Millisecond}})
	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodPost,
		URL:    server.URL,
		Header: http.Header{"Content-Type": {"application/soap+xml"}},
		Body:   []byte("<payload/>"),
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.StatusCode != http.StatusOK || string(resp.Body) != "ok" {
		t.Errorf("unexpected response: %d %q", resp.StatusCode, resp.Body)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestDo_NonRetryableStatusIsNotRetried(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(t, Config{Retry: RetryConfig{MaxRetries: 3, Initial: time.Millisecond}})
	if _, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL}); err == nil {
		t.Fatal("expected error")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected 1 attempt, got %d", got)
	}
}

func TestDo_RetryAfterIsHonored(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := newTestClient(t, Config{Retry: RetryConfig{MaxRetries: 2, Initial: time.Millisecond, MaxDelay: 1500 * time.Millisecond}})

	start := time.Now()
	if _, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("expected Retry-After delay to be respected, got %s", elapsed)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

func TestDo_ExpectReturnsResponseWithoutErrorLog(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<Fault/>"))
	}))
	defer server.Close()

	log, logs := newLogger(slog.LevelDebug)
	client := newTestClient(t, Config{Log: log, LogLabel: "TEST", Retry: RetryConfig{MaxRetries: 3, Initial: time.Millisecond}})
	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodPost,
		URL:    server.URL,
		Expect: func(status int) bool { return status == http.StatusInternalServerError },
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError || string(resp.Body) != "<Fault/>" {
		t.Errorf("unexpected response: %d %q", resp.StatusCode, resp.Body)
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected statuses are not retried, got %d attempts", got)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Errorf("expected status must not be logged as an error:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), "TEST Response") {
		t.Errorf("expected a debug response record:\n%s", logs.String())
	}
}

func TestDo_MaxBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	client := newTestClient(t, Config{})

	resp, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL, MaxBytes: 10})
	if err != nil || len(resp.Body) != 10 {
		t.Fatalf("body at the limit: got %q, %v", resp.Body, err)
	}

	_, err = client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL, MaxBytes: 9})
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("expected ErrResponseTooLarge, got %v", err)
	}
}

func TestDo_RedactsBodiesInLogsAndErrors(t *testing.T) {
	const secret = "12345678000195"
	redact := func(b []byte) []byte {
		return bytes.ReplaceAll(b, []byte(secret), []byte(MaskIdentifier(secret)))
	}

	t.Run("trace response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<CNPJ>" + secret + "</CNPJ>"))
		}))
		defer server.Close()

		log, logs := newLogger(logger.LevelTrace)
		client := newTestClient(t, Config{Log: log, RedactBody: redact})
		resp, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL})
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		if !bytes.Contains(resp.Body, []byte(secret)) {
			t.Error("Response.Body must stay raw")
		}
		if strings.Contains(logs.String(), secret) {
			t.Errorf("trace log leaks the identifier:\n%s", logs.String())
		}
		if !strings.Contains(logs.String(), "12**********95") {
			t.Errorf("expected the masked identifier in the trace body:\n%s", logs.String())
		}
	})

	t.Run("error record and message", func(t *testing.T) {
		// Long enough to also produce the trace-level full-body record.
		body := "<CNPJ>" + secret + "</CNPJ>" + strings.Repeat("x", MaxErrorLogBodyBytes)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(body))
		}))
		defer server.Close()

		log, logs := newLogger(logger.LevelTrace)
		client := newTestClient(t, Config{Log: log, LogLabel: "TEST", RedactBody: redact})
		_, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL})

		var statusErr *StatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected *StatusError, got %v", err)
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error message leaks the identifier: %s", err.Error())
		}
		if !strings.HasPrefix(err.Error(), "TEST error GET ") {
			t.Errorf("unexpected error message: %s", err.Error())
		}
		if statusErr.Body != body {
			t.Error("StatusError.Body must stay raw")
		}
		out := logs.String()
		if strings.Contains(out, secret) {
			t.Errorf("error logs leak the identifier:\n%s", out)
		}
		if !strings.Contains(out, "TEST Error Response Body") {
			t.Errorf("expected the trace-level full error body record:\n%s", out)
		}
	})
}

func TestDo_RedactsURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	log, logs := newLogger(logger.LevelTrace)
	client := newTestClient(t, Config{Log: log, RedactURL: func(string) string { return "redacted" }})
	_, err := client.Do(context.Background(), Request{Method: http.MethodGet, URL: server.URL + "/secret"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "/secret") || strings.Contains(logs.String(), "/secret") {
		t.Errorf("URL must be redacted:\nerr: %v\nlogs:\n%s", err, logs.String())
	}
}
