package httpclient

import (
	"testing"
	"time"
)

func Test_parseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		maxDelay time.Duration
		want     time.Duration
	}{
		{"empty string", "", 10 * time.Second, 0},
		{"valid seconds", "5", 10 * time.Second, 5 * time.Second},
		{"valid seconds with spaces", " 3 ", 10 * time.Second, 3 * time.Second},
		{"exceeds max delay", "20", 10 * time.Second, 10 * time.Second},
		{"invalid format", "abc", 10 * time.Second, 0},
		{"negative number", "-5", 10 * time.Second, 0},
		{"no max delay limit", "15", 0, 15 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseRetryAfter(tt.raw, tt.maxDelay); got != tt.want {
				t.Errorf("parseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isRetryableStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{"200 OK", 200, false},
		{"400 Bad Request", 400, false},
		{"404 Not Found", 404, false},
		{"408 Request Timeout", 408, true},
		{"425 Too Early", 425, true},
		{"429 Too Many Requests", 429, true},
		{"500 Internal Server Error", 500, true},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
		{"504 Gateway Timeout", 504, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableStatus(tt.status); got != tt.want {
				t.Errorf("isRetryableStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
