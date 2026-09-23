package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

// fakeSource replays scripted batches keyed by the requested cursor. A cursor
// without a script gets an empty batch that is Done with caught_up.
type fakeSource struct {
	kind     nfse.SyncSource
	policy   SourcePolicy
	batches  map[int64]Batch
	outcomes map[int64]ItemOutcome // per NSU; missing means a plain insert
	failures map[int64]error       // per NSU; ProcessItem returns it every time
	onFetch  func(cursor int64)
	cursors  []int64
}

func (f *fakeSource) Kind() nfse.SyncSource { return f.kind }

func (f *fakeSource) Policy() SourcePolicy { return f.policy }

func (f *fakeSource) Fetch(_ context.Context, _ *nfse.Company, cursor int64) (Batch, error) {
	f.cursors = append(f.cursors, cursor)
	if f.onFetch != nil {
		f.onFetch(cursor)
	}
	if batch, ok := f.batches[cursor]; ok {
		return batch, nil
	}
	return Batch{NextCursor: cursor, Done: true, StopReason: nfse.SyncStopReasonCaughtUp}, nil
}

func (f *fakeSource) ProcessItem(ctx context.Context, _ *nfse.Company, _ SourceState, item Item, commit CommitFunc) (ItemOutcome, error) {
	if err := f.failures[item.NSU]; err != nil {
		return ItemOutcome{}, err
	}
	outcome, ok := f.outcomes[item.NSU]
	if !ok {
		outcome = ItemOutcome{Inserted: true}
	}
	return commit(ctx, func(*sql.Tx) (ItemOutcome, error) {
		return outcome, nil
	})
}

func items(nsus ...int64) []Item {
	out := make([]Item, 0, len(nsus))
	for _, nsu := range nsus {
		out = append(out, Item{NSU: nsu})
	}
	return out
}

func (h *testHelper) runSource(src Source) error {
	h.t.Helper()
	return NewSyncService(h.store, src, discardLogger()).Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil)
}

// sourceCursor returns last_checked_nsu and max_nsu of the source's sync_state.
func (h *testHelper) sourceCursor(source nfse.SyncSource) (int64, sql.NullInt64) {
	h.t.Helper()
	var cursor int64
	var maxNSU sql.NullInt64
	err := h.db.QueryRowContext(context.Background(), `
		SELECT last_checked_nsu, max_nsu FROM sync_state WHERE company_id = ? AND source = ?
	`, string(h.company.ID), string(source)).Scan(&cursor, &maxNSU)
	if err != nil {
		h.t.Fatalf("failed to query %s sync_state: %v", source, err)
	}
	return cursor, maxNSU
}

func (h *testHelper) countRows(query string, args ...any) int {
	h.t.Helper()
	var count int
	if err := h.db.QueryRowContext(context.Background(), query, args...).Scan(&count); err != nil {
		h.t.Fatalf("count query failed: %v", err)
	}
	return count
}

func (h *testHelper) assertSourceRun(source nfse.SyncSource, wantStatus nfse.SyncStatus, wantReason nfse.SyncStopReason) {
	h.t.Helper()
	var status string
	var reason sql.NullString
	err := h.db.QueryRowContext(context.Background(), `
		SELECT status, stop_reason FROM sync_runs
		WHERE company_id = ? AND source = ?
		ORDER BY rowid DESC LIMIT 1
	`, string(h.company.ID), string(source)).Scan(&status, &reason)
	if err != nil {
		h.t.Fatalf("failed to query %s run: %v", source, err)
	}
	if nfse.SyncStatus(status) != wantStatus || nfse.SyncStopReason(reason.String) != wantReason {
		h.t.Errorf("%s run = %s/%s, want %s/%s", source, status, reason.String, wantStatus, wantReason)
	}
}

func TestSourceLoopCursorFollowsNextCursorPastLastItem(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind: nfse.SyncSourceNFe,
		batches: map[int64]Batch{
			0:  {Items: items(1, 2), UltNSU: 10, MaxNSU: 20, NextCursor: 10},
			10: {Items: items(15), UltNSU: 20, MaxNSU: 20, NextCursor: 20, Done: true, StopReason: nfse.SyncStopReasonCaughtUp},
		},
	}

	if err := h.runSource(src); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(src.cursors) != 2 || src.cursors[0] != 0 || src.cursors[1] != 10 {
		t.Fatalf("fetch cursors = %v, want [0 10]", src.cursors)
	}
	cursor, maxNSU := h.sourceCursor(nfse.SyncSourceNFe)
	if cursor != 20 {
		t.Errorf("last_checked_nsu = %d, want 20", cursor)
	}
	if !maxNSU.Valid || maxNSU.Int64 != 20 {
		t.Errorf("max_nsu = %v, want 20", maxNSU)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
}

