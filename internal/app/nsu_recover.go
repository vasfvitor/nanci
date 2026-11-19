package app

import (
	"context"

	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// RecoverNSUInput names the company whose missing NSUs to ask SEFAZ for.
// Limit caps the requests of the run; 0 spends what the hour has left minus
// the sync.RecoverNSUReserve kept for the pull.
type RecoverNSUInput struct {
	CNPJ  string
	Limit int
}

// RecoverNSUResult counts what one recovery did; see sync.RecoverNSUResult.
type RecoverNSUResult = sync.RecoverNSUResult

// RecoverNSU asks the NF-e distribution for each NSU it skipped without a
// document (consNSU), spending the hour's budget; see sync.Manager.RecoverNSU.
func (s *NFeService) RecoverNSU(ctx context.Context, in RecoverNSUInput) (RecoverNSUResult, error) {
	return s.SyncManager.RecoverNSU(ctx, sync.RecoverNSUInput{CNPJ: in.CNPJ, Source: syncstate.SyncSourceNFe, Limit: in.Limit})
}

// RecoverNSU asks the CT-e distribution for each NSU it skipped without a
// document (consNSU), spending the hour's budget; see sync.Manager.RecoverNSU.
func (s *CTeService) RecoverNSU(ctx context.Context, in RecoverNSUInput) (RecoverNSUResult, error) {
	return s.SyncManager.RecoverNSU(ctx, sync.RecoverNSUInput{CNPJ: in.CNPJ, Source: syncstate.SyncSourceCTe, Limit: in.Limit})
}
