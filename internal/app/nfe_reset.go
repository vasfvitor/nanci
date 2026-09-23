package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// NFeResetResult is what an NF-e reset removed, or would remove, for one
// company.
type NFeResetResult struct {
	CompanyName string
	CNPJ        string
	Environment nfse.Environment
	nfe.ResetCounts
}

// PreviewReset returns what Reset would remove, changing nothing.
func (s *NFeService) PreviewReset(ctx context.Context, cnpj string) (NFeResetResult, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return NFeResetResult{}, err
	}
	counts, err := s.NFeRepo.PreviewResetCompany(ctx, comp.ID)
	if err != nil {
		return NFeResetResult{}, fmt.Errorf("prever redefinição de NF-e: %w", err)
	}
	return resetResult(comp, counts), nil
}

// Reset removes the company's NF-e documents, events and export marks, then
// its NF-e sync cursor and initial-sync flag, so the next pull starts over
// from NSU 0. The manifestações nanci sent stay as the audit trail, and a
// SEFAZ block stays in place. No NF-e pull of the company may run meanwhile.
//
// The documents go first: if the cursor reset then fails, the environment
// stays locked and running Reset again finishes the job.
func (s *NFeService) Reset(ctx context.Context, cnpj string) (NFeResetResult, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return NFeResetResult{}, err
	}
	release, err := s.SyncManager.ReserveSource(comp.ID, nfse.SyncSourceNFe)
	if err != nil {
		return NFeResetResult{}, err
	}
	defer release()

	counts, err := s.NFeRepo.ResetCompany(ctx, comp.ID)
	if err != nil {
		return NFeResetResult{}, fmt.Errorf("redefinir NF-e: %w", err)
	}
	result := resetResult(comp, counts)
	err = s.SyncRepo.ResetSyncState(ctx, nfse.ResetSyncStateParams{CompanyID: comp.ID, Source: nfse.SyncSourceNFe})
	if err != nil {
		return result, fmt.Errorf("as NF-e foram removidas, mas o estado de sincronização não foi redefinido; execute a redefinição novamente: %w", err)
	}
	return result, nil
}

func resetResult(comp *nfse.Company, counts nfe.ResetCounts) NFeResetResult {
	return NFeResetResult{
		CompanyName: comp.Name,
		CNPJ:        comp.CNPJ,
		Environment: comp.Environment,
		ResetCounts: counts,
	}
}