func TestSourceLoopCaughtUpMarksInitialSyncOnlyForThatSource(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{kind: nfse.SyncSourceNFe}

	if err := h.runSource(src); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	nfeState, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if nfeState.InitialSyncDoneAt == nil {
		t.Error("nfe initial sync not marked after caught_up")
	}
	h.assertInitialSyncCompleted(false) // the nfse flag and its companies mirror stay untouched
}

func TestSourceLoopWaitUntilBlocksNextSyncWithoutRun(t *testing.T) {
	h := newTestHelper(t)
	waitUntil := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	src := &fakeSource{
		kind: nfse.SyncSourceNFe,
		batches: map[int64]Batch{
			0: {NextCursor: 0, Done: true, StopReason: nfse.SyncStopReasonConsumoIndevido, WaitUntil: &waitUntil},
		},
	}

	if err := h.runSource(src); err != nil {
		t.Fatalf("first Sync: %v", err)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonConsumoIndevido)

	state, err := h.store.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if state.BlockedUntil == nil || !state.BlockedUntil.Equal(waitUntil) {
		t.Fatalf("blocked_until = %v, want %v", state.BlockedUntil, waitUntil)
	}
	if state.BlockedReason != nfse.SyncStopReasonConsumoIndevido {
		t.Errorf("blocked_reason = %q, want consumo_indevido", state.BlockedReason)
	}

	err = h.runSource(src)
	var blocked *BlockedError
	if !errors.As(err, &blocked) || !errors.Is(err, ErrSourceBlocked) {
		t.Fatalf("second Sync error = %v, want *BlockedError", err)
	}
	if blocked.Source != nfse.SyncSourceNFe || !blocked.Until.Equal(waitUntil) || blocked.Reason != nfse.SyncStopReasonConsumoIndevido {
		t.Errorf("BlockedError = %+v", blocked)
	}
	if got := h.countRows(`SELECT COUNT(*) FROM sync_runs WHERE source = 'nfe'`); got != 1 {
		t.Errorf("nfe runs = %d, want 1 (a blocked Sync must not start a run)", got)
	}
	if len(src.cursors) != 1 {
		t.Errorf("fetches = %d, want 1", len(src.cursors))
	}
}

func TestSourceLoopRequestBudgetStopsAtLimit(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind:   nfse.SyncSourceNFe,
		policy: SourcePolicy{RequestsPerHour: 2},
		batches: map[int64]Batch{
			0: {Items: items(1), NextCursor: 1},
			1: {Items: items(2), NextCursor: 2},
			2: {Items: items(3), NextCursor: 3},
		},
	}
	src.onFetch = func(int64) {
		// The request is recorded before it is sent.
		if got := h.countRows(`SELECT COUNT(*) FROM sync_requests WHERE source = 'nfe'`); got != len(src.cursors) {
			t.Errorf("sync_requests during fetch %d = %d, want %d", len(src.cursors), got, len(src.cursors))
		}
	}

	if err := h.runSource(src); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if len(src.cursors) != 2 {
		t.Fatalf("fetch cursors = %v, want 2 fetches", src.cursors)
	}
	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 2 {
		t.Errorf("last_checked_nsu = %d, want 2", cursor)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonRateBudget)

	// The budget and the block survive a new Store on the same database.
	reopened := NewStore(h.db)
	count, oldest, err := reopened.RequestsSince(context.Background(), h.company.ID, nfse.SyncSourceNFe, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || oldest == nil {
		t.Fatalf("RequestsSince = %d, %v; want 2 and the oldest time", count, oldest)
	}
	state, err := reopened.SourceState(context.Background(), h.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if state.BlockedUntil == nil || !state.BlockedUntil.Equal(oldest.Add(time.Hour)) || state.BlockedReason != nfse.SyncStopReasonRateBudget {
		t.Fatalf("source state = %+v, want blocked until %v for rate_budget", state, oldest.Add(time.Hour))
	}

	if err := h.runSource(src); !errors.Is(err, ErrSourceBlocked) {
		t.Fatalf("next Sync error = %v, want ErrSourceBlocked", err)
	}
}

func TestSourceLoopUnsupportedItemAdvancesCheckpoint(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind: nfse.SyncSourceNFe,
		batches: map[int64]Batch{
			0: {Items: items(4, 5), NextCursor: 5, Done: true, StopReason: nfse.SyncStopReasonCaughtUp},
		},
		outcomes: map[int64]ItemOutcome{5: {Unsupported: true}},
	}

	if err := h.runSource(src); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 5 {
		t.Errorf("last_checked_nsu = %d, want 5", cursor)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
}

func TestSourceLoopRunsOfDifferentSourcesDoNotInterruptEachOther(t *testing.T) {
	h := newTestHelper(t)
	nfseSrc := &fakeSource{
		kind:    nfse.SyncSourceNFSe,
		batches: map[int64]Batch{0: {Items: items(1), NextCursor: 1, Done: true, StopReason: nfse.SyncStopReasonEmptyLimit}},
	}
	nfeSrc := &fakeSource{
		kind:    nfse.SyncSourceNFe,
		batches: map[int64]Batch{0: {Items: items(7), NextCursor: 7, Done: true, StopReason: nfse.SyncStopReasonCaughtUp}},
	}
	nfeSrc.onFetch = func(int64) {
		// The NF-e run is in flight: a whole NFS-e run happens meanwhile.
		if err := h.runSource(nfseSrc); err != nil {
			t.Errorf("nfse Sync: %v", err)
		}
	}

	if err := h.runSource(nfeSrc); err != nil {
		t.Fatalf("nfe Sync: %v", err)
	}

	h.assertSourceRun(nfse.SyncSourceNFSe, nfse.SyncStatusCompleted, nfse.SyncStopReasonEmptyLimit)
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFSe); cursor != 1 {
		t.Errorf("nfse cursor = %d, want 1", cursor)
	}
	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 7 {
		t.Errorf("nfe cursor = %d, want 7", cursor)
	}
}

