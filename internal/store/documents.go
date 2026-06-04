package store

import (
	"context"
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
