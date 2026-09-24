package nfe

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// Document is the canonical NF-e, built from a resNFe (resumo) or a procNFe
// (completa). Fields that a resumo does not carry stay empty or zero.
type Document struct {
	ID                string
	ChaveAcesso       dfe.AccessKey
	Modelo            string     // "55"
	Serie             string     // ide/serie (completa) or the key slot (resumo)
	Numero            string     // ide/nNF (completa) or the key slot (resumo)
	IssueDate         time.Time  // dhEmi
	Competence        string     // "YYYY-MM" from dhEmi, in the offset it was written with
	AuthorizedAt      *time.Time // resNFe/dhRecbto or protNFe/infProt/dhRecbto
	Protocolo         string     // nProt
	EmitenteCNPJ      string     // CNPJ or CPF
	EmitenteName      string
	EmitenteIE        string
	EmitenteUF        string   // enderEmit/UF (completa) or the key's cUF (resumo)
	DestinatarioCNPJ  string   // CNPJ, CPF or idEstrangeiro; empty on resumo
	DestinatarioName  string   // empty on resumo
	TransportadorCNPJ string   // transp/transporta CNPJ or CPF; empty on resumo
	AutorizadosCNPJ   []string // autXML CNPJ or CPF; empty on resumo
	TpNF              string   // "0" entrada, "1" saída
	FinNFe            string   // empty on resumo
	NatOp             string   // empty on resumo
	TotalValue        dfe.Money
	ICMSValue         dfe.Money // total/ICMSTot/vICMS; zero on resumo
	IPIValue          dfe.Money // total/ICMSTot/vIPI; zero on resumo
	Situacao          Situacao
	Completeness      Completeness
	LayoutVersion     string // infNFe@versao or resNFe@versao
	TpAmb             string // ide/tpAmb: "1" produção, "2" homologação; a resumo carries none
	RawHash           string // hash of the current blob (procNFe once completa)
	ResumoRawHash     string // hash of the resNFe blob, kept after the upgrade to completa
	ParseWarnings     []string
}

// CompanyDocument is one managed company's view of a canonical NF-e.
type CompanyDocument struct {
	Document
	RelationID       string
	CompanyID        dfe.CompanyID
	CompanyRole      CompanyRole
	VisibilityReason VisibilityReason
	Manifestacao     Manifestacao
	// ManifestacaoAt is when the event that set Manifestacao was registered.
	ManifestacaoAt *time.Time
	FirstSeenNSU   *int64
	LastSeenNSU    *int64
	FirstSyncedAt  time.Time
	LastSyncedAt   time.Time
	// EventCount is how many events nanci holds for the chave.
	EventCount int
}
