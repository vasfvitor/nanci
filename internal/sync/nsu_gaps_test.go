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
	now := time.Now()

	gaps := []sync.NSURange{
		{First: 11, Last: 12, Cause: sync.NSUGapConsumoIndevido},
		{First: 20, Last: 20, Cause: sync.NSUGapBatchHole},
	}
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env, now, gaps); err != nil {
		t.Fatal(err)
	}
	// The same jump detected again, wider, adds only the NSU no gap holds.
	wider := []sync.NSURange{{First: 11, Last: 13, Cause: sync.NSUGapConsumoIndevido}}
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env, now, wider); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, env, 4)
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, env, 0)

	open, err := repo.OpenNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 3 || open[0].First != 11 || open[0].Last != 12 || open[1].First != 13 || open[1].Last != 13 || open[2].First != 20 {
		t.Fatalf("open gaps = %+v, want 11-12, 13 and 20", open)
	}
	if open[0].Cause != sync.NSUGapConsumoIndevido || open[0].Since.IsZero() {
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
	// An item skipped by policy stores nothing and counts as empty.
	skipped := repo.ApplyAndAdvanceNSUGap(open[1], 13)
	if _, err := skipped(ctx, func(*sql.Tx) (sync.ItemOutcome, error) {
		return sync.ItemOutcome{SkippedByPolicy: true, IsEvent: true}, nil
	}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, env, 1)

	old := time.Now().Add(-sync.NSUGapWindow - time.Hour).UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `UPDATE sync_nsu_gaps SET nsus_since = ? WHERE first_nsu = 20`, old); err != nil {
		t.Fatal(err)
	}
	expired, err := repo.ExpireNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env)
	if err != nil {
		t.Fatal(err)
	}
	if expired != 1 {
		t.Fatalf("expired NSUs = %d, want 1", expired)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, env, 0)

	counts := gapCounts(t, db)
	want := map[int64][3]any{
		11: {"done", 1, 1},
		13: {"done", 0, 1},
		20: {"expired", 0, 0},
	}
	for first, w := range want {
		if counts[first] != w {
			t.Errorf("gap at %d = %v, want %v", first, counts[first], w)
		}
	}

	// A range across finished gaps keeps only the NSUs outside them.
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env, now, []sync.NSURange{{First: 10, Last: 22, Cause: sync.NSUGapConsumoIndevido}}); err != nil {
		t.Fatal(err)
	}
	open, err = repo.OpenNSUGaps(ctx, comp.ID, syncstate.SyncSourceNFe, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 3 || open[0].First != 10 || open[0].Last != 10 ||
		open[1].First != 14 || open[1].Last != 19 || open[2].First != 21 || open[2].Last != 22 {
		t.Fatalf("open gaps = %+v, want 10, 14-19 and 21-22", open)
	}
}

func TestMissingNSUsLeavesOutGapsPastTheWindow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	comp := seedCompany(t, db, "company-1", "11222333000181")

	// NSUs skipped after the distribution last answered 100 days ago are
	// already past the window when the gap is found.
	old := time.Now().Add(-100 * 24 * time.Hour)
	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceCTe, comp.Environment, old, []sync.NSURange{{First: 5, Last: 9, Cause: sync.NSUGapBatchHole}}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, comp.Environment, 0)

	if err := repo.RecordNSUGaps(ctx, comp.ID, syncstate.SyncSourceCTe, comp.Environment, time.Now(), []sync.NSURange{{First: 20, Last: 21, Cause: sync.NSUGapBatchHole}}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, comp.Environment, 2)
}

func TestResetSyncStateDeletesTheSourceGaps(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	comp := seedCompany(t, db, "company-1", "11222333000181")
	gap := []sync.NSURange{{First: 5, Last: 9, Cause: sync.NSUGapBatchHole}}
	for _, source := range []syncstate.SyncSource{syncstate.SyncSourceNFe, syncstate.SyncSourceCTe} {
		if err := repo.RecordNSUGaps(ctx, comp.ID, source, comp.Environment, time.Now(), gap); err != nil {
			t.Fatal(err)
		}
	}

	if err := repo.ResetSyncState(ctx, syncstate.ResetSyncStateParams{CompanyID: comp.ID, Source: syncstate.SyncSourceNFe}); err != nil {
		t.Fatal(err)
	}

	assertMissing(t, repo, comp.ID, syncstate.SyncSourceNFe, comp.Environment, 0)
	assertMissing(t, repo, comp.ID, syncstate.SyncSourceCTe, comp.Environment, 5)
}

// gapCounts maps the first NSU of every recorded gap to its status and its
// recovered and empty counters.
func gapCounts(t *testing.T, db *sql.DB) map[int64][3]any {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `SELECT first_nsu, status, recovered, empty FROM sync_nsu_gaps`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()

	counts := map[int64][3]any{}
	for rows.Next() {
		var first int64
		var status string
		var recovered, empty int
		if err := rows.Scan(&first, &status, &recovered, &empty); err != nil {
			t.Fatal(err)
		}
		counts[first] = [3]any{status, recovered, empty}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return counts
}

func assertMissing(t *testing.T, repo *sync.Store, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, want int64) {
	t.Helper()
	got, err := repo.MissingNSUs(context.Background(), companyID, source, env)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("missing %s NSUs = %d, want %d", source, got, want)
	}
}
