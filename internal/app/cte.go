package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// CTeService owns the CT-e use cases: pull, status, list, events, export,
// reset and the connection test. Every listing, count and export shows only
// the documents of the company's current environment.
type CTeService struct {
	Log          *slog.Logger
	CompanyStore *company.Store
	CTeRepo      *store.CTeRepository
	SyncRepo     *sync.Store
	SyncManager  *sync.Manager
	XMLStore     files.XMLStore
	Certificates *sync.CertificateLoader
}

func NewCTeService(d Dependencies, certificates *sync.CertificateLoader, syncManager *sync.Manager) *CTeService {
	return &CTeService{
		Log:          d.Log,
		CompanyStore: d.CompanyStore,
		CTeRepo:      d.CTeRepo,
		SyncRepo:     d.SyncRepo,
		SyncManager:  syncManager,
		XMLStore:     d.XMLStore,
		Certificates: certificates,
	}
}

// CTePullResult is the outcome of one CT-e distribution pull.
type CTePullResult struct {
	CompanyName string
	CNPJ        string
	Status      string // completed | failed | interrupted
	StopReason  string // caught_up | consumo_indevido | rate_budget | ...
	LastNSU     int64  // cursor after the pull (the last ultNSU)
	MaxNSU      *int64 // highest NSU SEFAZ reported; nil when unknown
	// DocumentsSaved counts the documents stored in this pull, new or
	// updated. CT-e distribution has no resumo.
	DocumentsSaved   int
	EventsSaved      int
	Errors           int
	NextAllowedAt    *time.Time // set while SEFAZ must not be queried
	RequestsLastHour int
	RequestBudget    int
	Duration         time.Duration
}

// Pull walks the company's CT-e distribution queue. A blocked source fails
// with ErrSourceBlocked before any password prompt.
func (s *CTeService) Pull(ctx context.Context, cnpj string) (CTePullResult, error) {
	res, err := s.SyncManager.Pull(ctx, sync.PullInput{CNPJ: cnpj, Source: nfse.SyncSourceCTe})
	if err != nil {
		return CTePullResult{}, err
	}
	return CTePullResult{
		CompanyName:      res.CompanyName,
		CNPJ:             res.CNPJ,
		Status:           res.Status,
		StopReason:       res.StopReason,
		LastNSU:          res.LastProcessedNSU,
		MaxNSU:           res.MaxNSU,
		DocumentsSaved:   res.CompletasSaved,
		EventsSaved:      res.EventsSaved,
		Errors:           res.Errors,
		NextAllowedAt:    res.NextAllowedAt,
		RequestsLastHour: res.RequestsLastHour,
		RequestBudget:    res.RequestBudget,
		Duration:         res.Duration,
	}, nil
}

// CTeStatusResult describes a company's CT-e sync state and totals. The
// totals count the documents of the current environment by primary role.
type CTeStatusResult struct {
	CompanyName       string
	CNPJ              string
	UF                string
	TpAmb             string // "1" produção, "2" homologação
	LastNSU           int64
	MaxNSU            *int64 // nil when unknown
	LastSyncAt        *time.Time
	LastRunStatus     string
	LastRunStopReason string
	InitialSyncDoneAt *time.Time
	NextAllowedAt     *time.Time // set while SEFAZ must not be queried
	BlockedReason     string     // caught_up | consumo_indevido | rate_budget; empty when not blocked
	RequestsLastHour  int
	RequestBudget     int
	TotalTomador      int
	TotalDestinatario int
	TotalRemetente    int
	TotalOutros       int // expedidor, recebedor, emitente, autorizado and none
}

// Status reports the company's CT-e sync state and totals. It never
// contacts SEFAZ.
func (s *CTeService) Status(ctx context.Context, cnpj string) (CTeStatusResult, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return CTeStatusResult{}, err
	}
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return CTeStatusResult{}, err
	}

	result := CTeStatusResult{
		CompanyName: comp.Name,
		CNPJ:        comp.CNPJ,
		UF:          comp.UF,
		TpAmb:       tpAmb,
	}

	snapshot, err := s.SyncRepo.LatestSyncSnapshot(ctx, comp.ID, nfse.SyncSourceCTe, comp.Environment, comp.CNPJ)
	if err != nil {
		return CTeStatusResult{}, fmt.Errorf("carregar snapshot de sincronização: %w", err)
	}
	if snapshot.State != nil {
		result.LastNSU = snapshot.State.LastProcessedNSU
		result.MaxNSU = snapshot.State.MaxNSU
		result.LastSyncAt = snapshot.State.LastSuccessAt
	}
	if snapshot.Run != nil {
		result.LastRunStatus = string(snapshot.Run.Status)
		result.LastRunStopReason = string(snapshot.Run.StopReason)
		if snapshot.Run.FinishedAt != nil {
			result.LastSyncAt = snapshot.Run.FinishedAt
		}
	}

	sourceState, err := s.SyncRepo.SourceState(ctx, comp.ID, nfse.SyncSourceCTe)
	if err != nil {
		return CTeStatusResult{}, fmt.Errorf("carregar estado da origem: %w", err)
	}
	result.InitialSyncDoneAt = sourceState.InitialSyncDoneAt
	limits, err := s.SyncManager.SourceLimits(ctx, comp.ID, nfse.SyncSourceCTe)
	if err != nil {
		return CTeStatusResult{}, err
	}
	result.NextAllowedAt = limits.NextAllowedAt
	result.BlockedReason = string(limits.BlockedReason)
	result.RequestsLastHour = limits.RequestsLastHour
	result.RequestBudget = limits.RequestBudget

	counts, err := s.CTeRepo.CountSummary(ctx, comp.ID, tpAmb)
	if err != nil {
		return CTeStatusResult{}, fmt.Errorf("contar CT-e: %w", err)
	}
	for role, total := range counts.ByRole {
		switch role {
		case cte.CompanyRoleTomador:
			result.TotalTomador += total
		case cte.CompanyRoleDestinatario:
			result.TotalDestinatario += total
		case cte.CompanyRoleRemetente:
			result.TotalRemetente += total
		default:
			result.TotalOutros += total
		}
	}
	return result, nil
}

