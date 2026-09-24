package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// ErrorPayload is the value a bound method's promise rejects with. Code names
// the errors the frontend branches on ("canceled", "sefaz_blocked",
// "sync_running") and is empty for any other error.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StatusResult struct {
	CompanyName        string
	CNPJ               string
	Environment        string
	ConsultationCNPJ   string
	CredentialCNPJ     string
	CredentialNotAfter *time.Time
	LastProcessedNSU   int64
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	LastRunStatus      string
	LastRunStopReason  string
	TotalEmitidas      int64
	TotalTomadas       int64
}

type CompanySummary struct {
	ID                 string
	CNPJ               string
	CNPJRoot           string
	Name               string
	CredentialID       string
	CredentialLabel    string
	CredentialCertPath string
	Environment        string
	UF                 string
	LastFoundNSU       *int64
	LastSyncAt         *time.Time
	SyncStartPolicy    string
	SyncStartDate      *time.Time
	InitialSyncDoneAt  *time.Time
	LastRunStatus      string
	LastRunStopReason  string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CredentialSummary struct {
	ID                string
	Label             string
	CertPath          string
	OwnerCNPJ         string
	OwnerCNPJRoot     string
	FingerprintSHA256 string
	SubjectName       string
	NotBefore         *time.Time
	NotAfter          *time.Time
	InspectedAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type DocumentRow struct {
	ID                 string
	ChaveAcesso        string
	IssueDate          time.Time
	Competence         string
	PrestadorCNPJ      string
	PrestadorName      string
	TomadorCNPJ        string
	TomadorName        string
	IntermediarioCNPJ  string
	IntermediarioName  string
	ServiceValue       int64
	ISSValue           int64
	IRRFValue          int64
	INSSValue          int64
	PISValue           int64
	COFINSValue        int64
	CSLLValue          int64
	TotalRetentions    int64
	Status             string
	LayoutVersion      string
	XMLPath            string
	RawHash            string
	ParseWarnings      []string
	NFSeNumber         string
	ServiceDescription string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	RelationID         string
	CompanyID          string
	DocumentID         string
	CompanyRole        string
	VisibilityReason   string
	FirstSeenNSU       *int64
	LastSeenNSU        *int64
	FirstSyncedAt      time.Time
	LastSyncedAt       time.Time
	ViewedAt           *time.Time
}

type DocumentEvent struct {
	ID                     string
	Type                   string
	EventAt                *time.Time
	ReplacementChaveAcesso string
	Description            string
	RawXMLPath             string
}

type AddCompanyInput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	CertPath        string
	Environment     string // "producao" | "producao_restrita"
	UF              string // state sigla such as "SP"; empty when unknown
	SyncStartPolicy string // "all" | "since_date" | "from_now"
	SyncStartDate   string // "YYYY-MM-DD" when SyncStartPolicy is since_date
}

type UpdateCompanyInput struct {
	CNPJ            string
	Name            string
	Environment     string // "producao" | "producao_restrita"
	UF              string // state sigla such as "SP"; empty when unknown
	SyncStartPolicy string // "all" | "since_date" | "from_now"
	SyncStartDate   string // "YYYY-MM-DD" when SyncStartPolicy is since_date
}

type AddCredentialInput struct {
	Label    string
	CertPath string
}

type UpdateCredentialPathInput struct {
	CredentialID string
	CertPath     string
}

type AssignCredentialInput struct {
	CompanyCNPJ  string
	CredentialID string
}

type UpdateCredentialDataInput struct {
	CredentialID string
	Label        string
}

type ListInput struct {
	CNPJ       string
	Competence string
	Direction  string
	OnlyUnread bool
}

type PullInput struct {
	CNPJ string
	Mode string
}

type PullResult struct {
	CompanyName              string
	CNPJ                     string
	CredentialLabel          string
	CredentialCNPJ           string
	ConsultationBasis        string
	Status                   string
	StopReason               string
	LastProcessedNSU         int64
	LastFoundNSU             *int64
	EmptyStreak              int
	DocumentsFound           int
	EventsFound              int
	DocumentsSaved           int
	EventsSaved              int
	DocumentsSkippedByPolicy int
	EventsSkippedByPolicy    int
	Errors                   int
	Duration                 time.Duration
}

type QueryNFSeInput struct {
	CompanyCNPJ string
	ChaveAcesso string
}

type ResetSyncInput struct {
	CompanyCNPJ string
}

type ExportDocumentsInput struct {
	CNPJ         string
	Competence   string
	Direction    string
	Format       string
	OutPath      string
	Incremental  bool
	ChavesAcesso []string
}

type ExportDANFSeInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

type ExportXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

type ExportResult struct {
	OutPath       string
	Format        string
	Incremental   bool
	ExportedCount int
}

func CompanySummaries(companies []nfse.Company) []CompanySummary {
	out := make([]CompanySummary, len(companies))
	for i, company := range companies {
		out[i] = CompanySummary{
			ID:                 string(company.ID),
			CNPJ:               company.CNPJ,
			CNPJRoot:           company.CNPJRoot,
			Name:               company.Name,
			CredentialID:       string(company.CredentialID),
			CredentialLabel:    company.CredentialLabel,
			CredentialCertPath: company.CredentialCertPath,
			Environment:        string(company.Environment),
			UF:                 company.UF,
			LastFoundNSU:       company.LastFoundNSU,
			LastSyncAt:         company.LastSyncAt,
			SyncStartPolicy:    string(company.SyncStartPolicy),
			SyncStartDate:      company.SyncStartDate,
			InitialSyncDoneAt:  company.InitialSyncDoneAt,
			LastRunStatus:      string(company.LastRunStatus),
			LastRunStopReason:  string(company.LastRunStopReason),
			CreatedAt:          company.CreatedAt,
			UpdatedAt:          company.UpdatedAt,
		}
	}
	return out
}

func CredentialSummaries(credentials []nfse.Credential) []CredentialSummary {
	out := make([]CredentialSummary, len(credentials))
	for i, credential := range credentials {
		out[i] = CredentialSummary{
			ID:                string(credential.ID),
			Label:             credential.Label,
			CertPath:          credential.CertPath,
			OwnerCNPJ:         credential.OwnerCNPJ,
			OwnerCNPJRoot:     credential.OwnerCNPJRoot,
			FingerprintSHA256: credential.FingerprintSHA256,
			SubjectName:       credential.SubjectName,
			NotBefore:         credential.NotBefore,
			NotAfter:          credential.NotAfter,
			InspectedAt:       credential.InspectedAt,
			CreatedAt:         credential.CreatedAt,
			UpdatedAt:         credential.UpdatedAt,
		}
	}
	return out
}

func DocumentRows(documents []nfse.CompanyDocument) []DocumentRow {
	out := make([]DocumentRow, len(documents))
	for i, document := range documents {
		out[i] = DocumentRow{
			ID:                 string(document.ID),
			ChaveAcesso:        string(document.ChaveAcesso),
			IssueDate:          document.IssueDate,
			Competence:         document.Competence,
			PrestadorCNPJ:      document.PrestadorCNPJ,
			PrestadorName:      document.PrestadorName,
			TomadorCNPJ:        document.TomadorCNPJ,
			TomadorName:        document.TomadorName,
			IntermediarioCNPJ:  document.IntermediarioCNPJ,
			IntermediarioName:  document.IntermediarioName,
			ServiceValue:       document.ServiceValue.Cents(),
			ISSValue:           document.ISSValue.Cents(),
			IRRFValue:          document.IRRFValue.Cents(),
			INSSValue:          document.INSSValue.Cents(),
			PISValue:           document.PISValue.Cents(),
			COFINSValue:        document.COFINSValue.Cents(),
			CSLLValue:          document.CSLLValue.Cents(),
			TotalRetentions:    document.TotalRetentions.Cents(),
			Status:             string(document.Status),
			LayoutVersion:      document.LayoutVersion,
			XMLPath:            document.XMLPath,
			RawHash:            document.RawHash,
			ParseWarnings:      document.ParseWarnings,
			NFSeNumber:         document.NFSeNumber,
			ServiceDescription: document.ServiceDescription,
			CreatedAt:          document.CreatedAt,
			UpdatedAt:          document.UpdatedAt,
			RelationID:         document.RelationID,
			CompanyID:          string(document.CompanyID),
			DocumentID:         string(document.DocumentID),
			CompanyRole:        string(document.CompanyRole),
			VisibilityReason:   string(document.VisibilityReason),
			FirstSeenNSU:       document.FirstSeenNSU,
			LastSeenNSU:        document.LastSeenNSU,
			FirstSyncedAt:      document.FirstSyncedAt,
			LastSyncedAt:       document.LastSyncedAt,
			ViewedAt:           document.ViewedAt,
		}
	}
	return out
}

func DocumentEvents(events []app.EventView) []DocumentEvent {
	out := make([]DocumentEvent, len(events))
	for i, event := range events {
		out[i] = DocumentEvent(event)
	}
	return out
}

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type ConnectionTestResult struct {
	CertLoaded        bool   `json:"certLoaded"`
	CertSubject       string `json:"certSubject"`
	CertExpiration    string `json:"certExpiration"`
	MTLSAccepted      bool   `json:"mtlsAccepted"`
	EndpointReached   bool   `json:"endpointReached"`
	ResponseCode      string `json:"responseCode"`
	ResponseDetail    string `json:"responseDetail"`
	StatusExplanation string `json:"statusExplanation"`
}

// --- NF-e ---
//
// Enum fields carry the app-layer strings unchanged (situação, completude,
// manifestação, papel, outcome), in Portuguese. Money is in cents. A deadline
// is nil when the NF-e has neither an authorization nor an issue date.

// ListNFeInput filters a company's NF-e. Empty fields do not filter.
type ListNFeInput struct {
	CNPJ         string
	Competence   string // "YYYY-MM" of the issue date
	Situacao     string // autorizada | denegada | cancelada
	Completeness string // resumo | completa
	Manifestacao string // nenhuma | ciencia | confirmada | desconhecida | nao_realizada
	Role         string // destinatario | emitente | transportador | autorizado | none
	EmitenteCNPJ string
	ChavesAcesso []string
}

type NFeRow struct {
	ID               string // company-document relation id
	DocumentID       string
	ChaveAcesso      string
	Serie            string
	Numero           string
	IssueDate        time.Time
	AuthorizedAt     *time.Time
	Protocolo        string
	TpNF             string // "0" entrada, "1" saída
	EmitenteCNPJ     string
	EmitenteName     string
	EmitenteIE       string
	DestinatarioCNPJ string
	DestinatarioName string
	TotalValue       int64
	Situacao         string
	Completeness     string
	Manifestacao     string
	ManifestacaoAt   *time.Time
	CienciaDue       *time.Time
	ConclusiveDue    *time.Time
	CompanyRole      string
	EventCount       int
	FirstSyncedAt    time.Time
	LastSyncedAt     time.Time
	// DaysLeft is how many calendar days are left until ConclusiveDue: 0 on
	// the due day, negative once it passed, nil without a deadline.
	DaysLeft *int
	// CienciaDaysLeft counts the same way until CienciaDue.
	CienciaDaysLeft *int
	// TacitlyConfirmed is true once ConclusiveDue passed without a
	// conclusive manifestação.
	TacitlyConfirmed bool
	// CienciaBlockReason and ConclusiveBlockReason say why the NF-e cannot
	// receive that manifestação; empty when it can.
	CienciaBlockReason    string
	ConclusiveBlockReason string
}

// NFeKeyInput identifies one of the company's NF-e.
type NFeKeyInput struct {
	CNPJ        string
	ChaveAcesso string
}

type NFeEvent struct {
	ID            string
	TpEvento      string // 110111, 110110, 210210, 210200, 210220, 210240, ...
	NSeqEvento    int
	Description   string
	EventAt       *time.Time
	RegisteredAt  *time.Time
	Protocolo     string
	CStat         string
	XMotivo       string
	Justificativa string
	Correcao      string
	AutorCNPJ     string
	Completeness  string
	Registered    bool
	SentByNanci   bool
}

type NFePendingInput struct {
	CNPJ string
	// DueWithinDays keeps rows whose conclusive deadline is at most this many
	// days away; 0 keeps every row.
	DueWithinDays int
}

// NFePendingRow is an authorized NF-e addressed to the company that still
// lacks a conclusive manifestação.
type NFePendingRow struct {
	NFeRow
	Kind           string // sem_ciencia | sem_conclusiva
	CienciaOverdue bool   // no manifestação and CienciaDue has passed
}

// RegisterNFeCienciaInput selects the NF-e for Ciência da Operação.
type RegisterNFeCienciaInput struct {
	CNPJ         string
	ChavesAcesso []string
}

// NFeSkipped is a requested chave that will not be sent, and why.
type NFeSkipped struct {
	ChaveAcesso string
	Reason      string
}

// NFeCienciaPlan is what RegisterNFeCiencia would send, for the confirmation
// dialog.
type NFeCienciaPlan struct {
	Eligible []NFeRow
	Skipped  []NFeSkipped
}

// RegisterNFeManifestacaoInput is one conclusive manifestação.
type RegisterNFeManifestacaoInput struct {
	CNPJ        string
	ChaveAcesso string
	// Tipo is confirmacao, desconhecimento or nao_realizada, or the tpEvento
	// code 210200, 210220 or 210240.
	Tipo string
	// Justificativa is required (15 to 255 characters) for nao_realizada.
	Justificativa string
}

type NFeEventResult struct {
	ChaveAcesso  string
	TpEvento     string
	Status       string // registrada | ja_registrada | rejeitada | nao_enviada
	CStat        string // empty when SEFAZ did not answer
	XMotivo      string
	Protocolo    string
	RegisteredAt *time.Time
}

// NFeEventBatchResult is the result of RegisterNFeCiencia: one result per
// eligible chave, in the order sent.
type NFeEventBatchResult struct {
	Results []NFeEventResult
	Skipped []NFeSkipped
	// Interrupted is the error that stopped the sending; the lotes after it
	// were not sent. Empty when every lote was sent.
	Interrupted string
}

type PullNFeInput struct {
	CNPJ string
}

type PullNFeResult struct {
	CompanyName      string
	CNPJ             string
	Status           string // completed | failed | interrupted
	StopReason       string // caught_up | consumo_indevido | rate_budget | ...
	LastNSU          int64
	MaxNSU           *int64 // nil when unknown
	CompletasSaved   int
	ResumosSaved     int
	EventsSaved      int
	Errors           int
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
	Duration         time.Duration
}

type NFeStatusResult struct {
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
	NextAllowedAt     *time.Time
	BlockedReason     string // caught_up | consumo_indevido | rate_budget; empty when not blocked
	RequestsLastHour  int
	RequestBudget     int
	TotalDestinatario int
	TotalEmitente     int
	TotalOutros       int
	TotalResumos      int
	TotalCompletas    int
	PendingCiencia    int
	PendingConclusiva int
	CienciaOverdue    int
}

// NFeResetResult is what ResetNFe removed for one company. The manifestações
// sent stay as the audit trail.
type NFeResetResult struct {
	CompanyName       string
	CNPJ              string
	CompanyDocuments  int // the company's notes
	Documents         int // notes no other company sees
	Events            int
	ExportMarks       int
	ManifestacoesKept int
}

type ExportNFeXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// ExportNFeZIPInput selects the NF-e for the XML ZIP. The export filters by
// competência, papel and chaves only.
type ExportNFeZIPInput struct {
	CNPJ           string
	Competence     string
	Role           string
	ChavesAcesso   []string
	IncludeResumos bool
	Incremental    bool
	OutPath        string
}

type NFeExportResult struct {
	ExportResult
	SkippedResumos int
}

func NFeRows(documents []app.NFeDocument) []NFeRow {
	out := make([]NFeRow, len(documents))
	for i, document := range documents {
		out[i] = nfeRow(document)
	}
	return out
}

func nfeRow(document app.NFeDocument) NFeRow {
	row := NFeRow{
		ID:                    document.RelationID,
		DocumentID:            document.ID,
		ChaveAcesso:           string(document.ChaveAcesso),
		Serie:                 document.Serie,
		Numero:                document.Numero,
		IssueDate:             document.IssueDate,
		AuthorizedAt:          document.AuthorizedAt,
		Protocolo:             document.Protocolo,
		TpNF:                  document.TpNF,
		EmitenteCNPJ:          document.EmitenteCNPJ,
		EmitenteName:          document.EmitenteName,
		EmitenteIE:            document.EmitenteIE,
		DestinatarioCNPJ:      document.DestinatarioCNPJ,
		DestinatarioName:      document.DestinatarioName,
		TotalValue:            document.TotalValue.Cents(),
		Situacao:              string(document.Situacao),
		Completeness:          string(document.Completeness),
		Manifestacao:          string(document.Manifestacao),
		ManifestacaoAt:        document.ManifestacaoAt,
		CienciaDue:            optionalTime(document.CienciaDue),
		ConclusiveDue:         optionalTime(document.ConclusiveDue),
		CompanyRole:           string(document.CompanyRole),
		EventCount:            document.EventCount,
		FirstSyncedAt:         document.FirstSyncedAt,
		LastSyncedAt:          document.LastSyncedAt,
		TacitlyConfirmed:      document.TacitlyConfirmed,
		CienciaBlockReason:    document.CienciaBlockReason,
		ConclusiveBlockReason: document.ConclusiveBlockReason,
	}
	if !document.ConclusiveDue.IsZero() {
		row.DaysLeft = &document.DaysLeft
	}
	if !document.CienciaDue.IsZero() {
		row.CienciaDaysLeft = &document.CienciaDaysLeft
	}
	return row
}

func NFeEvents(events []nfe.Event) []NFeEvent {
	out := make([]NFeEvent, len(events))
	for i, event := range events {
		out[i] = NFeEvent{
			ID:            event.ID,
			TpEvento:      event.TpEvento,
			NSeqEvento:    event.NSeqEvento,
			Description:   event.Description,
			EventAt:       event.EventAt,
			RegisteredAt:  event.RegisteredAt,
			Protocolo:     event.Protocolo,
			CStat:         event.CStat,
			XMotivo:       event.XMotivo,
			Justificativa: event.Justificativa,
			Correcao:      event.Correcao,
			AutorCNPJ:     event.AutorCNPJ,
			Completeness:  string(event.Completeness),
			Registered:    event.Registered,
			SentByNanci:   event.SentByNanci,
		}
	}
	return out
}

func NFePendingRows(pending []app.NFePendingManifestacao) []NFePendingRow {
	out := make([]NFePendingRow, len(pending))
	for i, p := range pending {
		out[i] = NFePendingRow{
			NFeRow:         nfeRow(p.NFeDocument),
			Kind:           p.Kind,
			CienciaOverdue: p.CienciaOverdue,
		}
	}
	return out
}

func NFeCienciaPlanFrom(plan app.NFeCienciaPlan) NFeCienciaPlan {
	return NFeCienciaPlan{
		Eligible: NFeRows(plan.Eligible),
		Skipped:  nfeSkipped(plan.Skipped),
	}
}

func NFeEventResults(summary app.NFeManifestacaoSummary) NFeEventBatchResult {
	results := make([]NFeEventResult, len(summary.Outcomes))
	for i, outcome := range summary.Outcomes {
		results[i] = NFeEventResultFrom(outcome)
	}
	return NFeEventBatchResult{
		Results:     results,
		Skipped:     nfeSkipped(summary.Skipped),
		Interrupted: summary.Interrupted,
	}
}

func NFeEventResultFrom(outcome app.NFeEventOutcome) NFeEventResult {
	return NFeEventResult{
		ChaveAcesso:  outcome.ChaveAcesso,
		TpEvento:     outcome.TpEvento,
		Status:       outcome.Status,
		CStat:        outcome.CStat,
		XMotivo:      outcome.XMotivo,
		Protocolo:    outcome.Protocolo,
		RegisteredAt: outcome.RegisteredAt,
	}
}

func nfeSkipped(skipped []app.NFeSkipped) []NFeSkipped {
	out := make([]NFeSkipped, len(skipped))
	for i, s := range skipped {
		out[i] = NFeSkipped(s)
	}
	return out
}

// --- CT-e ---
//
// Enum fields carry the domain strings unchanged (situação, papel, tipo de
// documento, tipo de evento), in Portuguese. TpCTe, TpServ and Modal are the
// codes as written in the XML. Money is in cents.

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

// CTeMunicipio is where the transport starts or ends.
type CTeMunicipio struct {
	Codigo string // IBGE code
	Nome   string
	UF     string
}

type CTeRow struct {
	ID            string // company-document relation id
	DocumentID    string
	ChaveAcesso   string
	TpAmb         string // "1" produção, "2" homologação
	Modelo        string // "57", "64" or "67"
	TipoDocumento string // cte | cte_os | gtve | cte_simplificado
	Serie         string
	Numero        string
	CFOP          string
	NatOp         string
	IssueDate     time.Time
	Competence    string
	AuthorizedAt  *time.Time
	Protocolo     string
	TpCTe         string
	TpServ        string
	Modal         string
	MunIni        CTeMunicipio
	MunFim        CTeMunicipio

	EmitenteCNPJ     string
	EmitenteName     string
	RemetenteCNPJ    string
	RemetenteName    string
	DestinatarioCNPJ string
	DestinatarioName string
	ExpedidorCNPJ    string
	ExpedidorName    string
	RecebedorCNPJ    string
	RecebedorName    string
	TomadorCNPJ      string
	TomadorName      string
	TomadorIE        string
	TomadorUF        string
	// TomadorIndicador is the raw toma code; empty on a CT-e OS.
	TomadorIndicador string

	TotalValue          int64
	ReceivableValue     int64
	ICMSValue           int64
	TotTribValue        int64
	CargaValue          int64
	ProdutoPredominante string
	// NFeChaves are the NF-e the CT-e transported, in document order.
	NFeChaves []string

	Situacao         string
	CompanyRole      string
	Papeis           []string // every role the company plays, primary first
	VisibilityReason string
	EventCount       int
	FirstSeenNSU     *int64
	LastSeenNSU      *int64
	FirstSyncedAt    time.Time
	LastSyncedAt     time.Time
	LayoutVersion    string
	ParseWarnings    []string
}

// CTeKeyInput identifies one of the company's CT-e.
type CTeKeyInput struct {
	CNPJ        string
	ChaveAcesso string
}

type CTeEvent struct {
	ID            string
	TpEvento      string // 110111, 110110, 110180, 610110, 310610, ...
	Type          string // cancelamento | carta_correcao | comprovante_entrega | ... | unknown
	NSeqEvento    int
	Description   string
	EventAt       *time.Time
	RegisteredAt  *time.Time
	Protocolo     string
	CStat         string
	XMotivo       string
	AutorCNPJ     string
	Justificativa string
	Observacao    string
	Correcao      string
	Registered    bool
}

type PullCTeInput struct {
	CNPJ string
}

type PullCTeResult struct {
	CompanyName      string
	CNPJ             string
	Status           string // completed | failed | interrupted
	StopReason       string // caught_up | consumo_indevido | rate_budget | ...
	LastNSU          int64
	MaxNSU           *int64 // nil when unknown
	DocumentsSaved   int
	EventsSaved      int
	Errors           int
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
	Duration         time.Duration
}

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
	NextAllowedAt     *time.Time
	BlockedReason     string // caught_up | consumo_indevido | rate_budget; empty when not blocked
	RequestsLastHour  int
	RequestBudget     int
	TotalTomador      int
	TotalDestinatario int
	TotalRemetente    int
	TotalOutros       int // expedidor, recebedor, emitente, autorizado and none
}

