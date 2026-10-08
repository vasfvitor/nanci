package main

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	nsync "github.com/vasfvitor/nanci/internal/sync"
)

// call is one call a fake service received: the method name and the
// arguments after ctx.
type call struct {
	method string
	args   []any
}

type recorder struct {
	calls []call
}

func (r *recorder) record(method string, args ...any) {
	r.calls = append(r.calls, call{method: method, args: args})
}

// assertCalls fails unless r received exactly want, in order.
func assertCalls(t *testing.T, r *recorder, want ...call) {
	t.Helper()
	if len(r.calls) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(r.calls, want) {
		t.Errorf("calls =\n  %#v\nwant\n  %#v", r.calls, want)
	}
}

// newTestApp returns an App wired to the given services, as startup would
// leave it, without a database or the Wails runtime.
func newTestApp(svc services) *App {
	return &App{
		ctx:      context.Background(),
		svc:      svc,
		cred:     newWailsCredentialProvider(),
		logLevel: new(slog.LevelVar),
	}
}

// fakeSet is a fake of every core service App calls.
type fakeSet struct {
	companies   *fakeCompanies
	credentials *fakeCredentials
	documents   *fakeDocuments
	exports     *fakeExports
	query       *fakeQuery
	sync        *fakeSync
	nfe         *fakeNFe
	cte         *fakeCTe
}

// newFakeApp returns an App wired to a fresh fakeSet whose services all
// return err.
func newFakeApp(err error) (fakeSet, *App) {
	f := fakeSet{
		companies:   &fakeCompanies{err: err},
		credentials: &fakeCredentials{err: err},
		documents:   &fakeDocuments{err: err},
		exports:     &fakeExports{err: err},
		query:       &fakeQuery{err: err},
		sync:        &fakeSync{err: err},
		nfe:         &fakeNFe{err: err},
		cte:         &fakeCTe{err: err},
	}
	return f, newTestApp(services{
		companies:   f.companies,
		credentials: f.credentials,
		documents:   f.documents,
		exports:     f.exports,
		query:       f.query,
		sync:        f.sync,
		nfe:         f.nfe,
		cte:         f.cte,
	})
}

// assertNoCalls fails if any service of the set was called.
func (f fakeSet) assertNoCalls(t *testing.T) {
	t.Helper()
	assertCalls(t, &f.companies.recorder)
	assertCalls(t, &f.credentials.recorder)
	assertCalls(t, &f.documents.recorder)
	assertCalls(t, &f.exports.recorder)
	assertCalls(t, &f.query.recorder)
	assertCalls(t, &f.sync.recorder)
	assertCalls(t, &f.nfe.recorder)
	assertCalls(t, &f.cte.recorder)
}

// fakeEvents records what would be emitted to the frontend and signals each
// emit on emitted.
type fakeEvents struct {
	emitted chan call
}

func (f *fakeEvents) Emit(_ context.Context, name string, data ...any) {
	f.emitted <- call{method: name, args: data}
}

type fakeCompanies struct {
	recorder
	companies []company.Company
	err       error
}

func (f *fakeCompanies) AddCompany(_ context.Context, in company.AddCompanyInput) error {
	f.record("AddCompany", in)
	return f.err
}

func (f *fakeCompanies) UpdateCompany(_ context.Context, in company.UpdateCompanyInput) error {
	f.record("UpdateCompany", in)
	return f.err
}

func (f *fakeCompanies) AssignCredentialToCompany(_ context.Context, in company.AssignCredentialInput) error {
	f.record("AssignCredentialToCompany", in)
	return f.err
}

func (f *fakeCompanies) ListCompanies(_ context.Context) ([]company.Company, error) {
	f.record("ListCompanies")
	return f.companies, f.err
}

type fakeCredentials struct {
	recorder
	credentials []credential.Credential
	err         error
}

func (f *fakeCredentials) AddCredential(_ context.Context, in credential.AddCredentialInput) error {
	f.record("AddCredential", in)
	return f.err
}

func (f *fakeCredentials) ListCredentials(_ context.Context) ([]credential.Credential, error) {
	f.record("ListCredentials")
	return f.credentials, f.err
}

func (f *fakeCredentials) UpdateCredentialPath(_ context.Context, in credential.UpdateCredentialPathInput) error {
	f.record("UpdateCredentialPath", in)
	return f.err
}

