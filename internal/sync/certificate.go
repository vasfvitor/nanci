package sync

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// LoadedCredential is a company's certificate, loaded and checked against
// the company it consults for.
type LoadedCredential struct {
	Credential *credential.Credential
	TLS        tls.Certificate // PrivateKey is a crypto.Signer (RSA for ICP-Brasil A1)
	Basis      syncstate.ConsultationBasis
}

// CertificateLoader loads the certificate a company consults with. Pull,
// direct query and manifestação share it.
type CertificateLoader struct {
	Log         *slog.Logger
	Credentials credentialProvider
	Passwords   CredentialProvider
}

// LoadForCompany validates the certificate path, asks for the password
// (purpose tells the user why), loads the PKCS#12, persists the certificate
// inspection and checks that the certificate may consult for the company.
// The password is zeroed before returning.
func (l *CertificateLoader) LoadForCompany(ctx context.Context, comp *company.Company, purpose string) (LoadedCredential, error) {
	cred, err := l.Credentials.CredentialByID(ctx, comp.CredentialID)
	if err != nil {
		return LoadedCredential{}, fmt.Errorf("resolver credencial da empresa %s: %w", comp.Name, err)
	}
	if err := credential.ValidateCertificatePath(cred.CertPath); err != nil {
		return LoadedCredential{}, err
	}

	pass, err := l.Passwords.GetCertPassword(ctx, CertPasswordRequest{
		RequestID:       dfe.GenerateID(),
		CompanyID:       string(comp.ID),
		CompanyName:     comp.Name,
		TargetCNPJ:      comp.CNPJ,
		CredentialID:    string(cred.ID),
		CredentialLabel: cred.Label,
		CertPath:        cred.CertPath,
		Purpose:         purpose,
	})
	if err != nil {
		return LoadedCredential{}, fmt.Errorf("obter senha do certificado: %w", err)
	}
	defer cert.ZeroBytes(pass)

	l.Log.DebugContext(ctx, "Carregando certificado TLS", slog.String("cert_path", cred.CertPath))
	loaded, err := loadPKCS12(cred.CertPath, pass)
	if err != nil {
		return LoadedCredential{}, fmt.Errorf("carregar certificado: %w", err)
	}

	inspection := loaded.Inspection
	cred.OwnerCNPJ = inspection.OwnerCNPJ
	cred.OwnerCNPJRoot = inspection.OwnerCNPJRoot
	cred.FingerprintSHA256 = inspection.FingerprintSHA256
	cred.SubjectName = inspection.SubjectName
	cred.NotBefore = &inspection.NotBefore
	cred.NotAfter = &inspection.NotAfter
	now := time.Now().UTC()
	cred.InspectedAt = &now
	if err := l.Credentials.UpdateCredential(ctx, cred); err != nil {
		return LoadedCredential{}, fmt.Errorf("persistir inspeção da credencial: %w", err)
	}

	basis, err := validateConsultationCompatibility(comp, cred)
	if err != nil {
		return LoadedCredential{}, err
	}

	return LoadedCredential{
		Credential: cred,
		TLS:        loaded.TLS,
		Basis:      basis,
	}, nil
}

func validateConsultationCompatibility(comp *company.Company, cred *credential.Credential) (syncstate.ConsultationBasis, error) {
	if cred.OwnerCNPJ == "" || cred.OwnerCNPJRoot == "" {
		return "", company.ErrCredentialNoOwner
	}
	if comp.Environment == "" {
		return "", company.ErrCompanyNoEnvironment
	}
	if comp.CNPJRoot != cred.OwnerCNPJRoot {
		return "", fmt.Errorf("%w: credencial (raiz %s) vs empresa (%s)", company.ErrCredentialMismatch, cred.OwnerCNPJRoot, cnpj.Format(comp.CNPJ))
	}
	if comp.CNPJ == cred.OwnerCNPJ {
		return syncstate.ConsultationBasisExactCertificateCNPJ, nil
	}
	return syncstate.ConsultationBasisSameRootCertificate, nil
}
