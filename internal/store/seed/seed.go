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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
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