func (f *fakeCredentials) UpdateCredentialData(_ context.Context, in credential.UpdateCredentialDataInput) error {
	f.record("UpdateCredentialData", in)
	return f.err
}

type fakeDocuments struct {
	recorder
	documents []nfse.CompanyDocument
	events    []app.EventView
	viewed    int
	err       error
}

func (f *fakeDocuments) ListDocuments(_ context.Context, in app.ListInput) ([]nfse.CompanyDocument, error) {
	f.record("ListDocuments", in)
	return f.documents, f.err
}

func (f *fakeDocuments) ListEventsForDocument(_ context.Context, documentID string) ([]app.EventView, error) {
	f.record("ListEventsForDocument", documentID)
	return f.events, f.err
}

func (f *fakeDocuments) MarkDocumentsViewed(_ context.Context, cnpj string, chaves []string) (int, error) {
	f.record("MarkDocumentsViewed", cnpj, chaves)
	return f.viewed, f.err
}

type fakeQuery struct {
	recorder
	events     string
	connection app.ConnectionTestResult
	err        error
}

func (f *fakeQuery) QueryNFSeEvents(_ context.Context, in app.QueryNFSeInput) (string, error) {
	f.record("QueryNFSeEvents", in)
	return f.events, f.err
}

func (f *fakeQuery) TestConnection(_ context.Context, cnpj string) (app.ConnectionTestResult, error) {
	f.record("TestConnection", cnpj)
	return f.connection, f.err
}

type fakeSync struct {
	recorder
	pull   nsync.PullResult
	status nsync.StatusResult
	err    error
}

func (f *fakeSync) Pull(_ context.Context, in nsync.PullInput) (nsync.PullResult, error) {
	f.record("Pull", in)
	return f.pull, f.err
}

func (f *fakeSync) Status(_ context.Context, cnpj string) (nsync.StatusResult, error) {
	f.record("Status", cnpj)
	return f.status, f.err
}

func (f *fakeSync) ResetSyncState(_ context.Context, in nsync.ResetSyncInput) error {
	f.record("ResetSyncState", in)
	return f.err
}

type fakeExports struct {
	recorder
	result app.ExportResult
	count  int
	err    error
}

func (f *fakeExports) ExportCSV(_ context.Context, in app.ExportInput) (app.ExportResult, error) {
	f.record("ExportCSV", in)
	return f.result, f.err
}

func (f *fakeExports) ExportXLSX(_ context.Context, in app.ExportInput) (app.ExportResult, error) {
	f.record("ExportXLSX", in)
	return f.result, f.err
}

func (f *fakeExports) ExportZIP(_ context.Context, in app.ExportInput) (app.ExportResult, error) {
	f.record("ExportZIP", in)
	return f.result, f.err
}

func (f *fakeExports) ExportDANFSeZIP(_ context.Context, in app.ExportInput) (app.ExportResult, error) {
	f.record("ExportDANFSeZIP", in)
	return f.result, f.err
}

func (f *fakeExports) ExportDANFSe(_ context.Context, in app.ExportDANFSeInput) error {
	f.record("ExportDANFSe", in)
	return f.err
}

func (f *fakeExports) ExportXML(_ context.Context, in app.ExportXMLInput) error {
	f.record("ExportXML", in)
	return f.err
}

func (f *fakeExports) CountPendingExportDocuments(_ context.Context, in app.ExportInput, kind string) (int, error) {
	f.record("CountPendingExportDocuments", in, kind)
	return f.count, f.err
}

type fakeNFe struct {
	recorder
	pull      app.NFePullResult
	status    app.NFeStatusResult
	documents []app.NFeDocument
	viewed    int
	events    []nfe.Event
	pending   []app.NFePendingManifestacao
	plan      app.NFeCienciaPlan
	summary   app.NFeManifestacaoSummary
	outcome   app.NFeEventOutcome
	zip       app.NFeExportResult
	reset     app.NFeResetResult
	err       error
}

func (f *fakeNFe) Pull(_ context.Context, cnpj string) (app.NFePullResult, error) {
	f.record("Pull", cnpj)
	return f.pull, f.err
}

func (f *fakeNFe) Status(_ context.Context, cnpj string) (app.NFeStatusResult, error) {
	f.record("Status", cnpj)
	return f.status, f.err
}

