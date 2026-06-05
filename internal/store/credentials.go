package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// CreateCredential inserts a reusable credential into the database.
func (s *SQLiteStore) CreateCredential(ctx context.Context, c *nfse.Credential) error {
	query := `
		INSERT INTO credentials (
			id, label, cert_path, environment, owner_cnpj, owner_cnpj_root,
			fingerprint_sha256, subject_name, not_before, not_after, inspected_at,
			created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC()
	nowRFC3339 := now.Format(time.RFC3339)

	_, err := s.db.ExecContext(ctx, query,
		c.ID, c.Label, c.CertPath, c.Environment, nullableText(c.OwnerCNPJ), nullableText(c.OwnerCNPJRoot),
		nullableText(c.FingerprintSHA256), nullableText(c.SubjectName), nullableTimeString(c.NotBefore),
		nullableTimeString(c.NotAfter), nullableTimeString(c.InspectedAt), nowRFC3339, nowRFC3339,
	)
	if err != nil {
		return err
	}

	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

// GetCredential retrieves a reusable credential by ID.
func (s *SQLiteStore) GetCredential(ctx context.Context, id string) (*nfse.Credential, error) {
	query := `
		SELECT id, label, cert_path, environment, owner_cnpj, owner_cnpj_root,
		       fingerprint_sha256, subject_name, not_before, not_after, inspected_at,
		       created_at, updated_at
		FROM credentials
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	credential, err := scanCredential(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return credential, nil
}

// ListCredentials returns all reusable credentials.
func (s *SQLiteStore) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	query := `
		SELECT id, label, cert_path, environment, owner_cnpj, owner_cnpj_root,
		       fingerprint_sha256, subject_name, not_before, not_after, inspected_at,
		       created_at, updated_at
		FROM credentials
		ORDER BY label ASC, created_at ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []nfse.Credential
	for rows.Next() {
		credential, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, *credential)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return credentials, nil
}

// UpdateCredentialPath updates the path to the PKCS#12 file for a credential.
func (s *SQLiteStore) UpdateCredentialPath(ctx context.Context, id, certPath string) error {
	query := `
		UPDATE credentials
		SET cert_path = ?, updated_at = ?, inspected_at = NULL
		WHERE id = ?
	`
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, query, certPath, now, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("credencial não encontrada para atualização")
	}
	return nil
}

// UpdateCredentialInspection stores metadata derived from the leaf certificate.
func (s *SQLiteStore) UpdateCredentialInspection(ctx context.Context, c *nfse.Credential) error {
	query := `
		UPDATE credentials
		SET owner_cnpj = ?, owner_cnpj_root = ?, fingerprint_sha256 = ?, subject_name = ?,
		    not_before = ?, not_after = ?, inspected_at = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, query,
		nullableText(c.OwnerCNPJ), nullableText(c.OwnerCNPJRoot),
		nullableText(c.FingerprintSHA256), nullableText(c.SubjectName),
		nullableTimeString(c.NotBefore), nullableTimeString(c.NotAfter),
		nullableTimeString(c.InspectedAt), now, c.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("credencial não encontrada para inspeção")
	}
	return nil
}

type credentialScanner interface {
	Scan(dest ...any) error
}

func scanCredential(scanner credentialScanner) (*nfse.Credential, error) {
	var c nfse.Credential
	var ownerCNPJ, ownerCNPJRoot, fingerprint, subjectName sql.NullString
	var notBefore, notAfter, inspectedAt sql.NullString
	var createdAt, updatedAt string

	err := scanner.Scan(
		&c.ID, &c.Label, &c.CertPath, &c.Environment, &ownerCNPJ, &ownerCNPJRoot,
		&fingerprint, &subjectName, &notBefore, &notAfter, &inspectedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	c.OwnerCNPJ = ownerCNPJ.String
	c.OwnerCNPJRoot = ownerCNPJRoot.String
	c.FingerprintSHA256 = fingerprint.String
	c.SubjectName = subjectName.String
	c.NotBefore = parseNullableTime(notBefore)
	c.NotAfter = parseNullableTime(notAfter)
	c.InspectedAt = parseNullableTime(inspectedAt)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &c, nil
}

func parseNullableTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, value.String)
	if err != nil {
		return nil
	}
	return &t
}

func nullableTimeString(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}
