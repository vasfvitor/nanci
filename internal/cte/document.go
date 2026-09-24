package cte

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// Party is one of the parties named in a CT-e.
type Party struct {
	CNPJ string // CNPJ or CPF
	Name string // xNome
	IE   string
	UF   string // UF of the party's address
}

// Municipio is where the transport starts or ends.
type Municipio struct {
	Codigo string // IBGE code
	Nome   string
	UF     string
}

// Document is the canonical CT-e, CT-e OS, GTV-e or CT-e Simplificado, built
// from the full signed XML plus its authorization protocol. CT-e distribution
// has no resumo. Fields a document kind does not carry stay empty or zero.
type Document struct {
	ID            string
	ChaveAcesso   dfe.AccessKey
	TpAmb         string // ide/tpAmb: "1" produção, "2" homologação
	Modelo        string // "57", "64" or "67"
	TipoDocumento TipoDocumento
	Serie         string
	Numero        string // ide/nCT
	CFOP          string
	NatOp         string
	IssueDate     time.Time  // dhEmi
	Competence    string     // "YYYY-MM" from dhEmi, in the offset it was written with
	AuthorizedAt  *time.Time // protCTe/infProt/dhRecbto
	Protocolo     string     // nProt
	// TpCTe, TpServ and Modal are kept as written: their domains changed
	// between layouts 3.00 and 4.00, so the labels live in the frontend.
	TpCTe  string
	TpServ string
	Modal  string
	MunIni Municipio
	MunFim Municipio

	Emitente     Party
	Remetente    Party
	Destinatario Party
	Expedidor    Party
	Recebedor    Party
	// Tomador is resolved by the parser: a copy of the party TomadorIndicador
	// points to, or the party the document names on its own.
	Tomador Party
	// TomadorIndicador is the raw toma code: "0" remetente, "1" expedidor
	// ("1" destinatário on a GTV-e), "2" recebedor, "3" destinatário,
	// "4" outro. Empty on a CT-e OS.
	TomadorIndicador string
	AutorizadosCNPJ  []string // autXML CNPJ or CPF

	TotalValue          dfe.Money // vPrest/vTPrest
	ReceivableValue     dfe.Money // vPrest/vRec
	ICMSValue           dfe.Money // imp/ICMS/*/vICMS
	TotTribValue        dfe.Money // imp/vTotTrib
	CargaValue          dfe.Money // infCarga/vCarga
	ProdutoPredominante string    // infCarga/proPred
	// NFeChaves are the NF-e access keys the CT-e transported, in document
	// order. Keys masked with 9s for autXML parties are left out.
	NFeChaves []string

	Situacao      Situacao
	LayoutVersion string // infCte@versao
	RawHash       string
	ParseWarnings []string
}

// CompanyDocument is one managed company's view of a canonical CT-e.
type CompanyDocument struct {
	Document
	RelationID  string
	CompanyID   dfe.CompanyID
	CompanyRole CompanyRole
	// Papeis are all the roles the company plays in the document, in
	// priority order. CompanyRole is the first one.
	Papeis           []CompanyRole
	VisibilityReason VisibilityReason
	FirstSeenNSU     *int64
	LastSeenNSU      *int64
	FirstSyncedAt    time.Time
	LastSyncedAt     time.Time
	// EventCount is how many events nanci holds for the chave.
	EventCount int
}
