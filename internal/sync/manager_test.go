package sync

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/nfse"
	dbstore "github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

type dummyCompanyProvider struct {
	company *nfse.Company
}

func (d *dummyCompanyProvider) CompanyByCNPJ(ctx context.Context, cnpj string) (*nfse.Company, error) {
	return d.company, nil
}

type dummyCredentialProvider struct {
	credential *nfse.Credential
}

func (d *dummyCredentialProvider) CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error) {
	return d.credential, nil
}
func (d *dummyCredentialProvider) UpdateCredential(ctx context.Context, c *nfse.Credential) error {
	return nil
}

type dummyDocumentProvider struct{}

func (d *dummyDocumentProvider) CountDocumentsByRole(ctx context.Context, companyID nfse.CompanyID) (map[string]int64, error) {
	return nil, nil
}

type providerStub struct{}

func (providerStub) GetCertPassword(context.Context, CertPasswordRequest) (string, error) {
	return "secret", nil
}

type captureXMLStore struct {
	storeCalls []string
}

func (s *captureXMLStore) Store(hash string, data []byte) error {
	s.storeCalls = append(s.storeCalls, hash)
	return nil
}

func (s *captureXMLStore) Get(string) ([]byte, error) { return nil, nil }

type syncRunnerStub struct {
	sync func(context.Context, *nfse.Company, *nfse.Credential, string, nfse.SyncMode, nfse.ProgressFunc) error
}

func (s syncRunnerStub) Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error {
	return s.sync(ctx, company, credential, consultationBasis, mode, progress)
}

func TestPullUsesInjectedXMLStore(t *testing.T) {
	db := storetest.OpenTestDB(t)
	companyStore := company.NewStore(db)
	credentialStore := credential.NewStore(db)

	comp := &nfse.Company{ //nolint:gosec
		ID:           "company-1",
		CNPJ:         "11222333000181",
		CNPJRoot:     "11222333",
		Name:         "Company",
		CredentialID: "credential-1",
		Environment:  nfse.EnvironmentProduction,
	}
	if err := companyStore.CreateCompany(context.Background(), comp); err != nil {
		t.Fatal(err)
	}

	certPath := filepath.Join(t.TempDir(), "cert.pfx")
	if err := os.WriteFile(certPath, []byte("stub"), 0o600); err != nil {
		t.Fatal(err)
	}

	cred := &nfse.Credential{
		ID:       "credential-1",
		Label:    "Credential",
		CertPath: certPath,
	}
	if err := credentialStore.CreateCredential(context.Background(), cred); err != nil {
		t.Fatal(err)
	}

	xmlStoreVal := &captureXMLStore{}
	mgr := &Manager{
		Log:                slog.New(slog.DiscardHandler),
		CompanyProvider:    companyStore,
		CredentialProvider: credentialStore,
		DocProvider:        dbstore.NewDocumentRepository(db),
		SyncRepo:           NewStore(db),
		XMLStore:           xmlStoreVal,
		PassProvider:       providerStub{},
	}

	originalLoadPKCS12 := loadPKCS12
	originalNewADNClient := newADNClient
	originalNewSyncRunner := newSyncRunner
	t.Cleanup(func() {
		loadPKCS12 = originalLoadPKCS12
		newADNClient = originalNewADNClient
		newSyncRunner = originalNewSyncRunner
	})

	loadPKCS12 = func(string, string) (cert.LoadedCertificate, error) {
		now := time.Now().UTC()
		return cert.LoadedCertificate{
			TLS: tls.Certificate{},
			Inspection: cert.Inspection{
				OwnerCNPJ:         "11222333000181",
				OwnerCNPJRoot:     "11222333",
				FingerprintSHA256: "fingerprint",
				SubjectName:       "CN=Company",
				NotBefore:         now,
				NotAfter:          now.Add(24 * time.Hour),
			},
		}, nil
	}
	newADNClient = func(adn.ClientConfig) (*adn.Client, error) {
		return &adn.Client{}, nil
	}

	var receivedStore files.XMLStore
	newSyncRunner = func(repo *Store, client *adn.Client, store files.XMLStore, log *slog.Logger) syncRunner {
		receivedStore = store
		return syncRunnerStub{
			sync: func(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error {
				if progress != nil {
					progress(nfse.ProgressEvent{DocsFound: 1})
				}
				return store.Store("hash-1", []byte("<NFSe/>"))
			},
		}
	}

	result, err := mgr.Pull(context.Background(), PullInput{CNPJ: "11222333000181"})
	if err != nil {
		t.Fatal(err)
	}
	if receivedStore != xmlStoreVal {
		t.Fatal("expected Pull to pass the injected XMLStore to the sync service")
	}
	if len(xmlStoreVal.storeCalls) != 1 || xmlStoreVal.storeCalls[0] != "hash-1" {
		t.Fatalf("unexpected XMLStore usage: %v", xmlStoreVal.storeCalls)
	}
	if result.DocumentsFound != 1 {
		t.Fatalf("DocumentsFound = %d, want 1", result.DocumentsFound)
	}
}
