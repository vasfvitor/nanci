package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfse"
)

type ResetSyncInput struct {
	CNPJ string
}

// ResetSyncState clears the persisted sync cursor for a company without deleting documents.
func (a *App) ResetSyncState(ctx context.Context, input ResetSyncInput) error {
	company, err := a.companyByCNPJ(ctx, input.CNPJ)
	if err != nil {
		return err
	}

	if err := a.SyncRepo.ResetSyncState(ctx, nfse.ResetSyncStateParams{
		CompanyID: company.ID,
	}); err != nil {
		return fmt.Errorf("resetar estado de sincronização: %w", err)
	}

	return nil
}
