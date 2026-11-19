package sync

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func recordGaps(t *testing.T, mgr *Manager, comp *company.Company, source syncstate.SyncSource, gaps ...NSURange) {
	t.Helper()
	if err := mgr.SyncRepo.RecordNSUGaps(context.Background(), comp.ID, source, comp.Environment, time.Now(), gaps); err != nil {
		t.Fatal(err)
	}
}

func spendRequests(t *testing.T, mgr *Manager, comp *company.Company, source syncstate.SyncSource, n int) {
	t.Helper()
	for range n {
		if err := mgr.SyncRepo.RecordRequest(context.Background(), comp.ID, source, comp.Environment, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecoverNSUAsksEachMissingNSUAndStoresWhatComes(t *testing.T) {
	passwords := &countingProvider{}
	fetcher := &scriptedFetcher{consNSU: map[int64]sefaz.DistResult{
		5: {CStat: sefaz.CStatDocumentoLocalizado, Docs: []sefaz.DocZip{docZip(t, 5, "procNFe_v4.00.xsd", "procnfe.xml")}},
		9: {CStat: sefaz.CStatDocumentoLocalizado, Docs: []sefaz.DocZip{docZip(t, 9, "resNFe_v1.01.xsd", "resnfe.xml")}},
	}}
	mgr, comp := newNFePullTestManager(t, passwords, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe,
		NSURange{First: 5, Last: 7, Cause: NSUGapConsumoIndevido},
		NSURange{First: 9, Last: 9, Cause: NSUGapBatchHole})

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}

	if want := []int64{5, 6, 7, 9}; !slices.Equal(fetcher.asked, want) {
		t.Errorf("asked NSUs = %v, want %v", fetcher.asked, want)
	}
	if result.Asked != 4 || result.Recovered != 2 || result.Empty != 2 || result.Missing != 0 || result.StopReason != "" {
		t.Errorf("result = %+v, want 4 asked, 2 recovered, 2 empty, none missing", result)
	}
	if result.RequestsLastHour != 4 || result.RequestBudget != 20 {
		t.Errorf("requests = %d of %d, want 4 of 20", result.RequestsLastHour, result.RequestBudget)
	}
	if got := passwords.callCount(); got != 1 {
		t.Errorf("password prompts = %d, want 1", got)
	}
	var stored int
	if err := mgr.SyncRepo.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM company_nfe_documents WHERE company_id = ?`, string(comp.ID)).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == 0 {
		t.Error("recovered NF-e not stored for the company")
	}
	cursor, _ := mgr.SyncRepo.SourceState(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment)
	if cursor.BlockedUntil != nil {
		t.Errorf("source blocked after a clean recovery: %+v", cursor)
	}
}

func TestRecoverNSULeavesTheReserveForThePull(t *testing.T) {
	fetcher := &scriptedFetcher{}
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 20, Cause: NSUGapConsumoIndevido})
	spendRequests(t, mgr, comp, syncstate.SyncSourceNFe, 15)

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.Asked != 3 || result.StopReason != RecoverStopLimit || result.Missing != 7 || result.RequestsLastHour != 18 {
		t.Errorf("result = %+v, want 3 asked (20-15-2), stopped at the limit, 7 missing", result)
	}

	// The next run starts where this one stopped, and --limite may use the
	// reserve.
	result, err = mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe, Limit: 5})
	if err != nil {
		t.Fatalf("RecoverNSU with a limit: %v", err)
	}
	if result.Asked != 2 || result.Missing != 5 {
		t.Errorf("result with limit = %+v, want the 2 left in the hour", result)
	}
	if want := []int64{11, 12, 13, 14, 15}; !slices.Equal(fetcher.asked, want) {
		t.Errorf("asked NSUs = %v, want %v", fetcher.asked, want)
	}
}

func TestRecoverNSURefusesBeforeThePasswordWhenNothingIsLeft(t *testing.T) {
	passwords := &countingProvider{}
	mgr, comp := newNFePullTestManager(t, passwords, &scriptedFetcher{})
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 12, Cause: NSUGapBatchHole})
	spendRequests(t, mgr, comp, syncstate.SyncSourceNFe, 18)

	_, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if !errors.Is(err, ErrNoRequestsLeft) {
		t.Fatalf("RecoverNSU error = %v, want ErrNoRequestsLeft", err)
	}
	if got := passwords.callCount(); got != 0 {
		t.Errorf("password prompts = %d, want 0", got)
	}
}

func TestRecoverNSUStopsAndBlocksOnConsumoIndevido(t *testing.T) {
	passwords := &countingProvider{}
	fetcher := &scriptedFetcher{consNSU: map[int64]sefaz.DistResult{
		12: {CStat: sefaz.CStatConsumoIndevido},
	}}
	mgr, comp := newNFePullTestManager(t, passwords, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 13, Cause: NSUGapConsumoIndevido})

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.StopReason != RecoverStopConsumoIndevido || result.Empty != 1 || result.Missing != 2 {
		t.Errorf("result = %+v, want stopped by 656 with NSUs 12 and 13 still missing", result)
	}
	if result.NextAllowedAt == nil || time.Until(*result.NextAllowedAt) < 55*time.Minute {
		t.Errorf("NextAllowedAt = %v, want about an hour from now", result.NextAllowedAt)
	}

	_, err = mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	var blocked *BlockedError
	if !errors.As(err, &blocked) || blocked.Reason != syncstate.SyncStopReasonConsumoIndevido {
		t.Fatalf("second RecoverNSU error = %v, want *BlockedError", err)
	}
	if got := passwords.callCount(); got != 1 {
		t.Errorf("password prompts = %d, want 1 (the blocked run must not prompt)", got)
	}
}

func TestRecoverNSUClosesGapsPastTheWindowWithoutAsking(t *testing.T) {
	passwords := &countingProvider{}
	fetcher := &scriptedFetcher{}
	mgr, comp := newNFePullTestManager(t, passwords, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 14, Cause: NSUGapConsumoIndevido})
	old := time.Now().Add(-NSUGapWindow - time.Hour).UTC().Format(time.RFC3339)
	if _, err := mgr.SyncRepo.db.ExecContext(context.Background(), `UPDATE sync_nsu_gaps SET nsus_since = ?`, old); err != nil {
		t.Fatal(err)
	}

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.Expired != 4 || result.Asked != 0 || result.Missing != 0 {
		t.Errorf("result = %+v, want 4 expired and nothing asked", result)
	}
	if got := passwords.callCount(); got != 0 || len(fetcher.asked) != 0 {
		t.Errorf("password prompts = %d, requests = %d, want none", got, len(fetcher.asked))
	}
}

func TestRecoverNSURefusesWhileThePullRuns(t *testing.T) {
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
	release, err := mgr.ReserveSource(comp.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	_, err = mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if !errors.Is(err, ErrSyncRunning) {
		t.Fatalf("RecoverNSU error = %v, want ErrSyncRunning", err)
	}
}

func TestRecoverNSURejectsNFSe(t *testing.T) {
	mgr, comp := newPullTestManager(t, &countingProvider{})
	if _, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFSe}); err == nil {
		t.Fatal("RecoverNSU for NFS-e succeeded, want an error")
	}
}

func TestRecoverNSUOfCTeUsesTheCTeService(t *testing.T) {
	fetcher := &scriptedFetcher{consNSU: map[int64]sefaz.DistResult{
		3: {CStat: sefaz.CStatDocumentoLocalizado, Docs: []sefaz.DocZip{cteDocZip(t, 3, "procCTe_v4.00.xsd", "procte.xml")}},
	}}
	mgr, comp := newCTePullTestManager(t, &countingProvider{}, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceCTe, NSURange{First: 3, Last: 3, Cause: NSUGapBatchHole})

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceCTe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.Recovered != 1 || result.Missing != 0 || !slices.Equal(fetcher.services, []string{"cte"}) {
		t.Errorf("result = %+v, services = %v, want 1 recovered through the CT-e service", result, fetcher.services)
	}
}

func TestRecoverNSUMovesPastAnNSUSEFAZRejects(t *testing.T) {
	fetcher := &scriptedFetcher{consNSU: map[int64]sefaz.DistResult{
		11: {CStat: sefaz.CStatNSUAcimaDoMaximo, XMotivo: "NSU superior ao maior NSU"},
		12: {CStat: sefaz.CStatForaDePrazo, XMotivo: "fora de prazo"},
	}}
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 13, Cause: NSUGapConsumoIndevido})

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.Asked != 3 || result.Empty != 3 || result.Missing != 0 {
		t.Errorf("result = %+v, want the rejected NSUs counted as empty and the gap closed", result)
	}
}

func TestRecoverNSUStopsOnARejectionOfTheRequest(t *testing.T) {
	fetcher := &scriptedFetcher{consNSU: map[int64]sefaz.DistResult{
		11: {CStat: 593, XMotivo: "CNPJ-Base consultado difere do CNPJ-Base do certificado"},
	}}
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 12, Cause: NSUGapConsumoIndevido})

	_, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	var rejection *sefaz.RejectionError
	if !errors.As(err, &rejection) || rejection.CStat != 593 {
		t.Fatalf("RecoverNSU error = %v, want the 593 rejection", err)
	}
	missing, err := mgr.SyncRepo.MissingNSUs(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if missing != 2 {
		t.Errorf("missing = %d, want the gap left untouched", missing)
	}
}

func TestRecoverNSUCountsAnEventItCannotKeepAsEmpty(t *testing.T) {
	fetcher := &scriptedFetcher{consNSU: map[int64]sefaz.DistResult{
		// A ciência by another CNPJ on an NF-e the company does not have.
		11: {CStat: sefaz.CStatDocumentoLocalizado, Docs: []sefaz.DocZip{docZip(t, 11, "procEventoNFe_v1.00.xsd", "proceventonfe-ciencia.xml")}},
	}}
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 11, Cause: NSUGapBatchHole})

	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.Recovered != 0 || result.Empty != 1 || result.Missing != 0 {
		t.Errorf("result = %+v, want the skipped event counted as empty", result)
	}
}
