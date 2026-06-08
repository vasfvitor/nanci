package seed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func SeedDevelopment(ctx context.Context, db *sql.DB) error {
	company := nfse.Company{
		ID:           "dev-company-70860312000150",
		CNPJ:         "70860312000150",
		CNPJRoot:     "70860312",
		Name:         "Empresa Mock Teste",
		CredentialID: "dev-credential-70860312000150",
		Environment:  nfse.EnvironmentRestricted,
		LastNSU:      0,
	}

	credential := nfse.Credential{
		ID:                "dev-credential-70860312000150",
		Label:             "Certificado Mock 70860312000150",
		CertPath:          "devdata/certs/cert_a1_mock_70860312000150.pfx",
		Environment:       nfse.EnvironmentRestricted,
		OwnerCNPJ:         "70860312000150",
		OwnerCNPJRoot:     "70860312",
		FingerprintSHA256: "mock-fingerprint",
	}

	if err := UpsertCredential(ctx, db, credential); err != nil {
		return fmt.Errorf("upsert credential: %w", err)
	}

	if err := UpsertCompany(ctx, db, company); err != nil {
		return fmt.Errorf("upsert company: %w", err)
	}

	return nil
}

func UpsertCredential(ctx context.Context, db *sql.DB, c nfse.Credential) error {
	query := `
		INSERT INTO credentials (
			id, label, cert_path, environment, owner_cnpj, owner_cnpj_root,
			fingerprint_sha256, subject_name, not_before, not_after, inspected_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(id) DO UPDATE SET
			label = excluded.label,
			cert_path = excluded.cert_path,
			environment = excluded.environment,
			owner_cnpj = excluded.owner_cnpj,
			owner_cnpj_root = excluded.owner_cnpj_root,
			fingerprint_sha256 = excluded.fingerprint_sha256,
			subject_name = excluded.subject_name,
			not_before = excluded.not_before,
			not_after = excluded.not_after,
			inspected_at = excluded.inspected_at,
			updated_at = excluded.updated_at;
	`
	_, err := db.ExecContext(ctx, query,
		c.ID, c.Label, c.CertPath, string(c.Environment), c.OwnerCNPJ, c.OwnerCNPJRoot,
		c.FingerprintSHA256, c.SubjectName, c.NotBefore, c.NotAfter, c.InspectedAt,
	)
	return err
}

func UpsertCompany(ctx context.Context, db *sql.DB, c nfse.Company) error {
	query := `
		INSERT INTO companies (
			id, cnpj, cnpj_root, name, credential_id, environment,
			last_nsu, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(id) DO UPDATE SET
			cnpj = excluded.cnpj,
			cnpj_root = excluded.cnpj_root,
			name = excluded.name,
			credential_id = excluded.credential_id,
			environment = excluded.environment,
			last_nsu = excluded.last_nsu,
			updated_at = excluded.updated_at;
	`
	_, err := db.ExecContext(ctx, query,
		c.ID, c.CNPJ, c.CNPJRoot, c.Name, c.CredentialID, string(c.Environment), c.LastNSU,
	)
	return err
}

func UpsertDocument(ctx context.Context, db *sql.DB, d nfse.Document) error {
	query := `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
			intermediario_cnpj, intermediario_name,
			service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value, total_retentions,
			status, layout_version, xml_path, raw_hash, parse_warnings,
			nfse_number, service_description, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
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
			parse_warnings = excluded.parse_warnings,
			nfse_number = excluded.nfse_number,
			service_description = excluded.service_description,
			updated_at = excluded.updated_at;
	`
	// parse_warnings omitted for seed mock simplicity
	_, err := db.ExecContext(ctx, query,
		d.ID, d.ChaveAcesso, d.IssueDate.Format("2006-01-02T15:04:05Z07:00"), d.Competence,
		d.PrestadorCNPJ, d.PrestadorName, d.TomadorCNPJ, d.TomadorName,
		d.IntermediarioCNPJ, d.IntermediarioName,
		d.ServiceValue, d.ISSValue, d.IRRFValue, d.INSSValue, d.PISValue, d.COFINSValue, d.CSLLValue, d.TotalRetentions,
		d.Status, d.LayoutVersion, d.XMLPath, d.RawHash, nil,
		d.NFSeNumber, d.ServiceDescription,
	)
	return err
}

func UpsertCompanyDocument(ctx context.Context, db *sql.DB, cd nfse.CompanyDocument) error {
	query := `
		INSERT INTO company_documents (
			relation_id, company_id, document_id, company_role, visibility_reason,
			first_seen_nsu, last_seen_nsu, first_seen_nsu_valid, last_seen_nsu_valid,
			first_synced_at, last_synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(relation_id) DO UPDATE SET
			company_role = excluded.company_role,
			visibility_reason = excluded.visibility_reason,
			last_seen_nsu = excluded.last_seen_nsu,
			last_seen_nsu_valid = excluded.last_seen_nsu_valid,
			last_synced_at = excluded.last_synced_at;
	`
	var firstSeenValid, lastSeenValid int
	if cd.FirstSeenNSUValid {
		firstSeenValid = 1
	}
	if cd.LastSeenNSUValid {
		lastSeenValid = 1
	}

	_, err := db.ExecContext(ctx, query,
		cd.RelationID, cd.CompanyID, cd.DocumentID, string(cd.CompanyRole), string(cd.VisibilityReason),
		cd.FirstSeenNSU, cd.LastSeenNSU, firstSeenValid, lastSeenValid,
	)
	return err
}
