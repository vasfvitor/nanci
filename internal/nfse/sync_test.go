package nfse_test

import (
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
			} else if err == nil {
				t.Errorf("Expected error for input %s, got nil", tt.input)
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
			} else if err == nil {
				t.Errorf("Expected error for input %s, got nil", tt.input)
			}
		})
	}
}

func TestSyncSource(t *testing.T) {
	tests := []struct {
		input    string
		expected nfse.SyncSource
		valid    bool
	}{
		{"nfse", nfse.SyncSourceNFSe, true},
		{"nfe", nfse.SyncSourceNFe, true},
		{"cte", nfse.SyncSourceCTe, true},
		{"mdfe", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			source, err := nfse.ParseSyncSource(tt.input)
			if tt.valid {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if source != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, source)
				}
			} else if err == nil {
				t.Errorf("Expected error for input %q, got nil", tt.input)
			}
		})
	}
}
