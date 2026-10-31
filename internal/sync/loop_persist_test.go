package sync

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/syncstate"
)

// failWrites makes every insert and update on table abort with "disk full"
// when the row matches when (a SQL condition on NEW; empty matches all).
func (h *testHelper) failWrites(table, when string) {
	h.t.Helper()
	condition := ""
	if when != "" {
		condition = "WHEN " + when
	}
	for _, event := range []string{"INSERT", "UPDATE"} {
		stmt := "CREATE TRIGGER fail_" + strings.ToLower(event) + "_" + table + // #nosec G202 -- fixed names from the tests.
			" BEFORE " + event + " ON " + table + " " + condition +
			" BEGIN SELECT RAISE(ABORT, 'disk full'); END"
		if _, err := h.db.ExecContext(context.Background(), stmt); err != nil {
			h.t.Fatalf("create trigger on %s: %v", table, err)
		}
	}
}

// assertPersistError checks that err ends the run as a failed checkpoint
// write: failed, process_error, code persist_error, with the cause kept.
func assertPersistError(t *testing.T, err error, wantMsg string) {
	t.Helper()
	if err == nil {
		t.Fatal("Sync succeeded, want a persist_error failure")
	}
	status, reason, code, msg := classifySyncError(err)
	if status != syncstate.SyncStatusFailed || reason != syncstate.SyncStopReasonProcessError || code != "persist_error" {
		t.Errorf("classified as %s/%s/%s, want failed/process_error/persist_error (err: %v)", status, reason, code, err)
	}
	if !strings.Contains(msg, wantMsg) || !strings.Contains(msg, "disk full") {
		t.Errorf("error message = %q, want %q caused by the disk full write", msg, wantMsg)
	}
}

func TestSourceLoopFailedWaitWriteFailsRunBeforeItems(t *testing.T) {
	h := newTestHelper(t)
	waitUntil := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	processed := 0
	src := &fakeSource{
		kind: syncstate.SyncSourceNFe,
		batches: map[int64]Batch{
			0: {Items: items(1, 2), UltNSU: 2, MaxNSU: 2, NextCursor: 2, Done: true, StopReason: syncstate.SyncStopReasonCaughtUp, WaitUntil: &waitUntil},
		},
		onItem: func(int64) { processed++ },
	}
	h.failWrites("company_sync_sources", "")

	err := h.runSource(src)

	assertPersistError(t, err, "failed to record source wait")
	h.assertSourceRun(syncstate.SyncSourceNFe, syncstate.SyncStatusFailed, syncstate.SyncStopReasonProcessError)
	if processed != 0 {
		t.Errorf("items processed = %d, want 0: the wait is recorded before any item", processed)
	}
	if cursor, _ := h.sourceCursor(syncstate.SyncSourceNFe); cursor != 0 {
		t.Errorf("cursor = %d, want 0", cursor)
	}
}

func TestSourceLoopFailedBatchCheckpointKeepsCommittedCursor(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind: syncstate.SyncSourceNFe,
		batches: map[int64]Batch{
			// NF-e moves the cursor to ultNSU, past the last item, in the
			// batch checkpoint after the items committed.
			0: {Items: items(1, 2), UltNSU: 10, MaxNSU: 20, NextCursor: 10},
		},
	}
	h.failWrites("sync_state", "NEW.last_checked_nsu = 10")

	err := h.runSource(src)

	assertPersistError(t, err, "failed to persist batch checkpoint")
	h.assertSourceRun(syncstate.SyncSourceNFe, syncstate.SyncStatusFailed, syncstate.SyncStopReasonProcessError)
	if len(src.cursors) != 1 {
		t.Errorf("fetch cursors = %v, want the run to stop after the first batch", src.cursors)
	}
	if cursor, _ := h.sourceCursor(syncstate.SyncSourceNFe); cursor != 2 {
		t.Errorf("cursor = %d, want 2 (the last committed item)", cursor)
	}
	if toNSU := h.countRows(`SELECT to_nsu FROM sync_runs WHERE company_id = ? AND source = 'nfe'`, string(h.company.ID)); toNSU != 2 {
		t.Errorf("run to_nsu = %d, want 2", toNSU)
	}
}

