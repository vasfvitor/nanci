package store

import (
	"context"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// CreateSyncRun inserts a new sync run record into the database.
func (s *SQLiteStore) CreateSyncRun(ctx context.Context, run *nfse.SyncRun) error {
	query := `
		INSERT INTO sync_runs (
			id, company_id, credential_id, credential_cnpj, consultation_cnpj, consultation_basis,
			started_at, from_nsu, to_nsu, documents_found, errors_count, status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	startedAt := run.StartedAt.UTC().Format(time.RFC3339)

	_, err := s.db.ExecContext(ctx, query,
		run.ID, run.CompanyID, nullableString(run.CredentialID), nullableString(run.CredentialCNPJ),
		nullableString(run.ConsultationCNPJ), nullableString(run.ConsultationBasis),
		startedAt, run.FromNSU, run.ToNSU, run.DocumentsFound, run.ErrorsCount, run.Status,
	)
	return err
}

// UpdateSyncRun updates an existing sync run record.
func (s *SQLiteStore) UpdateSyncRun(ctx context.Context, run *nfse.SyncRun) error {
	query := `
		UPDATE sync_runs
		SET finished_at = ?, to_nsu = ?, documents_found = ?, errors_count = ?, status = ?
		WHERE id = ?
	`
	var finishedAt *string
	if run.FinishedAt != nil {
		f := run.FinishedAt.UTC().Format(time.RFC3339)
		finishedAt = &f
	}

	_, err := s.db.ExecContext(ctx, query,
		finishedAt, run.ToNSU, run.DocumentsFound, run.ErrorsCount, run.Status, run.ID,
	)
	return err
}
