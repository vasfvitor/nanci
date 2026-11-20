package cli

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/sync"
)

// The types below are what --json prints. They keep the Go field names, as
// the desktop DTOs do, carry money as int64 cents, optional times as
// *time.Time (null when unknown) and enums as strings. They leave out
// internal fields: hashes, row IDs and file paths of the local store.

// optionalTime is nil for the zero time, so JSON shows null instead of
// 0001-01-01.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// optionalDaysLeft is days, or nil when there is no deadline to count to.
func optionalDaysLeft(due time.Time, days int) *int {
	if due.IsZero() {
		return nil
	}
	return &days
}

// optionalIdleDays is days, or nil when it is 0: the app sets it only when
// the idle warning applies.
func optionalIdleDays(days int) *int {
	if days == 0 {
		return nil
	}
	return &days
}

// nonNilStrings keeps a list as [] in JSON when it is empty.
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// nfseDocumentOutput is one row of `list`.
type nfseDocumentOutput struct {
	ChaveAcesso        string
	NFSeNumber         string
	IssueDate          *time.Time
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
	ServiceDescription string
	CompanyRole        string
	VisibilityReason   string
	ParseWarnings      []string
	FirstSyncedAt      time.Time
	LastSyncedAt       time.Time
	ViewedAt           *time.Time
}

func newNFSeDocumentOutput(d nfse.CompanyDocument) nfseDocumentOutput {
	return nfseDocumentOutput{
		ChaveAcesso:        string(d.ChaveAcesso),
		NFSeNumber:         d.NFSeNumber,
		IssueDate:          optionalTime(d.IssueDate),
		Competence:         d.Competence,
		PrestadorCNPJ:      d.PrestadorCNPJ,
		PrestadorName:      d.PrestadorName,
		TomadorCNPJ:        d.TomadorCNPJ,
		TomadorName:        d.TomadorName,
		IntermediarioCNPJ:  d.IntermediarioCNPJ,
		IntermediarioName:  d.IntermediarioName,
		ServiceValue:       d.ServiceValue.Cents(),
		ISSValue:           d.ISSValue.Cents(),
		IRRFValue:          d.IRRFValue.Cents(),
		INSSValue:          d.INSSValue.Cents(),
		PISValue:           d.PISValue.Cents(),
		COFINSValue:        d.COFINSValue.Cents(),
		CSLLValue:          d.CSLLValue.Cents(),
		TotalRetentions:    d.TotalRetentions.Cents(),
		Status:             string(d.Status),
		ServiceDescription: d.ServiceDescription,
		CompanyRole:        string(d.CompanyRole),
		VisibilityReason:   string(d.VisibilityReason),
		ParseWarnings:      nonNilStrings(d.ParseWarnings),
		FirstSyncedAt:      d.FirstSyncedAt,
		LastSyncedAt:       d.LastSyncedAt,
		ViewedAt:           d.ViewedAt,
	}
}

// nfeDocumentOutput is one NF-e of `nfe list`, with its manifestação
// deadlines. DaysLeft and CienciaDaysLeft are null when there is no
// deadline.
type nfeDocumentOutput struct {
	ChaveAcesso           string
	Modelo                string
	Serie                 string
	Numero                string
	IssueDate             *time.Time
	Competence            string
	AuthorizedAt          *time.Time
	Protocolo             string
	TpNF                  string
	TpAmb                 string
	FinNFe                string
	NatOp                 string
	EmitenteCNPJ          string
	EmitenteName          string
	EmitenteIE            string
	EmitenteUF            string
	DestinatarioCNPJ      string
	DestinatarioName      string
	TransportadorCNPJ     string
	AutorizadosCNPJ       []string
	TotalValue            int64
	ICMSValue             int64
	IPIValue              int64
	Situacao              string
	Completeness          string
	ParseWarnings         []string
	CompanyRole           string
	VisibilityReason      string
	Manifestacao          string
	ManifestacaoAt        *time.Time
	CienciaDue            *time.Time
	ConclusiveDue         *time.Time
	DaysLeft              *int
	CienciaDaysLeft       *int
	TacitlyConfirmed      bool
	CienciaBlockReason    string
	ConclusiveBlockReason string
	EventCount            int
	FirstSyncedAt         time.Time
	LastSyncedAt          time.Time
	ViewedAt              *time.Time
}

