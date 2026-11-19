package cli

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfse"
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
	Serie                 string
	Numero                string
	IssueDate             *time.Time
	Competence            string
	AuthorizedAt          *time.Time
	Protocolo             string
	TpNF                  string
	TpAmb                 string
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
		Serie:                 d.Serie,
		Numero:                d.Numero,
		IssueDate:             optionalTime(d.IssueDate),
		Competence:            d.Competence,
		AuthorizedAt:          d.AuthorizedAt,
		Protocolo:             d.Protocolo,
		TpNF:                  d.TpNF,
		TpAmb:                 d.TpAmb,
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
		TotalEmitidas:      r.TotalEmitidas,
		TotalTomadas:       r.TotalTomadas,
	}
}

// sefazSourceStatusOutput is the part of `nfe status` and `cte status` that
// both SEFAZ sources share. MaxNSU is null when unknown; IdleDays is 0 unless
// the idle warning applies.
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
	IdleDays          int
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
		IdleDays:          s.IdleDays,
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
	LastSyncAt      *time.Time
}

func newCompanyOutput(c company.Company) companyOutput {
	return companyOutput{
		CNPJ:            c.CNPJ,
		Name:            c.Name,
		CredentialID:    string(c.CredentialID),
		CredentialLabel: c.CredentialLabel,
		Environment:     string(c.Environment),
		UF:              c.UF,
		SyncStartPolicy: string(c.SyncStartPolicy),
		SyncStartDate:   c.SyncStartDate,
		LastSyncAt:      c.LastSyncAt,
	}
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
type connectionTestOutput struct {
	CertLoaded        bool
	CertSubject       string
	CertExpiration    string
	EndpointReached   bool
	StatusExplanation string
}

func newConnectionTestOutput(r app.ConnectionTestResult) connectionTestOutput {
	return connectionTestOutput{
		CertLoaded:        r.CertLoaded,
		CertSubject:       r.CertSubject,
		CertExpiration:    r.CertExpiration,
		EndpointReached:   r.EndpointReached,
		StatusExplanation: r.StatusExplanation,
	}
}
