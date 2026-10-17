// CT-e DTOs and their mappers. Enum fields carry the domain strings unchanged
// (situação, papel, tipo de documento, tipo de evento), in Portuguese. TpCTe,
// TpServ and Modal are the codes as written in the XML. Money is in cents.

package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
)

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
	OnlyUnread   bool
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
	HasDACTE      bool   // false for the kinds that have no DACTE
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
	ViewedAt         *time.Time // nil while the CT-e is new
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
	IdleDays          int // days since the last successful distribution query when it reached sync.DistIdleWarningDays; 0 otherwise
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

// ExportCTeDocumentsInput selects the CT-e of a list export and its Format:
// csv, xlsx, zip (the XMLs) or dacte-zip. The export filters by competência,
// papel and chaves only.
type ExportCTeDocumentsInput struct {
	CNPJ         string
	Competence   string
	Role         string
	Format       string
	ChavesAcesso []string
	Incremental  bool
	OutPath      string
}

// ExportDACTEInput names the CT-e whose DACTE is written to OutPath.
type ExportDACTEInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// CTeExportResult is ExportResult plus the CT-e a DACTE export left out
// because their kind has no DACTE.
type CTeExportResult struct {
	ExportResult
	SkippedSemDACTE int
}

func CTeExportResultFrom(res app.CTeExportResult) CTeExportResult {
	return CTeExportResult{
		ExportResult:    ExportResult(res.ExportResult),
		SkippedSemDACTE: res.SkippedSemDACTE,
	}
}

func CTeRows(documents []cte.CompanyDocument) []CTeRow {
	out := make([]CTeRow, len(documents))
	for i, document := range documents {
		out[i] = cteRow(document)
	}
	return out
}

func cteRow(document cte.CompanyDocument) CTeRow {
	return CTeRow{
		ID:                  document.RelationID,
		DocumentID:          document.ID,
		ChaveAcesso:         string(document.ChaveAcesso),
		TpAmb:               document.TpAmb,
		Modelo:              document.Modelo,
		TipoDocumento:       string(document.TipoDocumento),
		HasDACTE:            document.HasDACTE(),
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
		Papeis:           cte.PapeisStrings(document.Papeis),
		VisibilityReason: string(document.VisibilityReason),
		EventCount:       document.EventCount,
		FirstSeenNSU:     document.FirstSeenNSU,
		LastSeenNSU:      document.LastSeenNSU,
		FirstSyncedAt:    document.FirstSyncedAt,
		LastSyncedAt:     document.LastSyncedAt,
		ViewedAt:         document.ViewedAt,
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
