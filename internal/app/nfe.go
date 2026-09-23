package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/sync"
)

// NFeRepository is the NF-e storage the app uses. *store.NFeRepository
// satisfies it.
type NFeRepository interface {
	ListCompanyDocuments(ctx context.Context, companyID nfse.CompanyID, f nfe.DocumentFilter) ([]nfe.CompanyDocument, error)
	ListPendingExport(ctx context.Context, companyID nfse.CompanyID, f nfe.DocumentFilter, kind string) ([]nfe.CompanyDocument, error)
	ListEventsByChave(ctx context.Context, chave string) ([]nfe.Event, error)
	ListEventsByChaves(ctx context.Context, chaves []string) ([]nfe.Event, error)
	CompanyDocumentByChave(ctx context.Context, companyID nfse.CompanyID, chave string) (*nfe.CompanyDocument, error)
	CountSummary(ctx context.Context, companyID nfse.CompanyID) (nfe.Counts, error)
	MarkExported(ctx context.Context, companyID nfse.CompanyID, kind string, docs []nfe.CompanyDocument) error
	RecordManifestations(ctx context.Context, items []nfe.ManifestationRecord) error
	ResetCompany(ctx context.Context, companyID nfse.CompanyID) (nfe.ResetCounts, error)
	PreviewResetCompany(ctx context.Context, companyID nfse.CompanyID) (nfe.ResetCounts, error)
}

// sefazClient is the part of *sefaz.Client the NF-e use cases call.
type sefazClient interface {
	CheckTLS(ctx context.Context) error
	EnviarEventos(ctx context.Context, signer *sefaz.Signer, idLote string, eventos []sefaz.Evento) (sefaz.LoteResult, error)
}

// newSEFAZClient builds the SEFAZ client; tests swap it.
var newSEFAZClient = func(cfg sefaz.ClientConfig) (sefazClient, error) {
	return sefaz.NewClient(cfg)
}

// Kinds of pending manifestação.
const (
	NFePendingSemCiencia    = "sem_ciencia"    // no manifestação at all
	NFePendingSemConclusiva = "sem_conclusiva" // ciência, but no conclusive manifestação
)

// NFeService owns the NF-e use cases: pull, status, list, manifestação do
// destinatário, pending manifestações, export and the connection test.
type NFeService struct {
	Log          *slog.Logger
	CompanyStore *company.Store
	NFeRepo      NFeRepository
	SyncRepo     *sync.Store
	SyncManager  *sync.Manager
	XMLStore     files.XMLStore
	Certificates *sync.CertificateLoader

	now func() time.Time
}

func NewNFeService(d Dependencies, certificates *sync.CertificateLoader, syncManager *sync.Manager) *NFeService {
	return &NFeService{
		Log:          d.Log,
		CompanyStore: d.CompanyStore,
		NFeRepo:      d.NFeRepo,
		SyncRepo:     d.SyncRepo,
		SyncManager:  syncManager,
		XMLStore:     d.XMLStore,
		Certificates: certificates,
		now:          time.Now,
	}
}

// NFePullResult is the outcome of one NF-e distribution pull.
type NFePullResult struct {
	CompanyName      string
	CNPJ             string
	Status           string // completed | failed | interrupted
	StopReason       string // caught_up | consumo_indevido | rate_budget | ...
	UltNSU           int64  // cursor after the pull
	MaxNSU           int64  // highest NSU SEFAZ reported; 0 when unknown
	CompletasSaved   int
	ResumosSaved     int
	EventsSaved      int
	Errors           int
	NextAllowedAt    *time.Time // set while SEFAZ must not be queried
	RequestsLastHour int
	RequestBudget    int
	Duration         time.Duration
}

// Pull walks the company's NF-e distribution queue. A blocked source fails
// with ErrSourceBlocked before any password prompt.
func (s *NFeService) Pull(ctx context.Context, cnpj string) (NFePullResult, error) {
	res, err := s.SyncManager.Pull(ctx, sync.PullInput{CNPJ: cnpj, Source: nfse.SyncSourceNFe})
	if err != nil {
		return NFePullResult{}, err
	}
	return NFePullResult{
		CompanyName:      res.CompanyName,
		CNPJ:             res.CNPJ,
		Status:           res.Status,
		StopReason:       res.StopReason,
		UltNSU:           res.LastProcessedNSU,
		MaxNSU:           res.MaxNSU,
		CompletasSaved:   res.FullDocumentsSaved,
		ResumosSaved:     res.PartialDocumentsSaved,
		EventsSaved:      res.EventsSaved,
		Errors:           res.Errors,
		NextAllowedAt:    res.NextAllowedAt,
		RequestsLastHour: res.RequestsLastHour,
		RequestBudget:    res.RequestBudget,
		Duration:         res.Duration,
	}, nil
}

