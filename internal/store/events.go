package store

import (
	"context"
	"fmt"
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

	doc, err := s.GetDocumentByChave(ctx, event.ChaveAcesso)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("documento não encontrado para evento: %s", event.ChaveAcesso)
	}

	query := `
		INSERT INTO events (
			id, document_id, company_id, chave_acesso, event_type, event_date, description, raw_hash, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(chave_acesso, event_type) DO NOTHING
	`
	now := time.Now().UTC().Format(time.RFC3339)
	issueDate := event.IssueDate.UTC().Format(time.RFC3339)

	res, err := tx.ExecContext(ctx, query,
		event.ID, doc.ID, event.CompanyID, event.ChaveAcesso, event.Type, issueDate, event.Details, event.RawHash, now,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 && event.Type == "cancelamento" {
		// Update document status to canceled
		updateDoc := `UPDATE documents SET status = 'cancelada', updated_at = ? WHERE chave_acesso = ?`
		if _, err := tx.ExecContext(ctx, updateDoc, now, event.ChaveAcesso); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error) {
	// Simple stub to satisfy interface for now
	return []nfse.Event{}, nil
}
