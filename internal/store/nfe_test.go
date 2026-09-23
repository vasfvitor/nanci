package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

const (
	nfeKeyProc      = "35260911222333000181550010000012341123456787"
	nfeKeyCancelada = "35260911222333000181550010000012351234567894"
	nfeKeyDenegada  = "35260911222333000181550010000012361345678900"

	cnpjMock       = "70860312000150"
	cnpjEmitente   = "11222333000181"
	cnpjAutorizado = "45678901000175"
)

// nfeFixture holds a migrated database with three companies: the
// destinatário of the fixtures, their emitente and an autXML party.
type nfeFixture struct {
	t    *testing.T
	db   *sql.DB
	repo *store.NFeRepository
}

func newNFeFixture(t *testing.T) *nfeFixture {
	t.Helper()
	db := storetest.OpenTestDB(t)
	const now = "2026-09-01T00:00:00Z"
	for _, c := range []struct{ id, cnpj string }{
		{"mock", cnpjMock},
		{"emitente", cnpjEmitente},
		{"autorizado", cnpjAutorizado},
	} {
		mustExec(t, db, `
			INSERT INTO companies (id, cnpj, cnpj_root, name, environment, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'producao', ?, ?)
		`, c.id, c.cnpj, c.cnpj[:8], c.id, now, now)
	}
	return &nfeFixture{t: t, db: db, repo: store.NewNFeRepository(db)}
}

func (f *nfeFixture) readFixture(name string) string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", name)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

// resNFe parses a resNFe fixture after applying the replacements (old, new
// pairs) and gives it rawHash.
func (f *nfeFixture) resNFe(name, rawHash string, replacements ...string) nfe.Document {
	f.t.Helper()
	data := strings.NewReplacer(replacements...).Replace(f.readFixture(name))
	doc, err := nfe.ParseResNFe([]byte(data))
	if err != nil {
		f.t.Fatalf("parse %s: %v", name, err)
	}
	doc.RawHash = rawHash
	return doc
}

func (f *nfeFixture) procNFe(name, rawHash string) nfe.Document {
	f.t.Helper()
	doc, err := nfe.ParseProcNFe([]byte(f.readFixture(name)))
	if err != nil {
		f.t.Fatalf("parse %s: %v", name, err)
	}
	doc.RawHash = rawHash
	return doc
}

func (f *nfeFixture) procEvento(name, rawHash string) nfe.Event {
	f.t.Helper()
	ev, err := nfe.ParseProcEventoNFe([]byte(f.readFixture(name)))
	if err != nil {
		f.t.Fatalf("parse %s: %v", name, err)
	}
	ev.RawHash = rawHash
	return ev
}

func (f *nfeFixture) applyDocument(companyID, companyCNPJ string, doc nfe.Document, nsu int64) bool {
	f.t.Helper()
	var inserted bool
	f.inTx(func(tx *sql.Tx) error {
		var err error
		inserted, err = f.repo.ApplyDocumentTx(context.Background(), tx, store.ApplyNFeDocumentParams{
			Document:    doc,
			CompanyID:   nfse.CompanyID(companyID),
			CompanyCNPJ: companyCNPJ,
			NSU:         nsu,
		})
		return err
	})
	return inserted
}

func (f *nfeFixture) applyEvent(ev nfe.Event) bool {
	f.t.Helper()
	var inserted bool
	f.inTx(func(tx *sql.Tx) error {
		var err error
		inserted, err = f.repo.ApplyEventTx(context.Background(), tx, store.ApplyNFeEventParams{Event: ev})
		return err
	})
	return inserted
}

func (f *nfeFixture) inTx(fn func(tx *sql.Tx) error) {
	f.t.Helper()
	tx, err := f.db.BeginTx(context.Background(), nil)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		f.t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		f.t.Fatal(err)
	}
}

func (f *nfeFixture) companyDocument(companyID, chave string) nfe.CompanyDocument {
	f.t.Helper()
	doc, err := f.repo.CompanyDocumentByChave(context.Background(), nfse.CompanyID(companyID), chave)
	if err != nil {
		f.t.Fatalf("CompanyDocumentByChave(%s, %s): %v", companyID, chave, err)
	}
	return *doc
}