// ListCTeInput filters a company's CT-e. Empty fields do not filter.
type ListCTeInput struct {
	CNPJ       string
	Competence string // "YYYY-MM" of the issue date
	Situacao   string // autorizada | denegada | cancelada
	// Role matches the primary role or any other role the company plays:
	// tomador | destinatario | remetente | expedidor | recebedor | emitente | autorizado | none
	Role         string
	Modelo       string // 57 | 64 | 67
	EmitenteCNPJ string
	TomadorCNPJ  string
	// NFeChave keeps the CT-e that transported this NF-e.
	NFeChave     string
	ChavesAcesso []string
	Limit        int // 0 means no limit
}

// ListDocuments returns the company's CT-e of its current environment,
// newest issue date first.
func (s *CTeService) ListDocuments(ctx context.Context, in ListCTeInput) ([]cte.CompanyDocument, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return nil, err
	}
	filter, err := cteFilter(comp, in)
	if err != nil {
		return nil, err
	}
	docs, err := s.CTeRepo.ListCompanyDocuments(ctx, comp.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("listar CT-e: %w", err)
	}
	return docs, nil
}

// ListEvents returns the events nanci holds for one of the company's CT-e,
// oldest first.
func (s *CTeService) ListEvents(ctx context.Context, cnpj, chave string) ([]cte.Event, error) {
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return nil, err
	}
	doc, err := s.companyDocument(ctx, comp, chave)
	if err != nil {
		return nil, err
	}
	events, err := s.CTeRepo.ListEventsByChave(ctx, string(doc.ChaveAcesso))
	if err != nil {
		return nil, fmt.Errorf("listar eventos do CT-e: %w", err)
	}
	return events, nil
}

// TestConnection loads the certificate and opens a TLS connection to the
// CT-e distribution host. It sends no request, so it does not use the
// hourly budget; SEFAZ only checks the client certificate on a real query.
func (s *CTeService) TestConnection(ctx context.Context, cnpj string) (ConnectionTestResult, error) {
	var result ConnectionTestResult
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, cnpj)
	if err != nil {
		return result, err
	}

	loaded, err := s.Certificates.LoadForCompany(ctx, comp, "Teste de conexão CT-e")
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
	if err := client.CheckTLSCTe(ctx); err != nil {
		result.StatusExplanation = fmt.Sprintf("Falha na conexão TLS com a SEFAZ (CT-e): %v", err)
		return result, nil
	}
	result.EndpointReached = true
	result.StatusExplanation = "Conexão TLS com a SEFAZ (CT-e) estabelecida. Nenhuma consulta foi enviada, para não gastar o limite de consultas por hora; a SEFAZ só valida o certificado na primeira consulta."
	return result, nil
}

// cteFilter validates the list filters and pins them to the company's
// current environment.
func cteFilter(comp *nfse.Company, in ListCTeInput) (cte.DocumentFilter, error) {
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return cte.DocumentFilter{}, err
	}
	chaves, err := parseAccessKeys(in.ChavesAcesso)
	if err != nil {
		return cte.DocumentFilter{}, err
	}
	filter := cte.DocumentFilter{
		Competence:   in.Competence,
		EmitenteCNPJ: in.EmitenteCNPJ,
		TomadorCNPJ:  in.TomadorCNPJ,
		ChavesAcesso: chaves,
		TpAmb:        tpAmb,
		Limit:        in.Limit,
	}
	if in.Situacao != "" {
		if filter.Situacao, err = cte.ParseSituacao(in.Situacao); err != nil {
			return cte.DocumentFilter{}, fmt.Errorf("situação inválida: %w", err)
		}
	}
	if in.Role != "" {
		if filter.Role, err = cte.ParseCompanyRole(in.Role); err != nil {
			return cte.DocumentFilter{}, fmt.Errorf("papel inválido: %w", err)
		}
	}
	switch in.Modelo {
	case "", "57", "64", "67":
		filter.Modelo = in.Modelo
	default:
		return cte.DocumentFilter{}, fmt.Errorf("modelo inválido %q (use 57, 64 ou 67): %w", in.Modelo, dfe.ErrInvalidEnum)
	}
	if in.NFeChave != "" {
		chave, err := dfe.ParseAccessKey(in.NFeChave)
		if err != nil {
			return cte.DocumentFilter{}, fmt.Errorf("chave de NF-e inválida: %w", err)
		}
		filter.NFeChave = string(chave)
	}
	return filter, nil
}

// companyDocument returns the company's row for the chave in its current
// environment, or an error matching cte.ErrDocumentNotFound.
func (s *CTeService) companyDocument(ctx context.Context, comp *nfse.Company, rawChave string) (cte.CompanyDocument, error) {
	chave, err := dfe.ParseAccessKey(rawChave)
	if err != nil {
		return cte.CompanyDocument{}, fmt.Errorf("chave de acesso inválida: %w", err)
	}
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return cte.CompanyDocument{}, err
	}
	doc, err := s.CTeRepo.CompanyDocumentByChave(ctx, comp.ID, tpAmb, string(chave))
	if errors.Is(err, cte.ErrDocumentNotFound) {
		return cte.CompanyDocument{}, fmt.Errorf("chave %s: %w", chave, err)
	}
	if err != nil {
		return cte.CompanyDocument{}, fmt.Errorf("buscar CT-e: %w", err)
	}
	return *doc, nil
}