func newNFeDocumentOutput(d app.NFeDocument) nfeDocumentOutput {
	return nfeDocumentOutput{
		ChaveAcesso:           string(d.ChaveAcesso),
		Modelo:                d.Modelo,
		Serie:                 d.Serie,
		Numero:                d.Numero,
		IssueDate:             optionalTime(d.IssueDate),
		Competence:            d.Competence,
		AuthorizedAt:          d.AuthorizedAt,
		Protocolo:             d.Protocolo,
		TpNF:                  d.TpNF,
		TpAmb:                 d.TpAmb,
		FinNFe:                d.FinNFe,
		NatOp:                 d.NatOp,
		EmitenteCNPJ:          d.EmitenteCNPJ,
		EmitenteName:          d.EmitenteName,
		EmitenteIE:            d.EmitenteIE,
		EmitenteUF:            d.EmitenteUF,
		DestinatarioCNPJ:      d.DestinatarioCNPJ,
		DestinatarioName:      d.DestinatarioName,
		TransportadorCNPJ:     d.TransportadorCNPJ,
		AutorizadosCNPJ:       nonNilStrings(d.AutorizadosCNPJ),
		TotalValue:            d.TotalValue.Cents(),
		ICMSValue:             d.ICMSValue.Cents(),
		IPIValue:              d.IPIValue.Cents(),
		Situacao:              string(d.Situacao),
		Completeness:          string(d.Completeness),
		ParseWarnings:         nonNilStrings(d.ParseWarnings),
		CompanyRole:           string(d.CompanyRole),
		VisibilityReason:      string(d.VisibilityReason),
		Manifestacao:          string(d.Manifestacao),
		ManifestacaoAt:        d.ManifestacaoAt,
		CienciaDue:            optionalTime(d.CienciaDue),
		ConclusiveDue:         optionalTime(d.ConclusiveDue),
		DaysLeft:              optionalDaysLeft(d.ConclusiveDue, d.DaysLeft),
		CienciaDaysLeft:       optionalDaysLeft(d.CienciaDue, d.CienciaDaysLeft),
		TacitlyConfirmed:      d.TacitlyConfirmed,
		CienciaBlockReason:    d.CienciaBlockReason,
		ConclusiveBlockReason: d.ConclusiveBlockReason,
		EventCount:            d.EventCount,
		FirstSyncedAt:         d.FirstSyncedAt,
		LastSyncedAt:          d.LastSyncedAt,
		ViewedAt:              d.ViewedAt,
	}
}

// nfePendingOutput is one row of `nfe pendentes`: the NF-e plus why it is
// pending.
type nfePendingOutput struct {
	nfeDocumentOutput
	Kind           string // sem_ciencia | sem_conclusiva
	CienciaOverdue bool
}

func newNFePendingOutput(p app.NFePendingManifestacao) nfePendingOutput {
	return nfePendingOutput{
		nfeDocumentOutput: newNFeDocumentOutput(p.NFeDocument),
		Kind:              p.Kind,
		CienciaOverdue:    p.CienciaOverdue,
	}
}

// ctePartyOutput is one party named in a CT-e.
type ctePartyOutput struct {
	CNPJ string
	Name string
	IE   string
	UF   string
}

func newCTePartyOutput(p cte.Party) ctePartyOutput {
	return ctePartyOutput{CNPJ: p.CNPJ, Name: p.Name, IE: p.IE, UF: p.UF}
}

// cteMunicipioOutput is where a transport starts or ends.
type cteMunicipioOutput struct {
	Codigo string
	Nome   string
	UF     string
}

func newCTeMunicipioOutput(m cte.Municipio) cteMunicipioOutput {
	return cteMunicipioOutput{Codigo: m.Codigo, Nome: m.Nome, UF: m.UF}
}

// cteDocumentOutput is one CT-e of `cte list`. Papeis are all the roles the
// company plays, CompanyRole the first one.
type cteDocumentOutput struct {
	ChaveAcesso         string
	Modelo              string
	TipoDocumento       string
	Serie               string
	Numero              string
	CFOP                string
	NatOp               string
	IssueDate           *time.Time
	Competence          string
	AuthorizedAt        *time.Time
	Protocolo           string
	TpAmb               string
	TpCTe               string
	TpServ              string
	Modal               string
	MunIni              cteMunicipioOutput
	MunFim              cteMunicipioOutput
	Emitente            ctePartyOutput
	Remetente           ctePartyOutput
	Destinatario        ctePartyOutput
	Expedidor           ctePartyOutput
	Recebedor           ctePartyOutput
	Tomador             ctePartyOutput
	TomadorIndicador    string
	AutorizadosCNPJ     []string
	TotalValue          int64
	ReceivableValue     int64
	ICMSValue           int64
	TotTribValue        int64
	CargaValue          int64
	ProdutoPredominante string
	NFeChaves           []string
	MaskedKeys          bool
	Situacao            string
	ParseWarnings       []string
	CompanyRole         string
	Papeis              []string
	VisibilityReason    string
	EventCount          int
	FirstSyncedAt       time.Time
	LastSyncedAt        time.Time
	ViewedAt            *time.Time
}