func (f *nfeFixture) events(chave string) []nfe.Event {
	f.t.Helper()
	events, err := f.repo.ListEventsByChave(context.Background(), chave)
	if err != nil {
		f.t.Fatal(err)
	}
	return events
}

func (f *nfeFixture) record(items ...nfe.ManifestationRecord) {
	f.t.Helper()
	if err := f.repo.RecordManifestations(context.Background(), items); err != nil {
		f.t.Fatalf("RecordManifestations: %v", err)
	}
}

func (f *nfeFixture) list(companyID string, filter nfe.DocumentFilter) []string {
	f.t.Helper()
	docs, err := f.repo.ListCompanyDocuments(context.Background(), nfse.CompanyID(companyID), filter)
	if err != nil {
		f.t.Fatalf("ListCompanyDocuments: %v", err)
	}
	return chaves(docs)
}

func chaves(docs []nfe.CompanyDocument) []string {
	out := []string{}
	for _, d := range docs {
		out = append(out, string(d.ChaveAcesso))
	}
	return out
}

func manifestation(tpEvento, status string, registeredAt time.Time) nfe.ManifestationRecord {
	return nfe.ManifestationRecord{
		CompanyID:         "mock",
		CompanyCNPJ:       "70.860.312/0001-50",
		IDLote:            "1",
		ChaveAcesso:       nfeKeyProc,
		TpEvento:          tpEvento,
		NSeqEvento:        1,
		EventAt:           &registeredAt,
		Status:            status,
		CStat:             "135",
		XMotivo:           "Evento registrado e vinculado a NF-e",
		Protocolo:         "891260000000099",
		RegisteredAt:      &registeredAt,
		ProcEventoRawHash: "own-" + tpEvento,
	}
}

func TestNFeResumoUpgradesToCompleta(t *testing.T) {
	f := newNFeFixture(t)

	if !f.applyDocument("mock", cnpjMock, f.resNFe("resnfe.xml", "hash-resumo"), 10) {
		t.Error("first resumo should be inserted")
	}
	resumo := f.companyDocument("mock", nfeKeyProc)
	if resumo.Completeness != nfe.CompletenessResumo || resumo.VisibilityReason != nfe.VisibilityReasonResumoDestinatario {
		t.Fatalf("resumo row = (%s, %s)", resumo.Completeness, resumo.VisibilityReason)
	}

	if f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 12) {
		t.Error("upgrade should not count as a new company document")
	}
	got := f.companyDocument("mock", nfeKeyProc)
	if got.ID != resumo.ID || got.RelationID != resumo.RelationID {
		t.Errorf("ids changed: document %s -> %s, relation %s -> %s", resumo.ID, got.ID, resumo.RelationID, got.RelationID)
	}
	if got.Completeness != nfe.CompletenessCompleta || got.RawHash != "hash-completa" || got.ResumoRawHash != "hash-resumo" {
		t.Errorf("completeness/hashes = (%s, %s, %s)", got.Completeness, got.RawHash, got.ResumoRawHash)
	}
	if got.CompanyRole != nfe.CompanyRoleDestinatario || got.VisibilityReason != nfe.VisibilityReasonExactDestinatario {
		t.Errorf("participation = (%s, %s), want exact destinatário", got.CompanyRole, got.VisibilityReason)
	}
	if got.DestinatarioName != "EMPRESA MOCK LTDA" || got.NatOp != "VENDA DE MERCADORIA" {
		t.Errorf("completa fields missing: %+v", got.Document)
	}
	if *got.FirstSeenNSU != 10 || *got.LastSeenNSU != 12 {
		t.Errorf("NSUs = (%d, %d), want (10, 12)", *got.FirstSeenNSU, *got.LastSeenNSU)
	}
	if got.TotalValue.Cents() != 125050 || got.AuthorizedAt == nil || got.IssueDate.IsZero() {
		t.Errorf("value/dates = (%d, %v, %s)", got.TotalValue.Cents(), got.AuthorizedAt, got.IssueDate)
	}
}

