package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// SaveEvent inserts a canonical event ledger row and refreshes the document status.
func (s *SQLiteStore) SaveEvent(ctx context.Context, event *nfse.Event) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	doc, err := getDocumentByChaveTx(ctx, tx, event.ChaveAcesso)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("documento não encontrado para evento: %s", event.ChaveAcesso)
	}

	event.DocumentID = doc.ID

	query := `
		INSERT INTO events (
			id, document_id, chave_acesso, event_type, event_at, replacement_chave_acesso,
			description, raw_xml_path, raw_hash, parse_warnings, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(raw_hash) DO NOTHING
	`
	now := time.Now().UTC().Format(time.RFC3339)

	var warningsJSON []byte
	if len(event.ParseWarnings) > 0 {
		warningsJSON, _ = json.Marshal(event.ParseWarnings)
	}

	res, err := tx.ExecContext(ctx, query,
		event.ID,
		event.DocumentID,
		event.ChaveAcesso,
		event.Type,
		nullableTime(event.EventAt, event.EventAtValid),
		nullableString(event.ReplacementChaveAcesso),
		nullableString(event.Description),
		event.RawXMLPath,
		event.RawHash,
		nullableString(string(warningsJSON)),
		now,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		if err := recomputeDocumentStatusTx(ctx, tx, event.ChaveAcesso); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error) {
	query := `
		SELECT
			id, document_id, chave_acesso, event_type, event_at, replacement_chave_acesso,
			description, raw_xml_path, raw_hash, parse_warnings, created_at
		FROM events
		WHERE document_id = ?
		ORDER BY
			CASE WHEN event_at IS NULL THEN 1 ELSE 0 END,
			event_at ASC,
			created_at ASC,
			id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []nfse.Event
	for rows.Next() {
		var event nfse.Event
		var eventAt, createdAt sql.NullString
		var replacementChave, description, parseWarnings sql.NullString

		if err := rows.Scan(
			&event.ID,
			&event.DocumentID,
			&event.ChaveAcesso,
			&event.Type,
			&eventAt,
			&replacementChave,
			&description,
			&event.RawXMLPath,
			&event.RawHash,
			&parseWarnings,
			&createdAt,
		); err != nil {
			return nil, err
		}

		if eventAt.Valid {
			parsed, err := time.Parse(time.RFC3339, eventAt.String)
			if err == nil {
				event.EventAt = parsed
				event.EventAtValid = true
			}
		}
		if replacementChave.Valid {
			event.ReplacementChaveAcesso = replacementChave.String
		}
		if description.Valid {
			event.Description = description.String
		}
		if parseWarnings.Valid && parseWarnings.String != "" {
			_ = json.Unmarshal([]byte(parseWarnings.String), &event.ParseWarnings)
		}
		if createdAt.Valid {
			event.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (s *SQLiteStore) recomputeDocumentStatus(ctx context.Context, chaveAcesso string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := recomputeDocumentStatusTx(ctx, tx, chaveAcesso); err != nil {
		return err
	}

	return tx.Commit()
}

func recomputeDocumentStatusTx(ctx context.Context, tx *sql.Tx, chaveAcesso string) error {
	query := `
		SELECT event_type
		FROM events
		WHERE chave_acesso = ?
	`
	rows, err := tx.QueryContext(ctx, query, chaveAcesso)
	if err != nil {
		return err
	}
	defer rows.Close()

	status := "normal"
	hasCancellation := false
	hasSubstitution := false

	for rows.Next() {
		var eventType string
		if err := rows.Scan(&eventType); err != nil {
			return err
		}
		switch eventType {
		case "substituicao":
			hasSubstitution = true
		case "cancelamento":
			hasCancellation = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	switch {
	case hasSubstitution:
		status = "substituida"
	case hasCancellation:
		status = "cancelada"
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE documents SET status = ?, updated_at = ? WHERE chave_acesso = ?`,
		status,
		time.Now().UTC().Format(time.RFC3339),
		chaveAcesso,
	)
	return err
}

func getDocumentByChaveTx(ctx context.Context, tx *sql.Tx, chave string) (*nfse.Document, error) {
	query := `
		SELECT id, chave_acesso, status
		FROM documents
		WHERE chave_acesso = ?
	`

	var doc nfse.Document
	err := tx.QueryRowContext(ctx, query, chave).Scan(&doc.ID, &doc.ChaveAcesso, &doc.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &doc, nil
}

func nullableTime(value time.Time, valid bool) interface{} {
	if !valid || value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
