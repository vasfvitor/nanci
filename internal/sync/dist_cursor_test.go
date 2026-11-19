package sync

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// The NF-e and CT-e distributions treat a query whose ultNSU is not the one
// the SEFAZ returned last as out of sequence (cStat 656). These tests pin
// that the cursor stored after every response, including an empty lote (137)
// and a 656, is the ultNSU the SEFAZ returned, even when it jumps far past
// the last item, and never a value derived from the item NSUs.

// ultNSUJumpScript answers the first query with one item at NSU 1 but ultNSU
// 40, then an empty lote that jumps to 90, then a 656 that jumps to 120; doc
// is the single item.
func ultNSUJumpScript(doc sefaz.DocZip) map[int64]sefaz.DistResult {
	return map[int64]sefaz.DistResult{
		0:  {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 40, MaxNSU: 90, Docs: []sefaz.DocZip{doc}},
		40: {CStat: sefaz.CStatNenhumDocumento, UltNSU: 90, MaxNSU: 90},
		90: {CStat: sefaz.CStatConsumoIndevido, UltNSU: 120, MaxNSU: 120},
	}
}

func assertStoredCursor(t *testing.T, h *testHelper, source syncstate.SyncSource, want int64, when string) {
	t.Helper()
	if got, _ := h.sourceCursor(source); got != want {
		t.Errorf("%s: stored cursor = %d, want the returned ultNSU %d", when, got, want)
	}
}

// assertReturnedUltNSUFollowed runs three pulls through the jump script and
// checks the stored cursor after each one and the ultNSU of every request.
func assertReturnedUltNSUFollowed(t *testing.T, h *testHelper, source syncstate.SyncSource, fetcher *scriptedFetcher, run func() error) {
	t.Helper()
	steps := []struct {
		when string
		want int64
	}{
		{"after 138 then 137", 90},
		{"after 656", 120},
		{"after 137 at the same ultNSU", 120},
	}
	for _, step := range steps {
		if err := run(); err != nil {
			t.Fatalf("Sync %s: %v", step.when, err)
		}
		assertStoredCursor(t, h, source, step.want, step.when)
	}
	if want := []int64{0, 40, 90, 120}; !slices.Equal(fetcher.cursors, want) {
		t.Errorf("requested ultNSU = %v, want %v (each one the previous response's ultNSU)", fetcher.cursors, want)
	}

	// Only the 656 jump is a gap: the walk from 0 starts at the window's
	// oldest NSU, and a 137 found nothing to deliver.
	gaps, err := h.store.OpenNSUGaps(context.Background(), h.company.ID, source, h.company.Environment)
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 1 || gaps[0].First != 91 || gaps[0].Last != 120 || gaps[0].Cause != NSUGapConsumoIndevido {
		t.Errorf("open gaps = %+v, want 91-120 from the 656", gaps)
	}
}

func TestNFeSourceStoresReturnedUltNSUAfterEveryResponse(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: ultNSUJumpScript(docZip(t, 1, "resNFe_v1.01.xsd", "resnfe.xml"))}

	assertReturnedUltNSUFollowed(t, h.testHelper, syncstate.SyncSourceNFe, fetcher, func() error {
		_, err := h.run(fetcher)
		return err
	})
	if got := len(h.documents()); got != 1 {
		t.Errorf("documents = %d, want 1", got)
	}
}

func TestCTeSourceStoresReturnedUltNSUAfterEveryResponse(t *testing.T) {
	h := newCTeTestHelper(t)
	fetcher := newScriptedCTeFetcher(ultNSUJumpScript(cteDocZip(t, 1, "procCTe_v4.00.xsd", "procte.xml")))

	assertReturnedUltNSUFollowed(t, h.testHelper, syncstate.SyncSourceCTe, fetcher, func() error {
		_, err := h.run(fetcher)
		return err
	})
	if got := len(h.documents()); got != 1 {
		t.Errorf("documents = %d, want 1", got)
	}
}

// TestNSUGapsDateFromTheLastAnswer resumes a distribution whose last answer
// is 100 days old: SEFAZ starts the page at the oldest NSU it still keeps,
// and the NSUs before it, skipped since that answer, are past the window.
// A hole in a later page of the same run dates from the page before it.
func TestNSUGapsDateFromTheLastAnswer(t *testing.T) {
	h := newNFeTestHelper(t)
	h.setNFeCursor(100)
	old := time.Now().Add(-100 * 24 * time.Hour).UTC()
	if _, err := h.db.ExecContext(context.Background(), `UPDATE sync_state SET last_success_at = ?`, old.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	fetcher := &scriptedFetcher{responses: map[int64]sefaz.DistResult{
		100: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 300, MaxNSU: 310, Docs: []sefaz.DocZip{
			docZip(t, 300, "resNFe_v1.01.xsd", "resnfe.xml"),
		}},
		300: {CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 310, MaxNSU: 310, Docs: []sefaz.DocZip{
			docZip(t, 310, "procNFe_v4.00.xsd", "procnfe.xml"),
		}},
	}}

	if _, err := h.run(fetcher); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	gaps, err := h.store.OpenNSUGaps(context.Background(), h.company.ID, syncstate.SyncSourceNFe, h.company.Environment)
	if err != nil {
		t.Fatal(err)
	}
	if len(gaps) != 2 || gaps[0].First != 101 || gaps[0].Last != 299 || gaps[1].First != 301 || gaps[1].Last != 309 {
		t.Fatalf("open gaps = %+v, want 101-299 and 301-309", gaps)
	}
	if !gaps[0].Since.Equal(old.Truncate(time.Second)) {
		t.Errorf("first gap since = %v, want the old answer %v", gaps[0].Since, old)
	}
	if time.Since(gaps[1].Since) > time.Minute {
		t.Errorf("second gap since = %v, want the first page's answer", gaps[1].Since)
	}
	missing, err := h.store.MissingNSUs(context.Background(), h.company.ID, syncstate.SyncSourceNFe, h.company.Environment, time.Now().Add(-NSUGapWindow))
	if err != nil {
		t.Fatal(err)
	}
	if missing != 9 {
		t.Errorf("missing = %d, want only the 9 NSUs of the second page", missing)
	}
}