func TestNFeLateResumoDoesNotDowngrade(t *testing.T) {
	f := newNFeFixture(t)
	completa := f.procNFe("procnfe.xml", "hash-completa")
	f.applyDocument("mock", cnpjMock, completa, 10)
	f.applyDocument("autorizado", cnpjAutorizado, completa, 3)

	// The late summary reports the NF-e as cancelled.
	lateResumo := f.resNFe("resnfe.xml", "hash-resumo", "<cSitNFe>1</cSitNFe>", "<cSitNFe>3</cSitNFe>")
	f.applyDocument("mock", cnpjMock, lateResumo, 20)
	f.applyDocument("autorizado", cnpjAutorizado, lateResumo, 4)

	got := f.companyDocument("mock", nfeKeyProc)
	if got.Completeness != nfe.CompletenessCompleta || got.RawHash != "hash-completa" || got.ResumoRawHash != "hash-resumo" {
		t.Errorf("completeness/hashes = (%s, %s, %s)", got.Completeness, got.RawHash, got.ResumoRawHash)
	}
	if got.Situacao != nfe.SituacaoCancelada {
		t.Errorf("situacao = %s, want cancelada", got.Situacao)
	}
	if got.NatOp != "VENDA DE MERCADORIA" || got.VisibilityReason != nfe.VisibilityReasonExactDestinatario {
		t.Errorf("completa data lost: natOp %q, visibility %s", got.NatOp, got.VisibilityReason)
	}
	// The autXML party is only known from the completa, so it must survive
	// the reclassification done for the late resumo.
	if role := f.companyDocument("autorizado", nfeKeyProc).CompanyRole; role != nfe.CompanyRoleAutorizado {
		t.Errorf("autXML company role = %s, want autorizado", role)
	}
}

func TestNFeCancelamentoMakesDocumentCancelada(t *testing.T) {
	autorizada := func(f *nfeFixture) nfe.Document {
		return f.resNFe("resnfe-cancelada.xml", "hash-resumo", "<cSitNFe>3</cSitNFe>", "<cSitNFe>1</cSitNFe>")
	}

	t.Run("event after document", func(t *testing.T) {
		f := newNFeFixture(t)
		f.applyDocument("mock", cnpjMock, autorizada(f), 1)
		if s := f.companyDocument("mock", nfeKeyCancelada).Situacao; s != nfe.SituacaoAutorizada {
			t.Fatalf("situacao before the event = %s", s)
		}
		if !f.applyEvent(f.procEvento("proceventonfe-cancelamento.xml", "hash-cancel")) {
			t.Error("event should be inserted")
		}
		got := f.companyDocument("mock", nfeKeyCancelada)
		if got.Situacao != nfe.SituacaoCancelada || got.EventCount != 1 {
			t.Errorf("(situacao, events) = (%s, %d), want (cancelada, 1)", got.Situacao, got.EventCount)
		}
	})

	t.Run("event before document", func(t *testing.T) {
		f := newNFeFixture(t)
		if !f.applyEvent(f.procEvento("proceventonfe-cancelamento.xml", "hash-cancel")) {
			t.Error("event should be inserted")
		}
		f.applyDocument("mock", cnpjMock, autorizada(f), 1)
		got := f.companyDocument("mock", nfeKeyCancelada)
		if got.Situacao != nfe.SituacaoCancelada || got.EventCount != 1 {
			t.Errorf("(situacao, events) = (%s, %d), want (cancelada, 1)", got.Situacao, got.EventCount)
		}
		var linked string
		if err := f.db.QueryRowContext(context.Background(), `SELECT nfe_document_id FROM nfe_events WHERE chave_acesso = ?`, nfeKeyCancelada).Scan(&linked); err != nil {
			t.Fatal(err)
		}
		if linked != got.ID {
			t.Errorf("event linked to %q, want %q", linked, got.ID)
		}
	})
}

