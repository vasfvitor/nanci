package sync

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	gosync "sync"
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

type providerStub struct{}

func (providerStub) GetCertPassword(context.Context, CertPasswordRequest) ([]byte, error) {
	return []byte("secret"), nil
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
		Certificates:       &CertificateLoader{Log: slog.New(slog.DiscardHandler), Credentials: credentialStore, Passwords: providerStub{}},
	}

	originalLoadPKCS12 := loadPKCS12
	originalNewADNClient := newADNClient
	originalNewSyncRunner := newSyncRunner
	t.Cleanup(func() {
		loadPKCS12 = originalLoadPKCS12
		newADNClient = originalNewADNClient
		newSyncRunner = originalNewSyncRunner
	})

	loadPKCS12 = func(string, []byte) (cert.LoadedCertificate, error) {
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
	newSyncRunner = func(repo *Store, src Source, log *slog.Logger) syncRunner {
		nfseSrc, ok := src.(*nfseSource)
		if !ok {
			t.Fatalf("source = %T, want *nfseSource", src)
		}
		receivedStore = nfseSrc.xml
		store := nfseSrc.xml
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

type countingProvider struct {
	mu    gosync.Mutex
	calls int
}

func (p *countingProvider) GetCertPassword(context.Context, CertPasswordRequest) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return []byte("secret"), nil
}

func (p *countingProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// newPullTestManager returns a Manager over a real database whose certificate
// loading and ADN client are stubbed, plus its only company.
func newPullTestManager(t *testing.T, passwords CredentialProvider) (*Manager, *nfse.Company) {
	t.Helper()
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
	if err := credentialStore.CreateCredential(context.Background(), &nfse.Credential{ID: "credential-1", Label: "Credential", CertPath: certPath}); err != nil {
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
	loadPKCS12 = func(string, []byte) (cert.LoadedCertificate, error) {
		now := time.Now().UTC()
		return cert.LoadedCertificate{Inspection: cert.Inspection{
			OwnerCNPJ:     "11222333000181",
			OwnerCNPJRoot: "11222333",
			NotBefore:     now,
			NotAfter:      now.Add(24 * time.Hour),
		}}, nil
	}
	newADNClient = func(adn.ClientConfig) (*adn.Client, error) {
		return &adn.Client{}, nil
	}

	return &Manager{
		Log:                slog.New(slog.DiscardHandler),
		CompanyProvider:    companyStore,
		CredentialProvider: credentialStore,
		DocProvider:        dbstore.NewDocumentRepository(db),
		SyncRepo:           NewStore(db),
		XMLStore:           &captureXMLStore{},
		Certificates:       &CertificateLoader{Log: slog.New(slog.DiscardHandler), Credentials: credentialStore, Passwords: passwords},
	}, comp
}

func TestPullReturnsBlockedErrorBeforePasswordPrompt(t *testing.T) {
	passwords := &countingProvider{}
	mgr, comp := newPullTestManager(t, passwords)
	until := time.Now().Add(time.Hour)
	if err := mgr.SyncRepo.SetBlockedUntil(context.Background(), comp.ID, nfse.SyncSourceNFSe, until, nfse.SyncStopReasonConsumoIndevido); err != nil {
		t.Fatal(err)
	}

	_, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ})
	var blocked *BlockedError
	if !errors.As(err, &blocked) || !errors.Is(err, ErrSourceBlocked) {
		t.Fatalf("Pull error = %v, want *BlockedError", err)
	}
	if blocked.Reason != nfse.SyncStopReasonConsumoIndevido {
		t.Errorf("Reason = %q, want consumo_indevido", blocked.Reason)
	}
	if got := passwords.callCount(); got != 0 {
		t.Errorf("password prompts = %d, want 0", got)
	}
}

func TestPullRefusesSecondPullOfSameCompanyAndSource(t *testing.T) {
	passwords := &countingProvider{}
	mgr, comp := newPullTestManager(t, passwords)

	started := make(chan struct{})
	release := make(chan struct{})
	newSyncRunner = func(*Store, Source, *slog.Logger) syncRunner {
		return syncRunnerStub{
			sync: func(context.Context, *nfse.Company, *nfse.Credential, string, nfse.SyncMode, nfse.ProgressFunc) error {
				close(started)
				<-release
				return nil
			},
		}
	}

	firstErr := make(chan error, 1)
	go func() {
		_, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ})
		firstErr <- err
	}()
	<-started

	if _, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ}); !errors.Is(err, ErrSyncRunning) {
		t.Errorf("second Pull error = %v, want ErrSyncRunning", err)
	}
	if got := passwords.callCount(); got != 1 {
		t.Errorf("password prompts = %d, want 1 (the second pull must not prompt)", got)
	}

	close(release)
	if err := <-firstErr; err != nil {
		t.Fatalf("first Pull: %v", err)
	}

	// Once the first pull ends, the pair is free again.
	newSyncRunner = func(*Store, Source, *slog.Logger) syncRunner {
		return syncRunnerStub{sync: func(context.Context, *nfse.Company, *nfse.Credential, string, nfse.SyncMode, nfse.ProgressFunc) error {
			return nil
		}}
	}
	if _, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ}); err != nil {
		t.Fatalf("third Pull: %v", err)
	}
}

func TestPullRejectsSourcesWithoutALoop(t *testing.T) {
	passwords := &countingProvider{}
	mgr, comp := newPullTestManager(t, passwords)

	for _, source := range []nfse.SyncSource{nfse.SyncSourceCTe, "bogus"} {
		if _, err := mgr.Pull(context.Background(), PullInput{CNPJ: comp.CNPJ, Source: source}); err == nil {
			t.Errorf("Pull with source %q succeeded, want an error", source)
		}
	}
	if got := passwords.callCount(); got != 0 {
		t.Errorf("password prompts = %d, want 0", got)
	}
}
