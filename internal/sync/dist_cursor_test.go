package sync

import (
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
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

func assertStoredCursor(t *testing.T, h *testHelper, source nfse.SyncSource, want int64, when string) {
	t.Helper()
	if got, _ := h.sourceCursor(source); got != want {
		t.Errorf("%s: stored cursor = %d, want the returned ultNSU %d", when, got, want)
	}
}

// assertReturnedUltNSUFollowed runs three pulls through the jump script and
// checks the stored cursor after each one and the ultNSU of every request.
func assertReturnedUltNSUFollowed(t *testing.T, h *testHelper, source nfse.SyncSource, fetcher *scriptedFetcher, run func() error) {
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
}

func TestNFeSourceStoresReturnedUltNSUAfterEveryResponse(t *testing.T) {
	h := newNFeTestHelper(t)
	fetcher := &scriptedFetcher{responses: ultNSUJumpScript(docZip(t, 1, "resNFe_v1.01.xsd", "resnfe.xml"))}

	assertReturnedUltNSUFollowed(t, h.testHelper, nfse.SyncSourceNFe, fetcher, func() error {
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

	assertReturnedUltNSUFollowed(t, h.testHelper, nfse.SyncSourceCTe, fetcher, func() error {
		_, err := h.run(fetcher)
		return err
	})
	if got := len(h.documents()); got != 1 {
		t.Errorf("documents = %d, want 1", got)
	}
}
