package syncstate_test

import (
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func TestSyncRun_Instantiation(t *testing.T) {
	now := time.Now()
	run := syncstate.SyncRun{
		ID:               "run123",
		CompanyID:        "comp1",
		Environment:      dfe.EnvironmentProduction,
		ConsultationCNPJ: "11111111000111",
		Mode:             syncstate.SyncModeNormal,
		Status:           syncstate.SyncStatusRunning,
		StartedAt:        now,
	}

	if run.ID != "run123" {
		t.Errorf("Expected run123, got %s", run.ID)
	}
	if run.Status != syncstate.SyncStatusRunning {
		t.Errorf("Expected status running, got %s", run.Status)
	}
}

func TestSyncStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected syncstate.SyncStatus
		valid    bool
	}{
		{"running", syncstate.SyncStatusRunning, true},
		{"completed", syncstate.SyncStatusCompleted, true},
		{"failed", syncstate.SyncStatusFailed, true},
		{"interrupted", syncstate.SyncStatusInterrupted, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			status, err := syncstate.ParseSyncStatus(tt.input)
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
		expected syncstate.SyncMode
		valid    bool
	}{
		{"normal", syncstate.SyncModeNormal, true},
		{"first_setup", syncstate.SyncModeFirstSetup, true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := syncstate.ParseSyncMode(tt.input)
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
		expected syncstate.SyncSource
		valid    bool
	}{
		{"nfse", syncstate.SyncSourceNFSe, true},
		{"nfe", syncstate.SyncSourceNFe, true},
		{"cte", syncstate.SyncSourceCTe, true},
		{"mdfe", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			source, err := syncstate.ParseSyncSource(tt.input)
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