func newCTeDocumentOutput(d cte.CompanyDocument) cteDocumentOutput {
	return cteDocumentOutput{
		ChaveAcesso:         string(d.ChaveAcesso),
		Modelo:              d.Modelo,
		TipoDocumento:       string(d.TipoDocumento),
		Serie:               d.Serie,
		Numero:              d.Numero,
		CFOP:                d.CFOP,
		NatOp:               d.NatOp,
		IssueDate:           optionalTime(d.IssueDate),
		Competence:          d.Competence,
		AuthorizedAt:        d.AuthorizedAt,
		Protocolo:           d.Protocolo,
		TpAmb:               d.TpAmb,
		TpCTe:               d.TpCTe,
		TpServ:              d.TpServ,
		Modal:               d.Modal,
		MunIni:              newCTeMunicipioOutput(d.MunIni),
		MunFim:              newCTeMunicipioOutput(d.MunFim),
		Emitente:            newCTePartyOutput(d.Emitente),
		Remetente:           newCTePartyOutput(d.Remetente),
		Destinatario:        newCTePartyOutput(d.Destinatario),
		Expedidor:           newCTePartyOutput(d.Expedidor),
		Recebedor:           newCTePartyOutput(d.Recebedor),
		Tomador:             newCTePartyOutput(d.Tomador),
		TomadorIndicador:    d.TomadorIndicador,
		AutorizadosCNPJ:     nonNilStrings(d.AutorizadosCNPJ),
		TotalValue:          d.TotalValue.Cents(),
		ReceivableValue:     d.ReceivableValue.Cents(),
		ICMSValue:           d.ICMSValue.Cents(),
		TotTribValue:        d.TotTribValue.Cents(),
		CargaValue:          d.CargaValue.Cents(),
		ProdutoPredominante: d.ProdutoPredominante,
		NFeChaves:           nonNilStrings(d.NFeChaves),
		MaskedKeys:          d.MaskedKeys,
		Situacao:            string(d.Situacao),
		ParseWarnings:       nonNilStrings(d.ParseWarnings),
		CompanyRole:         string(d.CompanyRole),
		Papeis:              nonNilStrings(cte.PapeisStrings(d.Papeis)),
		VisibilityReason:    string(d.VisibilityReason),
		EventCount:          d.EventCount,
		FirstSyncedAt:       d.FirstSyncedAt,
		LastSyncedAt:        d.LastSyncedAt,
		ViewedAt:            d.ViewedAt,
	}
}

// nfseStatusOutput is `status`, the NFS-e sync state of a company.
type nfseStatusOutput struct {
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
	NextAllowedAt      *time.Time
	BlockedReason      string
	TotalEmitidas      int64
	TotalTomadas       int64
}

func newNFSeStatusOutput(r sync.StatusResult) nfseStatusOutput {
	return nfseStatusOutput{
		CompanyName:        r.CompanyName,
		CNPJ:               r.CNPJ,
		Environment:        r.Environment,
		ConsultationCNPJ:   r.ConsultationCNPJ,
		CredentialCNPJ:     r.CredentialCNPJ,
		CredentialNotAfter: r.CredentialNotAfter,
		LastProcessedNSU:   r.LastProcessedNSU,
		LastFoundNSU:       r.LastFoundNSU,
		LastSyncAt:         r.LastSyncAt,
		LastRunStatus:      r.LastRunStatus,
		LastRunStopReason:  r.LastRunStopReason,
		NextAllowedAt:      r.NextAllowedAt,
		BlockedReason:      r.BlockedReason,
		TotalEmitidas:      r.TotalEmitidas,
		TotalTomadas:       r.TotalTomadas,
	}
}

// sefazSourceStatusOutput is the part of `nfe status` and `cte status` that
// both SEFAZ sources share. MaxNSU is null when unknown; IdleDays is null
// unless the idle warning applies.
type sefazSourceStatusOutput struct {
	CompanyName       string
	CNPJ              string
	UF                string
	TpAmb             string
	LastNSU           int64
	MaxNSU            *int64
	LastSyncAt        *time.Time
	LastRunStatus     string
	LastRunStopReason string
	LastRunError      string
	InitialSyncDoneAt *time.Time
	NextAllowedAt     *time.Time
	BlockedReason     string
	RequestsLastHour  int
	RequestBudget     int
	IdleDays          *int
	NSUFaltantes      int64
	RecoveryAllowance int
	RecoveryReadyAt   *time.Time
}

