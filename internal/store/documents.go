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
			status, layout_version, xml_path, raw_hash, parse_warnings,
			nfse_number, service_description, created_at, updated_at
		FROM documents
		WHERE chave_acesso = ?
	`

	var doc nfse.Document
	var issueDate, createdAt, updatedAt string
	var parseWarnings sql.NullString

	err := s.db.QueryRowContext(ctx, query, chave).Scan(
		&doc.ID, &doc.ChaveAcesso, &issueDate, &doc.Competence,
		&doc.PrestadorCNPJ, &doc.PrestadorName, &doc.TomadorCNPJ, &doc.TomadorName,
		&doc.IntermediarioCNPJ, &doc.IntermediarioName,
		&doc.ServiceValue, &doc.ISSValue, &doc.IRRFValue, &doc.INSSValue, &doc.PISValue, &doc.COFINSValue, &doc.CSLLValue, &doc.TotalRetentions,
		&doc.Status, &doc.LayoutVersion, &doc.XMLPath, &doc.RawHash, &parseWarnings,
		&doc.NFSeNumber, &doc.ServiceDescription, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query document by chave: %w", err)
	}

	doc.IssueDate, err = parseRequiredTime("document issue_date", issueDate)
	if err != nil {
		return nil, err
	}
	doc.CreatedAt, err = parseRequiredTime("document created_at", createdAt)
	if err != nil {
		return nil, err
	}
	doc.UpdatedAt, err = parseRequiredTime("document updated_at", updatedAt)
	if err != nil {
		return nil, err
	}
	if err := decodeWarnings(parseWarnings, &doc.ParseWarnings); err != nil {
		return nil, fmt.Errorf("document parse_warnings: %w", err)
	}

	return &doc, nil
}

// ListCompanyDocuments retrieves company-facing documents based on the provided filters.
func (s *DocumentRepository) ListCompanyDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter) ([]nfse.CompanyDocument, error) {
	query := `
		SELECT
			d.id, d.chave_acesso, d.issue_date, d.competence,
			d.prestador_cnpj, d.prestador_name, d.tomador_cnpj, d.tomador_name,
			d.intermediario_cnpj, d.intermediario_name,
			d.service_value, d.iss_value, d.irrf_value, d.inss_value, d.pis_value, d.cofins_value, d.csll_value, d.total_retentions,
			d.status, d.layout_version, d.xml_path, d.raw_hash, d.parse_warnings, d.created_at, d.updated_at,
			d.nfse_number, d.service_description,
			cd.relation_id, cd.company_id, cd.document_id, cd.company_role, cd.visibility_reason,
			cd.first_seen_nsu, cd.last_seen_nsu,
			cd.first_seen_nsu_valid, cd.last_seen_nsu_valid,
			cd.first_synced_at, cd.last_synced_at
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
	if filter.FromNSU != nil {
		query += " AND cd.last_seen_nsu_valid = 1 AND cd.last_seen_nsu >= ?"
		args = append(args, *filter.FromNSU)
	}
	if filter.ToNSU != nil {
		query += " AND cd.first_seen_nsu_valid = 1 AND cd.first_seen_nsu <= ?"
		args = append(args, *filter.ToNSU)
	}

	query += " ORDER BY d.issue_date DESC, d.chave_acesso DESC"
	if filter.Limit != nil && *filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, *filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var docs []nfse.CompanyDocument
	for rows.Next() {
		var d nfse.CompanyDocument
		var issueDate, createdAt, updatedAt string
		var parseWarnings sql.NullString
		var firstSeenValid, lastSeenValid int64
		var firstSyncedAt, lastSyncedAt string

		if err := rows.Scan(
			&d.ID, &d.ChaveAcesso, &issueDate, &d.Competence,
			&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
			&d.IntermediarioCNPJ, &d.IntermediarioName,
			&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue, &d.TotalRetentions,
			&d.Status, &d.LayoutVersion, &d.XMLPath, &d.RawHash, &parseWarnings, &createdAt, &updatedAt,
			&d.NFSeNumber, &d.ServiceDescription,
			&d.RelationID, &d.CompanyID, &d.DocumentID, &d.CompanyRole, &d.VisibilityReason,
			&d.FirstSeenNSU, &d.LastSeenNSU, &firstSeenValid, &lastSeenValid,
			&firstSyncedAt, &lastSyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		d.IssueDate, err = parseRequiredTime("document issue_date", issueDate)
		if err != nil {
			return nil, err
		}
		d.CreatedAt, err = parseRequiredTime("document created_at", createdAt)
		if err != nil {
			return nil, err
		}
		d.UpdatedAt, err = parseRequiredTime("document updated_at", updatedAt)
		if err != nil {
			return nil, err
		}
		if err := decodeWarnings(parseWarnings, &d.ParseWarnings); err != nil {
			return nil, fmt.Errorf("document parse_warnings: %w", err)
		}
		d.FirstSeenNSUValid = firstSeenValid != 0
		d.LastSeenNSUValid = lastSeenValid != 0
		d.FirstSyncedAt, err = parseRequiredTime("company document first_synced_at", firstSyncedAt)
		if err != nil {
			return nil, err
		}
		d.LastSyncedAt, err = parseRequiredTime("company document last_synced_at", lastSyncedAt)
		if err != nil {
			return nil, err
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
			id, document_id, chave_acesso, type, event_at, event_at_valid, replacement_chave_acesso,
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
		var documentID, eventAt, parseWarnings sql.NullString
		var eventAtValid int64
		var createdAt string

		if err := rows.Scan(
			&event.ID,
			&documentID,
			&event.ChaveAcesso,
			&event.Type,
			&eventAt,
			&eventAtValid,
			&event.ReplacementChaveAcesso,
			&event.Description,
			&event.RawXMLPath,
			&event.RawHash,
			&parseWarnings,
			&createdAt,
		); err != nil {
			return nil, err
		}

		if documentID.Valid {
			event.DocumentID = nfse.DocumentID(documentID.String)
		}
		if eventAtValid != 0 {
			if !eventAt.Valid {
				return nil, errors.New("event event_at is required when event_at_valid is set")
			}
			event.EventAt, err = parseRequiredTime("event event_at", eventAt.String)
			if err != nil {
				return nil, err
			}
			event.EventAtValid = true
		}
		if err := decodeWarnings(parseWarnings, &event.ParseWarnings); err != nil {
			return nil, fmt.Errorf("event parse_warnings: %w", err)
		}
		event.CreatedAt, err = parseRequiredTime("event created_at", createdAt)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func parseRequiredTime(field, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: %w", field, err)
	}
	return parsed, nil
}

func decodeWarnings(value sql.NullString, dst *[]string) error {
	if !value.Valid || value.String == "" {
		return nil
	}
	return json.Unmarshal([]byte(value.String), dst)
}
