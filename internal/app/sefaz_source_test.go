package app

import (
	"context"
	"errors"
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

// failingPassword answers every prompt with err.
type failingPassword struct{ err error }

func (p failingPassword) GetCertPassword(context.Context, CertPasswordRequest) ([]byte, error) {
	return nil, p.err
}

// TestSefazConnectionTestStopsWhenThePromptIsCancelled checks that a
// connection test whose password prompt ends without a password returns the
// cancellation, whether the user closed the dialog or the desktop call was
// cancelled, instead of a result that reads as a certificate failure.
func TestSefazConnectionTestStopsWhenThePromptIsCancelled(t *testing.T) {
	for _, cancelErr := range []error{ErrOperationCanceled, context.Canceled} {
		t.Run(cancelErr.Error(), func(t *testing.T) {
			env := newNFeTestEnv(t)
			stub := useFakeSEFAZ(t)
			env.app.NFe.Certificates.Passwords = failingPassword{err: cancelErr}

			result, err := env.app.NFe.TestConnection(context.Background(), nfeTestCNPJ)
			if !errors.Is(err, cancelErr) {
				t.Fatalf("TestConnection = %+v, %v; want %v", result, err, cancelErr)
			}
			if stub.tlsCalls != 0 {
				t.Errorf("TLS checks = %d, want 0", stub.tlsCalls)
			}
		})
	}
}
