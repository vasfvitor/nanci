package store

import (
	"context"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// SaveEvent inserts a new event into the database and updates the document status if needed.
func (s *SQLiteStore) SaveEvent(ctx context.Context, event *nfse.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO events (
			id, company_id, chave_acesso, event_type, issue_date, details, raw_hash, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chave_acesso, event_type) DO NOTHING
	`
	now := time.Now().UTC().Format(time.RFC3339)
	issueDate := event.IssueDate.UTC().Format(time.RFC3339)

	res, err := tx.ExecContext(ctx, query,
		event.ID, event.CompanyID, event.ChaveAcesso, event.Type, issueDate, event.Details, event.RawHash, now,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 && event.Type == "cancelamento" {
		// Update document status to canceled
		updateDoc := `UPDATE documents SET status = 'cancelado', updated_at = ? WHERE chave_acesso = ? AND company_id = ?`
		if _, err := tx.ExecContext(ctx, updateDoc, now, event.ChaveAcesso, event.CompanyID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error) {
	// Simple stub to satisfy interface for now
	return []nfse.Event{}, nil
}
