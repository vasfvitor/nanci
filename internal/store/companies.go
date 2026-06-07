package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// CreateCompany inserts a new company into the database.
func (s *SQLiteStore) CreateCompany(ctx context.Context, c *nfse.Company) error {
	query := `
		INSERT INTO companies (id, cnpj, cnpj_root, name, credential_id, last_nsu, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.ExecContext(ctx, query,
		c.ID, c.CNPJ, c.CNPJRoot, c.Name, c.CredentialID, c.LastNSU, now, now,
	)
	if err != nil {
		return err
	}

	c.CreatedAt, _ = time.Parse(time.RFC3339, now)
	c.UpdatedAt = c.CreatedAt
	return nil
}

// GetCompany retrieves a company by its CNPJ.
func (s *SQLiteStore) GetCompany(ctx context.Context, cnpj string) (*nfse.Company, error) {
	query := `
		SELECT c.id, c.cnpj, c.cnpj_root, c.name, c.credential_id, cred.label, cred.cert_path, cred.environment,
		       c.last_nsu, c.created_at, c.updated_at
		FROM companies c
		JOIN credentials cred ON cred.id = c.credential_id
		WHERE c.cnpj = ?
	`
	row := s.db.QueryRowContext(ctx, query, cnpj)

	var c nfse.Company
	var createdAt, updatedAt string

	err := row.Scan(
		&c.ID, &c.CNPJ, &c.CNPJRoot, &c.Name, &c.CredentialID, &c.CredentialLabel, &c.CredentialCertPath, &c.Environment, &c.LastNSU, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &c, nil
}

// ListCompanies returns all registered companies.
func (s *SQLiteStore) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	query := `
		SELECT c.id, c.cnpj, c.cnpj_root, c.name, c.credential_id, cred.label, cred.cert_path, cred.environment,
		       c.last_nsu, c.created_at, c.updated_at
		FROM companies c
		JOIN credentials cred ON cred.id = c.credential_id
		ORDER BY c.name ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var companies []nfse.Company
	for rows.Next() {
		var c nfse.Company
		var createdAt, updatedAt string

		if err := rows.Scan(
			&c.ID, &c.CNPJ, &c.CNPJRoot, &c.Name, &c.CredentialID, &c.CredentialLabel, &c.CredentialCertPath, &c.Environment, &c.LastNSU, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}

		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		companies = append(companies, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return companies, nil
}

// AssignCredentialToCompany updates the active credential assignment for a company.
func (s *SQLiteStore) AssignCredentialToCompany(ctx context.Context, companyID, credentialID string) error {
	query := `
		UPDATE companies
		SET credential_id = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := s.db.ExecContext(ctx, query, credentialID, now, companyID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("empresa não encontrada para atualização")
	}

	return nil
}

// UpdateLastNSU updates the last processed NSU for a company.
func (s *SQLiteStore) UpdateLastNSU(ctx context.Context, companyID string, nsu int64) error {
	query := `
		UPDATE companies
		SET last_nsu = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := s.db.ExecContext(ctx, query, nsu, now, companyID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("empresa não encontrada para atualização")
	}

	return nil
}
