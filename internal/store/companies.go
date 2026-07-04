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
	syncStartPolicy := c.SyncStartPolicy
	if syncStartPolicy == "" {
		syncStartPolicy = nfse.SyncStartPolicyFromNow
		c.SyncStartPolicy = syncStartPolicy
	}
	if syncStartPolicy == nfse.SyncStartPolicyFromNow && c.SyncStartDate == nil {
		today, _ := time.Parse(dateOnlyLayout, time.Now().Format(dateOnlyLayout))
		c.SyncStartDate = &today
	}
	err := r.queries.CreateCompany(ctx, sqlgen.CreateCompanyParams{
		ID:                     string(c.ID),
		Cnpj:                   c.CNPJ,
		CnpjRoot:               c.CNPJRoot,
		Name:                   c.Name,
		CredentialID:           sql.NullString{String: string(c.CredentialID), Valid: c.CredentialID != ""},
		CredentialLabel:        sql.NullString{String: c.CredentialLabel, Valid: c.CredentialLabel != ""},
		CredentialCertPath:     sql.NullString{String: c.CredentialCertPath, Valid: c.CredentialCertPath != ""},
		Environment:            string(c.Environment),
		SyncStartPolicy:        string(syncStartPolicy),
		SyncStartDate:          nullableTime(c.SyncStartDate, dateOnlyLayout),
		InitialSyncCompletedAt: nullableTime(c.InitialSyncDoneAt, time.RFC3339),
		CreatedAt:              now.Format(time.RFC3339),
		UpdatedAt:              now.Format(time.RFC3339),
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

	c := companyFromRow(row)
	return c, nil
}

func (r *CompanyRepository) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	rows, err := r.queries.ListCompanies(ctx)
	if err != nil {
		return nil, err
	}

	companies := make([]nfse.Company, 0, len(rows))
	for _, row := range rows {
		companies = append(companies, *companyFromRow(row))
	}

	return companies, nil
}

func companyFromRow(row sqlgen.Company) *nfse.Company {
	c := &nfse.Company{
		ID:                 nfse.CompanyID(row.ID),
		CNPJ:               row.Cnpj,
		CNPJRoot:           row.CnpjRoot,
		Name:               row.Name,
		CredentialID:       nfse.CredentialID(row.CredentialID.String),
		CredentialLabel:    row.CredentialLabel.String,
		CredentialCertPath: row.CredentialCertPath.String,
		Environment:        nfse.Environment(row.Environment),
		SyncStartPolicy:    nfse.SyncStartPolicy(row.SyncStartPolicy),
		SyncStartDate:      parseNullableDate(row.SyncStartDate),
		InitialSyncDoneAt:  ParseNullableTime(row.InitialSyncCompletedAt),
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)

	return c
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

func (r *CompanyRepository) UpdateCompany(ctx context.Context, c *nfse.Company) error {
	now := time.Now().UTC().Format(time.RFC3339)
	err := r.queries.UpdateCompany(ctx, sqlgen.UpdateCompanyParams{
		Name:            c.Name,
		Environment:     string(c.Environment),
		SyncStartPolicy: string(c.SyncStartPolicy),
		SyncStartDate:   nullableTime(c.SyncStartDate, dateOnlyLayout),
		UpdatedAt:       now,
		ID:              string(c.ID),
	})
	if err != nil {
		return err
	}
	return nil
}

const dateOnlyLayout = "2006-01-02"

func nullableTime(t *time.Time, layout string) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(layout), Valid: true}
}

func parseNullableDate(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed, err := time.Parse(dateOnlyLayout, value.String)
	if err != nil {
		return nil
	}
	return &parsed
}