// NFeStatusResult describes a company's NF-e sync state and totals.
type NFeStatusResult struct {
	CompanyName       string
	CNPJ              string
	UF                string
	Environment       string
	TpAmb             string // "1" produção, "2" homologação
	AmbienteLabel     string // "Produção" | "Homologação"
	LastNSU           int64
	MaxNSU            int64 // 0 when unknown
	LastSyncAt        *time.Time
	LastRunStatus     string
	LastRunStopReason string
	InitialSyncDoneAt *time.Time
	NextAllowedAt     *time.Time // set while SEFAZ must not be queried
	BlockedReason     string     // caught_up | consumo_indevido | rate_budget; empty when not blocked
	RequestsLastHour  int
	RequestBudget     int
	TotalDestinatario int
	TotalEmitente     int
	TotalOutros       int
	TotalResumos      int
	TotalCompletas    int
	PendingCiencia    int
	PendingConclusiva int
	CienciaOverdue    int // pending ciência past its warning date
}

// Status reports the company's NF-e sync state and totals. It never
// contacts SEFAZ.
func (s *NFeService) Status(ctx context.Context, cnpj string) (NFeStatusResult, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return NFeStatusResult{}, err
	}
	tpAmb, err := sefaz.TpAmb(comp.Environment)
	if err != nil {
		return NFeStatusResult{}, fmt.Errorf("ambiente da empresa: %w", err)
	}
	now := s.now()

	result := NFeStatusResult{
		CompanyName:   comp.Name,
		CNPJ:          comp.CNPJ,
		UF:            comp.UF,
		Environment:   string(comp.Environment),
		TpAmb:         tpAmb,
		AmbienteLabel: "Homologação",
	}
	if tpAmb == sefaz.TpAmbProducao {
		result.AmbienteLabel = "Produção"
	}

	snapshot, err := s.SyncRepo.LatestSyncSnapshot(ctx, comp.ID, nfse.SyncSourceNFe, comp.Environment, comp.CNPJ)
	if err != nil {
		return NFeStatusResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}
	if snapshot.State != nil {
		result.LastNSU = snapshot.State.LastProcessedNSU
		if snapshot.State.MaxNSU != nil {
			result.MaxNSU = *snapshot.State.MaxNSU
		}
		result.LastSyncAt = snapshot.State.LastSuccessAt
	}
	if snapshot.Run != nil {
		result.LastRunStatus = string(snapshot.Run.Status)
		result.LastRunStopReason = string(snapshot.Run.StopReason)
		if snapshot.Run.FinishedAt != nil {
			result.LastSyncAt = snapshot.Run.FinishedAt
		}
	}

	sourceState, err := s.SyncRepo.SourceState(ctx, comp.ID, nfse.SyncSourceNFe)
	if err != nil {
		return NFeStatusResult{}, fmt.Errorf("carregar estado da origem: %w", err)
	}
	result.InitialSyncDoneAt = sourceState.InitialSyncDoneAt
	limits, err := s.SyncManager.SourceLimits(ctx, comp.ID, nfse.SyncSourceNFe)
	if err != nil {
		return NFeStatusResult{}, err
	}
	result.NextAllowedAt = limits.NextAllowedAt
	result.BlockedReason = string(limits.BlockedReason)
	result.RequestsLastHour = limits.RequestsLastHour
	result.RequestBudget = limits.RequestBudget

	counts, err := s.NFeRepo.CountSummary(ctx, comp.ID)
	if err != nil {
		return NFeStatusResult{}, fmt.Errorf("contar NF-e: %w", err)
	}
	for role, total := range counts.ByRole {
		switch role {
		case nfe.CompanyRoleDestinatario:
			result.TotalDestinatario += total
		case nfe.CompanyRoleEmitente:
			result.TotalEmitente += total
		default:
			result.TotalOutros += total
		}
	}
	result.TotalResumos = counts.Resumos
	result.TotalCompletas = counts.Completas

	pending, err := s.pendingManifestations(ctx, comp.ID, now)
	if err != nil {
		return NFeStatusResult{}, err
	}
	for _, p := range pending {
		switch p.Kind {
		case NFePendingSemCiencia:
			result.PendingCiencia++
		case NFePendingSemConclusiva:
			result.PendingConclusiva++
		}
		if p.CienciaOverdue {
			result.CienciaOverdue++
		}
	}
	return result, nil
}