func newSefazSourceStatusOutput(s app.SefazSourceStatus) sefazSourceStatusOutput {
	return sefazSourceStatusOutput{
		CompanyName:       s.CompanyName,
		CNPJ:              s.CNPJ,
		UF:                s.UF,
		TpAmb:             s.TpAmb,
		LastNSU:           s.LastNSU,
		MaxNSU:            s.MaxNSU,
		LastSyncAt:        s.LastSyncAt,
		LastRunStatus:     s.LastRunStatus,
		LastRunStopReason: s.LastRunStopReason,
		LastRunError:      s.LastRunError,
		InitialSyncDoneAt: s.InitialSyncDoneAt,
		NextAllowedAt:     s.NextAllowedAt,
		BlockedReason:     s.BlockedReason,
		RequestsLastHour:  s.RequestsLastHour,
		RequestBudget:     s.RequestBudget,
		IdleDays:          optionalIdleDays(s.IdleDays),
		NSUFaltantes:      s.NSUFaltantes,
		RecoveryAllowance: s.RecoveryAllowance,
		RecoveryReadyAt:   s.RecoveryReadyAt,
	}
}

// nfeStatusOutput is `nfe status`.
type nfeStatusOutput struct {
	sefazSourceStatusOutput
	TotalDestinatario int
	TotalEmitente     int
	TotalOutros       int
	TotalResumos      int
	TotalCompletas    int
	PendingCiencia    int
	PendingConclusiva int
	CienciaOverdue    int
}

func newNFeStatusOutput(r app.NFeStatusResult) nfeStatusOutput {
	return nfeStatusOutput{
		sefazSourceStatusOutput: newSefazSourceStatusOutput(r.SefazSourceStatus),
		TotalDestinatario:       r.TotalDestinatario,
		TotalEmitente:           r.TotalEmitente,
		TotalOutros:             r.TotalOutros,
		TotalResumos:            r.TotalResumos,
		TotalCompletas:          r.TotalCompletas,
		PendingCiencia:          r.PendingCiencia,
		PendingConclusiva:       r.PendingConclusiva,
		CienciaOverdue:          r.CienciaOverdue,
	}
}

// cteStatusOutput is `cte status`, with the totals by primary role.
type cteStatusOutput struct {
	sefazSourceStatusOutput
	TotalTomador      int
	TotalDestinatario int
	TotalRemetente    int
	TotalOutros       int
}

func newCTeStatusOutput(r app.CTeStatusResult) cteStatusOutput {
	return cteStatusOutput{
		sefazSourceStatusOutput: newSefazSourceStatusOutput(r.SefazSourceStatus),
		TotalTomador:            r.TotalTomador,
		TotalDestinatario:       r.TotalDestinatario,
		TotalRemetente:          r.TotalRemetente,
		TotalOutros:             r.TotalOutros,
	}
}

// companyOutput is one row of `company list`. CredentialID is the value
// `company assign-credential --credential-id` takes.
type companyOutput struct {
	CNPJ            string
	Name            string
	CredentialID    string
	CredentialLabel string
	Environment     string
	UF              string
	SyncStartPolicy string
	SyncStartDate   *time.Time
	// SyncStartPolicyLocked is true while the NFS-e has a sync cursor: the
	// start policy and date only change after `nanci reset --confirmar`.
	SyncStartPolicyLocked bool
	LastSyncAt            *time.Time
}

func newCompanyOutput(c company.Company) companyOutput {
	return companyOutput{
		CNPJ:                  c.CNPJ,
		Name:                  c.Name,
		CredentialID:          string(c.CredentialID),
		CredentialLabel:       c.CredentialLabel,
		Environment:           string(c.Environment),
		UF:                    c.UF,
		SyncStartPolicy:       string(c.SyncStartPolicy),
		SyncStartDate:         c.SyncStartDate,
		SyncStartPolicyLocked: c.SyncStartPolicyLocked,
		LastSyncAt:            c.LastSyncAt,
	}
}

// initOutput is what `init` prints: the folder that holds the database.
type initOutput struct {
	DataDir string
}

// credentialOutput is one row of `credential list`. OwnerCNPJ is empty until
// the certificate was read.
type credentialOutput struct {
	ID                string
	Label             string
	OwnerCNPJ         string
	SubjectName       string
	FingerprintSHA256 string
	NotBefore         *time.Time
	NotAfter          *time.Time
}

