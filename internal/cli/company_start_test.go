package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// TestCompanyUpdate_SyncStartDate walks the fix for a start date set too
// late: the update is refused while the NFS-e has a cursor and goes through
// after `nanci reset --confirmar`. Every update passes the date, since flag
// values stick to a command tree.
func TestCompanyUpdate_SyncStartDate(t *testing.T) {
	env := newNFeTestRoot(t)
	ctx := context.Background()
	startDate := func() string {
		t.Helper()
		c, err := company.NewStore(env.db).CompanyByCNPJ(ctx, nfeTestCNPJ)
		if err != nil {
			t.Fatal(err)
		}
		if c.SyncStartPolicy != syncstate.SyncStartPolicySinceDate || c.SyncStartDate == nil {
			t.Fatalf("start policy = %q, %v; want since_date with a date", c.SyncStartPolicy, c.SyncStartDate)
		}
		return c.SyncStartDate.Format("2006-01-02")
	}

	// Before any NFS-e sync, a date alone sets since_date.
	if err := env.run("company", "update", "-c", nfeTestCNPJ, "--sync-start-date", "2024-01-01"); err != nil {
		t.Fatalf("update before the first sync: %v", err)
	}
	if got := startDate(); got != "2024-01-01" {
		t.Errorf("start date = %s, want 2024-01-01", got)
	}

	seedNFSeCursors(t, env)
	err := env.run("company", "update", "-c", nfeTestCNPJ, "--sync-start-date", "2023-01-01")
	if !errors.Is(err, company.ErrSyncStartPolicyLocked) || !strings.Contains(err.Error(), "nanci reset -c "+nfeTestCNPJ+" --confirmar") {
		t.Fatalf("update with an NFS-e cursor = %v, want ErrSyncStartPolicyLocked with the reset hint", err)
	}
	if got := startDate(); got != "2024-01-01" {
		t.Errorf("start date after the refusal = %s, want 2024-01-01", got)
	}

	if err := env.run("reset", "-c", nfeTestCNPJ, "--confirmar"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := env.run("company", "update", "-c", nfeTestCNPJ, "--sync-start-date", "2023-01-01"); err != nil {
		t.Fatalf("update after the reset: %v", err)
	}
	if got := startDate(); got != "2023-01-01" {
		t.Errorf("start date after the reset = %s, want 2023-01-01", got)
	}
}

func TestCompanyUpdate_RejectsInvalidSyncStart(t *testing.T) {
	env := newNFeTestRoot(t)
	err := env.run("company", "update", "-c", nfeTestCNPJ, "--sync-start-policy", "all", "--sync-start-date", "2024-01-01")
	if err == nil || !strings.Contains(err.Error(), "erro na política de sincronização") {
		t.Fatalf("update with all and a date = %v, want a policy error", err)
	}
}
