package app

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

type syncRunnerStub struct {
	sync func(context.Context, *nfse.Company, *nfse.Credential, string, nfse.SyncMode, nfse.ProgressFunc) error
}

func (s syncRunnerStub) Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error {
	return s.sync(ctx, company, credential, consultationBasis, mode, progress)
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

func TestPullUsesInjectedXMLStore(t *testing.T) {
	dataDir := t.TempDir()
	db, err := store.OpenDB(context.Background(), filepath.Join(dataDir, "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	companyRepo := store.NewCompanyRepository(db)
	credentialRepo := store.NewCredentialRepository(db)
	company := &nfse.Company{ //nolint:gosec // intentional: mock test credentials
		ID:           "company-1",
		CNPJ:         "11222333000181",
		CNPJRoot:     "11222333",
		Name:         "Company",
		CredentialID: "credential-1",
		Environment:  nfse.EnvironmentProduction,
	}
	if err := companyRepo.CreateCompany(context.Background(), company); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(t.TempDir(), "cert.pfx")
	if err := os.WriteFile(certPath, []byte("stub"), 0o600); err != nil {
		t.Fatal(err)
	}
	credential := &nfse.Credential{
		ID:       "credential-1",
		Label:    "Credential",
		CertPath: certPath,
	}
	if err := credentialRepo.CreateCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}

	xmlStore := &captureXMLStore{}
	application, err := New(Dependencies{
		Log:                slog.New(slog.DiscardHandler),
		CompanyRepo:        companyRepo,
		CredentialRepo:     credentialRepo,
		SyncRepo:           store.NewSyncRepository(db),
		DocumentReader:     store.NewDocumentRepository(db),
		DocumentTracker:    store.NewDocumentRepository(db),
		XMLStore:           xmlStore,
		DataDir:            dataDir,
		CredentialProvider: providerStub{},
	})
	if err != nil {
		t.Fatal(err)
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
	newSyncRunner = func(repo SyncRepository, client *adn.Client, store files.XMLStore, log *slog.Logger) syncRunner {
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

	result, err := application.Pull(context.Background(), PullInput{CNPJ: "11222333000181"})
	if err != nil {
		t.Fatal(err)
	}
	if receivedStore != xmlStore {
		t.Fatal("expected Pull to pass the injected XMLStore to the sync service")
	}
	if len(xmlStore.storeCalls) != 1 || xmlStore.storeCalls[0] != "hash-1" {
		t.Fatalf("unexpected XMLStore usage: %v", xmlStore.storeCalls)
	}
	if result.DocumentsFound != 1 {
		t.Fatalf("DocumentsFound = %d, want 1", result.DocumentsFound)
	}
}
