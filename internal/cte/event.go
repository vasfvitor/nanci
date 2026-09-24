package cte

import (
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// Event is a CT-e event (cancelamento, carta de correção, comprovante de
// entrega, MDF-e, ...) built from a procEventoCTe.
type Event struct {
	ID            string
	ChaveAcesso   dfe.AccessKey
	TpAmb         string // eventoCTe/infEvento/tpAmb
	COrgao        string // code of the body that registered the event
	TpEvento      string
	Type          EventType
	NSeqEvento    int
	EventAt       *time.Time // dhEvento
	RegisteredAt  *time.Time // retEventoCTe dhRegEvento
	Protocolo     string     // nProt of the event registration
	AutorCNPJ     string     // CNPJ or CPF of the event author
	Description   string     // detEvento descEvento, falling back to retEventoCTe xEvento
	Justificativa string     // detEvento xJust
	Observacao    string     // detEvento xObs
	// Correcao lists the infCorrecao groups of a carta de correção as
	// "grupo.campo=valor; grupo.campo=valor".
	Correcao    string
	CondicaoUso string // detEvento xCondUso
	// Registered is true when retEventoCTe carries one of the registration
	// cStat codes (134, 135, 136).
	Registered bool
	// CStat and XMotivo are the retEventoCTe answer.
	CStat         string
	XMotivo       string
	RawHash       string
	ParseWarnings []string
}