func TestSourceLoopFailedEmptyBatchCheckpointKeepsCursor(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind: syncstate.SyncSourceNFe,
		batches: map[int64]Batch{
			0: {UltNSU: 5, MaxNSU: 5, NextCursor: 5},
		},
	}
	h.failWrites("sync_state", "NEW.last_checked_nsu = 5")

	err := h.runSource(src)

	assertPersistError(t, err, "failed to persist batch checkpoint")
	h.assertSourceRun(syncstate.SyncSourceNFe, syncstate.SyncStatusFailed, syncstate.SyncStopReasonProcessError)
	if cursor, maxNSU := h.sourceCursor(syncstate.SyncSourceNFe); cursor != 0 || maxNSU.Valid {
		t.Errorf("cursor, max_nsu = %d, %v; want 0 and none", cursor, maxNSU)
	}
}

func TestSourceLoopFailedInitialSyncMarkFailsRun(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind: syncstate.SyncSourceNFe,
		batches: map[int64]Batch{
			0: {Items: items(1), UltNSU: 1, MaxNSU: 1, NextCursor: 1, Done: true, StopReason: syncstate.SyncStopReasonCaughtUp},
		},
	}
	h.failWrites("company_sync_sources", "")

	err := h.runSource(src)

	assertPersistError(t, err, "failed to mark initial sync completed")
	h.assertSourceRun(syncstate.SyncSourceNFe, syncstate.SyncStatusFailed, syncstate.SyncStopReasonProcessError)
	if cursor, _ := h.sourceCursor(syncstate.SyncSourceNFe); cursor != 1 {
		t.Errorf("cursor = %d, want 1 (the committed item)", cursor)
	}
	state, err := h.store.SourceState(context.Background(), h.company.ID, syncstate.SyncSourceNFe, h.company.Environment)
	if err != nil {
		t.Fatal(err)
	}
	if state.InitialSyncDoneAt != nil {
		t.Errorf("initial sync marked at %v, want unmarked", state.InitialSyncDoneAt)
	}
}

func TestSourceLoopFailedBudgetWriteFailsRunBeforeFetch(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		spent    int // requests already made in the last hour
		wantMsg  string
		wantRows int // sync_requests rows after the run
	}{
		{name: "record request", table: "sync_requests", spent: 0, wantMsg: "failed to record request", wantRows: 0},
		{name: "block at the limit", table: "company_sync_sources", spent: 2, wantMsg: "failed to block source after request budget", wantRows: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHelper(t)
			src := &fakeSource{
				kind:    syncstate.SyncSourceNFe,
				policy:  SourcePolicy{RequestsPerHour: 2},
				batches: map[int64]Batch{0: {Items: items(1), NextCursor: 1}},
			}
			for range tt.spent {
				if err := h.store.RecordRequest(context.Background(), h.company.ID, syncstate.SyncSourceNFe, h.company.Environment, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			h.failWrites(tt.table, "")

			err := h.runSource(src)

			assertPersistError(t, err, tt.wantMsg)
			h.assertSourceRun(syncstate.SyncSourceNFe, syncstate.SyncStatusFailed, syncstate.SyncStopReasonProcessError)
			if len(src.cursors) != 0 {
				t.Errorf("fetch cursors = %v, want no fetch", src.cursors)
			}
			if cursor, _ := h.sourceCursor(syncstate.SyncSourceNFe); cursor != 0 {
				t.Errorf("cursor = %d, want 0", cursor)
			}
			if n := h.countRows(`SELECT COUNT(*) FROM sync_requests WHERE company_id = ?`, string(h.company.ID)); n != tt.wantRows {
				t.Errorf("sync_requests = %d, want %d", n, tt.wantRows)
			}
		})
	}
}
