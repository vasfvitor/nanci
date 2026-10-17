package syncstate_test

import (
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// enumCase is one input to a Parse* function: valid inputs come back
// unchanged, invalid ones as the zero value and dfe.ErrInvalidEnum.
type enumCase struct {
	input string
	valid bool
}

func checkParse[T ~string](t *testing.T, parse func(string) (T, error), valid func(T) bool, cases []enumCase) {
	t.Helper()
	for _, tc := range cases {
		got, err := parse(tc.input)
		if tc.valid {
			if err != nil {
				t.Errorf("parse(%q): %v", tc.input, err)
				continue
			}
			if string(got) != tc.input {
				t.Errorf("parse(%q) = %q", tc.input, got)
			}
			if !valid(got) {
				t.Errorf("parse(%q).Valid() = false", tc.input)
			}
			continue
		}
		if !errors.Is(err, dfe.ErrInvalidEnum) {
			t.Errorf("parse(%q) error = %v, want ErrInvalidEnum", tc.input, err)
		}
		if got != "" {
			t.Errorf("parse(%q) = %q on error, want empty", tc.input, got)
		}
		if valid(T(tc.input)) {
			t.Errorf("%q.Valid() = true", tc.input)
		}
	}
}

func TestParseSyncStartPolicy(t *testing.T) {
	checkParse(t, syncstate.ParseSyncStartPolicy, syncstate.SyncStartPolicy.Valid, []enumCase{
		{"all", true},
		{"since_date", true},
		{"from_now", true},
		{"", false},
		{"ALL", false},
		{"since-date", false},
	})
}

func TestParseSyncStopReason(t *testing.T) {
	checkParse(t, syncstate.ParseSyncStopReason, syncstate.SyncStopReason.Valid, []enumCase{
		{"empty_limit", true},
		{"context_canceled", true},
		{"fetch_error", true},
		{"process_error", true},
		{"caught_up", true},
		{"consumo_indevido", true},
		{"rate_budget", true},
		{"", false},
		{"timeout", false},
	})
}

func TestParseConsultationBasis(t *testing.T) {
	checkParse(t, syncstate.ParseConsultationBasis, syncstate.ConsultationBasis.Valid, []enumCase{
		{"exact_certificate_cnpj", true},
		{"same_root_certificate", true},
		{"", false},
		{"other_cnpj", false},
	})
}

func TestParseSyncEnumsRejectUnknown(t *testing.T) {
	checkParse(t, syncstate.ParseSyncSource, syncstate.SyncSource.Valid, []enumCase{{"NFSE", false}, {"mdfe", false}})
	checkParse(t, syncstate.ParseSyncMode, syncstate.SyncMode.Valid, []enumCase{{"", false}, {"Normal", false}})
	checkParse(t, syncstate.ParseSyncStatus, syncstate.SyncStatus.Valid, []enumCase{{"", false}, {"done", false}})
}

func TestSyncEnumStrings(t *testing.T) {
	tests := []struct {
		got  string
		want string
	}{
		{syncstate.SyncSourceNFSe.String(), "nfse"},
		{syncstate.SyncSourceNFe.String(), "nfe"},
		{syncstate.SyncSourceCTe.String(), "cte"},
		{syncstate.SyncStatusInterrupted.String(), "interrupted"},
		{syncstate.ConsultationBasisSameRootCertificate.String(), "same_root_certificate"},
		{syncstate.SyncModeFirstSetup.String(), "first_setup"},
		{syncstate.SyncStopReasonRateBudget.String(), "rate_budget"},
		{syncstate.SyncStartPolicySinceDate.String(), "since_date"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("String() = %q, want %q", tt.got, tt.want)
		}
	}
}
