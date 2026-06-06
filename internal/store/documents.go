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

// UpsertDocument inserts or updates a canonical document identified by chave_acesso.
func (s *SQLiteStore) UpsertDocument(ctx context.Context, doc *nfse.Document) error {
	query := `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
			intermediario_cnpj, intermediario_name,
			service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value, total_retentions,
			status, layout_version, xml_path, raw_hash, parse_error, parse_warnings, created_at, updated_at,
			nfse_number, service_description
		) VALUES (
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?
		)
		ON CONFLICT(chave_acesso) DO UPDATE SET
			issue_date = excluded.issue_date,
			competence = excluded.competence,
			prestador_cnpj = excluded.prestador_cnpj,
			prestador_name = excluded.prestador_name,
			tomador_cnpj = excluded.tomador_cnpj,
			tomador_name = excluded.tomador_name,
			intermediario_cnpj = excluded.intermediario_cnpj,
			intermediario_name = excluded.intermediario_name,
			service_value = excluded.service_value,
			iss_value = excluded.iss_value,
			irrf_value = excluded.irrf_value,
			inss_value = excluded.inss_value,
			pis_value = excluded.pis_value,
			cofins_value = excluded.cofins_value,
			csll_value = excluded.csll_value,
			total_retentions = excluded.total_retentions,
			status = excluded.status,
			layout_version = excluded.layout_version,
			xml_path = excluded.xml_path,
			raw_hash = excluded.raw_hash,
			parse_error = excluded.parse_error,
			parse_warnings = excluded.parse_warnings,
			updated_at = excluded.updated_at,
			nfse_number = excluded.nfse_number,
			service_description = excluded.service_description
	`

	now := time.Now().UTC().Format(time.RFC3339)
	var issueDate string
	if !doc.IssueDate.IsZero() {
		issueDate = doc.IssueDate.UTC().Format(time.RFC3339)
	}
	
	var warningsJSON []byte
	if len(doc.ParseWarnings) > 0 {
		warningsJSON, _ = json.Marshal(doc.ParseWarnings)
	}

	_, err := s.db.ExecContext(ctx, query,
		doc.ID, doc.ChaveAcesso, issueDate, doc.Competence,
		doc.PrestadorCNPJ, doc.PrestadorName, doc.TomadorCNPJ, doc.TomadorName,
		doc.IntermediarioCNPJ, doc.IntermediarioName,
		doc.ServiceValue, doc.ISSValue, doc.IRRFValue, doc.INSSValue, doc.PISValue, doc.COFINSValue, doc.CSLLValue, doc.TotalRetentions,
		doc.Status, nullableString(doc.LayoutVersion), doc.XMLPath, doc.RawHash, nullableString(doc.ParseError), nullableString(string(warningsJSON)), now, now,
		nullableString(doc.NFSeNumber), nullableString(doc.ServiceDescription),
	)
	if err != nil {
		return err
	}

	return s.recomputeDocumentStatus(ctx, doc.ChaveAcesso)
}

// GetDocumentByChave retrieves a canonical document by its access key.
func (s *SQLiteStore) GetDocumentByChave(ctx context.Context, chave string) (*nfse.Document, error) {
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
	var parseError, layoutVersion, parseWarnings, numero, descricao sql.NullString

	err := s.db.QueryRowContext(ctx, query, chave).Scan(
		&doc.ID, &doc.ChaveAcesso, &issueDate, &doc.Competence,
		&doc.PrestadorCNPJ, &doc.PrestadorName, &doc.TomadorCNPJ, &doc.TomadorName,
		&doc.IntermediarioCNPJ, &doc.IntermediarioName,
		&doc.ServiceValue, &doc.ISSValue, &doc.IRRFValue, &doc.INSSValue, &doc.PISValue, &doc.COFINSValue, &doc.CSLLValue, &doc.TotalRetentions,
		&doc.Status, &layoutVersion, &doc.XMLPath, &doc.RawHash, &parseError, &parseWarnings, &createdAt, &updatedAt,
		&numero, &descricao,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query document by chave: %w", err)
	}

	if issueDate != "" {
		doc.IssueDate, _ = time.Parse(time.RFC3339, issueDate)
	}
	doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	doc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if parseError.Valid {
		doc.ParseError = parseError.String
	}
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
func (s *SQLiteStore) UpsertCompanyDocument(ctx context.Context, doc *nfse.CompanyDocument) error {
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

// ListDocuments retrieves company-facing documents based on the provided filters.
func (s *SQLiteStore) ListDocuments(ctx context.Context, companyID string, filter DocumentFilter) ([]nfse.CompanyDocument, error) {
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
	args := []interface{}{companyID}

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
		var parseError, layoutVersion, parseWarnings, numero, descricao sql.NullString
		var firstSeenNSU, lastSeenNSU sql.NullInt64
		var firstSyncedAt, lastSyncedAt string

		if err := rows.Scan(
			&d.Document.ID, &d.ChaveAcesso, &issueDate, &d.Competence,
			&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
			&d.IntermediarioCNPJ, &d.IntermediarioName,
			&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue, &d.TotalRetentions,
			&d.Status, &layoutVersion, &d.XMLPath, &d.RawHash, &parseError, &parseWarnings, &createdAt, &updatedAt,
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
		if parseError.Valid {
			d.ParseError = parseError.String
		}
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

// GetCompanyStats returns aggregated statistics for a company.
func (s *SQLiteStore) GetCompanyStats(ctx context.Context, companyID string) (*CompanyStats, error) {
	query := `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN company_role = 'tomada' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN company_role = 'prestada' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN d.status = 'cancelada' THEN 1 ELSE 0 END), 0)
		FROM company_documents cd
		INNER JOIN documents d ON d.id = cd.document_id
		WHERE cd.company_id = ?
	`

	stats := &CompanyStats{}
	if err := s.db.QueryRowContext(ctx, query, companyID).Scan(
		&stats.TotalDocuments,
		&stats.TotalTomadas,
		&stats.TotalPrestadas,
		&stats.TotalCanceled,
	); err != nil {
		return nil, err
	}

	return stats, nil
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
