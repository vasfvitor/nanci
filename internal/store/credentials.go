package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

type CredentialRepository struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewCredentialRepository(db *sql.DB) *CredentialRepository {
	return &CredentialRepository{
		db:      db,
		queries: sqlgen.New(db),
	}
}

func (r *CredentialRepository) CreateCredential(ctx context.Context, c *nfse.Credential) error {
	now := time.Now().UTC()
	nowRFC3339 := now.Format(time.RFC3339)

	err := r.queries.CreateCredential(ctx, sqlgen.CreateCredentialParams{
		ID:                string(c.ID),
		Label:             c.Label,
		CertPath:          c.CertPath,
		Environment:       string(c.Environment),
		OwnerCnpj:         c.OwnerCNPJ,
		OwnerCnpjRoot:     c.OwnerCNPJRoot,
		FingerprintSha256: c.FingerprintSHA256,
		SubjectName:       c.SubjectName,
		NotBefore:         nullableTimeString(c.NotBefore),
		NotAfter:          nullableTimeString(c.NotAfter),
		InspectedAt:       nullableTimeString(c.InspectedAt),
		CreatedAt:         nowRFC3339,
		UpdatedAt:         nowRFC3339,
	})
	if err != nil {
		return err
	}

	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

func (r *CredentialRepository) CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error) {
	row, err := r.queries.GetCredential(ctx, string(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	c := &nfse.Credential{
		ID:                nfse.CredentialID(row.ID),
		Label:             row.Label,
		CertPath:          row.CertPath,
		Environment:       nfse.Environment(row.Environment),
		OwnerCNPJ:         row.OwnerCnpj,
		OwnerCNPJRoot:     row.OwnerCnpjRoot,
		FingerprintSHA256: row.FingerprintSha256,
		SubjectName:       row.SubjectName,
		NotBefore:         parseNullableTime(row.NotBefore),
		NotAfter:          parseNullableTime(row.NotAfter),
		InspectedAt:       parseNullableTime(row.InspectedAt),
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)

	return c, nil
}

func (r *CredentialRepository) ListCredentials(ctx context.Context) ([]nfse.Credential, error) {
	rows, err := r.queries.ListCredentials(ctx)
	if err != nil {
		return nil, err
	}

	creds := make([]nfse.Credential, 0, len(rows))
	for _, row := range rows {
		c := nfse.Credential{
			ID:                nfse.CredentialID(row.ID),
			Label:             row.Label,
			CertPath:          row.CertPath,
			Environment:       nfse.Environment(row.Environment),
			OwnerCNPJ:         row.OwnerCnpj,
			OwnerCNPJRoot:     row.OwnerCnpjRoot,
			FingerprintSHA256: row.FingerprintSha256,
			SubjectName:       row.SubjectName,
			NotBefore:         parseNullableTime(row.NotBefore),
			NotAfter:          parseNullableTime(row.NotAfter),
			InspectedAt:       parseNullableTime(row.InspectedAt),
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, row.CreatedAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, row.UpdatedAt)
		creds = append(creds, c)
	}

	return creds, nil
}

func (r *CredentialRepository) DeleteCredential(ctx context.Context, id nfse.CredentialID) error {
	return r.queries.DeleteCredential(ctx, string(id))
}

func (r *CredentialRepository) UpdateCredential(ctx context.Context, c *nfse.Credential) error {
	now := time.Now().UTC()
	err := r.queries.UpdateCredential(ctx, sqlgen.UpdateCredentialParams{
		ID:                string(c.ID),
		Label:             c.Label,
		CertPath:          c.CertPath,
		Environment:       string(c.Environment),
		OwnerCnpj:         c.OwnerCNPJ,
		OwnerCnpjRoot:     c.OwnerCNPJRoot,
		FingerprintSha256: c.FingerprintSHA256,
		SubjectName:       c.SubjectName,
		NotBefore:         nullableTimeString(c.NotBefore),
		NotAfter:          nullableTimeString(c.NotAfter),
		InspectedAt:       nullableTimeString(c.InspectedAt),
		UpdatedAt:         now.Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	c.UpdatedAt = now
	return nil
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

func nullableTimeString(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339), Valid: true}
}