func TestNFeOwnCienciaCollapsesWithDistributedCopy(t *testing.T) {
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)

	registeredAt := time.Date(2026, 9, 2, 10, 0, 5, 0, time.FixedZone("-03", -3*3600))
	f.record(manifestation(nfe.TpEventoCiencia, nfe.ManifestationStatusRegistrada, registeredAt))

	got := f.companyDocument("mock", nfeKeyProc)
	if got.Manifestacao != nfe.ManifestacaoCiencia {
		t.Fatalf("manifestacao after own ciência = %s", got.Manifestacao)
	}
	if got.ManifestacaoAt == nil || !got.ManifestacaoAt.Equal(registeredAt) {
		t.Errorf("ManifestacaoAt = %v, want %s", got.ManifestacaoAt, registeredAt)
	}

	if f.applyEvent(f.procEvento("proceventonfe-ciencia.xml", "hash-distributed")) {
		t.Error("distributed copy of our ciência should not be a new event")
	}
	// A later summary of the same event must not replace the full copy.
	resumo := f.procEvento("proceventonfe-ciencia.xml", "hash-resumo")
	resumo.Completeness = nfe.CompletenessResumo
	f.applyEvent(resumo)

	events := f.events(nfeKeyProc)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	e := events[0]
	if !e.SentByNanci || e.RawHash != "hash-distributed" || e.Completeness != nfe.CompletenessCompleta || e.Protocolo != "891260000000001" {
		t.Errorf("event = %+v, want the distributed copy flagged as sent by nanci", e)
	}
	if got := f.companyDocument("mock", nfeKeyProc); got.Manifestacao != nfe.ManifestacaoCiencia || got.EventCount != 1 {
		t.Errorf("(manifestacao, events) = (%s, %d), want (ciencia, 1)", got.Manifestacao, got.EventCount)
	}

	var attempts int
	if err := f.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM nfe_manifestations WHERE company_id = 'mock'`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Errorf("nfe_manifestations rows = %d, want 1", attempts)
	}
}

func TestNFeManifestationKeepsTpAmb(t *testing.T) {
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)

	sent := manifestation(nfe.TpEventoCiencia, nfe.ManifestationStatusRegistrada, time.Now())
	sent.TpAmb = "2"
	f.record(sent)

	var tpAmb string
	if err := f.db.QueryRowContext(context.Background(), `SELECT tp_amb FROM nfe_manifestations WHERE company_id = 'mock'`).Scan(&tpAmb); err != nil {
		t.Fatal(err)
	}
	if tpAmb != "2" {
		t.Errorf("tp_amb = %q, want 2", tpAmb)
	}
}

func TestNFeConclusiveAfterCiencia(t *testing.T) {
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)
	f.applyEvent(f.procEvento("proceventonfe-ciencia.xml", "hash-ciencia"))

	confirmedAt := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	rejected := manifestation(nfe.TpEventoDesconhecimento, nfe.ManifestationStatusRejeitada, confirmedAt.Add(time.Hour))
	rejected.CStat, rejected.XMotivo, rejected.RegisteredAt = "596", "Rejeicao: prazo", nil
	f.record(
		manifestation(nfe.TpEventoConfirmacao, nfe.ManifestationStatusRegistrada, confirmedAt),
		rejected,
	)

	got := f.companyDocument("mock", nfeKeyProc)
	if got.Manifestacao != nfe.ManifestacaoConfirmada {
		t.Errorf("manifestacao = %s, want confirmada", got.Manifestacao)
	}
	if got.ManifestacaoAt == nil || !got.ManifestacaoAt.Equal(confirmedAt) {
		t.Errorf("ManifestacaoAt = %v, want %s", got.ManifestacaoAt, confirmedAt)
	}
	if got.EventCount != 2 {
		t.Errorf("events = %d, want 2 (the rejected attempt is not an event)", got.EventCount)
	}
}

func TestNFeAlreadyRegisteredManifestation(t *testing.T) {
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)
	f.applyEvent(f.procEvento("proceventonfe-ciencia.xml", "hash-distributed"))

	dup := manifestation(nfe.TpEventoCiencia, nfe.ManifestationStatusJaRegistrada, time.Now())
	dup.CStat, dup.Protocolo, dup.RegisteredAt, dup.ProcEventoRawHash = "573", "", nil, ""
	f.record(dup)

	events := f.events(nfeKeyProc)
	if len(events) != 1 || events[0].RawHash != "hash-distributed" || events[0].SentByNanci {
		t.Errorf("an already-registered answer must not touch the stored event: %+v", events)
	}

	// Without a stored copy, the duplicate answer still records the ciência.
	g := newNFeFixture(t)
	g.applyDocument("mock", cnpjMock, g.procNFe("procnfe.xml", "hash-completa"), 1)
	g.record(dup)
	if m := g.companyDocument("mock", nfeKeyProc).Manifestacao; m != nfe.ManifestacaoCiencia {
		t.Errorf("manifestacao after already-registered answer = %s, want ciencia", m)
	}
}

// A ciência answered with 655 (conclusive manifestação already at SEFAZ) is
// recorded as a rejected attempt and must not look like a registered event.
func TestNFeCienciaAfterConclusiveIsNotAnEvent(t *testing.T) {
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)

	late := manifestation(nfe.TpEventoCiencia, nfe.ManifestationStatusRejeitada, time.Now())
	late.CStat, late.XMotivo = "655", "Rejeicao: Ciencia da Operacao informada apos a manifestacao final"
	late.Protocolo, late.RegisteredAt, late.ProcEventoRawHash = "", nil, ""
	f.record(late)

	if events := f.events(nfeKeyProc); len(events) != 0 {
		t.Errorf("events = %+v, want none", events)
	}
	if got := f.companyDocument("mock", nfeKeyProc); got.Manifestacao != nfe.ManifestacaoNenhuma {
		t.Errorf("manifestacao = %s, want nenhuma", got.Manifestacao)
	}
	var status, cStat string
	err := f.db.QueryRowContext(context.Background(),
		`SELECT status, c_stat FROM nfe_manifestations WHERE company_id = 'mock'`).Scan(&status, &cStat)
	if err != nil {
		t.Fatal(err)
	}
	if status != nfe.ManifestationStatusRejeitada || cStat != "655" {
		t.Errorf("attempt = (%s, %s), want (rejeitada, 655)", status, cStat)
	}
}

// seedFilterDocuments stores, for company mock (destinatário):
//
//	procnfe.xml           completa  autorizada  2026-09-01
//	resnfe-cancelada.xml  resumo    cancelada   2026-08-31
//	procnfe-denegada.xml  completa  denegada    2026-09-05
//
// and procnfe.xml again for company emitente.
func seedFilterDocuments(f *nfeFixture) {
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-a"), 1)
	f.applyDocument("mock", cnpjMock, f.resNFe("resnfe-cancelada.xml", "hash-b"), 2)
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe-denegada.xml", "hash-c"), 3)
	f.applyDocument("emitente", cnpjEmitente, f.procNFe("procnfe.xml", "hash-a"), 7)
}

func TestNFeListFilters(t *testing.T) {
	f := newNFeFixture(t)
	seedFilterDocuments(f)
	f.record(manifestation(nfe.TpEventoCiencia, nfe.ManifestationStatusRegistrada, time.Now()))

	viewed, err := f.repo.MarkViewed(context.Background(), "mock", nfe.DocumentFilter{Competence: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if viewed != 1 {
		t.Errorf("MarkViewed = %d, want 1", viewed)
	}

	issueFloor := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	all := []string{nfeKeyDenegada, nfeKeyProc, nfeKeyCancelada}
	tests := []struct {
		name   string
		filter nfe.DocumentFilter
		want   []string
	}{
		{"no filter, newest first", nfe.DocumentFilter{}, all},
		{"competence", nfe.DocumentFilter{Competence: "2026-08"}, []string{nfeKeyCancelada}},
		{"situacao", nfe.DocumentFilter{Situacao: nfe.SituacaoDenegada}, []string{nfeKeyDenegada}},
		{"completeness", nfe.DocumentFilter{Completeness: nfe.CompletenessResumo}, []string{nfeKeyCancelada}},
		{"role", nfe.DocumentFilter{Role: nfe.CompanyRoleDestinatario}, all},
		{"role without match", nfe.DocumentFilter{Role: nfe.CompanyRoleEmitente}, []string{}},
		{"manifestacao", nfe.DocumentFilter{Manifestacao: nfe.ManifestacaoCiencia}, []string{nfeKeyProc}},
		{"emitente cnpj", nfe.DocumentFilter{EmitenteCNPJ: "11.222.333/0001-81"}, all},
		{"emitente cnpj without match", nfe.DocumentFilter{EmitenteCNPJ: cnpjMock}, []string{}},
		{"chaves", nfe.DocumentFilter{ChavesAcesso: []string{nfeKeyCancelada, nfeKeyDenegada}}, []string{nfeKeyDenegada, nfeKeyCancelada}},
		{"only unread", nfe.DocumentFilter{OnlyUnread: true}, []string{nfeKeyDenegada, nfeKeyProc}},
		{"issue date floor", nfe.DocumentFilter{IssueDateGTE: &issueFloor}, []string{nfeKeyDenegada}},
		{"issue date floor is inclusive", nfe.DocumentFilter{IssueDateGTE: new(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))}, []string{nfeKeyDenegada, nfeKeyProc}},
		{"pending manifestation", nfe.DocumentFilter{PendingManifestation: true}, []string{nfeKeyProc}},
		{"limit", nfe.DocumentFilter{Limit: 1}, []string{nfeKeyDenegada}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.list("mock", tt.filter); !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	if got := f.list("emitente", nfe.DocumentFilter{Role: nfe.CompanyRoleEmitente}); !slices.Equal(got, []string{nfeKeyProc}) {
		t.Errorf("emitente company rows = %v", got)
	}
}

func TestNFePendingManifestation(t *testing.T) {
	f := newNFeFixture(t)
	seedFilterDocuments(f)
	pending := nfe.DocumentFilter{PendingManifestation: true}

	// Autorizada without manifestação: pending. Cancelada and denegada: never.
	if got := f.list("mock", pending); !slices.Equal(got, []string{nfeKeyProc}) {
		t.Errorf("pending = %v, want only the autorizada", got)
	}
	// The emitente never manifests on its own NF-e.
	if got := f.list("emitente", pending); len(got) != 0 {
		t.Errorf("emitente pending = %v, want none", got)
	}
	// Ciência keeps it pending (conclusive still missing).
	f.applyEvent(f.procEvento("proceventonfe-ciencia.xml", "hash-ciencia"))
	if got := f.list("mock", pending); !slices.Equal(got, []string{nfeKeyProc}) {
		t.Errorf("pending after ciência = %v", got)
	}

	counts, err := f.repo.CountSummary(context.Background(), "mock")
	if err != nil {
		t.Fatal(err)
	}
	want := nfe.Counts{
		ByRole:            map[nfe.CompanyRole]int{nfe.CompanyRoleDestinatario: 3},
		Resumos:           1,
		Completas:         2,
		PendingCiencia:    0,
		PendingConclusiva: 1,
	}
	if counts.Resumos != want.Resumos || counts.Completas != want.Completas ||
		counts.PendingCiencia != want.PendingCiencia || counts.PendingConclusiva != want.PendingConclusiva ||
		len(counts.ByRole) != 1 || counts.ByRole[nfe.CompanyRoleDestinatario] != 3 {
		t.Errorf("CountSummary = %+v, want %+v", counts, want)
	}

	// A conclusive manifestação ends the pendency.
	f.record(manifestation(nfe.TpEventoConfirmacao, nfe.ManifestationStatusRegistrada, time.Now()))
	if got := f.list("mock", pending); len(got) != 0 {
		t.Errorf("pending after confirmação = %v, want none", got)
	}
}

func TestNFeCountSummaryEmitente(t *testing.T) {
	f := newNFeFixture(t)
	seedFilterDocuments(f)
	counts, err := f.repo.CountSummary(context.Background(), "emitente")
	if err != nil {
		t.Fatal(err)
	}
	if counts.ByRole[nfe.CompanyRoleEmitente] != 1 || counts.PendingCiencia != 0 || counts.Completas != 1 {
		t.Errorf("CountSummary = %+v", counts)
	}
}

func TestNFeExportMarks(t *testing.T) {
	ctx := context.Background()
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.resNFe("resnfe.xml", "hash-resumo"), 1)

	pending := func() []nfe.CompanyDocument {
		t.Helper()
		docs, err := f.repo.ListPendingExport(ctx, "mock", nfe.DocumentFilter{}, nfe.ExportKindXML)
		if err != nil {
			t.Fatal(err)
		}
		return docs
	}

	docs := pending()
	if !slices.Equal(chaves(docs), []string{nfeKeyProc}) {
		t.Fatalf("pending before export = %v", chaves(docs))
	}
	if err := f.repo.MarkExported(ctx, "mock", nfe.ExportKindXML, docs); err != nil {
		t.Fatal(err)
	}
	if docs := pending(); len(docs) != 0 {
		t.Errorf("pending after export = %v, want none", chaves(docs))
	}

	// The upgrade to completa changes the raw hash, so it is pending again.
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 2)
	docs = pending()
	if len(docs) != 1 || docs[0].RawHash != "hash-completa" {
		t.Fatalf("pending after upgrade = %+v", docs)
	}
	if err := f.repo.MarkExported(ctx, "mock", nfe.ExportKindXML, docs); err != nil {
		t.Fatal(err)
	}
	if docs := pending(); len(docs) != 0 {
		t.Errorf("pending after second export = %v, want none", chaves(docs))
	}

	if _, err := f.repo.ListPendingExport(ctx, "mock", nfe.DocumentFilter{}, ""); err == nil {
		t.Error("ListPendingExport without a kind should fail")
	}
}

func TestNFeCompanyDocumentLookup(t *testing.T) {
	ctx := context.Background()
	f := newNFeFixture(t)
	f.applyDocument("mock", cnpjMock, f.resNFe("resnfe.xml", "hash-resumo"), 1)

	exists, err := f.repo.CompanyDocumentExists(ctx, "mock", nfeKeyProc)
	if err != nil || !exists {
		t.Errorf("CompanyDocumentExists(mock) = %v, %v", exists, err)
	}
	exists, err = f.repo.CompanyDocumentExists(ctx, "emitente", nfeKeyProc)
	if err != nil || exists {
		t.Errorf("CompanyDocumentExists(emitente) = %v, %v", exists, err)
	}
	if _, err := f.repo.CompanyDocumentByChave(ctx, "emitente", nfeKeyProc); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("CompanyDocumentByChave(emitente) error = %v, want ErrNotFound", err)
	}
}

func TestNFeListEventsByChaves(t *testing.T) {
	f := newNFeFixture(t)
	f.applyEvent(f.procEvento("proceventonfe-ciencia.xml", "hash-ciencia"))
	f.applyEvent(f.procEvento("proceventonfe-cce.xml", "hash-cce"))
	f.applyEvent(f.procEvento("proceventonfe-cancelamento.xml", "hash-cancel"))
	f.applyEvent(f.procEvento("proceventonfe-ciencia-real.xml", "hash-other"))
	ctx := context.Background()

	got, err := f.repo.ListEventsByChaves(ctx, []string{nfeKeyProc, nfeKeyCancelada, nfeKeyDenegada})
	if err != nil {
		t.Fatal(err)
	}
	// Grouped by chave, each group equal to ListEventsByChave.
	want := append(f.events(nfeKeyProc), f.events(nfeKeyCancelada)...)
	if len(want) != 3 || len(got) != len(want) {
		t.Fatalf("got %d events, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].RawHash != want[i].RawHash {
			t.Errorf("event %d = %s %s, want %s %s", i, got[i].ChaveAcesso, got[i].RawHash, want[i].ChaveAcesso, want[i].RawHash)
		}
	}

	none, err := f.repo.ListEventsByChaves(ctx, nil)
	if err != nil || len(none) != 0 {
		t.Errorf("ListEventsByChaves(nil) = %v, %v; want none", none, err)
	}
}

func TestNFeResetCompany(t *testing.T) {
	ctx := context.Background()
	f := newNFeFixture(t)
	// Only mock sees nfeKeyProc; mock and its emitente both see nfeKeyCancelada.
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)
	f.applyDocument("mock", cnpjMock, f.resNFe("resnfe-cancelada.xml", "hash-cancelada"), 2)
	f.applyDocument("emitente", cnpjEmitente, f.resNFe("resnfe-cancelada.xml", "hash-cancelada"), 1)
	f.applyEvent(f.procEvento("proceventonfe-ciencia.xml", "hash-ciencia")) // mock's own, on nfeKeyProc
	f.applyEvent(f.procEvento("proceventonfe-cce.xml", "hash-cce"))         // by the emitente, on nfeKeyProc
	f.applyEvent(f.procEvento("proceventonfe-cancelamento.xml", "hash-canc"))
	confirmation := manifestation(nfe.TpEventoConfirmacao, nfe.ManifestationStatusRegistrada, time.Now())
	orphan := manifestation(nfe.TpEventoCiencia, nfe.ManifestationStatusRegistrada, time.Now())
	orphan.ChaveAcesso = nfeKeyDenegada // mock's own event without a document
	f.record(confirmation, orphan)
	docs, err := f.repo.ListCompanyDocuments(ctx, "mock", nfe.DocumentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.MarkExported(ctx, "mock", nfe.ExportKindXML, docs); err != nil {
		t.Fatal(err)
	}

	want := nfe.ResetCounts{
		CompanyDocuments:   2,
		Documents:          1, // nfeKeyCancelada stays for the emitente
		Events:             3, // ciência and confirmação on nfeKeyProc, and the orphan ciência
		ExportMarks:        2,
		ManifestationsKept: 2,
	}
	preview, err := f.repo.PreviewResetCompany(ctx, "mock")
	if err != nil {
		t.Fatal(err)
	}
	if preview != want {
		t.Errorf("PreviewResetCompany = %+v, want %+v", preview, want)
	}
	if got := f.list("mock", nfe.DocumentFilter{}); len(got) != 2 {
		t.Fatalf("documents after preview = %v, want both kept", got)
	}

	got, err := f.repo.ResetCompany(ctx, "mock")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("ResetCompany = %+v, want %+v", got, want)
	}

	if docs := f.list("mock", nfe.DocumentFilter{}); len(docs) != 0 {
		t.Errorf("mock documents after reset = %v, want none", docs)
	}
	if docs := f.list("emitente", nfe.DocumentFilter{}); !slices.Equal(docs, []string{nfeKeyCancelada}) {
		t.Errorf("emitente documents after reset = %v, want the shared note", docs)
	}
	if got := f.companyDocument("emitente", nfeKeyCancelada).Situacao; got != nfe.SituacaoCancelada {
		t.Errorf("shared note situação = %s, want cancelada", got)
	}
	if events := f.events(nfeKeyDenegada); len(events) != 0 {
		t.Errorf("orphan own events after reset = %+v", events)
	}
	// The CC-e belongs to another registered company: kept, unlinked.
	events := f.events(nfeKeyProc)
	if len(events) != 1 || events[0].TpEvento != "110110" {
		t.Fatalf("events of the removed note = %+v, want only the emitente's CC-e", events)
	}
	var linked int
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nfe_events WHERE chave_acesso = ? AND nfe_document_id IS NOT NULL`, nfeKeyProc).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != 0 {
		t.Errorf("events still linked to the removed document = %d", linked)
	}

	var manifestations, marks int
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nfe_manifestations WHERE company_id = 'mock'`).Scan(&manifestations); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM company_nfe_export_marks WHERE company_id = 'mock'`).Scan(&marks); err != nil {
		t.Fatal(err)
	}
	if manifestations != 2 || marks != 0 {
		t.Errorf("(manifestations, export marks) after reset = (%d, %d), want (2, 0)", manifestations, marks)
	}

	// The note comes back whole when the distribution delivers it again.
	f.applyDocument("mock", cnpjMock, f.procNFe("procnfe.xml", "hash-completa"), 1)
	if got := f.companyDocument("mock", nfeKeyProc); got.EventCount != 1 {
		t.Errorf("events of the note delivered again = %d, want the CC-e relinked", got.EventCount)
	}
}
