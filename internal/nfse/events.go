package nfse

import "time"

// Event represents an event that happened to a document (e.g., cancellation).
type Event struct {
	ID                     string
	DocumentID             string
	ChaveAcesso            string
	Type                   string // e.g., "cancelamento", "substituicao", "unknown"
	EventAt                time.Time
	EventAtValid           bool
	ReplacementChaveAcesso string
	Description            string
	RawXMLPath             string
	RawHash                string
	ParseWarnings          []string
	CreatedAt              time.Time
}
