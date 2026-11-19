package sync

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/syncstate"
)

// otherProcess is a second Manager on the same database: it shares the
// leases in source_locks but not the in-memory marks, as the desktop app and
// the CLI do.
func otherProcess(mgr *Manager, holder string) *Manager {
	return &Manager{Log: mgr.Log, SyncRepo: mgr.SyncRepo, Holder: holder}
}

func TestReserveSourceRefusesAnotherProcess(t *testing.T) {
	ctx := context.Background()
	cli, comp := newPullTestManager(t, &countingProvider{})
	cli.Holder = "CLI"
	desktop := otherProcess(cli, "aplicativo")

	release, err := desktop.ReserveSource(ctx, comp.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cli.ReserveSource(ctx, comp.ID, syncstate.SyncSourceNFe)
	if !errors.Is(err, ErrSyncRunning) || !strings.Contains(err.Error(), "outro processo do nanci (aplicativo)") {
		t.Fatalf("second process = %v, want ErrSyncRunning naming the app", err)
	}
	// Another source of the same company is free.
	releaseCTe, err := cli.ReserveSource(ctx, comp.ID, syncstate.SyncSourceCTe)
	if err != nil {
		t.Fatalf("CT-e while NF-e is held: %v", err)
	}
	releaseCTe()

	release()
	releaseAgain, err := cli.ReserveSource(ctx, comp.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatalf("after the release: %v", err)
	}
	releaseAgain()
}

func TestReserveSourceTakesOverAnExpiredLease(t *testing.T) {
	ctx := context.Background()
	cli, comp := newPullTestManager(t, &countingProvider{})
	// A process that crashed while holding the source left its row behind.
	if _, err := cli.SyncRepo.db.ExecContext(ctx, `
		INSERT INTO source_locks (company_id, source, owner, holder, pid, acquired_at, expires_at)
		VALUES (?, 'nfe', 'gone', 'aplicativo', 1, '2026-01-01T10:00:00.000Z', '2026-01-01T10:01:00.000Z')
	`, string(comp.ID)); err != nil {
		t.Fatal(err)
	}

	release, err := cli.ReserveSource(ctx, comp.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatalf("expired lease not taken over: %v", err)
	}
	release()
}

func TestReserveSourceRenewsTheLeaseWhileHeld(t *testing.T) {
	original := sourceLockTTL
	sourceLockTTL = 300 * time.Millisecond
	t.Cleanup(func() { sourceLockTTL = original })

	ctx := context.Background()
	desktop, comp := newPullTestManager(t, &countingProvider{})
	desktop.Holder = "aplicativo"
	cli := otherProcess(desktop, "CLI")

	release, err := desktop.ReserveSource(ctx, comp.ID, syncstate.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	// Three TTLs later the renewals still keep the other process out.
	time.Sleep(3 * sourceLockTTL)
	if _, err := cli.ReserveSource(ctx, comp.ID, syncstate.SyncSourceNFe); !errors.Is(err, ErrSyncRunning) {
		t.Fatalf("other process during a renewed hold = %v, want ErrSyncRunning", err)
	}
}