func newCredentialOutput(c credential.Credential) credentialOutput {
	return credentialOutput{
		ID:                string(c.ID),
		Label:             c.Label,
		OwnerCNPJ:         c.OwnerCNPJ,
		SubjectName:       c.SubjectName,
		FingerprintSHA256: c.FingerprintSHA256,
		NotBefore:         c.NotBefore,
		NotAfter:          c.NotAfter,
	}
}

// connectionTestOutput is `nfe testar-conexao` and `cte testar-conexao`.
// CertExpiration is null when the certificate was not read.
type connectionTestOutput struct {
	CertLoaded        bool
	CertSubject       string
	CertExpiration    *time.Time
	EndpointReached   bool
	StatusExplanation string
}

func newConnectionTestOutput(r app.ConnectionTestResult) connectionTestOutput {
	return connectionTestOutput{
		CertLoaded:        r.CertLoaded,
		CertSubject:       r.CertSubject,
		CertExpiration:    optionalTime(r.CertNotAfter),
		EndpointReached:   r.EndpointReached,
		StatusExplanation: r.StatusExplanation,
	}
}

// nfsePullOutput is `pull`, one NFS-e sync run. DurationMs is in
// milliseconds.
type nfsePullOutput struct {
	CompanyName              string
	CNPJ                     string
	Status                   string
	StopReason               string
	LastProcessedNSU         int64
	LastFoundNSU             *int64
	DocumentsFound           int
	EventsFound              int
	DocumentsSaved           int
	EventsSaved              int
	DocumentsSkippedByPolicy int
	EventsSkippedByPolicy    int
	Errors                   int
	DurationMs               int64
}

func newNFSePullOutput(r sync.PullResult) nfsePullOutput {
	return nfsePullOutput{
		CompanyName:              r.CompanyName,
		CNPJ:                     r.CNPJ,
		Status:                   r.Status,
		StopReason:               r.StopReason,
		LastProcessedNSU:         r.LastProcessedNSU,
		LastFoundNSU:             r.LastFoundNSU,
		DocumentsFound:           r.DocumentsFound,
		EventsFound:              r.EventsFound,
		DocumentsSaved:           r.DocumentsSaved,
		EventsSaved:              r.EventsSaved,
		DocumentsSkippedByPolicy: r.DocumentsSkippedByPolicy,
		EventsSkippedByPolicy:    r.EventsSkippedByPolicy,
		Errors:                   r.Errors,
		DurationMs:               r.Duration.Milliseconds(),
	}
}

// nfePullOutput is `nfe pull`. DurationMs is in milliseconds.
type nfePullOutput struct {
	CompanyName      string
	CNPJ             string
	Status           string
	StopReason       string
	LastNSU          int64
	MaxNSU           *int64
	CompletasSaved   int
	ResumosSaved     int
	EventsSaved      int
	Errors           int
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
	DurationMs       int64
}

func newNFePullOutput(r app.NFePullResult) nfePullOutput {
	return nfePullOutput{
		CompanyName:      r.CompanyName,
		CNPJ:             r.CNPJ,
		Status:           r.Status,
		StopReason:       r.StopReason,
		LastNSU:          r.LastNSU,
		MaxNSU:           r.MaxNSU,
		CompletasSaved:   r.CompletasSaved,
		ResumosSaved:     r.ResumosSaved,
		EventsSaved:      r.EventsSaved,
		Errors:           r.Errors,
		NextAllowedAt:    r.NextAllowedAt,
		RequestsLastHour: r.RequestsLastHour,
		RequestBudget:    r.RequestBudget,
		DurationMs:       r.Duration.Milliseconds(),
	}
}

// ctePullOutput is `cte pull`. DurationMs is in milliseconds.
type ctePullOutput struct {
	CompanyName      string
	CNPJ             string
	Status           string
	StopReason       string
	LastNSU          int64
	MaxNSU           *int64
	DocumentsSaved   int
	EventsSaved      int
	Errors           int
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
	DurationMs       int64
}

func newCTePullOutput(r app.CTePullResult) ctePullOutput {
	return ctePullOutput{
		CompanyName:      r.CompanyName,
		CNPJ:             r.CNPJ,
		Status:           r.Status,
		StopReason:       r.StopReason,
		LastNSU:          r.LastNSU,
		MaxNSU:           r.MaxNSU,
		DocumentsSaved:   r.DocumentsSaved,
		EventsSaved:      r.EventsSaved,
		Errors:           r.Errors,
		NextAllowedAt:    r.NextAllowedAt,
		RequestsLastHour: r.RequestsLastHour,
		RequestBudget:    r.RequestBudget,
		DurationMs:       r.Duration.Milliseconds(),
	}
}

