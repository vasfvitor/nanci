package sync

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// runStatusSource runs src for the company of a newPullTestManager.
func runStatusSource(t *testing.T, mgr *Manager, comp *company.Company, src Source) {
	t.Helper()
	cred, err := mgr.CredentialProvider.CredentialByID(context.Background(), comp.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewSyncService(mgr.SyncRepo, src, discardLogger()).Sync(context.Background(), comp, cred, "exact_certificate_cnpj", syncstate.SyncModeNormal, nil); err != nil {
		t.Fatalf("%s Sync: %v", src.Kind(), err)
	}
}

// applyStatusDocument stores one NFS-e of the company in role.
func applyStatusDocument(t *testing.T, mgr *Manager, comp *company.Company, n int, role nfse.CompanyRole) {
	t.Helper()
	id := fmt.Sprintf("doc-%d", n)
	_, err := mgr.SyncRepo.ApplyDocument(context.Background(), nfse.ApplyDocumentParams{
		Document: nfse.Document{
			ID:                 nfse.DocumentID(id),
			ChaveAcesso:        nfse.AccessKey(fmt.Sprintf("chave-%d", n)),
			IssueDate:          time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC),
			Competence:         "2026-06",
			PrestadorCNPJ:      "99887766000155",
			TomadorCNPJ:        comp.CNPJ,
			Status:             nfse.DocumentStatusNormal,
			LayoutVersion:      "1.01",
			XMLPath:            id + ".xml",
			RawHash:            "hash-" + id,
			NFSeNumber:         id,
			ServiceDescription: "service",
		},
		Participation: nfse.CompanyParticipation{CompanyRole: role, VisibilityReason: "unknown"},
		CompanyID:     comp.ID,
		NSU:           int64(n),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatusWithoutStateReportsCompanyAndCredential(t *testing.T) {
	mgr, comp := newPullTestManager(t, &countingProvider{})
	ctx := context.Background()
	cred, err := mgr.CredentialProvider.CredentialByID(ctx, comp.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	notAfter := time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC)
	cred.OwnerCNPJ = "11222333000181"
	cred.NotAfter = &notAfter
	if err := mgr.CredentialProvider.UpdateCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}

	got, err := mgr.Status(ctx, "11.222.333/0001-81")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if got.CredentialNotAfter == nil || !got.CredentialNotAfter.Equal(notAfter) {
		t.Errorf("CredentialNotAfter = %v, want %v", got.CredentialNotAfter, notAfter)
	}
	got.CredentialNotAfter = nil
	want := StatusResult{
		CompanyName:      "Company",
		CNPJ:             comp.CNPJ,
		Environment:      string(dfe.EnvironmentProduction),
		ConsultationCNPJ: comp.CNPJ,
		CredentialCNPJ:   "11222333000181",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Status = %+v, want %+v", got, want)
	}
}

func TestStatusReportsTheNFSeRunAndDocumentTotals(t *testing.T) {
	mgr, comp := newPullTestManager(t, &countingProvider{})
	runStatusSource(t, mgr, comp, &fakeSource{
		kind:    syncstate.SyncSourceNFSe,
		batches: map[int64]Batch{0: {Items: items(1, 2, 3), NextCursor: 3, Done: true, StopReason: syncstate.SyncStopReasonEmptyLimit}},
	})
	// A later NF-e run of the same company does not show up: Status is NFS-e only.
	runStatusSource(t, mgr, comp, &fakeSource{
		kind:    syncstate.SyncSourceNFe,
		batches: map[int64]Batch{0: {Items: items(40), NextCursor: 40, Done: true, StopReason: syncstate.SyncStopReasonCaughtUp}},
	})
	applyStatusDocument(t, mgr, comp, 1, nfse.CompanyRoleTomada)
	applyStatusDocument(t, mgr, comp, 2, nfse.CompanyRoleTomada)
	applyStatusDocument(t, mgr, comp, 3, nfse.CompanyRolePrestada)

	got, err := mgr.Status(context.Background(), comp.CNPJ)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if got.LastProcessedNSU != 3 || got.LastFoundNSU == nil || *got.LastFoundNSU != 3 {
		t.Errorf("NSUs = %d/%v, want 3/3", got.LastProcessedNSU, got.LastFoundNSU)
	}
	if got.LastRunStatus != string(syncstate.SyncStatusCompleted) || got.LastRunStopReason != string(syncstate.SyncStopReasonEmptyLimit) {
		t.Errorf("last run = %s/%s, want completed/empty_limit", got.LastRunStatus, got.LastRunStopReason)
	}
	if got.LastSyncAt == nil || time.Since(*got.LastSyncAt) > time.Minute {
		t.Errorf("LastSyncAt = %v, want the run's finish time", got.LastSyncAt)
	}
	if got.TotalTomadas != 2 || got.TotalEmitidas != 1 {
		t.Errorf("totals = %d tomadas, %d emitidas; want 2 and 1", got.TotalTomadas, got.TotalEmitidas)
	}
}

func TestStatusWithStateButNoRunUsesLastSuccess(t *testing.T) {
	mgr, comp := newPullTestManager(t, &countingProvider{})
	lastSuccess := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	storetest.SetSyncStateLastSuccess(t, mgr.SyncRepo.db, comp.ID, syncstate.SyncSourceNFSe, comp.Environment, comp.CNPJ, lastSuccess)

	got, err := mgr.Status(context.Background(), comp.CNPJ)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if got.LastProcessedNSU != 10 || got.LastFoundNSU != nil {
		t.Errorf("NSUs = %d/%v, want 10/nil", got.LastProcessedNSU, got.LastFoundNSU)
	}
	if got.LastSyncAt == nil || !got.LastSyncAt.Equal(lastSuccess) {
		t.Errorf("LastSyncAt = %v, want the last success %v", got.LastSyncAt, lastSuccess)
	}
	if got.LastRunStatus != "" || got.LastRunStopReason != "" {
		t.Errorf("last run = %q/%q, want none", got.LastRunStatus, got.LastRunStopReason)
	}
}

// TestStatusIgnoresAnActiveBlock pins that a block of the source neither
// fails Status nor changes what it reports: StatusResult has no block field.
func TestStatusIgnoresAnActiveBlock(t *testing.T) {
	mgr, comp := newPullTestManager(t, &countingProvider{})
	ctx := context.Background()
	runStatusSource(t, mgr, comp, &fakeSource{
		kind:    syncstate.SyncSourceNFSe,
		batches: map[int64]Batch{0: {Items: items(1), NextCursor: 1, Done: true, StopReason: syncstate.SyncStopReasonEmptyLimit}},
	})
	before, err := mgr.Status(ctx, comp.CNPJ)
	if err != nil {
		t.Fatalf("Status before the block: %v", err)
	}
	if err := mgr.SyncRepo.SetBlockedUntil(ctx, comp.ID, syncstate.SyncSourceNFSe, comp.Environment, time.Now().Add(time.Hour), syncstate.SyncStopReasonConsumoIndevido); err != nil {
		t.Fatal(err)
	}

	after, err := mgr.Status(ctx, comp.CNPJ)
	if err != nil {
		t.Fatalf("Status while blocked = %v, want no error", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Errorf("Status while blocked = %+v, want %+v", after, before)
	}
}

func TestStatusRejectsInvalidAndUnknownCNPJ(t *testing.T) {
	mgr, _ := newPullTestManager(t, &countingProvider{})
	tests := []struct {
		raw         string
		wantInvalid bool
	}{
		{raw: "123", wantInvalid: true},
		{raw: "11444777000161", wantInvalid: false}, // valid, but no such company
	}
	for _, tt := range tests {
		_, err := mgr.Status(context.Background(), tt.raw)
		if err == nil {
			t.Errorf("Status(%q) succeeded, want an error", tt.raw)
			continue
		}
		if invalid := strings.Contains(err.Error(), "CNPJ inválido"); invalid != tt.wantInvalid {
			t.Errorf("Status(%q) = %v, want invalid CNPJ: %t", tt.raw, err, tt.wantInvalid)
		}
	}
}
