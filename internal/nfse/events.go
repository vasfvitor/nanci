package nfse

import "time"

// Event represents an event that happened to a document (e.g., cancellation).
type Event struct {
	ID          string
	CompanyID   string
	ChaveAcesso string
	Type        string // e.g., "cancelamento", "substituicao"
	IssueDate   time.Time
	Details     string
	RawHash     string
	CreatedAt   time.Time
}
