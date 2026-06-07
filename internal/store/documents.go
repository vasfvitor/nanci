package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

type DocumentRepository struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{
		db:      db,
		queries: sqlgen.New(db),
	}
}

// GetDocumentByChave retrieves a canonical document by its access key.
func (s *DocumentRepository) GetDocumentByChave(ctx context.Context, chave string) (*nfse.Document, error) {
	query := `
		SELECT
			id, chave_acesso, issue_date, competence,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
			intermediario_cnpj, intermediario_name,
			service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value, total_retentions,
			status, layout_version, xml_path, raw_hash, parse_error, parse_warnings, created_at, updated_at,
			nfse_number, service_description
		FROM documents
		WHERE chave_acesso = ?
	`

	var doc nfse.Document
	var issueDate, createdAt, updatedAt string
	var layoutVersion, parseWarnings, numero, descricao sql.NullString

	err := s.db.QueryRowContext(ctx, query, chave).Scan(
		&doc.ID, &doc.ChaveAcesso, &issueDate, &doc.Competence,
		&doc.PrestadorCNPJ, &doc.PrestadorName, &doc.TomadorCNPJ, &doc.TomadorName,
		&doc.IntermediarioCNPJ, &doc.IntermediarioName,
		&doc.ServiceValue, &doc.ISSValue, &doc.IRRFValue, &doc.INSSValue, &doc.PISValue, &doc.COFINSValue, &doc.CSLLValue, &doc.TotalRetentions,
		&doc.Status, &layoutVersion, &doc.XMLPath, &doc.RawHash, &parseWarnings, &createdAt, &updatedAt,
		&numero, &descricao,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query document by chave: %w", err)
	}

	if issueDate != "" {
		doc.IssueDate, _ = time.Parse(time.RFC3339, issueDate)
	}
	doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	doc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if layoutVersion.Valid {
		doc.LayoutVersion = layoutVersion.String
	}
	if parseWarnings.Valid && parseWarnings.String != "" {
		_ = json.Unmarshal([]byte(parseWarnings.String), &doc.ParseWarnings)
	}
	if numero.Valid {
		doc.NFSeNumber = numero.String
	}
	if descricao.Valid {
		doc.ServiceDescription = descricao.String
	}

	return &doc, nil
}

// UpsertCompanyDocument inserts or updates a company-document relation.
func (s *DocumentRepository) UpsertCompanyDocument(ctx context.Context, doc *nfse.CompanyDocument) error {
	query := `
		INSERT INTO company_documents (
			id, company_id, document_id, company_role, visibility_reason,
			first_seen_nsu, last_seen_nsu, first_synced_at, last_synced_at, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?
		)
		ON CONFLICT(company_id, document_id) DO UPDATE SET
			company_role = excluded.company_role,
			visibility_reason = excluded.visibility_reason,
			first_seen_nsu = CASE
				WHEN company_documents.first_seen_nsu IS NULL THEN excluded.first_seen_nsu
				WHEN excluded.first_seen_nsu IS NULL THEN company_documents.first_seen_nsu
				WHEN excluded.first_seen_nsu < company_documents.first_seen_nsu THEN excluded.first_seen_nsu
				ELSE company_documents.first_seen_nsu
			END,
			last_seen_nsu = CASE
				WHEN company_documents.last_seen_nsu IS NULL THEN excluded.last_seen_nsu
				WHEN excluded.last_seen_nsu IS NULL THEN company_documents.last_seen_nsu
				WHEN excluded.last_seen_nsu > company_documents.last_seen_nsu THEN excluded.last_seen_nsu
				ELSE company_documents.last_seen_nsu
			END,
			first_synced_at = COALESCE(company_documents.first_synced_at, excluded.first_synced_at),
			last_synced_at = excluded.last_synced_at,
			updated_at = excluded.updated_at
	`

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	_, err := s.db.ExecContext(ctx, query,
		doc.RelationID, doc.CompanyID, doc.DocumentID, doc.CompanyRole, doc.VisibilityReason,
		nullableInt64(doc.FirstSeenNSU, doc.FirstSeenNSUValid),
		nullableInt64(doc.LastSeenNSU, doc.LastSeenNSUValid),
		nowStr, nowStr, nowStr, nowStr,
	)

	return err
}