// exportOutput is a bulk export of `export`. OutPath is empty when nothing
// was exported, since no file is written then.
type exportOutput struct {
	OutPath       string
	Format        string
	Incremental   bool
	ExportedCount int
}

func newExportOutput(r app.ExportResult) exportOutput {
	return exportOutput{
		OutPath:       r.OutPath,
		Format:        r.Format,
		Incremental:   r.Incremental,
		ExportedCount: r.ExportedCount,
	}
}

// nfeExportOutput is a bulk export of `nfe export`, with the resumos it
// left out.
type nfeExportOutput struct {
	exportOutput
	SkippedResumos int
}

func newNFeExportOutput(r app.NFeExportResult) nfeExportOutput {
	return nfeExportOutput{
		exportOutput:   newExportOutput(r.ExportResult),
		SkippedResumos: r.SkippedResumos,
	}
}

// cteExportOutput is a bulk export of `cte export`, with the CT-e a DACTE
// export left out.
type cteExportOutput struct {
	exportOutput
	SkippedSemDACTE int
}

func newCTeExportOutput(r app.CTeExportResult) cteExportOutput {
	return cteExportOutput{
		exportOutput:    newExportOutput(r.ExportResult),
		SkippedSemDACTE: r.SkippedSemDACTE,
	}
}

// fileOutput is the file a single-document export wrote.
type fileOutput struct {
	OutPath string
}

// importedFileOutput is one file of `nfe import` or `cte import`. Kind and
// ChaveAcesso are empty when the file was rejected before they were known.
type importedFileOutput struct {
	Path        string
	Kind        string
	ChaveAcesso string
	Outcome     string // imported | updated | rejected
	Message     string
}

// importOutput is `nfe import` or `cte import`, one entry per file.
type importOutput struct {
	Files    []importedFileOutput
	Imported int
	Updated  int
	Rejected int
}

func newImportOutput(r app.ImportXMLResult) importOutput {
	files := make([]importedFileOutput, 0, len(r.Files))
	for _, f := range r.Files {
		files = append(files, importedFileOutput{
			Path:        f.Path,
			Kind:        f.Kind,
			ChaveAcesso: f.ChaveAcesso,
			Outcome:     f.Outcome,
			Message:     f.Message,
		})
	}
	return importOutput{
		Files:    files,
		Imported: r.Imported,
		Updated:  r.Updated,
		Rejected: r.Rejected,
	}
}

// fetchOutput is `nfe buscar`.
type fetchOutput struct {
	ChaveAcesso      string
	Outcome          string // completa | resumo | not_found
	Message          string
	RequestsLastHour int
	RequestBudget    int
}

func newFetchOutput(r app.FetchByChaveResult) fetchOutput {
	return fetchOutput{
		ChaveAcesso:      r.ChaveAcesso,
		Outcome:          r.Outcome,
		Message:          r.Message,
		RequestsLastHour: r.RequestsLastHour,
		RequestBudget:    r.RequestBudget,
	}
}

// recoverNSUOutput is `recuperar-nsu`. StopReason is set when the run
// stopped with NSUs left to ask for.
type recoverNSUOutput struct {
	CompanyName      string
	CNPJ             string
	Asked            int
	Recovered        int
	Empty            int
	Unreadable       int
	Expired          int64
	Missing          int64
	StopReason       string
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
}

func newRecoverNSUOutput(r sync.RecoverNSUResult) recoverNSUOutput {
	return recoverNSUOutput{
		CompanyName:      r.CompanyName,
		CNPJ:             r.CNPJ,
		Asked:            r.Asked,
		Recovered:        r.Recovered,
		Empty:            r.Empty,
		Unreadable:       r.Unreadable,
		Expired:          r.Expired,
		Missing:          r.Missing,
		StopReason:       r.StopReason,
		NextAllowedAt:    r.NextAllowedAt,
		RequestsLastHour: r.RequestsLastHour,
		RequestBudget:    r.RequestBudget,
	}
}

// discardNSUOutput is `recuperar-nsu --descartar`: how many NSUs were
// discarded or, without --confirmar, would be.
type discardNSUOutput struct {
	Discarded int64
	Confirmed bool
}

// nfeSkippedOutput is a chave a manifestação leaves out, and why.
type nfeSkippedOutput struct {
	ChaveAcesso string
	Reason      string
}

