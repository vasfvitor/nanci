package app

import (
	"context"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

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

			if got := status(env, source); got.IdleDays != 0 {
				t.Errorf("never synced: IdleDays = %d, want 0", got.IdleDays)
			}

			// A stale query in the other environment does not count.
			storetest.SetSyncStateLastSuccess(t, env.db, env.company.ID, source, dfe.EnvironmentRestricted, env.company.CNPJ, time.Now().Add(-90*24*time.Hour))
			if got := status(env, source); got.IdleDays > 0 {
				t.Error("an idle homologação cursor warns for produção")
			}

			storetest.SetSyncStateLastSuccess(t, env.db, env.company.ID, source, dfe.EnvironmentProduction, env.company.CNPJ, time.Now().Add(-44*24*time.Hour))
			if got := status(env, source); got.IdleDays != 0 {
				t.Errorf("44 days: IdleDays = %d, want 0", got.IdleDays)
			}

			storetest.SetSyncStateLastSuccess(t, env.db, env.company.ID, source, dfe.EnvironmentProduction, env.company.CNPJ, time.Now().Add(-45*24*time.Hour))
			if got := status(env, source); got.IdleDays != 45 {
				t.Errorf("45 days: IdleDays = %d, want 45", got.IdleDays)
			}
		})
	}
}