// CTeResetResult is what ResetCTe removed, or PreviewResetCTe would remove,
// for one company.
type CTeResetResult struct {
	CompanyName      string
	CNPJ             string
	CompanyDocuments int // the company's CT-e
	Documents        int // CT-e no other company sees
	Events           int
	ExportMarks      int
}

type ExportCTeXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// ExportCTeZIPInput selects the CT-e for the XML ZIP. The export filters by
// competência, papel and chaves only.
type ExportCTeZIPInput struct {
	CNPJ         string
	Competence   string
	Role         string
	ChavesAcesso []string
	Incremental  bool
	OutPath      string
}

func CTeRows(documents []cte.CompanyDocument) []CTeRow {
	out := make([]CTeRow, len(documents))
	for i, document := range documents {
		out[i] = cteRow(document)
	}
	return out
}

func cteRow(document cte.CompanyDocument) CTeRow {
	papeis := make([]string, len(document.Papeis))
	for i, papel := range document.Papeis {
		papeis[i] = string(papel)
	}
	return CTeRow{
		ID:                  document.RelationID,
		DocumentID:          document.ID,
		ChaveAcesso:         string(document.ChaveAcesso),
		TpAmb:               document.TpAmb,
		Modelo:              document.Modelo,
		TipoDocumento:       string(document.TipoDocumento),
		Serie:               document.Serie,
		Numero:              document.Numero,
		CFOP:                document.CFOP,
		NatOp:               document.NatOp,
		IssueDate:           document.IssueDate,
		Competence:          document.Competence,
		AuthorizedAt:        document.AuthorizedAt,
		Protocolo:           document.Protocolo,
		TpCTe:               document.TpCTe,
		TpServ:              document.TpServ,
		Modal:               document.Modal,
		MunIni:              CTeMunicipio(document.MunIni),
		MunFim:              CTeMunicipio(document.MunFim),
		EmitenteCNPJ:        document.Emitente.CNPJ,
		EmitenteName:        document.Emitente.Name,
		RemetenteCNPJ:       document.Remetente.CNPJ,
		RemetenteName:       document.Remetente.Name,
		DestinatarioCNPJ:    document.Destinatario.CNPJ,
		DestinatarioName:    document.Destinatario.Name,
		ExpedidorCNPJ:       document.Expedidor.CNPJ,
		ExpedidorName:       document.Expedidor.Name,
		RecebedorCNPJ:       document.Recebedor.CNPJ,
		RecebedorName:       document.Recebedor.Name,
		TomadorCNPJ:         document.Tomador.CNPJ,
		TomadorName:         document.Tomador.Name,
		TomadorIE:           document.Tomador.IE,
		TomadorUF:           document.Tomador.UF,
		TomadorIndicador:    document.TomadorIndicador,
		TotalValue:          document.TotalValue.Cents(),
		ReceivableValue:     document.ReceivableValue.Cents(),
		ICMSValue:           document.ICMSValue.Cents(),
		TotTribValue:        document.TotTribValue.Cents(),
		CargaValue:          document.CargaValue.Cents(),
		ProdutoPredominante: document.ProdutoPredominante,
		// Empty lists reach the frontend as [] instead of null.
		NFeChaves:        append([]string{}, document.NFeChaves...),
		Situacao:         string(document.Situacao),
		CompanyRole:      string(document.CompanyRole),
		Papeis:           papeis,
		VisibilityReason: string(document.VisibilityReason),
		EventCount:       document.EventCount,
		FirstSeenNSU:     document.FirstSeenNSU,
		LastSeenNSU:      document.LastSeenNSU,
		FirstSyncedAt:    document.FirstSyncedAt,
		LastSyncedAt:     document.LastSyncedAt,
		LayoutVersion:    document.LayoutVersion,
		ParseWarnings:    append([]string{}, document.ParseWarnings...),
	}
}

func CTeEvents(events []cte.Event) []CTeEvent {
	out := make([]CTeEvent, len(events))
	for i, event := range events {
		out[i] = CTeEvent{
			ID:            event.ID,
			TpEvento:      event.TpEvento,
			Type:          string(event.Type),
			NSeqEvento:    event.NSeqEvento,
			Description:   event.Description,
			EventAt:       event.EventAt,
			RegisteredAt:  event.RegisteredAt,
			Protocolo:     event.Protocolo,
			CStat:         event.CStat,
			XMotivo:       event.XMotivo,
			AutorCNPJ:     event.AutorCNPJ,
			Justificativa: event.Justificativa,
			Observacao:    event.Observacao,
			Correcao:      event.Correcao,
			Registered:    event.Registered,
		}
	}
	return out
}

func CTeResetResultFrom(res app.CTeResetResult) CTeResetResult {
	return CTeResetResult{
		CompanyName:      res.CompanyName,
		CNPJ:             res.CNPJ,
		CompanyDocuments: res.CompanyDocuments,
		Documents:        res.Documents,
		Events:           res.Events,
		ExportMarks:      res.ExportMarks,
	}
}

// optionalTime returns nil for the zero time.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
