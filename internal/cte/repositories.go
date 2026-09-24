package cte

import "errors"

// ExportKindXML is the only export kind tracked for CT-e.
const ExportKindXML = "xml"

// ErrDocumentNotFound is returned when the company does not see a chave.
var ErrDocumentNotFound = errors.New("CT-e não encontrado para a empresa")

// DocumentFilter selects company CT-e rows. Zero values do not filter.
type DocumentFilter struct {
	Competence string // "YYYY-MM"
	Situacao   Situacao
	// Role matches the primary role or any of the company's Papeis.
	Role         CompanyRole
	Modelo       string // "57", "64" or "67"
	EmitenteCNPJ string
	TomadorCNPJ  string
	// NFeChave keeps the CT-e that transported this NF-e.
	NFeChave     string
	ChavesAcesso []string
	TpAmb        string // "1" produção, "2" homologação
	Limit        int
}

// Counts summarizes one company's CT-e by primary role.
type Counts struct {
	ByRole map[CompanyRole]int
}

// ResetCounts is what a company CT-e reset removes, or would remove.
type ResetCounts struct {
	CompanyDocuments int // the company's rows in company_cte_documents
	Documents        int // cte_documents no other company sees
	Events           int // cte_events of those documents
	ExportMarks      int
}
