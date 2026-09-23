package nfe

import "time"

// Event is an NF-e event (cancelamento, carta de correção, manifestação, ...)
// built from a resEvento (resumo) or a procEventoNFe (completa).
type Event struct {
	ID            string
	ChaveAcesso   AccessKey
	TpEvento      string
	Type          EventType
	NSeqEvento    int
	EventAt       *time.Time // dhEvento
	RegisteredAt  *time.Time // retEvento dhRegEvento or resEvento dhRecbto
	Protocolo     string     // nProt of the event registration
	AutorCNPJ     string     // CNPJ or CPF of the event author
	Description   string     // descEvento, falling back to xEvento
	Justificativa string     // detEvento/xJust
	Correcao      string     // detEvento/xCorrecao (carta de correção)
	Completeness  Completeness
	// Registered is true when SEFAZ accepted the event: always for a
	// resEvento, and for a procEventoNFe only when retEvento carries one of
	// the registration cStat codes.
	Registered    bool
	RawHash       string
	ParseWarnings []string
}
