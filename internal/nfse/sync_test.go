package nfse_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestSyncRun_Instantiation(t *testing.T) {
	now := time.Now()
	run := nfse.SyncRun{
		ID:               "run123",
		CompanyID:        "comp1",
		Environment:      nfse.EnvironmentProduction,
		ConsultationCNPJ: "11111111000111",
		Mode:             nfse.SyncModeNormal,
		Status:           nfse.SyncStatusRunning,
		StartedAt:        now,
	}

	if run.ID != "run123" {
		t.Errorf("Expected run123, got %s", run.ID)
	}
	if run.Status != nfse.SyncStatusRunning {
		t.Errorf("Expected status running, got %s", run.Status)
	}
}

func TestSyncStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected nfse.SyncStatus
		valid    bool
	}{
		{"running", nfse.SyncStatusRunning, true},
		{"completed", nfse.SyncStatusCompleted, true},
		{"failed", nfse.SyncStatusFailed, true},
		{"interrupted", nfse.SyncStatusInterrupted, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			status, err := nfse.ParseSyncStatus(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if status != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, status)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %s, got nil", tt.input)
				}
			}
		})
	}
}

func TestSyncMode(t *testing.T) {
	tests := []struct {
		input    string
		expected nfse.SyncMode
		valid    bool
	}{
		{"normal", nfse.SyncModeNormal, true},
		{"first_setup", nfse.SyncModeFirstSetup, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := nfse.ParseSyncMode(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if mode != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, mode)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %s, got nil", tt.input)
				}
			}
		})
	}
}

func TestDecodePayload(t *testing.T) {
	// Create a dummy XML payload and compress it
	xmlData := []byte("<xml>test</xml>")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(xmlData)
	_ = gz.Close()

	payloadBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	limits := nfse.PayloadLimits{
		CompressedBytes:   1024,
		UncompressedBytes: 1024,
	}

	decoded, err := nfse.DecodePayload(payloadBase64, limits)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if string(decoded.XML) != string(xmlData) {
		t.Errorf("Expected XML %s, got %s", string(xmlData), string(decoded.XML))
	}
	if decoded.SHA256 == "" {
		t.Errorf("Expected a SHA256 hash, got empty string")
	}
}

func TestDecodePayload_ExceedsLimits(t *testing.T) {
	// Create a dummy XML payload and compress it
	xmlData := []byte("<xml>test data that exceeds the limit</xml>")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(xmlData)
	_ = gz.Close()

	payloadBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	limits := nfse.PayloadLimits{
		CompressedBytes:   10, // Intentionally small
		UncompressedBytes: 10,
	}

	_, err := nfse.DecodePayload(payloadBase64, limits)
	if err == nil {
		t.Fatalf("Expected error due to size limits, got nil")
	}
}
