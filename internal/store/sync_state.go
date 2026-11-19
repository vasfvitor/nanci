package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/vasfvitor/nanci/internal/syncstate"
)

// ResetSyncStateTx deletes the source's sync cursor and NSU gaps and clears
// its initial-sync flag inside tx, in every environment of the company. It
// keeps blocked_until: a local reset does not lift a wait imposed by the tax
// authority.
func ResetSyncStateTx(ctx context.Context, tx *sql.Tx, params syncstate.ResetSyncStateParams) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM sync_state WHERE company_id = ? AND source = ?`,
		string(params.CompanyID), string(params.Source),
	); err != nil {
		return err
	}
	// The cursor goes back to 0, so the next pull walks every gap again.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM sync_nsu_gaps WHERE company_id = ? AND source = ?`,
		string(params.CompanyID), string(params.Source),
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE company_sync_sources
		SET initial_sync_completed_at = NULL, updated_at = ?
		WHERE company_id = ? AND source = ?
	`, now, string(params.CompanyID), string(params.Source)); err != nil {
		return err
	}
	return nil
}
