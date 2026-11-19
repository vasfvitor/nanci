package sync_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func TestNSUGapsRecordAdvanceAndClose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	comp := seedCompany(t, db, "company-1", "11222333000181")
	env := comp.Environment
	since := time.Now().Add(-time.Hour)

	gaps := []sync.NSURange{
		{First: 11, Last: 12, Cause: sync.NSUGapConsumoIndevido},
		{First: 20, Last: 20, Cause: sync.NSUGapBatchHole},
	}
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env, gaps); err != nil {
		t.Fatal(err)
	}
	// The same jump detected again, wider, keeps one row with the wider end.
	wider := []sync.NSURange{{First: 11, Last: 13, Cause: sync.NSUGapConsumoIndevido}}
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env, wider); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, env, since, 4)
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, env, since, 0)

	open, err := repo.OpenNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 2 || open[0].First != 11 || open[0].Last != 13 || open[0].Next != 11 || open[1].First != 20 {
		t.Fatalf("open gaps = %+v", open)
	}
	if open[0].Cause != sync.NSUGapConsumoIndevido || open[0].DetectedAt.IsZero() {
		t.Fatalf("first gap = %+v", open[0])
	}

	// NSU 11 is empty; 12 is recovered together with the item's write.
	if err := repo.AdvanceNSUGap(ctx, open[0], 11); err != nil {
		t.Fatal(err)
	}
	commit := repo.ApplyAndAdvanceNSUGap(open[0], 12)
	if _, err := commit(ctx, func(*sql.Tx) (sync.ItemOutcome, error) { return sync.ItemOutcome{Inserted: true}, nil }); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, env, since, 2)

	if err := repo.AdvanceNSUGap(ctx, open[0], 13); err != nil {
		t.Fatal(err)
	}
	if err := repo.ExpireNSUGap(ctx, open[1]); err != nil {
		t.Fatal(err)
	}
	open, err = repo.OpenNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 0 {
		t.Fatalf("open gaps after closing = %+v", open)
	}

	var status string
	var recovered, empty int
	if err := db.QueryRowContext(ctx, `SELECT status, recovered, empty FROM sync_nsu_gaps WHERE first_nsu = 11`).Scan(&status, &recovered, &empty); err != nil {
		t.Fatal(err)
	}
	if status != "done" || recovered != 1 || empty != 2 {
		t.Fatalf("closed gap = %s recovered=%d empty=%d, want done 1 2", status, recovered, empty)
	}

	// Detected again with a wider end, a closed gap opens again.
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env, []sync.NSURange{{First: 11, Last: 15, Cause: sync.NSUGapConsumoIndevido}}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, env, since, 2)
}

func TestMissingNSUsLeavesOutGapsDetectedBeforeSince(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	comp := seedCompany(t, db, "company-1", "11222333000181")

	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceCTe, comp.Environment, []sync.NSURange{{First: 5, Last: 9, Cause: sync.NSUGapBatchHole}}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, comp.Environment, time.Now().Add(-time.Hour), 5)
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, comp.Environment, time.Now().Add(time.Hour), 0)
}

func TestResetSyncStateDeletesTheSourceGaps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	comp := seedCompany(t, db, "company-1", "11222333000181")
	since := time.Now().Add(-time.Hour)
	gap := []sync.NSURange{{First: 5, Last: 9, Cause: sync.NSUGapBatchHole}}
	for _, source := range []syncstate.SyncSource{syncstate.SyncSourceNFe, syncstate.SyncSourceCTe} {
		if err := repo.RecordNSUGaps(ctx, comp.ID, source, comp.Environment, gap); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.ResetSyncState(ctx, syncstate.ResetSyncStateParams{CompanyID: comp.ID, Source: syncstate.SyncSourceNFe}); err != nil {
		t.Fatal(err)
	}

	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, comp.Environment, since, 0)
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, comp.Environment, since, 5)
}

func assertMissing(t *testing.T, repo *sync.Store, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, since time.Time, want int64) {
	t.Helper()
	got, err := repo.MissingNSUs(context.Background(), companyID, source, env, since)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("missing %s NSUs = %d, want %d", source, got, want)
	}
}
