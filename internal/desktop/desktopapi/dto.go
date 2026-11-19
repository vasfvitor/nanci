// DTOs shared by several areas: the error payload, the viewed marks input,
// the export result, build info, the connection test, the XML import and the
// fetch by chave, plus the small helpers the mappers share.

package desktopapi

import (
	"time"

	"github.com/vasfvitor/nanci/internal/app"
)

// ErrorPayload is the cause of the error a bound method's promise rejects
// with. Code names the errors the frontend branches on ("canceled",
// "sefaz_blocked", "sync_running", "update_busy", "startup_failed") and is
// empty for any other error.
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

// BuildInfo is the Sobre section and the header of the diagnostic package.
// WebView2Version is the installed WebView2 runtime, empty off Windows or
// when it cannot be read.
type BuildInfo struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	Date            string `json:"date"`
	WebView2Version string `json:"webview2Version"`
}

// UpdateInfo answers CheckForUpdate. Enabled is false in a build that does
// not update itself (a dev build, or one without the public key), and then
// nothing else is set but CurrentVersion. Version and Notes describe the
// release found when Available is true.
type UpdateInfo struct {
	Enabled        bool   `json:"enabled"`
	CurrentVersion string `json:"currentVersion"`
	Available      bool   `json:"available"`
	Version        string `json:"version"`
	Notes          string `json:"notes"`
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

// RecoverNSUInput names the company whose missing NSUs to ask SEFAZ for.
type RecoverNSUInput struct {
	CNPJ string
}

// RecoverNSUResult counts what one NSU recovery did and how much of the
// hourly budget is spent after it.
type RecoverNSUResult struct {
	Asked            int    // consNSU requests answered
	Recovered        int    // NSUs whose document was stored
	Empty            int    // NSUs with nothing to store (never generated, rejected alone or skipped)
	Unreadable       int    // NSUs whose document could not be read; see the log
	Expired          int64  // NSUs closed unasked, past the 90-day window
	Missing          int64  // NSUs still missing after the run
	StopReason       string // limit | rate_budget | consumo_indevido; empty when nothing is left to ask
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
}

func RecoverNSUResultFrom(res app.RecoverNSUResult) RecoverNSUResult {
	return RecoverNSUResult{
		Asked:            res.Asked,
		Recovered:        res.Recovered,
		Empty:            res.Empty,
		Unreadable:       res.Unreadable,
		Expired:          res.Expired,
		Missing:          res.Missing,
		StopReason:       res.StopReason,
		NextAllowedAt:    res.NextAllowedAt,
		RequestsLastHour: res.RequestsLastHour,
		RequestBudget:    res.RequestBudget,
	}
}

// optionalTime returns nil for the zero time.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