func newNFeSkippedOutput(s app.NFeSkipped) nfeSkippedOutput {
	return nfeSkippedOutput{ChaveAcesso: s.ChaveAcesso, Reason: s.Reason}
}

// nfeCienciaPlanOutput is `nfe ciencia` without --confirmar: what would be
// sent. Nothing is.
type nfeCienciaPlanOutput struct {
	Eligible []nfeDocumentOutput
	Skipped  []nfeSkippedOutput
}

func newNFeCienciaPlanOutput(p app.NFeCienciaPlan) nfeCienciaPlanOutput {
	eligible := make([]nfeDocumentOutput, 0, len(p.Eligible))
	for _, d := range p.Eligible {
		eligible = append(eligible, newNFeDocumentOutput(d))
	}
	skipped := make([]nfeSkippedOutput, 0, len(p.Skipped))
	for _, s := range p.Skipped {
		skipped = append(skipped, newNFeSkippedOutput(s))
	}
	return nfeCienciaPlanOutput{Eligible: eligible, Skipped: skipped}
}

// nfeEventOutcomeOutput is the result of one manifestação event. Status is
// registrada, ja_registrada, rejeitada or nao_enviada.
type nfeEventOutcomeOutput struct {
	ChaveAcesso  string
	TpEvento     string
	Status       string
	CStat        string
	XMotivo      string
	Protocolo    string
	RegisteredAt *time.Time
}

func newNFeEventOutcomeOutput(o app.NFeEventOutcome) nfeEventOutcomeOutput {
	return nfeEventOutcomeOutput{
		ChaveAcesso:  o.ChaveAcesso,
		TpEvento:     o.TpEvento,
		Status:       o.Status,
		CStat:        o.CStat,
		XMotivo:      o.XMotivo,
		Protocolo:    o.Protocolo,
		RegisteredAt: o.RegisteredAt,
	}
}

// nfeCienciaSummaryOutput is `nfe ciencia --confirmar`: one outcome per
// chave sent. Interrupted is the error that stopped the sending, or empty.
type nfeCienciaSummaryOutput struct {
	Outcomes    []nfeEventOutcomeOutput
	Skipped     []nfeSkippedOutput
	Interrupted string
}

func newNFeCienciaSummaryOutput(s app.NFeManifestacaoSummary) nfeCienciaSummaryOutput {
	outcomes := make([]nfeEventOutcomeOutput, 0, len(s.Outcomes))
	for _, o := range s.Outcomes {
		outcomes = append(outcomes, newNFeEventOutcomeOutput(o))
	}
	skipped := make([]nfeSkippedOutput, 0, len(s.Skipped))
	for _, sk := range s.Skipped {
		skipped = append(skipped, newNFeSkippedOutput(sk))
	}
	return nfeCienciaSummaryOutput{Outcomes: outcomes, Skipped: skipped, Interrupted: s.Interrupted}
}

// nfeManifestacaoPlanOutput is `nfe manifestar` without --confirmar: the
// event that would be sent and the NF-e it is for. Nothing is sent.
type nfeManifestacaoPlanOutput struct {
	Tipo             string
	TpEvento         string
	Justificativa    string
	ChaveAcesso      string
	Numero           string
	Serie            string
	IssueDate        *time.Time
	TotalValue       int64
	EmitenteCNPJ     string
	EmitenteName     string
	Manifestacao     string
	CienciaDue       *time.Time
	ConclusiveDue    *time.Time
	DaysLeft         *int
	TacitlyConfirmed bool
}

func newNFeManifestacaoPlanOutput(p app.NFeManifestacaoPlan) nfeManifestacaoPlanOutput {
	return nfeManifestacaoPlanOutput{
		Tipo:             string(p.Tipo),
		TpEvento:         p.Tipo.TpEvento(),
		Justificativa:    p.Justificativa,
		ChaveAcesso:      string(p.Document.ChaveAcesso),
		Numero:           p.Document.Numero,
		Serie:            p.Document.Serie,
		IssueDate:        optionalTime(p.Document.IssueDate),
		TotalValue:       p.Document.TotalValue.Cents(),
		EmitenteCNPJ:     p.Document.EmitenteCNPJ,
		EmitenteName:     p.Document.EmitenteName,
		Manifestacao:     string(p.Document.Manifestacao),
		CienciaDue:       optionalTime(p.CienciaDue),
		ConclusiveDue:    optionalTime(p.ConclusiveDue),
		DaysLeft:         optionalDaysLeft(p.ConclusiveDue, p.DaysLeft),
		TacitlyConfirmed: p.TacitlyConfirmed,
	}
}

