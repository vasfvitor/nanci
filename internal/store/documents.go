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

// CompanyDocumentByChave retrieves one company-visible document by access key.
func (s *DocumentRepository) CompanyDocumentByChave(ctx context.Context, companyID nfse.CompanyID, chave string) (*nfse.CompanyDocument, error) {
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
			cd.first_synced_at, cd.last_synced_at, cd.viewed_at
		FROM company_documents cd
		INNER JOIN documents d ON d.id = cd.document_id
		WHERE cd.company_id = ? AND d.chave_acesso = ?
		LIMIT 1
	`

	var d nfse.CompanyDocument
	var issueDate, createdAt, updatedAt string
	var parseWarnings sql.NullString
	var firstSeen, lastSeen int64
	var firstSeenValid, lastSeenValid int64
	var firstSyncedAt, lastSyncedAt string
	var viewedAt sql.NullString

	err := s.db.QueryRowContext(ctx, query, string(companyID), chave).Scan(
		&d.ID, &d.ChaveAcesso, &issueDate, &d.Competence,
		&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
		&d.IntermediarioCNPJ, &d.IntermediarioName,
		&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue, &d.TotalRetentions,
		&d.Status, &d.LayoutVersion, &d.XMLPath, &d.RawHash, &parseWarnings, &createdAt, &updatedAt,
		&d.NFSeNumber, &d.ServiceDescription,
		&d.RelationID, &d.CompanyID, &d.DocumentID, &d.CompanyRole, &d.VisibilityReason,
		&firstSeen, &lastSeen, &firstSeenValid, &lastSeenValid,
		&firstSyncedAt, &lastSyncedAt, &viewedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query company document by chave: %w", err)
	}

	if err := hydrateCompanyDocument(&d, issueDate, createdAt, updatedAt, parseWarnings, firstSeen, lastSeen, firstSeenValid, lastSeenValid, firstSyncedAt, lastSyncedAt, viewedAt); err != nil {
		return nil, err
	}
	return &d, nil
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
			cd.first_synced_at, cd.last_synced_at, cd.viewed_at
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
	if filter.OnlyUnread {
		query += " AND cd.viewed_at IS NULL"
	}
	if filter.IssueDateGTE != nil {
		query += " AND d.issue_date >= ?"
		args = append(args, filter.IssueDateGTE.Format("2006-01-02"))
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
	defer func() { _ = rows.Close() }()

	var docs []nfse.CompanyDocument
	for rows.Next() {
		var d nfse.CompanyDocument
		var issueDate, createdAt, updatedAt string
		var parseWarnings sql.NullString
		var firstSeen, lastSeen int64
		var firstSeenValid, lastSeenValid int64
		var firstSyncedAt, lastSyncedAt string
		var viewedAt sql.NullString

		if err := rows.Scan(
			&d.ID, &d.ChaveAcesso, &issueDate, &d.Competence,
			&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
			&d.IntermediarioCNPJ, &d.IntermediarioName,
			&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue, &d.TotalRetentions,
			&d.Status, &d.LayoutVersion, &d.XMLPath, &d.RawHash, &parseWarnings, &createdAt, &updatedAt,
			&d.NFSeNumber, &d.ServiceDescription,
			&d.RelationID, &d.CompanyID, &d.DocumentID, &d.CompanyRole, &d.VisibilityReason,
			&firstSeen, &lastSeen, &firstSeenValid, &lastSeenValid,
			&firstSyncedAt, &lastSyncedAt, &viewedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		if err := hydrateCompanyDocument(&d, issueDate, createdAt, updatedAt, parseWarnings, firstSeen, lastSeen, firstSeenValid, lastSeenValid, firstSyncedAt, lastSyncedAt, viewedAt); err != nil {
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
	defer func() { _ = rows.Close() }()

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

func hydrateCompanyDocument(d *nfse.CompanyDocument, issueDate, createdAt, updatedAt string, parseWarnings sql.NullString, firstSeen, lastSeen, firstSeenValid, lastSeenValid int64, firstSyncedAt, lastSyncedAt string, viewedAt sql.NullString) error {
	var err error
	d.IssueDate, err = parseRequiredTime("document issue_date", issueDate)
	if err != nil {
		return err
	}
	d.CreatedAt, err = parseRequiredTime("document created_at", createdAt)
	if err != nil {
		return err
	}
	d.UpdatedAt, err = parseRequiredTime("document updated_at", updatedAt)
	if err != nil {
		return err
	}
	if err := decodeWarnings(parseWarnings, &d.ParseWarnings); err != nil {
		return fmt.Errorf("document parse_warnings: %w", err)
	}
	d.FirstSeenNSU = ptrFromValidInt64(firstSeen, firstSeenValid)
	d.LastSeenNSU = ptrFromValidInt64(lastSeen, lastSeenValid)
	d.FirstSyncedAt, err = parseRequiredTime("company document first_synced_at", firstSyncedAt)
	if err != nil {
		return err
	}
	d.LastSyncedAt, err = parseRequiredTime("company document last_synced_at", lastSyncedAt)
	if err != nil {
		return err
	}
	if viewedAt.Valid && viewedAt.String != "" {
		parsedViewedAt, err := parseRequiredTime("company document viewed_at", viewedAt.String)
		if err != nil {
			return err
		}
		d.ViewedAt = &parsedViewedAt
	}
	return nil
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

// ListPendingExportDocuments retrieves documents that have not been exported for the given kind, or where the hash changed.
func (s *DocumentRepository) ListPendingExportDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter, kind string) ([]nfse.CompanyDocument, error) {
	// Re-use ListCompanyDocuments logic but add a JOIN/WHERE for pending export
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
			cd.first_synced_at, cd.last_synced_at, cd.viewed_at
		FROM company_documents cd
		INNER JOIN documents d ON d.id = cd.document_id
		LEFT JOIN company_document_export_marks m ON m.company_id = cd.company_id AND m.document_id = cd.document_id AND m.export_kind = ?
		WHERE cd.company_id = ? AND (m.exported_at IS NULL OR m.exported_hash != d.raw_hash)
	`
	args := []interface{}{kind, string(companyID)}

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
	if filter.OnlyUnread {
		query += " AND cd.viewed_at IS NULL"
	}
	if filter.IssueDateGTE != nil {
		query += " AND d.issue_date >= ?"
		args = append(args, filter.IssueDateGTE.Format("2006-01-02"))
	}

	query += " ORDER BY d.issue_date DESC, d.chave_acesso DESC"
	if filter.Limit != nil && *filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, *filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending export documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var docs []nfse.CompanyDocument
	for rows.Next() {
		var d nfse.CompanyDocument
		var issueDate, createdAt, updatedAt string
		var parseWarnings sql.NullString
		var firstSeen, lastSeen int64
		var firstSeenValid, lastSeenValid int64
		var firstSyncedAt, lastSyncedAt string
		var viewedAt sql.NullString

		if err := rows.Scan(
			&d.ID, &d.ChaveAcesso, &issueDate, &d.Competence,
			&d.PrestadorCNPJ, &d.PrestadorName, &d.TomadorCNPJ, &d.TomadorName,
			&d.IntermediarioCNPJ, &d.IntermediarioName,
			&d.ServiceValue, &d.ISSValue, &d.IRRFValue, &d.INSSValue, &d.PISValue, &d.COFINSValue, &d.CSLLValue, &d.TotalRetentions,
			&d.Status, &d.LayoutVersion, &d.XMLPath, &d.RawHash, &parseWarnings, &createdAt, &updatedAt,
			&d.NFSeNumber, &d.ServiceDescription,
			&d.RelationID, &d.CompanyID, &d.DocumentID, &d.CompanyRole, &d.VisibilityReason,
			&firstSeen, &lastSeen, &firstSeenValid, &lastSeenValid,
			&firstSyncedAt, &lastSyncedAt, &viewedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		if err := hydrateCompanyDocument(&d, issueDate, createdAt, updatedAt, parseWarnings, firstSeen, lastSeen, firstSeenValid, lastSeenValid, firstSyncedAt, lastSyncedAt, viewedAt); err != nil {
			return nil, err
		}

		docs = append(docs, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return docs, nil
}

// CountPendingExportDocuments counts pending export documents.
func (s *DocumentRepository) CountPendingExportDocuments(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter, kind string) (int, error) {
	query := `
		SELECT COUNT(1)
		FROM company_documents cd
		INNER JOIN documents d ON d.id = cd.document_id
		LEFT JOIN company_document_export_marks m ON m.company_id = cd.company_id AND m.document_id = cd.document_id AND m.export_kind = ?
		WHERE cd.company_id = ? AND (m.exported_at IS NULL OR m.exported_hash != d.raw_hash)
	`
	args := []interface{}{kind, string(companyID)}

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
	if filter.OnlyUnread {
		query += " AND cd.viewed_at IS NULL"
	}
	if filter.IssueDateGTE != nil {
		query += " AND d.issue_date >= ?"
		args = append(args, filter.IssueDateGTE.Format("2006-01-02"))
	}

	var count int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// MarkDocumentsExported marks documents as exported for a specific kind.
func (s *DocumentRepository) MarkDocumentsExported(ctx context.Context, companyID nfse.CompanyID, kind string, marks []nfse.DocumentExportMark) error {
	if len(marks) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO company_document_export_marks (company_id, document_id, export_kind, exported_hash, exported_at)
		VALUES (?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(company_id, document_id, export_kind) DO UPDATE SET
			exported_hash=excluded.exported_hash,
			exported_at=excluded.exported_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range marks {
		if _, err := stmt.ExecContext(ctx, string(companyID), m.DocumentID, kind, m.Hash); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// MarkDocumentsViewed marks documents matching the filter as viewed.
func (s *DocumentRepository) MarkDocumentsViewed(ctx context.Context, companyID nfse.CompanyID, filter nfse.DocumentFilter) (int, error) {
	// First we need to find the relation_ids that match, then update them.
	// Or we can do an UPDATE with a subquery.
	query := `
		UPDATE company_documents
		SET viewed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE company_id = ? AND viewed_at IS NULL AND relation_id IN (
			SELECT cd.relation_id
			FROM company_documents cd
			INNER JOIN documents d ON d.id = cd.document_id
			WHERE cd.company_id = ?
	`
	args := []interface{}{string(companyID), string(companyID)}

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
	if filter.IssueDateGTE != nil {
		query += " AND d.issue_date >= ?"
		args = append(args, filter.IssueDateGTE.Format("2006-01-02"))
	}
	query += ")"

	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}
