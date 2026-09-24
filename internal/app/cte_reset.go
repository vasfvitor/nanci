package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// CTeResetResult is what a CT-e reset removed, or would remove, for one
// company.
type CTeResetResult struct {
	CompanyName string
	CNPJ        string
	Environment nfse.Environment
	cte.ResetCounts
}

// PreviewReset returns what Reset would remove, changing nothing.
func (s *CTeService) PreviewReset(ctx context.Context, cnpj string) (CTeResetResult, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return CTeResetResult{}, err
	}
	counts, err := s.CTeRepo.PreviewResetCompany(ctx, comp.ID)
	if err != nil {
		return CTeResetResult{}, fmt.Errorf("prever redefinição de CT-e: %w", err)
	}
	return cteResetResult(comp, counts), nil
}

// Reset removes, in one transaction, the company's CT-e documents, events and
// export marks, and its CT-e sync cursor and initial-sync flag, so the next
// pull starts over from NSU 0. Documents another company also sees stay, and
// a SEFAZ block stays in place. No CT-e pull of the company may run
// meanwhile: it fails with ErrSyncRunning.
func (s *CTeService) Reset(ctx context.Context, cnpj string) (CTeResetResult, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return CTeResetResult{}, err
	}
	release, err := s.SyncManager.ReserveSource(comp.ID, nfse.SyncSourceCTe)
	if err != nil {
		return CTeResetResult{}, err
	}
	defer release()

	counts, err := s.CTeRepo.ResetCompany(ctx, comp.ID)
	if err != nil {
		return CTeResetResult{}, fmt.Errorf("redefinir CT-e: %w", err)
	}
	return cteResetResult(comp, counts), nil
}

func cteResetResult(comp *nfse.Company, counts cte.ResetCounts) CTeResetResult {
	return CTeResetResult{
		CompanyName: comp.Name,
		CNPJ:        comp.CNPJ,
		Environment: comp.Environment,
		ResetCounts: counts,
	}
}
