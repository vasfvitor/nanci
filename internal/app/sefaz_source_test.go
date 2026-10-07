package app

import (
	"context"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// setLastQuery stores a sync_state row whose last distNSU answer was at.
func (e *nfeTestEnv) setLastQuery(source syncstate.SyncSource, env dfe.Environment, at time.Time) {
	e.t.Helper()
	ts := at.UTC().Format(time.RFC3339)
	_, err := e.db.ExecContext(context.Background(), `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, last_success_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 10, ?, ?, ?)
		ON CONFLICT (company_id, source, environment, consultation_cnpj) DO UPDATE SET last_success_at = excluded.last_success_at
	`, string(e.company.ID), string(source), string(env), e.company.CNPJ, ts, ts, ts)
	if err != nil {
		e.t.Fatalf("set last query: %v", err)
	}
}

func TestSefazStatusWarnsWhenDistributionIdle(t *testing.T) {
	ctx := context.Background()
	status := func(env *nfeTestEnv, source syncstate.SyncSource) SefazSourceStatus {
		t.Helper()
		if source == syncstate.SyncSourceNFe {
			res, err := env.app.NFe.Status(ctx, nfeTestCNPJ)
			if err != nil {
				t.Fatal(err)
			}
			return res.SefazSourceStatus
		}
		res, err := env.app.CTe.Status(ctx, nfeTestCNPJ)
		if err != nil {
			t.Fatal(err)
		}
		return res.SefazSourceStatus
	}

	for _, source := range []syncstate.SyncSource{syncstate.SyncSourceNFe, syncstate.SyncSourceCTe} {
		t.Run(string(source), func(t *testing.T) {
			env := newNFeTestEnv(t)

			if got := status(env, source); got.NSUEmRisco || got.IdleDays != 0 {
				t.Errorf("never synced: (IdleDays, NSUEmRisco) = (%d, %v), want (0, false)", got.IdleDays, got.NSUEmRisco)
			}

			// A stale query in the other environment does not count.
			env.setLastQuery(source, dfe.EnvironmentRestricted, time.Now().Add(-90*24*time.Hour))
			if got := status(env, source); got.NSUEmRisco {
				t.Error("an idle homologação cursor warns for produção")
			}

			env.setLastQuery(source, dfe.EnvironmentProduction, time.Now().Add(-44*24*time.Hour))
			if got := status(env, source); got.NSUEmRisco || got.IdleDays != 44 {
				t.Errorf("44 days: (IdleDays, NSUEmRisco) = (%d, %v), want (44, false)", got.IdleDays, got.NSUEmRisco)
			}

			env.setLastQuery(source, dfe.EnvironmentProduction, time.Now().Add(-45*24*time.Hour))
			if got := status(env, source); !got.NSUEmRisco || got.IdleDays != 45 {
				t.Errorf("45 days: (IdleDays, NSUEmRisco) = (%d, %v), want (45, true)", got.IdleDays, got.NSUEmRisco)
			}
		})
	}
}