// NFeListInput filters a company's NF-e. Empty fields do not filter.
type NFeListInput struct {
	CNPJ         string
	Competence   string // "YYYY-MM" of the issue date
	Situacao     string // autorizada | denegada | cancelada
	Completeness string // resumo | completa
	Role         string // destinatario | emitente | transportador | autorizado | none
	Manifestacao string // nenhuma | ciencia | confirmada | desconhecida | nao_realizada
	EmitenteCNPJ string
	ChavesAcesso []string
	Limit        int // 0 means no limit
}

// ListDocuments returns the company's NF-e, newest issue date first. Unlike
// NFS-e, the company start policy is not applied: the SEFAZ queue already
// covers only the last 90 days.
func (s *NFeService) ListDocuments(ctx context.Context, in NFeListInput) ([]nfe.CompanyDocument, error) {
	comp, filter, err := s.buildFilter(ctx, in)
	if err != nil {
		return nil, err
	}
	docs, err := s.NFeRepo.ListCompanyDocuments(ctx, comp.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("listar NF-e: %w", err)
	}
	return docs, nil
}

// ListEvents returns the events nanci holds for one of the company's NF-e,
// oldest first.
func (s *NFeService) ListEvents(ctx context.Context, cnpj, chave string) ([]nfe.Event, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return nil, err
	}
	doc, err := s.companyDocument(ctx, comp.ID, chave)
	if err != nil {
		return nil, err
	}
	events, err := s.NFeRepo.ListEventsByChave(ctx, string(doc.ChaveAcesso))
	if err != nil {
		return nil, fmt.Errorf("listar eventos da NF-e: %w", err)
	}
	return events, nil
}

// NFePendingInput selects the pending manifestações of a company.
type NFePendingInput struct {
	CNPJ string
	// DueWithinDays keeps rows whose conclusive deadline is at most this many
	// days away (overdue rows included). 0 keeps every row.
	DueWithinDays int
}

// NFePendingManifestation is an authorized NF-e addressed to the company
// that still lacks a conclusive manifestação.
type NFePendingManifestation struct {
	ChaveAcesso    string
	Kind           string // sem_ciencia | sem_conclusiva
	Serie          string
	Numero         string
	EmitenteCNPJ   string
	EmitenteName   string
	IssueDate      time.Time
	AuthorizedAt   *time.Time
	TotalValue     nfse.Money
	Completeness   string // resumo | completa
	Manifestacao   string // nenhuma | ciencia
	CienciaDue     time.Time
	ConclusiveDue  time.Time // the deadline shown; nanci never blocks on it
	DaysLeft       int       // whole days until ConclusiveDue; negative once past
	CienciaOverdue bool      // no manifestação and CienciaDue has passed
	// Expired is true once ConclusiveDue has passed. The operation is then
	// deemed confirmed by law (nfe.TacitlyConfirmed).
	Expired bool
}

// ListPendingManifestations returns the company's pending manifestações,
// nearest conclusive deadline first.
func (s *NFeService) ListPendingManifestations(ctx context.Context, in NFePendingInput) ([]NFePendingManifestation, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return nil, err
	}
	pending, err := s.pendingManifestations(ctx, comp.ID, s.now())
	if err != nil {
		return nil, err
	}
	if in.DueWithinDays <= 0 {
		return pending, nil
	}
	var due []NFePendingManifestation
	for _, p := range pending {
		if p.DaysLeft <= in.DueWithinDays {
			due = append(due, p)
		}
	}
	return due, nil
}

func (s *NFeService) pendingManifestations(ctx context.Context, companyID nfse.CompanyID, now time.Time) ([]NFePendingManifestation, error) {
	docs, err := s.NFeRepo.ListCompanyDocuments(ctx, companyID, nfe.DocumentFilter{PendingManifestation: true})
	if err != nil {
		return nil, fmt.Errorf("listar manifestações pendentes: %w", err)
	}

	pending := make([]NFePendingManifestation, 0, len(docs))
	for _, doc := range docs {
		deadlines := nfe.ManifestationDeadlines(doc.Document)
		kind := NFePendingSemConclusiva
		if doc.Manifestacao == nfe.ManifestacaoNenhuma {
			kind = NFePendingSemCiencia
		}
		pending = append(pending, NFePendingManifestation{
			ChaveAcesso:    string(doc.ChaveAcesso),
			Kind:           kind,
			Serie:          doc.Serie,
			Numero:         doc.Numero,
			EmitenteCNPJ:   doc.EmitenteCNPJ,
			EmitenteName:   doc.EmitenteName,
			IssueDate:      doc.IssueDate,
			AuthorizedAt:   doc.AuthorizedAt,
			TotalValue:     doc.TotalValue,
			Completeness:   string(doc.Completeness),
			Manifestacao:   string(doc.Manifestacao),
			CienciaDue:     deadlines.CienciaDue,
			ConclusiveDue:  deadlines.ConclusiveDue,
			DaysLeft:       daysLeft(deadlines.ConclusiveDue, now),
			CienciaOverdue: kind == NFePendingSemCiencia && deadlines.CienciaWarning(now),
			Expired:        nfe.TacitlyConfirmed(doc, now),
		})
	}
	slices.SortFunc(pending, func(a, b NFePendingManifestation) int {
		return cmp.Or(a.ConclusiveDue.Compare(b.ConclusiveDue), cmp.Compare(a.ChaveAcesso, b.ChaveAcesso))
	})
	return pending, nil
}

