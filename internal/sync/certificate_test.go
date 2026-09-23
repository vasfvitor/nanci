package sync

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	companypkg "github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// recordingProvider keeps the request and the password slice it handed out.
type recordingProvider struct {
	request  CertPasswordRequest
	password []byte
}

func (p *recordingProvider) GetCertPassword(_ context.Context, req CertPasswordRequest) ([]byte, error) {
	p.request = req
	p.password = []byte("secret")
	return p.password, nil
}

func TestLoadForCompanyPersistsInspectionAndZeroesPassword(t *testing.T) {
	passwords := &recordingProvider{}
	mgr, comp := newPullTestManager(t, passwords)
	loader := CertificateLoader{Log: mgr.Log, Credentials: mgr.CredentialProvider, Passwords: passwords}

	loaded, err := loader.LoadForCompany(context.Background(), comp, "Consulta direta")
	if err != nil {
		t.Fatalf("LoadForCompany: %v", err)
	}

	if passwords.request.Purpose != "Consulta direta" {
		t.Errorf("Purpose = %q, want %q", passwords.request.Purpose, "Consulta direta")
	}
	if passwords.request.TargetCNPJ != comp.CNPJ || passwords.request.CredentialID != string(comp.CredentialID) {
		t.Errorf("password request = %+v", passwords.request)
	}
	for _, b := range passwords.password {
		if b != 0 {
			t.Fatalf("password not zeroed: %q", passwords.password)
		}
	}
	if loaded.Basis != nfse.ConsultationBasisExactCertificateCNPJ {
		t.Errorf("Basis = %q, want exact certificate CNPJ", loaded.Basis)
	}

	stored, err := mgr.CredentialProvider.CredentialByID(context.Background(), comp.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.OwnerCNPJ != "11222333000181" || stored.InspectedAt == nil {
		t.Errorf("stored credential = owner %q inspected %v, want the inspection persisted", stored.OwnerCNPJ, stored.InspectedAt)
	}
}

func TestLoadForCompanyRejectsCertificateOfAnotherRoot(t *testing.T) {
	passwords := &recordingProvider{}
	mgr, comp := newPullTestManager(t, passwords)
	loadPKCS12 = func(string, []byte) (cert.LoadedCertificate, error) {
		now := time.Now().UTC()
		return cert.LoadedCertificate{Inspection: cert.Inspection{
			OwnerCNPJ:     "99888777000100",
			OwnerCNPJRoot: "99888777",
			NotBefore:     now,
			NotAfter:      now.Add(24 * time.Hour),
		}}, nil
	}
	loader := CertificateLoader{Log: mgr.Log, Credentials: mgr.CredentialProvider, Passwords: passwords}

	_, err := loader.LoadForCompany(context.Background(), comp, "Consulta direta")
	if !errors.Is(err, companypkg.ErrCredentialMismatch) {
		t.Fatalf("LoadForCompany error = %v, want ErrCredentialMismatch", err)
	}
}

func TestPullAsksPasswordForSourceSync(t *testing.T) {
	passwords := &recordingProvider{}
	mgr, comp := newPullTestManager(t, passwords)
	newSyncRunner = func(*Store, Source, *slog.Logger) syncRunner {
		return syncRunnerStub{sync: func(context.Context, *nfse.Company, *nfse.Credential, string, nfse.SyncMode, nfse.ProgressFunc) error {
			return nil
		}}
	}

	if _, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ}); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if passwords.request.Purpose != "Sincronização NFS-e" {
		t.Errorf("Purpose = %q, want %q", passwords.request.Purpose, "Sincronização NFS-e")
	}
}
