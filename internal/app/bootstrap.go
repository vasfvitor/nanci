package app

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dacte"
	"github.com/vasfvitor/nanci/internal/danfe"
	"github.com/vasfvitor/nanci/internal/danfse"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/envfile"
	"github.com/vasfvitor/nanci/internal/foundation/paths"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

var ErrOperationCanceled = errors.New("operação cancelada pelo usuário")

// ErrSourceBlocked matches a pull refused because the source asked to wait;
// errors.As with *sync.BlockedError gives the time and the reason.
var ErrSourceBlocked = sync.ErrSourceBlocked

// ErrSyncRunning matches a pull refused because the same company and source
// are already syncing in this process.
var ErrSyncRunning = sync.ErrSyncRunning

// ErrWrongPassword matches a typed or given certificate password that does
// not open the certificate.
var ErrWrongPassword = errors.New("senha inválida para o certificado")

// ErrInvalidObservacao matches an observation of a CT-e event that the
// schema would refuse (sefaz.CheckXObsDesacordo).
var ErrInvalidObservacao = errors.New("observação inválida")

// ErrPasswordRequired matches a certificate password that is not saved and
// that the CredentialProvider has no way to ask for, as in the CLI run by a
// script or an AI agent without a terminal.
var ErrPasswordRequired = errors.New("a senha do certificado não está salva e não há terminal para pedi-la")

// CertPasswordRequest carries the context needed to ask for a certificate password.
type CertPasswordRequest = sync.CertPasswordRequest

// CredentialProvider abstracts how certificate passwords are obtained.
type CredentialProvider = sync.CredentialProvider

// App encapsulates the global dependencies of the application.
type App struct {
	Companies   *company.Manager
	Credentials *credential.Manager
	Documents   *DocumentService
	Exports     *ExportService
	Query       *QueryService
	NFe         *NFeService
	CTe         *CTeService
	SyncManager *sync.Manager
}

// Dependencies contains the infrastructure required by App.
type Dependencies struct {
	Log                *slog.Logger
	CompanyStore       *company.Store
	CredentialStore    *credential.Store
	SyncRepo           *sync.Store
	DocumentRepo       *store.DocumentRepository
	NFeRepo            *store.NFeRepository
	CTeRepo            *store.CTeRepository
	XMLStore           files.XMLStore
	DataDir            string
	CredentialProvider CredentialProvider
	DANFSeRenderer     danfse.Renderer
	// DANFERenderer and DACTERenderer render the NF-e and CT-e PDFs; nil
	// leaves those exports unavailable.
	DANFERenderer danfe.Renderer
	DACTERenderer dacte.Renderer
	// Holder names the process in the source leases ("aplicativo", "CLI"),
	// so the other process can say who holds a source.
	Holder string
}

// New constructs an App and rejects incomplete dependency graphs.
func New(deps Dependencies) (*App, error) {
	switch {
	case deps.Log == nil:
		return nil, errors.New("app: logger is required")
	case deps.CompanyStore == nil:
		return nil, errors.New("app: company repository is required")
	case deps.CredentialStore == nil:
		return nil, errors.New("app: credential repository is required")
	case deps.SyncRepo == nil:
		return nil, errors.New("app: sync repository is required")
	case deps.DocumentRepo == nil:
		return nil, errors.New("app: document repository is required")
	case deps.NFeRepo == nil:
		return nil, errors.New("app: NF-e repository is required")
	case deps.CTeRepo == nil:
		return nil, errors.New("app: CT-e repository is required")
	case deps.XMLStore == nil:
		return nil, errors.New("app: XML store is required")
	case deps.DataDir == "":
		return nil, errors.New("app: data directory is required")
	case deps.CredentialProvider == nil:
		return nil, errors.New("app: credential provider is required")
	}

	certificates := &sync.CertificateLoader{
		Log:         deps.Log,
		Credentials: deps.CredentialStore,
		Passwords:   deps.CredentialProvider,
	}

	syncManager := &sync.Manager{
		Log:                deps.Log,
		CompanyProvider:    deps.CompanyStore,
		CredentialProvider: deps.CredentialStore,
		DocProvider:        deps.DocumentRepo,
		SyncRepo:           deps.SyncRepo,
		XMLStore:           deps.XMLStore,
		Certificates:       certificates,
		NFeRepo:            deps.NFeRepo,
		CTeRepo:            deps.CTeRepo,
		Holder:             deps.Holder,
	}

	return &App{
		Companies:   company.NewManager(deps.CompanyStore, deps.CredentialStore, deps.SyncRepo),
		Credentials: credential.NewManager(deps.CredentialStore),
		Documents:   NewDocumentService(deps),
		Exports:     NewExportService(deps),
		Query:       NewQueryService(deps, certificates),
		NFe:         NewNFeService(deps, certificates, syncManager),
		CTe:         NewCTeService(deps, certificates, syncManager),
		SyncManager: syncManager,
	}, nil
}

// LoadRuntimeEnv loads supported .env.local files.
func LoadRuntimeEnv() error {
	if err := envfile.LoadLocal(); err != nil {
		return fmt.Errorf("carregar .env.local: %w", err)
	}
	return nil
}

func ResolveRuntimeDataDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dataDir, err := paths.DataDir()
	if err != nil {
		return "", fmt.Errorf("resolver diretório de dados: %w", err)
	}
	return dataDir, nil
}

func RuntimeDBPath(dataDir string) string {
	return filepath.Join(dataDir, "nanci-v1.db")
}
