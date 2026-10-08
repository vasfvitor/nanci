// Package seed writes the development database through the stores the app
// uses, so a schema change breaks it in the store or at compile time instead
// of in SQL kept only here.
package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// SeedDevelopment creates the mock credential and company, or brings them back
// to the mock values when they already exist, and returns the company.
func SeedDevelopment(ctx context.Context, db *sql.DB) (*company.Company, error) {
	cred := &credential.Credential{
		ID:                "dev-credential-70860312000150",
		Label:             "Certificado Mock 70860312000150",
		CertPath:          "devdata/certs/cert_a1_mock_70860312000150.pfx",
		OwnerCNPJ:         "70860312000150",
		OwnerCNPJRoot:     "70860312",
		FingerprintSHA256: "mock-fingerprint",
	}
	if err := saveCredential(ctx, credential.NewStore(db), cred); err != nil {
		return nil, fmt.Errorf("save credential: %w", err)
	}

	comp := &company.Company{ // #nosec G101 -- mock dev company; CredentialID is an ID, not a secret.
		ID:           "dev-company-70860312000150",
		CNPJ:         "70860312000150",
		CNPJRoot:     "70860312",
		Name:         "Empresa Mock Teste",
		CredentialID: cred.ID,
		Environment:  dfe.EnvironmentRestricted,
		// The fixtures are old; from_now, the store's default, would hide them.
		SyncStartPolicy: syncstate.SyncStartPolicyAll,
	}
	saved, err := saveCompany(ctx, company.NewStore(db), comp)
	if err != nil {
		return nil, fmt.Errorf("save company: %w", err)
	}
	return saved, nil
}

// SeedDocument parses an NFS-e XML and applies it to the company the way a
// sync applies a document received at nsu.
func SeedDocument(ctx context.Context, store *sync.Store, comp *company.Company, xmlPath string, nsu int64) error {
	data, err := os.ReadFile(xmlPath) // #nosec G304 -- dev seeder reads its own fixture files.
	if err != nil {
		return err
	}

	doc, warnings, err := nfse.ParseDocumentXML(data)
	if err != nil {
		return err
	}
	doc.ID = nfse.DocumentID("doc-" + string(doc.ChaveAcesso))
	doc.XMLPath = xmlPath
	doc.RawHash = "hash-" + string(doc.ChaveAcesso)
	doc.ParseWarnings = warnings

	_, err = store.ApplyDocument(ctx, nfse.ApplyDocumentParams{
		Document:      doc,
		Participation: nfse.ClassifyCompanyParticipation(&doc, comp.CNPJ),
		CompanyID:     comp.ID,
		NSU:           nsu,
	})
	return err
}

func saveCredential(ctx context.Context, store *credential.Store, c *credential.Credential) error {
	_, err := store.CredentialByID(ctx, c.ID)
	if errors.Is(err, credential.ErrCredentialNotFound) {
		return store.CreateCredential(ctx, c)
	}
	if err != nil {
		return err
	}
	return store.UpdateCredential(ctx, c)
}

// saveCompany looks the company up by CNPJ, so a company added by hand with the
// mock CNPJ is updated under its own ID.
func saveCompany(ctx context.Context, store *company.Store, c *company.Company) (*company.Company, error) {
	existing, err := store.CompanyByCNPJ(ctx, c.CNPJ)
	if errors.Is(err, company.ErrCompanyNotFound) {
		if err := store.CreateCompany(ctx, c); err != nil {
			return nil, err
		}
		return c, nil
	}
	if err != nil {
		return nil, err
	}

	existing.Name = c.Name
	existing.Environment = c.Environment
	existing.SyncStartPolicy = c.SyncStartPolicy
	existing.SyncStartDate = c.SyncStartDate
	if err := store.UpdateCompany(ctx, existing); err != nil {
		return nil, err
	}
	if err := store.AssignCredential(ctx, existing.ID, c.CredentialID); err != nil {
		return nil, err
	}
	existing.CredentialID = c.CredentialID
	return existing, nil
}
