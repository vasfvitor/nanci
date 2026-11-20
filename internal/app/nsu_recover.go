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

// RecoverNSU asks the NF-e distribution for each NSU it skipped without a
// document (consNSU), spending the hour's budget; see sync.Manager.RecoverNSU.
func (s *NFeService) RecoverNSU(ctx context.Context, in RecoverNSUInput) (sync.RecoverNSUResult, error) {
	return s.SyncManager.RecoverNSU(ctx, sync.RecoverNSUInput{CNPJ: in.CNPJ, Source: syncstate.SyncSourceNFe, Limit: in.Limit})
}

// RecoverNSU asks the CT-e distribution for each NSU it skipped without a
// document (consNSU), spending the hour's budget; see sync.Manager.RecoverNSU.
func (s *CTeService) RecoverNSU(ctx context.Context, in RecoverNSUInput) (sync.RecoverNSUResult, error) {
	return s.SyncManager.RecoverNSU(ctx, sync.RecoverNSUInput{CNPJ: in.CNPJ, Source: syncstate.SyncSourceCTe, Limit: in.Limit})
}

// DiscardNSUInput names the company whose missing NSUs to discard. Without
// Confirm nothing changes and the count is what would be discarded.
type DiscardNSUInput struct {
	CNPJ    string
	Confirm bool
}

// DiscardNSU closes the NF-e gaps so recovery never asks for them; see
// sync.Manager.DiscardNSU.
func (s *NFeService) DiscardNSU(ctx context.Context, in DiscardNSUInput) (int64, error) {
	return s.SyncManager.DiscardNSU(ctx, sync.DiscardNSUInput{CNPJ: in.CNPJ, Source: syncstate.SyncSourceNFe, Confirm: in.Confirm})
}

// DiscardNSU closes the CT-e gaps so recovery never asks for them; see
// sync.Manager.DiscardNSU.
func (s *CTeService) DiscardNSU(ctx context.Context, in DiscardNSUInput) (int64, error) {
	return s.SyncManager.DiscardNSU(ctx, sync.DiscardNSUInput{CNPJ: in.CNPJ, Source: syncstate.SyncSourceCTe, Confirm: in.Confirm})
}