// ListCompanyDocuments retrieves company-facing documents based on the provided filters.
func (s *DocumentRepository) ListCompanyDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter) ([]nfse.CompanyDocument, error) {
	query := `
		SELECT
			d.id, d.chave_acesso, d.issue_date, d.competence,
			d.prestador_cnpj, d.prestador_name, d.tomador_cnpj, d.tomador_name,
			d.intermediario_cnpj, d.intermediario_name,
			d.service_value, d.iss_value, d.irrf_value, d.inss_value, d.pis_value, d.cofins_value, d.csll_value, d.total_retentions,
			d.status, d.layout_version, d.xml_path, d.raw_hash, d.parse_error, d.parse_warnings, d.created_at, d.updated_at,
			d.nfse_number, d.service_description,
			cd.id, cd.company_id, cd.document_id, cd.company_role, cd.visibility_reason,
			cd.first_seen_nsu, cd.last_seen_nsu, cd.first_synced_at, cd.last_synced_at
		FROM company_documents cd
		INNER JOIN documents d ON d.id = cd.document_id
		WHERE cd.company_id = ?
	`
	args := []interface{}{string(companyID)}

	if filter.Competence != "" {
		query += " AND d.competence = ?"
		args = append(args, filter.Competence)
	}
	if filter.Direction != "" {
		query += " AND cd.company_role = ?"
		args = append(args, filter.Direction)
	}
	if filter.Status != "" {
		query += " AND d.status = ?"
		args = append(args, filter.Status)
	}

	query += " ORDER BY d.issue_date DESC, d.chave_acesso DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var docs []nfse.CompanyDocument
	for rows.Next() {
		var d nfse.CompanyDocument
		var issueDate, createdAt, updatedAt string
		var layoutVersion, parseWarnings, numero, descricao sql.NullString
		var firstSeenNSU, lastSeenNSU sql.NullInt64
		var firstSyncedAt, lastSyncedAt string

		if err := rows.Scan(
			&d.ID, &d.ChaveAcesso, &issueDate, &d.Competence,
			&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
			&d.IntermediarioCNPJ, &d.IntermediarioName,
			&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue, &d.TotalRetentions,
			&d.Status, &layoutVersion, &d.XMLPath, &d.RawHash, &parseWarnings, &createdAt, &updatedAt,
			&numero, &descricao,
			&d.RelationID, &d.CompanyID, &d.DocumentID, &d.CompanyRole, &d.VisibilityReason,
			&firstSeenNSU, &lastSeenNSU, &firstSyncedAt, &lastSyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		if issueDate != "" {
			d.IssueDate, _ = time.Parse(time.RFC3339, issueDate)
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if layoutVersion.Valid {
			d.LayoutVersion = layoutVersion.String
		}
		if parseWarnings.Valid && parseWarnings.String != "" {
			_ = json.Unmarshal([]byte(parseWarnings.String), &d.ParseWarnings)
		}
		if numero.Valid {
			d.NFSeNumber = numero.String
		}
		if descricao.Valid {
			d.ServiceDescription = descricao.String
		}
		if firstSeenNSU.Valid {
			d.FirstSeenNSU = firstSeenNSU.Int64
			d.FirstSeenNSUValid = true
		}
		if lastSeenNSU.Valid {
			d.LastSeenNSU = lastSeenNSU.Int64
			d.LastSeenNSUValid = true
		}
		if firstSyncedAt != "" {
			d.FirstSyncedAt, _ = time.Parse(time.RFC3339, firstSyncedAt)
		}
		if lastSyncedAt != "" {
			d.LastSyncedAt, _ = time.Parse(time.RFC3339, lastSyncedAt)
		}

		docs = append(docs, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return docs, nil
}

func (s *DocumentRepository) ListEventsByDocument(ctx context.Context, docID string) ([]nfse.Event, error) {
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



func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt64(value int64, valid bool) interface{} {
	if !valid {
		return nil
	}
	return value
}
