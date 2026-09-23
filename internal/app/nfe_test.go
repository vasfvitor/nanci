package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
)

// Fixtures of internal/nfe/testdata. The company is their destinatário.
const (
	nfeTestCNPJ        = "70860312000150"
	nfeChaveProc       = "35260911222333000181550010000012341123456787" // resnfe.xml, procnfe.xml, proceventonfe-ciencia.xml
	nfeChaveCancelada  = "35260911222333000181550010000012351234567894" // resnfe-cancelada.xml
	nfeChaveDenegada   = "35260911222333000181550010000012361345678900" // procnfe-denegada.xml
	nfeMockPFXPassword = "mockdata"
)

type nfePasswordStub struct {
	requests []CertPasswordRequest
}

func (p *nfePasswordStub) GetCertPassword(_ context.Context, req CertPasswordRequest) ([]byte, error) {
	p.requests = append(p.requests, req)
	return []byte(nfeMockPFXPassword), nil
}

type nfeTestEnv struct {
	t         *testing.T
	db        *sql.DB
	app       *App
	repo      *store.NFeRepository
	xml       files.XMLStore
	company   *nfse.Company
	passwords *nfePasswordStub
}

// newNFeTestEnv returns an App over a real database with one company, the
// destinatário of the fixtures, whose certificate is the mock PFX.
func newNFeTestEnv(t *testing.T) *nfeTestEnv {
	t.Helper()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)

	certPath, err := filepath.Abs(filepath.Join("..", "foundation", "cert", "testdata", "cert_a1_mock_70860312000150.pfx"))
	if err != nil {
		t.Fatal(err)
	}
	cred := &nfse.Credential{ID: "cred-1", Label: "Mock A1", CertPath: certPath}
	if err := credential.NewStore(db).CreateCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}
	comp := storetest.TestCompany("comp-1", nfeTestCNPJ, nfse.EnvironmentProduction, cred)
	comp.Name = "Empresa Mock"
	comp.UF = "SP"
	if err := company.NewStore(db).CreateCompany(ctx, comp); err != nil {
		t.Fatal(err)
	}

	repo := store.NewNFeRepository(db)
	xmlStore := files.NewBlobStore(t.TempDir())
	passwords := &nfePasswordStub{}
	application, err := New(Dependencies{
		Log:                slog.New(slog.DiscardHandler),
		CompanyStore:       company.NewStore(db),
		CredentialStore:    credential.NewStore(db),
		SyncRepo:           sync.NewStore(db),
		DocumentRepo:       store.NewDocumentRepository(db),
		NFeRepo:            repo,
		XMLStore:           xmlStore,
		DataDir:            t.TempDir(),
		CredentialProvider: passwords,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &nfeTestEnv{t: t, db: db, app: application, repo: repo, xml: xmlStore, company: comp, passwords: passwords}
}

func readNFeFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", name)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// seed stores the fixture (after the old, new replacements) as if the
// distribution had delivered it at nsu, and returns its blob hash.
func (e *nfeTestEnv) seed(fixture string, nsu int64, replacements ...string) string {
	e.t.Helper()
	data := []byte(strings.NewReplacer(replacements...).Replace(readNFeFixture(e.t, fixture)))
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if err := e.xml.Store(hash, data); err != nil {
		e.t.Fatal(err)
	}

	ctx := context.Background()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	switch {
	case strings.HasPrefix(fixture, "resnfe"), strings.HasPrefix(fixture, "procnfe"):
		parse := nfe.ParseResNFe
		if strings.HasPrefix(fixture, "procnfe") {
			parse = nfe.ParseProcNFe
		}
		doc, err := parse(data)
		if err != nil {
			e.t.Fatalf("parse %s: %v", fixture, err)
		}
		doc.RawHash = hash
		_, err = e.repo.ApplyDocumentTx(ctx, tx, store.ApplyNFeDocumentParams{Document: doc, CompanyID: e.company.ID, CompanyCNPJ: e.company.CNPJ, NSU: nsu})
		if err != nil {
			e.t.Fatal(err)
		}
	default:
		parse := nfe.ParseResEvento
		if strings.HasPrefix(fixture, "proceventonfe") {
			parse = nfe.ParseProcEventoNFe
		}
		ev, err := parse(data)
		if err != nil {
			e.t.Fatalf("parse %s: %v", fixture, err)
		}
		ev.RawHash = hash
		if _, err := e.repo.ApplyEventTx(ctx, tx, store.ApplyNFeEventParams{Event: ev}); err != nil {
			e.t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		e.t.Fatal(err)
	}
	return hash
}

// seedResumo stores a resNFe with the chave of number n, authorized at
// authorizedAt (RFC 3339), emitted by emitente.
func (e *nfeTestEnv) seedResumo(n int, authorizedAt, emitente string) string {
	e.t.Helper()
	chave := testChave(e.t, n)
	e.seed("resnfe.xml", int64(1000+n),
		nfeChaveProc, chave,
		"2026-09-01T09:15:42-03:00", authorizedAt,
		"<CNPJ>11222333000181</CNPJ>", "<CNPJ>"+emitente+"</CNPJ>",
	)
	return chave
}

// testChave builds a valid access key whose nNF and cNF are n.
func testChave(t *testing.T, n int) string {
	t.Helper()
	base := fmt.Sprintf("352609%s55001%09d1%08d", "11222333000181", n, n)
	sum, weight := 0, 2
	for i := len(base) - 1; i >= 0; i-- {
		sum += int(base[i]-'0') * weight
		weight++
		if weight > 9 {
			weight = 2
		}
	}
	dv := 0
	if r := sum % 11; r >= 2 {
		dv = 11 - r
	}
	chave := base + strconv.Itoa(dv)
	if _, err := nfe.ParseAccessKey(chave); err != nil {
		t.Fatal(err)
	}
	return chave
}

func chavesOf(docs []NFeDocument) []string {
	out := make([]string, 0, len(docs))
	for _, d := range docs {
		out = append(out, string(d.ChaveAcesso))
	}
	return out
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// seedFixtures stores a resumo cancelada (2026-08), a completa with the
// company's ciência and a completa denegada (both 2026-09).
func (e *nfeTestEnv) seedFixtures() {
	e.t.Helper()
	e.seed("resnfe-cancelada.xml", 1)
	e.seed("procnfe.xml", 2)
	e.seed("proceventonfe-ciencia.xml", 3)
	e.seed("procnfe-denegada.xml", 4)
}

func TestNFeListDocumentsFilters(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	ctx := context.Background()

	tests := []struct {
		name string
		in   NFeListInput
		want []string
	}{
		{"all, newest first", NFeListInput{}, []string{nfeChaveDenegada, nfeChaveProc, nfeChaveCancelada}},
		{"competence", NFeListInput{Competence: "2026-08"}, []string{nfeChaveCancelada}},
		{"situacao", NFeListInput{Situacao: "denegada"}, []string{nfeChaveDenegada}},
		{"completeness", NFeListInput{Completeness: "resumo"}, []string{nfeChaveCancelada}},
		{"role", NFeListInput{Role: "emitente"}, []string{}},
		{"manifestacao", NFeListInput{Manifestacao: "ciencia"}, []string{nfeChaveProc}},
		{"chaves", NFeListInput{ChavesAcesso: []string{nfeChaveCancelada}}, []string{nfeChaveCancelada}},
		{"limit", NFeListInput{Limit: 1}, []string{nfeChaveDenegada}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.CNPJ = nfeTestCNPJ
			docs, err := env.app.NFe.ListDocuments(ctx, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got := chavesOf(docs); !slices.Equal(got, tc.want) {
				t.Errorf("chaves = %v, want %v", got, tc.want)
			}
		})
	}

	if _, err := env.app.NFe.ListDocuments(ctx, NFeListInput{CNPJ: nfeTestCNPJ, Situacao: "rascunho"}); err == nil {
		t.Error("an invalid situação was accepted")
	}

	events, err := env.app.NFe.ListEvents(ctx, nfeTestCNPJ, nfeChaveProc)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].TpEvento != nfe.TpEventoCiencia {
		t.Errorf("events = %+v, want the ciência", events)
	}
	if _, err := env.app.NFe.ListEvents(ctx, nfeTestCNPJ, testChave(t, 7)); err == nil {
		t.Error("ListEvents of a chave the company does not see succeeded")
	}
}

func TestNFeStatusCounts(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	env.seedResumo(10, "2026-08-20T10:00:00-03:00", "11222333000181")
	ctx := context.Background()

	until := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	if err := env.app.NFe.SyncRepo.SetBlockedUntil(ctx, env.company.ID, nfse.SyncSourceNFe, until, nfse.SyncStopReasonConsumoIndevido); err != nil {
		t.Fatal(err)
	}

	status, err := env.app.NFe.Status(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	want := NFeStatusResult{
		CompanyName:       "Empresa Mock",
		CNPJ:              nfeTestCNPJ,
		UF:                "SP",
		TpAmb:             "1",
		NextAllowedAt:     &until,
		BlockedReason:     string(nfse.SyncStopReasonConsumoIndevido),
		RequestBudget:     20,
		TotalDestinatario: 4,
		TotalResumos:      2,
		TotalCompletas:    2,
		PendingCiencia:    1,
		PendingConclusiva: 1,
		CienciaOverdue:    1,
	}
	if status.NextAllowedAt == nil || !status.NextAllowedAt.Equal(until) {
		t.Errorf("NextAllowedAt = %v, want %v", status.NextAllowedAt, until)
	}
	status.NextAllowedAt = want.NextAllowedAt
	if status != want {
		t.Errorf("Status =\n%+v\nwant\n%+v", status, want)
	}

	_, err = env.app.NFe.Pull(ctx, nfeTestCNPJ)
	if !errors.Is(err, ErrSourceBlocked) {
		t.Errorf("Pull while blocked = %v, want ErrSourceBlocked", err)
	}
	if len(env.passwords.requests) != 0 {
		t.Errorf("password prompts = %d, want 0", len(env.passwords.requests))
	}
}

func TestNFeListPendingManifestationsOrderAndFlags(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	early := env.seedResumo(10, "2026-08-20T10:00:00-03:00", "11222333000181")
	env.seedResumo(11, "2026-08-21T10:00:00-03:00", nfeTestCNPJ) // emitted by the company: not pending
	ctx := context.Background()

	env.app.NFe.now = func() time.Time { return mustTime(t, "2026-09-15T12:00:00-03:00") }
	pending, err := env.app.NFe.ListPendingManifestations(ctx, NFePendingInput{CNPJ: nfeTestCNPJ})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || string(pending[0].ChaveAcesso) != early || pending[1].ChaveAcesso != nfeChaveProc {
		t.Fatalf("pending = %+v, want the 08-20 resumo then the ciência", pending)
	}

	first := pending[0]
	if first.Kind != NFePendingSemCiencia || !first.CienciaOverdue || first.TacitlyConfirmed || first.DaysLeft != 64 {
		t.Errorf("first = kind %s, overdue %t, tacit %t, days %d; want sem_ciencia, true, false, 64",
			first.Kind, first.CienciaOverdue, first.TacitlyConfirmed, first.DaysLeft)
	}
	if want := mustTime(t, "2026-11-18T10:00:00-03:00"); !first.ConclusiveDue.Equal(want) {
		t.Errorf("ConclusiveDue = %s, want %s", first.ConclusiveDue, want)
	}
	second := pending[1]
	if second.Kind != NFePendingSemConclusiva || second.CienciaOverdue || second.Manifestacao != "ciencia" {
		t.Errorf("second = kind %s, overdue %t, manifestacao %s; want sem_conclusiva, false, ciencia",
			second.Kind, second.CienciaOverdue, second.Manifestacao)
	}

	due, err := env.app.NFe.ListPendingManifestations(ctx, NFePendingInput{CNPJ: nfeTestCNPJ, DueWithinDays: 70})
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || string(due[0].ChaveAcesso) != early {
		t.Errorf("due within 70 days = %+v, want only the 08-20 resumo", due)
	}

	env.app.NFe.now = func() time.Time { return mustTime(t, "2026-12-01T12:00:00-03:00") }
	expired, err := env.app.NFe.ListPendingManifestations(ctx, NFePendingInput{CNPJ: nfeTestCNPJ})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range expired {
		if !p.TacitlyConfirmed || p.DaysLeft >= 0 {
			t.Errorf("%s: tacit %t, days %d; want tacitly confirmed with negative days", p.ChaveAcesso, p.TacitlyConfirmed, p.DaysLeft)
		}
	}
}

func zipEntries(t *testing.T, path string) map[string]string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	entries := make(map[string]string, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[f.Name] = string(data)
	}
	return entries
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func TestNFeExportXMLZipLayoutAndIncrementalMarks(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	env.seed("resevento-cancelamento.xml", 5) // a resumo event: never exported
	ctx := context.Background()
	outDir := t.TempDir()

	first := filepath.Join(outDir, "nfe.zip")
	res, err := env.app.NFe.ExportXMLZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: first})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 2 || res.SkippedResumos != 1 || res.OutPath != first || res.Format != "xml" {
		t.Errorf("result = %+v, want 2 exported and 1 resumo skipped", res)
	}
	entries := zipEntries(t, first)
	procEntry := "2026-09/destinatario/" + nfeChaveProc + "-procNFe.xml"
	want := []string{
		procEntry,
		"2026-09/destinatario/" + nfeChaveDenegada + "-procNFe.xml",
		"2026-09/destinatario/eventos/" + nfeChaveProc + "-210210-1.xml",
	}
	if got := sortedKeys(entries); !slices.Equal(got, want) {
		t.Fatalf("zip entries = %v, want %v", got, want)
	}
	if entries[procEntry] != readNFeFixture(t, "procnfe.xml") {
		t.Error("procNFe entry is not the stored XML")
	}
	if _, err := os.Stat(strings.TrimSuffix(first, ".zip") + ".tmp.zip"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind: %v", err)
	}

	second := filepath.Join(outDir, "again.zip")
	res, err = env.app.NFe.ExportXMLZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: second})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 0 || res.OutPath != "" || res.SkippedResumos != 1 {
		t.Errorf("second incremental = %+v, want nothing exported", res)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Errorf("an empty export wrote %s", second)
	}

	withResumos := filepath.Join(outDir, "resumos.zip")
	res, err = env.app.NFe.ExportXMLZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, IncludeResumos: true, OutPath: withResumos})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 1 || res.SkippedResumos != 0 {
		t.Errorf("with resumos = %+v, want the resumo only", res)
	}
	if got := sortedKeys(zipEntries(t, withResumos)); !slices.Equal(got, []string{"2026-08/destinatario/" + nfeChaveCancelada + "-resNFe.xml"}) {
		t.Errorf("resumo zip entries = %v", got)
	}

	single := filepath.Join(outDir, "single.xml")
	if err := env.app.NFe.ExportXML(ctx, NFeExportXMLInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveDenegada, OutPath: single}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(single) // #nosec G304 -- test temp dir.
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != readNFeFixture(t, "procnfe-denegada.xml") {
		t.Error("single export is not the stored XML")
	}
}