func TestSourceLoopSkipsPoisonItemAfterThreeFailedRuns(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind: nfse.SyncSourceNFe,
		batches: map[int64]Batch{
			0: {Items: items(1, 2), NextCursor: 2},
			1: {Items: items(2), NextCursor: 2},
		},
		failures: map[int64]error{2: &ProcessingError{Op: "parse document", NSU: 2, Err: errors.New("bad xml")}},
	}

	for run := 1; run < maxItemAttempts; run++ {
		err := h.runSource(src)
		var parseErr *ProcessingError
		if !errors.As(err, &parseErr) {
			t.Fatalf("run %d error = %v, want the ProcessingError", run, err)
		}
		if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 1 {
			t.Fatalf("run %d cursor = %d, want 1", run, cursor)
		}
		h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusFailed, nfse.SyncStopReasonProcessError)
	}

	if err := h.runSource(src); err != nil {
		t.Fatalf("run %d: %v", maxItemAttempts, err)
	}
	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 2 {
		t.Errorf("cursor after giving up = %d, want 2", cursor)
	}
	h.assertSourceRun(nfse.SyncSourceNFe, nfse.SyncStatusCompleted, nfse.SyncStopReasonCaughtUp)
}

func TestSourceLoopDoesNotSkipItemAfterNonParseFailures(t *testing.T) {
	h := newTestHelper(t)
	src := &fakeSource{
		kind:     nfse.SyncSourceNFe,
		batches:  map[int64]Batch{0: {Items: items(1), NextCursor: 1}},
		failures: map[int64]error{1: errors.New("disk full")},
	}

	for run := 1; run <= maxItemAttempts+1; run++ {
		if err := h.runSource(src); err == nil {
			t.Fatalf("run %d succeeded, want the storage error", run)
		}
	}
	if cursor, _ := h.sourceCursor(nfse.SyncSourceNFe); cursor != 0 {
		t.Errorf("cursor = %d, want 0", cursor)
	}
}

func TestNFSeSourceSkipsUnparseableDocumentAfterThreeRunsAndKeepsXML(t *testing.T) {
	h := newTestHelper(t)
	h.setSyncState(0, nil, 0)

	fetcher := &mockFetcher{
		handler: func(req adn.DistributionRequest) (*adn.DocumentResponse, error) {
			if req.LastNSU == 0 {
				return &adn.DocumentResponse{
					Docs: []adn.DocumentEnvelope{
						{NSU: 1, Schema: "procNFSe_v1.00.xsd", XMLGZipBase64: mustEncodeGzipBase64(t, "<NFSe>")},
					},
				}, nil
			}
			return &adn.DocumentResponse{}, nil
		},
	}
	originalDelay := syncRequestDelay
	syncRequestDelay = 0
	defer func() { syncRequestDelay = originalDelay }()

	xmlStore := &mockXMLStore{}
	svc := h.newNFSeService(fetcher, xmlStore)
	for run := 1; run < maxItemAttempts; run++ {
		if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err == nil {
			t.Fatalf("run %d succeeded, want a parse failure", run)
		}
	}
	if err := svc.Sync(context.Background(), h.company, h.credential, "exact_certificate_cnpj", nfse.SyncModeNormal, nil); err != nil {
		t.Fatalf("run %d: %v", maxItemAttempts, err)
	}

	h.assertDocumentsCount(0)
	h.assertSyncState(1, storetest.Int64Ptr(1), 1)
	if len(xmlStore.stored) != 1 {
		t.Fatalf("stored XML blobs = %d, want the unparsed document kept", len(xmlStore.stored))
	}
	for _, data := range xmlStore.stored {
		if string(data) != "<NFSe>" {
			t.Errorf("stored XML = %q, want the raw document", data)
		}
	}
}
