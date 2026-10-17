// DTOs shared by several areas: the error payload, the viewed marks input,
// the export result, build info, the connection test, the XML import and the
// fetch by chave, plus the small helpers the mappers share.

package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
)

// ErrorPayload is the value a bound method's promise rejects with. Code names
// the errors the frontend branches on ("canceled", "sefaz_blocked",
// "sync_running") and is empty for any other error.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MarkViewedInput names the documents of one company to mark as viewed.
type MarkViewedInput struct {
	CNPJ         string
	ChavesAcesso []string
}

type ExportResult struct {
	OutPath       string
	Format        string
	Incremental   bool
	ExportedCount int
}

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

type ConnectionTestResult struct {
	CertLoaded        bool   `json:"certLoaded"`
	CertSubject       string `json:"certSubject"`
	CertExpiration    string `json:"certExpiration"`
	MTLSAccepted      bool   `json:"mtlsAccepted"`
	EndpointReached   bool   `json:"endpointReached"`
	ResponseCode      string `json:"responseCode"`
	ResponseDetail    string `json:"responseDetail"`
	StatusExplanation string `json:"statusExplanation"`
}

// --- Import and fetch by chave ---

// ImportXMLInput lists the XML files the user picked to import for the
// company.
type ImportXMLInput struct {
	CNPJ  string
	Files []string
}

// ImportedFile is the result of one imported file. Kind and ChaveAcesso are
// empty when the file was rejected before they were known.
type ImportedFile struct {
	Path        string
	Kind        string // completa | resumo | evento
	ChaveAcesso string
	Outcome     string // imported | updated | rejected
	Message     string
}

// ImportXMLResult is one ImportedFile per input file, in input order, and
// the counts per outcome.
type ImportXMLResult struct {
	Files    []ImportedFile
	Imported int
	Updated  int
	Rejected int
}

func ImportXMLResultFrom(res app.ImportXMLResult) ImportXMLResult {
	files := make([]ImportedFile, len(res.Files))
	for i, f := range res.Files {
		files[i] = ImportedFile(f)
	}
	return ImportXMLResult{
		Files:    files,
		Imported: res.Imported,
		Updated:  res.Updated,
		Rejected: res.Rejected,
	}
}

// FetchByChaveInput names the NF-e to ask SEFAZ for.
type FetchByChaveInput struct {
	CNPJ        string
	ChaveAcesso string
}

// FetchByChaveResult is what SEFAZ delivered for the chave and how much of
// the hourly budget is spent after the request.
type FetchByChaveResult struct {
	ChaveAcesso      string
	Outcome          string // completa | resumo | not_found
	Message          string
	RequestsLastHour int
	RequestBudget    int
}

func FetchByChaveResultFrom(res app.FetchByChaveResult) FetchByChaveResult {
	return FetchByChaveResult(res)
}

// optionalTime returns nil for the zero time.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
