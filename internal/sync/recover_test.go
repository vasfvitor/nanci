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

func TestRecoverNSUReportsWhatItDidWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fetcher := &scriptedFetcher{
		consNSU: map[int64]sefaz.DistResult{
			11: {CStat: sefaz.CStatDocumentoLocalizado, Docs: []sefaz.DocZip{docZip(t, 11, "procNFe_v4.00.xsd", "procnfe.xml")}},
		},
		// The user cancels while the second NSU is being asked.
		onConsNSU: func(nsu int64) {
			if nsu == 12 {
				cancel()
			}
		},
	}
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 15, Cause: NSUGapConsumoIndevido})

	result, err := mgr.RecoverNSU(ctx, RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.StopReason != RecoverStopCanceled || result.Asked != 2 || result.Recovered != 1 || result.Empty != 1 || result.Missing != 3 {
		t.Errorf("result = %+v, want canceled after 2 asked, 3 missing", result)
	}
	if want := []int64{11, 12}; !slices.Equal(fetcher.asked, want) {
		t.Errorf("asked NSUs = %v, want %v", fetcher.asked, want)
	}
}

func TestRecoverNSURefusesWhileThePullRuns(t *testing.T) {
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
	release, err := mgr.ReserveSource(context.Background(), comp.ID, syncstate.SyncSourceNFe)
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
	missing, err := mgr.SyncRepo.MissingNSUs(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment)
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

func TestRecoveryLimits(t *testing.T) {
	t.Run("nothing missing", func(t *testing.T) {
		mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
		limits, err := mgr.RecoveryLimits(context.Background(), comp, syncstate.SyncSourceNFe)
		if err != nil {
			t.Fatal(err)
		}
		if limits != (RecoveryLimits{}) {
			t.Errorf("limits = %+v, want zero", limits)
		}
	})

	t.Run("ready, up to the hour left after the reserve", func(t *testing.T) {
		mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
		recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 40, Cause: NSUGapConsumoIndevido})
		spendRequests(t, mgr, comp, syncstate.SyncSourceNFe, 5)
		// The wait after catching up holds back distNSU only.
		if err := mgr.SyncRepo.SetBlockedUntil(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment, time.Now().Add(time.Hour), syncstate.SyncStopReasonCaughtUp); err != nil {
			t.Fatal(err)
		}
		limits, err := mgr.RecoveryLimits(context.Background(), comp, syncstate.SyncSourceNFe)
		if err != nil {
			t.Fatal(err)
		}
		if limits.Missing != 30 || limits.Allowance != 13 || limits.ReadyAt != nil {
			t.Errorf("limits = %+v, want 30 missing, 13 allowed (20-5-2), ready now", limits)
		}
	})

	t.Run("budget spent: ready when the oldest request leaves the hour", func(t *testing.T) {
		mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
		recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 12, Cause: NSUGapBatchHole})
		oldest := time.Now().Add(-30 * time.Minute).UTC().Truncate(time.Second)
		if err := mgr.SyncRepo.RecordRequest(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment, oldest); err != nil {
			t.Fatal(err)
		}
		spendRequests(t, mgr, comp, syncstate.SyncSourceNFe, 17)
		limits, err := mgr.RecoveryLimits(context.Background(), comp, syncstate.SyncSourceNFe)
		if err != nil {
			t.Fatal(err)
		}
		if limits.Allowance != 0 || limits.ReadyAt == nil || !limits.ReadyAt.Equal(oldest.Add(time.Hour)) {
			t.Errorf("limits = %+v, want nothing allowed until %v", limits, oldest.Add(time.Hour))
		}
	})

	t.Run("blocked by a 656: ready when the block ends", func(t *testing.T) {
		mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
		recordGaps(t, mgr, comp, syncstate.SyncSourceNFe, NSURange{First: 11, Last: 12, Cause: NSUGapBatchHole})
		until := time.Now().Add(40 * time.Minute).UTC().Truncate(time.Second)
		if err := mgr.SyncRepo.SetBlockedUntil(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment, until, syncstate.SyncStopReasonConsumoIndevido); err != nil {
			t.Fatal(err)
		}
		limits, err := mgr.RecoveryLimits(context.Background(), comp, syncstate.SyncSourceNFe)
		if err != nil {
			t.Fatal(err)
		}
		if limits.Allowance != 0 || limits.ReadyAt == nil || !limits.ReadyAt.Equal(until) {
			t.Errorf("limits = %+v, want nothing allowed until %v", limits, until)
		}
	})
}

func TestDiscardNSUCountsThenClosesEveryOpenGap(t *testing.T) {
	passwords := &countingProvider{}
	fetcher := &scriptedFetcher{}
	mgr, comp := newNFePullTestManager(t, passwords, fetcher)
	recordGaps(t, mgr, comp, syncstate.SyncSourceNFe,
		NSURange{First: 11, Last: 5010, Cause: NSUGapConsumoIndevido},
		NSURange{First: 6000, Last: 6001, Cause: NSUGapBatchHole})

	would, err := mgr.DiscardNSU(context.Background(), DiscardNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("DiscardNSU dry run: %v", err)
	}
	if would != 5002 {
		t.Errorf("dry run = %d, want 5002", would)
	}
	missing, err := mgr.SyncRepo.MissingNSUs(context.Background(), comp.ID, syncstate.SyncSourceNFe, comp.Environment)
	if err != nil || missing != 5002 {
		t.Fatalf("missing after the dry run = %d, %v, want 5002", missing, err)
	}

	discarded, err := mgr.DiscardNSU(context.Background(), DiscardNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe, Confirm: true})
	if err != nil || discarded != 5002 {
		t.Fatalf("DiscardNSU = %d, %v, want 5002", discarded, err)
	}
	result, err := mgr.RecoverNSU(context.Background(), RecoverNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe})
	if err != nil {
		t.Fatalf("RecoverNSU: %v", err)
	}
	if result.Asked != 0 || result.Missing != 0 || len(fetcher.asked) != 0 || passwords.callCount() != 0 {
		t.Errorf("recovery after discarding = %+v, asked %v, prompts %d; want nothing", result, fetcher.asked, passwords.callCount())
	}
}

func TestDiscardNSURefusesWhileThePullRuns(t *testing.T) {
	mgr, comp := newNFePullTestManager(t, &countingProvider{}, &scriptedFetcher{})
	release, err := mgr.ReserveSource(context.Background(), comp.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if _, err := mgr.DiscardNSU(context.Background(), DiscardNSUInput{CNPJ: comp.CNPJ, Source: syncstate.SyncSourceNFe, Confirm: true}); !errors.Is(err, ErrSyncRunning) {
		t.Fatalf("DiscardNSU error = %v, want ErrSyncRunning", err)
	}
}
