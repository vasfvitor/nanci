// NF-e DTOs and their mappers. Enum fields carry the app-layer strings
// unchanged (situação, completude, manifestação, papel, outcome), in
// Portuguese. Money is in cents. A deadline is nil when the NF-e has neither
// an authorization nor an issue date.

package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfe"
)

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
	OnlyUnread   bool
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
	TpAmb            string // "1" produção, "2" homologação
	NatOp            string // empty on resumo
	EmitenteCNPJ     string
	EmitenteName     string
	EmitenteIE       string
	EmitenteUF       string
	DestinatarioCNPJ string
	DestinatarioName string
	TotalValue       int64
	ICMSValue        int64 // zero on resumo
	IPIValue         int64 // zero on resumo
	Situacao         string
	Completeness     string
	LayoutVersion    string
	ParseWarnings    []string
	Manifestacao     string
	ManifestacaoAt   *time.Time
	CienciaDue       *time.Time
	ConclusiveDue    *time.Time
	CompanyRole      string
	EventCount       int
	FirstSeenNSU     *int64 // nil for an NF-e that came by import, not by the distribution
	FirstSyncedAt    time.Time
	LastSyncedAt     time.Time
	ViewedAt         *time.Time // nil while the NF-e is new
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
	LastRunError      string // message of the last run, set only when it failed
	InitialSyncDoneAt *time.Time
	NextAllowedAt     *time.Time
	BlockedReason     string // caught_up | consumo_indevido | rate_budget; empty when not blocked
	RequestsLastHour  int
	RequestBudget     int
	IdleDays          int // days since the last successful distribution query when it reached sync.DistIdleWarningDays; 0 otherwise
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

// ExportNFeDocumentsInput selects the NF-e of a list export and its Format:
// csv, xlsx, zip (the XMLs) or danfe-zip. The export filters by competência,
// papel and chaves only.
type ExportNFeDocumentsInput struct {
	CNPJ           string
	Competence     string
	Role           string
	Format         string
	ChavesAcesso   []string
	IncludeResumos bool
	Incremental    bool
	OutPath        string
}

// ExportDANFEInput names the NF-e whose DANFE is written to OutPath.
type ExportDANFEInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
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
		TpAmb:                 document.TpAmb,
		NatOp:                 document.NatOp,
		EmitenteCNPJ:          document.EmitenteCNPJ,
		EmitenteName:          document.EmitenteName,
		EmitenteIE:            document.EmitenteIE,
		EmitenteUF:            document.EmitenteUF,
		DestinatarioCNPJ:      document.DestinatarioCNPJ,
		DestinatarioName:      document.DestinatarioName,
		TotalValue:            document.TotalValue.Cents(),
		ICMSValue:             document.ICMSValue.Cents(),
		IPIValue:              document.IPIValue.Cents(),
		Situacao:              string(document.Situacao),
		Completeness:          string(document.Completeness),
		LayoutVersion:         document.LayoutVersion,
		Manifestacao:          string(document.Manifestacao),
		ManifestacaoAt:        document.ManifestacaoAt,
		CienciaDue:            optionalTime(document.CienciaDue),
		ConclusiveDue:         optionalTime(document.ConclusiveDue),
		CompanyRole:           string(document.CompanyRole),
		EventCount:            document.EventCount,
		FirstSeenNSU:          document.FirstSeenNSU,
		FirstSyncedAt:         document.FirstSyncedAt,
		LastSyncedAt:          document.LastSyncedAt,
		ViewedAt:              document.ViewedAt,
		TacitlyConfirmed:      document.TacitlyConfirmed,
		CienciaBlockReason:    document.CienciaBlockReason,
		ConclusiveBlockReason: document.ConclusiveBlockReason,
		// Empty lists reach the frontend as [] instead of null.
		ParseWarnings: append([]string{}, document.ParseWarnings...),
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