// nfeResetOutput is `nfe reset`: what was removed with --confirmar
// (Confirmed true), or what would be removed without it.
// nfseResetOutput is the plan or the result of `nanci reset`. LastProcessedNSU
// is the cursor as it stood before the reset; no document is removed.
type nfseResetOutput struct {
	CompanyName      string
	CNPJ             string
	Environment      string
	Confirmed        bool
	LastProcessedNSU int64
}

func newNFSeResetOutput(s sync.StatusResult, confirmed bool) nfseResetOutput {
	return nfseResetOutput{
		CompanyName:      s.CompanyName,
		CNPJ:             s.CNPJ,
		Environment:      s.Environment,
		Confirmed:        confirmed,
		LastProcessedNSU: s.LastProcessedNSU,
	}
}

type nfeResetOutput struct {
	CompanyName       string
	CNPJ              string
	Environment       string
	Confirmed         bool
	CompanyDocuments  int
	Documents         int
	Events            int
	ExportMarks       int
	ManifestacoesKept int
}

func newNFeResetOutput(r app.NFeResetResult, confirmed bool) nfeResetOutput {
	return nfeResetOutput{
		CompanyName:       r.CompanyName,
		CNPJ:              r.CNPJ,
		Environment:       string(r.Environment),
		Confirmed:         confirmed,
		CompanyDocuments:  r.CompanyDocuments,
		Documents:         r.Documents,
		Events:            r.Events,
		ExportMarks:       r.ExportMarks,
		ManifestacoesKept: r.ManifestacoesKept,
	}
}

// cteResetOutput is `cte reset`: what was removed with --confirmar
// (Confirmed true), or what would be removed without it.
type cteResetOutput struct {
	CompanyName      string
	CNPJ             string
	Environment      string
	Confirmed        bool
	CompanyDocuments int
	Documents        int
	Events           int
	ExportMarks      int
}

func newCTeResetOutput(r app.CTeResetResult, confirmed bool) cteResetOutput {
	return cteResetOutput{
		CompanyName:      r.CompanyName,
		CNPJ:             r.CNPJ,
		Environment:      string(r.Environment),
		Confirmed:        confirmed,
		CompanyDocuments: r.CompanyDocuments,
		Documents:        r.Documents,
		Events:           r.Events,
		ExportMarks:      r.ExportMarks,
	}
}

// cteDesacordoPlanOutput is `cte desacordo` without --confirmar: the event
// that would be sent and the CT-e it is for. Due and DaysLeft are null when
// the CT-e has no authorization date. Nothing is sent.
type cteDesacordoPlanOutput struct {
	TpEvento     string
	TpAmb        string
	ChaveAcesso  string
	Modelo       string
	Numero       string
	Serie        string
	IssueDate    *time.Time
	AuthorizedAt *time.Time
	TotalValue   int64
	EmitenteCNPJ string
	EmitenteName string
	NSeqEvento   int
	Observacao   string
	Due          *time.Time
	DaysLeft     *int
}

func newCTeDesacordoPlanOutput(p app.CTeDesacordoPlan) cteDesacordoPlanOutput {
	doc := p.Document
	return cteDesacordoPlanOutput{
		TpEvento:     sefaz.TpEventoPrestacaoDesacordo,
		TpAmb:        p.TpAmb,
		ChaveAcesso:  string(doc.ChaveAcesso),
		Modelo:       doc.Modelo,
		Numero:       doc.Numero,
		Serie:        doc.Serie,
		IssueDate:    optionalTime(doc.IssueDate),
		AuthorizedAt: doc.AuthorizedAt,
		TotalValue:   doc.TotalValue.Cents(),
		EmitenteCNPJ: doc.Emitente.CNPJ,
		EmitenteName: doc.Emitente.Name,
		NSeqEvento:   p.NSeqEvento,
		Observacao:   p.Observacao,
		Due:          optionalTime(p.Due),
		DaysLeft:     optionalDaysLeft(p.Due, p.DaysLeft),
	}
}

// newCTeDesacordoOutcomeOutput is `cte desacordo --confirmar`, in the shape
// of an NF-e event outcome.
func newCTeDesacordoOutcomeOutput(o app.CTeDesacordoOutcome) nfeEventOutcomeOutput {
	return nfeEventOutcomeOutput{
		ChaveAcesso:  o.ChaveAcesso,
		TpEvento:     sefaz.TpEventoPrestacaoDesacordo,
		Status:       o.Status,
		CStat:        o.CStat,
		XMotivo:      o.XMotivo,
		Protocolo:    o.Protocolo,
		RegisteredAt: o.RegisteredAt,
	}
}