func (f *fakeNFe) ListDocuments(_ context.Context, in app.NFeListInput) ([]app.NFeDocument, error) {
	f.record("ListDocuments", in)
	return f.documents, f.err
}

func (f *fakeNFe) MarkViewed(_ context.Context, cnpj string, chaves []string) (int, error) {
	f.record("MarkViewed", cnpj, chaves)
	return f.viewed, f.err
}

func (f *fakeNFe) ListEvents(_ context.Context, cnpj, chave string) ([]nfe.Event, error) {
	f.record("ListEvents", cnpj, chave)
	return f.events, f.err
}

func (f *fakeNFe) ListPendingManifestacoes(_ context.Context, in app.NFePendingInput) ([]app.NFePendingManifestacao, error) {
	f.record("ListPendingManifestacoes", in)
	return f.pending, f.err
}

func (f *fakeNFe) PlanCiencia(_ context.Context, in app.NFeCienciaInput) (app.NFeCienciaPlan, error) {
	f.record("PlanCiencia", in)
	return f.plan, f.err
}

func (f *fakeNFe) RegisterCiencia(_ context.Context, in app.NFeCienciaInput) (app.NFeManifestacaoSummary, error) {
	f.record("RegisterCiencia", in)
	return f.summary, f.err
}

func (f *fakeNFe) RegisterManifestacao(_ context.Context, in app.NFeManifestacaoInput) (app.NFeEventOutcome, error) {
	f.record("RegisterManifestacao", in)
	return f.outcome, f.err
}

func (f *fakeNFe) ExportXML(_ context.Context, in app.NFeExportXMLInput) error {
	f.record("ExportXML", in)
	return f.err
}

func (f *fakeNFe) ExportXMLZip(_ context.Context, in app.NFeExportInput) (app.NFeExportResult, error) {
	f.record("ExportXMLZip", in)
	return f.zip, f.err
}

func (f *fakeNFe) Reset(_ context.Context, cnpj string) (app.NFeResetResult, error) {
	f.record("Reset", cnpj)
	return f.reset, f.err
}

type fakeCTe struct {
	recorder
	pull       app.CTePullResult
	status     app.CTeStatusResult
	documents  []cte.CompanyDocument
	viewed     int
	events     []cte.Event
	connection app.ConnectionTestResult
	zip        app.ExportResult
	reset      app.CTeResetResult
	err        error
}

func (f *fakeCTe) Pull(_ context.Context, cnpj string) (app.CTePullResult, error) {
	f.record("Pull", cnpj)
	return f.pull, f.err
}

func (f *fakeCTe) Status(_ context.Context, cnpj string) (app.CTeStatusResult, error) {
	f.record("Status", cnpj)
	return f.status, f.err
}

func (f *fakeCTe) ListDocuments(_ context.Context, in app.ListCTeInput) ([]cte.CompanyDocument, error) {
	f.record("ListDocuments", in)
	return f.documents, f.err
}

func (f *fakeCTe) MarkViewed(_ context.Context, cnpj string, chaves []string) (int, error) {
	f.record("MarkViewed", cnpj, chaves)
	return f.viewed, f.err
}

func (f *fakeCTe) ListEvents(_ context.Context, cnpj, chave string) ([]cte.Event, error) {
	f.record("ListEvents", cnpj, chave)
	return f.events, f.err
}

func (f *fakeCTe) TestConnection(_ context.Context, cnpj string) (app.ConnectionTestResult, error) {
	f.record("TestConnection", cnpj)
	return f.connection, f.err
}

func (f *fakeCTe) ExportXML(_ context.Context, in app.CTeExportXMLInput) error {
	f.record("ExportXML", in)
	return f.err
}

func (f *fakeCTe) ExportXMLZip(_ context.Context, in app.CTeExportInput) (app.ExportResult, error) {
	f.record("ExportXMLZip", in)
	return f.zip, f.err
}

func (f *fakeCTe) PreviewReset(_ context.Context, cnpj string) (app.CTeResetResult, error) {
	f.record("PreviewReset", cnpj)
	return f.reset, f.err
}

func (f *fakeCTe) Reset(_ context.Context, cnpj string) (app.CTeResetResult, error) {
	f.record("Reset", cnpj)
	return f.reset, f.err
}
