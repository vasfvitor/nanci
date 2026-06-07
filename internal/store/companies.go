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
		ID:          string(c.ID),
		Cnpj:        c.CNPJ,
		CnpjRoot:    c.CNPJRoot,
		Name:        c.Name,
		Environment: string(c.Environment),
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
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
	err := r.queries.AssignCredentialToCompany(ctx, sqlgen.AssignCredentialToCompanyParams{
		CredentialID:       sql.NullString{String: string(credID), Valid: true},
		CredentialLabel:    sql.NullString{String: "Label pending", Valid: true}, // Would usually fetch label here, simplified for now
		CredentialCertPath: sql.NullString{String: "", Valid: true},
		UpdatedAt:          now,
		ID:                 string(companyID),
	})
	if err != nil {
		return err
	}
	return nil
}
