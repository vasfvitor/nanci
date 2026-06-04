package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// SaveDocument inserts a new document or ignores if it already exists (by chave_acesso).
func (s *SQLiteStore) SaveDocument(ctx context.Context, doc *nfse.Document) error {
	query := `
		INSERT INTO documents (
			id, company_id, chave_acesso, nsu, direction, issue_date, competence,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
			service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value,
			status, xml_path, raw_hash, parse_error, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?
		)
		ON CONFLICT(chave_acesso) DO UPDATE SET
			nsu = excluded.nsu,
			status = excluded.status,
			updated_at = excluded.updated_at
	`
	now := time.Now().UTC().Format(time.RFC3339)
	var issueDate string
	if !doc.IssueDate.IsZero() {
		issueDate = doc.IssueDate.UTC().Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx, query,
		doc.ID, doc.CompanyID, doc.ChaveAcesso, doc.NSU, doc.Direction, issueDate, doc.Competence,
		doc.PrestadorCNPJ, doc.PrestadorName, doc.TomadorCNPJ, doc.TomadorName,
		doc.ServiceValue, doc.ISSValue, doc.IRRFValue, doc.INSSValue, doc.PISValue, doc.COFINSValue, doc.CSLLValue,
		doc.Status, doc.XMLPath, doc.RawHash, doc.ParseError, now, now,
	)

	return err
}

// GetDocumentByChave retrieves a single document by its access key.
func (s *SQLiteStore) GetDocumentByChave(ctx context.Context, chave string) (*nfse.Document, error) {
	// Not implemented fully yet, just a stub for now to satisfy interface
	return nil, errors.New("GetDocumentByChave not implemented")
}

// ListDocuments retrieves documents based on the provided filters.
func (s *SQLiteStore) ListDocuments(ctx context.Context, companyID string, filter DocumentFilter) ([]nfse.Document, error) {
	query := `
		SELECT 
			id, company_id, chave_acesso, nsu, direction, issue_date, competence,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
			service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value,
			status, xml_path, raw_hash, parse_error, created_at, updated_at
		FROM documents
		WHERE company_id = ?
	`
	args := []interface{}{companyID}

	if filter.Competence != "" {
		query += " AND competence = ?"
		args = append(args, filter.Competence)
	}
	if filter.Direction != "" {
		query += " AND direction = ?"
		args = append(args, filter.Direction)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	query += " ORDER BY issue_date DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var docs []nfse.Document
	for rows.Next() {
		var d nfse.Document
		var issueDate, createdAt, updatedAt string
		var parseError sql.NullString

		if err := rows.Scan(
			&d.ID, &d.CompanyID, &d.ChaveAcesso, &d.NSU, &d.Direction, &issueDate, &d.Competence,
			&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
			&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue,
			&d.Status, &d.XMLPath, &d.RawHash, &parseError, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		if issueDate != "" {
			d.IssueDate, _ = time.Parse(time.RFC3339, issueDate)
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		d.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if parseError.Valid {
			d.ParseError = parseError.String
		}

		docs = append(docs, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return docs, nil
}

// GetCompanyStats returns aggregated statistics for a company.
func (s *SQLiteStore) GetCompanyStats(ctx context.Context, companyID string) (*CompanyStats, error) {
	// Not fully implemented, return empty for now
	return &CompanyStats{}, nil
}


