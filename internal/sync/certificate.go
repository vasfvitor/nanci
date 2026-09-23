package sync

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"time"

	companypkg "github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// LoadedCredential is a company's certificate, loaded and checked against
// the company it consults for.
type LoadedCredential struct {
	Credential *nfse.Credential
	TLS        tls.Certificate // PrivateKey is a crypto.Signer (RSA for ICP-Brasil A1)
	Basis      nfse.ConsultationBasis
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
func (l *CertificateLoader) LoadForCompany(ctx context.Context, company *nfse.Company, purpose string) (LoadedCredential, error) {
	credential, err := l.Credentials.CredentialByID(ctx, company.CredentialID)
	if err != nil {
		return LoadedCredential{}, fmt.Errorf("resolver credencial da empresa %s: %w", company.Name, err)
	}
	if err := validateCertificatePath(credential.CertPath); err != nil {
		return LoadedCredential{}, err
	}

	pass, err := l.Passwords.GetCertPassword(ctx, CertPasswordRequest{
		RequestID:       nfse.GenerateID(),
		CompanyID:       string(company.ID),
		CompanyName:     company.Name,
		TargetCNPJ:      company.CNPJ,
		CredentialID:    string(credential.ID),
		CredentialLabel: credential.Label,
		CertPath:        credential.CertPath,
		Purpose:         purpose,
	})
	if err != nil {
		return LoadedCredential{}, fmt.Errorf("obter senha do certificado: %w", err)
	}
	defer cert.ZeroBytes(pass)

	l.Log.DebugContext(ctx, "Carregando certificado TLS", slog.String("cert_path", credential.CertPath))
	loaded, err := loadPKCS12(credential.CertPath, pass)
	if err != nil {
		return LoadedCredential{}, fmt.Errorf("carregar certificado: %w", err)
	}

	inspection := loaded.Inspection
	credential.OwnerCNPJ = inspection.OwnerCNPJ
	credential.OwnerCNPJRoot = inspection.OwnerCNPJRoot
	credential.FingerprintSHA256 = inspection.FingerprintSHA256
	credential.SubjectName = inspection.SubjectName
	credential.NotBefore = &inspection.NotBefore
	credential.NotAfter = &inspection.NotAfter
	now := time.Now().UTC()
	credential.InspectedAt = &now
	if err := l.Credentials.UpdateCredential(ctx, credential); err != nil {
		return LoadedCredential{}, fmt.Errorf("persistir inspeção da credencial: %w", err)
	}

	basis, err := validateConsultationCompatibility(company, credential)
	if err != nil {
		return LoadedCredential{}, err
	}

	return LoadedCredential{
		Credential: credential,
		TLS:        loaded.TLS,
		Basis:      basis,
	}, nil
}

func validateCertificatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo de certificado não encontrado: %s", path)
		}
		return fmt.Errorf("verificar certificado: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("caminho do certificado aponta para um diretório: %s", path)
	}
	return nil
}

func validateConsultationCompatibility(company *nfse.Company, credential *nfse.Credential) (nfse.ConsultationBasis, error) {
	if credential.OwnerCNPJ == "" || credential.OwnerCNPJRoot == "" {
		return "", companypkg.ErrCredentialNoOwner
	}
	if company.Environment == "" {
		return "", companypkg.ErrCompanyNoEnvironment
	}
	if company.CNPJRoot != credential.OwnerCNPJRoot {
		return "", fmt.Errorf("%w: credencial (raiz %s) vs empresa (%s)", companypkg.ErrCredentialMismatch, credential.OwnerCNPJRoot, cnpj.Format(company.CNPJ))
	}
	if company.CNPJ == credential.OwnerCNPJ {
		return nfse.ConsultationBasisExactCertificateCNPJ, nil
	}
	return nfse.ConsultationBasisSameRootCertificate, nil
}