func TestNFeTestConnectionOnlyChecksTLS(t *testing.T) {
	env := newNFeTestEnv(t)
	stub := useFakeSEFAZ(t)

	result, err := env.app.NFe.TestConnection(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if !result.CertLoaded || !result.EndpointReached || stub.tlsCalls != 1 || len(stub.lotes) != 0 {
		t.Errorf("result = %+v, TLS checks = %d, lotes = %d", result, stub.tlsCalls, len(stub.lotes))
	}
	if len(env.passwords.requests) != 1 || env.passwords.requests[0].Purpose != "Teste de conexão NF-e" {
		t.Errorf("password requests = %+v", env.passwords.requests)
	}

	stub.tlsErr = errors.New("handshake failed")
	result, err = env.app.NFe.TestConnection(context.Background(), nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if result.EndpointReached || !strings.Contains(result.StatusExplanation, "handshake failed") {
		t.Errorf("failed check result = %+v", result)
	}
}

func TestNFeListDocumentsDerivesManifestationState(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	ctx := context.Background()

	list := func() map[string]NFeDocument {
		t.Helper()
		docs, err := env.app.NFe.ListDocuments(ctx, NFeListInput{CNPJ: nfeTestCNPJ})
		if err != nil {
			t.Fatal(err)
		}
		byChave := make(map[string]NFeDocument, len(docs))
		for _, d := range docs {
			byChave[string(d.ChaveAcesso)] = d
		}
		return byChave
	}

	env.app.NFe.now = func() time.Time { return mustTime(t, "2026-09-15T12:00:00-03:00") }
	docs := list()
	tests := []struct {
		chave               string
		ciencia, conclusive string
	}{
		{nfeChaveProc, "já manifestada (ciencia)", ""},
		{nfeChaveCancelada, "NF-e cancelada", "NF-e cancelada"},
		{nfeChaveDenegada, "NF-e denegada", "NF-e denegada"},
	}
	for _, tc := range tests {
		d := docs[tc.chave]
		if d.CienciaBlockReason != tc.ciencia || d.ConclusiveBlockReason != tc.conclusive {
			t.Errorf("%s: block reasons = %q, %q; want %q, %q", tc.chave, d.CienciaBlockReason, d.ConclusiveBlockReason, tc.ciencia, tc.conclusive)
		}
	}
	proc := docs[nfeChaveProc]
	if proc.ConclusiveDue.IsZero() || proc.DaysLeft <= 0 || proc.TacitlyConfirmed {
		t.Errorf("ciência row = due %s, days %d, tacit %t; want a future deadline", proc.ConclusiveDue, proc.DaysLeft, proc.TacitlyConfirmed)
	}

	env.app.NFe.now = func() time.Time { return mustTime(t, "2026-12-15T12:00:00-03:00") }
	proc = list()[nfeChaveProc]
	if proc.DaysLeft >= 0 || !proc.TacitlyConfirmed || proc.ConclusiveBlockReason != "" {
		t.Errorf("ciência row after the deadline = days %d, tacit %t, reason %q; want tacitly confirmed and still manifestable",
			proc.DaysLeft, proc.TacitlyConfirmed, proc.ConclusiveBlockReason)
	}
}

func TestDaysLeftCountsCalendarDays(t *testing.T) {
	brt := time.FixedZone("BRT", -3*3600)
	now := time.Date(2026, 9, 23, 0, 5, 0, 0, brt)
	tests := []struct {
		due  time.Time
		want int
	}{
		{time.Date(2026, 9, 23, 23, 59, 0, 0, brt), 0},
		{time.Date(2026, 9, 24, 0, 1, 0, 0, brt), 1},
		{time.Date(2026, 9, 22, 23, 59, 0, 0, brt), -1},
		{time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC), 0}, // 22:00 on the 23rd in BRT
		{time.Date(2026, 10, 3, 12, 0, 0, 0, brt), 10},
	}
	for _, tc := range tests {
		if got := daysLeft(tc.due, now); got != tc.want {
			t.Errorf("daysLeft(%s) = %d, want %d", tc.due, got, tc.want)
		}
	}
}