// TestConnection loads the certificate and opens a TLS connection to the
// SEFAZ distribution host. It sends no request, so it does not use the
// hourly budget; SEFAZ only checks the client certificate on a real query.
func (s *NFeService) TestConnection(ctx context.Context, cnpj string) (ConnectionTestResult, error) {
	var result ConnectionTestResult
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return result, err
	}

	loaded, err := s.Certificates.LoadForCompany(ctx, comp, "Teste de conexão NF-e")
	if err != nil {
		if errors.Is(err, ErrOperationCanceled) {
			return result, err
		}
		result.StatusExplanation = fmt.Sprintf("Erro ao carregar certificado/senha: %v", err)
		return result, nil
	}
	result.CertLoaded = true
	result.CertSubject = loaded.Credential.SubjectName
	if loaded.Credential.NotAfter != nil {
		result.CertExpiration = loaded.Credential.NotAfter.Format("02/01/2006 15:04:05")
	}

	client, err := newSEFAZClient(sefaz.ClientConfig{
		Environment: comp.Environment,
		Certificate: &loaded.TLS,
		Log:         s.Log,
	})
	if err != nil {
		result.StatusExplanation = fmt.Sprintf("Erro ao configurar cliente SEFAZ: %v", err)
		return result, nil
	}
	if err := client.CheckTLS(ctx); err != nil {
		result.StatusExplanation = fmt.Sprintf("Falha na conexão TLS com a SEFAZ: %v", err)
		return result, nil
	}
	result.EndpointReached = true
	result.StatusExplanation = "Conexão TLS com a SEFAZ estabelecida. Nenhuma consulta foi enviada, para não gastar o limite de consultas por hora; a SEFAZ só valida o certificado na primeira consulta."
	return result, nil
}

// buildFilter resolves the company and validates the list filters.
func (s *NFeService) buildFilter(ctx context.Context, in NFeListInput) (*nfse.Company, nfe.DocumentFilter, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return nil, nfe.DocumentFilter{}, err
	}
	filter := nfe.DocumentFilter{
		Competence:   in.Competence,
		EmitenteCNPJ: in.EmitenteCNPJ,
		ChavesAcesso: in.ChavesAcesso,
		Limit:        in.Limit,
	}
	if in.Situacao != "" {
		if filter.Situacao, err = nfe.ParseSituacao(in.Situacao); err != nil {
			return nil, nfe.DocumentFilter{}, fmt.Errorf("situação inválida %q", in.Situacao)
		}
	}
	if in.Completeness != "" {
		if filter.Completeness, err = nfe.ParseCompleteness(in.Completeness); err != nil {
			return nil, nfe.DocumentFilter{}, fmt.Errorf("completude inválida %q", in.Completeness)
		}
	}
	if in.Role != "" {
		if filter.Role, err = nfe.ParseCompanyRole(in.Role); err != nil {
			return nil, nfe.DocumentFilter{}, fmt.Errorf("papel inválido %q", in.Role)
		}
	}
	if in.Manifestacao != "" {
		if filter.Manifestacao, err = nfe.ParseManifestacao(in.Manifestacao); err != nil {
			return nil, nfe.DocumentFilter{}, fmt.Errorf("manifestação inválida %q", in.Manifestacao)
		}
	}
	return comp, filter, nil
}

// companyDocument returns the company's row for the chave.
func (s *NFeService) companyDocument(ctx context.Context, companyID nfse.CompanyID, rawChave string) (nfe.CompanyDocument, error) {
	chave, err := nfe.ParseAccessKey(rawChave)
	if err != nil {
		return nfe.CompanyDocument{}, fmt.Errorf("chave de acesso inválida: %w", err)
	}
	doc, err := s.NFeRepo.CompanyDocumentByChave(ctx, companyID, string(chave))
	if errors.Is(err, nfe.ErrDocumentNotFound) {
		return nfe.CompanyDocument{}, fmt.Errorf("NF-e %s não encontrada para a empresa", chave)
	}
	if err != nil {
		return nfe.CompanyDocument{}, fmt.Errorf("buscar NF-e: %w", err)
	}
	return *doc, nil
}
