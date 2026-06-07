package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

type CompanyRepository struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewCompanyRepository(db *sql.DB) *CompanyRepository {
	return &CompanyRepository{
		db:      db,
		queries: sqlgen.New(db),
	}
}

func (r *CompanyRepository) CreateCompany(ctx context.Context, c *nfse.Company) error {
	now := time.Now().UTC()
	err := r.queries.CreateCompany(ctx, sqlgen.CreateCompanyParams{
		ID:                 string(c.ID),
		Cnpj:               c.CNPJ,
		CnpjRoot:           c.CNPJRoot,
		Name:               c.Name,
		CredentialID:       sql.NullString{String: string(c.CredentialID), Valid: c.CredentialID != ""},
		CredentialLabel:    sql.NullString{String: c.CredentialLabel, Valid: c.CredentialLabel != ""},
		CredentialCertPath: sql.NullString{String: c.CredentialCertPath, Valid: c.CredentialCertPath != ""},
		Environment:        string(c.Environment),
		CreatedAt:          now.Format(time.RFC3339),
		UpdatedAt:          now.Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

func (r *CompanyRepository) CompanyByCNPJ(ctx context.Context, cnpjVal string) (*nfse.Company, error) {
	row, err := r.queries.GetCompanyByCNPJ(ctx, cnpjVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	c := &nfse.Company{
		ID:                 nfse.CompanyID(row.ID),
		CNPJ:               row.Cnpj,
		CNPJRoot:           row.CnpjRoot,
		Name:               row.Name,
		CredentialID:       nfse.CredentialID(row.CredentialID.String),
		CredentialLabel:    row.CredentialLabel.String,
		CredentialCertPath: row.CredentialCertPath.String,
		Environment:        nfse.Environment(row.Environment),
		LastNSU:            row.LastNsu,
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)

	return c, nil
}

func (r *CompanyRepository) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	rows, err := r.queries.ListCompanies(ctx)
	if err != nil {
		return nil, err
	}

	companies := make([]nfse.Company, 0, len(rows))
	for _, row := range rows {
		c := nfse.Company{
			ID:                 nfse.CompanyID(row.ID),
			CNPJ:               row.Cnpj,
			CNPJRoot:           row.CnpjRoot,
			Name:               row.Name,
			CredentialID:       nfse.CredentialID(row.CredentialID.String),
			CredentialLabel:    row.CredentialLabel.String,
			CredentialCertPath: row.CredentialCertPath.String,
			Environment:        nfse.Environment(row.Environment),
			LastNSU:            row.LastNsu,
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
		companies = append(companies, c)
	}

	return companies, nil
}

func (r *CompanyRepository) AssignCredential(ctx context.Context, companyID nfse.CompanyID, credID nfse.CredentialID) error {
	now := time.Now().UTC().Format(time.RFC3339)
	affected, err := r.queries.AssignCredentialToCompany(ctx, sqlgen.AssignCredentialToCompanyParams{
		CredentialID: sql.NullString{String: string(credID), Valid: credID != ""},
		UpdatedAt:    now,
		CompanyID:    string(companyID),
	})
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CompanyRepository) UpdateCompany(ctx context.Context, id nfse.CompanyID, name string, environment nfse.Environment) error {
	now := time.Now().UTC().Format(time.RFC3339)
	err := r.queries.UpdateCompany(ctx, sqlgen.UpdateCompanyParams{
		Name:        name,
		Environment: string(environment),
		UpdatedAt:   now,
		ID:          string(id),
	})
	if err != nil {
		return err
	}
	return nil
}
