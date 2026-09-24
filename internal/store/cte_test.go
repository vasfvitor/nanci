package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

// Access keys and CNPJs of the fixtures in internal/cte/testdata.
const (
	cteKeyProc  = "35260912345678000195570010000001011123456784" // mock is tomador and destinatário
	cteKeyToma4 = "35260912345678000195570010000001021234567891" // mock is autorizado
	cteKeyV200  = "35260912345678000195570010000001031345678907" // mock is tomador and remetente, tpAmb 2
	cteKeyOS    = "35260912345678000195670010000001041456789014" // mock is tomador
	cteKeyGTVe  = "35260912345678000195640010000001051567890127" // mock is tomador and destinatário

	cteNFeKeyA     = "35260911222333000181550010000012341123456787" // in procte.xml
	cteNFeKeyB     = "35260911222333000181550010000012351234567894" // in procte.xml
	cteNFeKeyToma4 = "35260911222333000181550010000012361345678900" // in procte-toma4.xml

	cteCNPJRemetente = "11222333000181" // remetente of procte.xml, destinatário of procte-v200-toma03.xml
	cteCNPJTerceiro  = "11223344000186" // tomador named in procte-toma4.xml
)

// cteFixture holds a migrated database with three companies: mock, the
// remetente of the fixtures and the third-party tomador.
type cteFixture struct {
	t    *testing.T
	db   *sql.DB
	repo *store.CTeRepository
}

func newCTeFixture(t *testing.T) *cteFixture {
	t.Helper()
	db := storetest.OpenTestDB(t)
	const now = "2026-09-01T00:00:00Z"
	for _, c := range []struct{ id, cnpj string }{
		{"mock", cnpjMock},
		{"remetente", cteCNPJRemetente},
		{"terceiro", cteCNPJTerceiro},
	} {
		mustExec(t, db, `
			INSERT INTO companies (id, cnpj, cnpj_root, name, environment, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'producao', ?, ?)
		`, c.id, c.cnpj, c.cnpj[:8], c.id, now, now)
	}
	return &cteFixture{t: t, db: db, repo: store.NewCTeRepository(db)}
}

func (f *cteFixture) readFixture(name string) string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "cte", "testdata", name)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		f.t.Fatal(err)
	}
	return string(data)
}

// document parses a document fixture after applying the replacements (old,
// new pairs) and gives it rawHash.
func (f *cteFixture) document(name, rawHash string, replacements ...string) cte.Document {
	f.t.Helper()
	data := strings.NewReplacer(replacements...).Replace(f.readFixture(name))
	doc, err := cte.ParseProcCTe([]byte(data))
	if err != nil {
		f.t.Fatalf("parse %s: %v", name, err)
	}
	doc.RawHash = rawHash
	return doc
}

func (f *cteFixture) event(name, rawHash string, replacements ...string) cte.Event {
	f.t.Helper()
	data := strings.NewReplacer(replacements...).Replace(f.readFixture(name))
	ev, err := cte.ParseProcEventoCTe([]byte(data))
	if err != nil {
		f.t.Fatalf("parse %s: %v", name, err)
	}
	ev.RawHash = rawHash
	return ev
}

func (f *cteFixture) applyDocument(companyID, companyCNPJ string, doc cte.Document, nsu int64) bool {
	f.t.Helper()
	var inserted bool
	f.inTx(func(tx *sql.Tx) error {
		var err error
		inserted, err = f.repo.ApplyDocumentTx(context.Background(), tx, store.ApplyCTeDocumentParams{
			Document:    doc,
			CompanyID:   dfe.CompanyID(companyID),
			CompanyCNPJ: companyCNPJ,
			NSU:         nsu,
		})
		return err
	})
	return inserted
}

func (f *cteFixture) applyEvent(ev cte.Event) bool {
	f.t.Helper()
	var inserted bool
	f.inTx(func(tx *sql.Tx) error {
		var err error
		inserted, err = f.repo.ApplyEventTx(context.Background(), tx, store.ApplyCTeEventParams{Event: ev})
		return err
	})
	return inserted
}

func (f *cteFixture) inTx(fn func(tx *sql.Tx) error) {
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

func (f *cteFixture) companyDocument(companyID, chave string) cte.CompanyDocument {
	f.t.Helper()
	doc, err := f.repo.CompanyDocumentByChave(context.Background(), dfe.CompanyID(companyID), chave)
	if err != nil {
		f.t.Fatalf("CompanyDocumentByChave(%s, %s): %v", companyID, chave, err)
	}
	return *doc
}

func (f *cteFixture) events(chave string) []cte.Event {
	f.t.Helper()
	events, err := f.repo.ListEventsByChave(context.Background(), chave)
	if err != nil {
		f.t.Fatal(err)
	}
	return events
}

func (f *cteFixture) list(companyID string, filter cte.DocumentFilter) []string {
	f.t.Helper()
	docs, err := f.repo.ListCompanyDocuments(context.Background(), dfe.CompanyID(companyID), filter)
	if err != nil {
		f.t.Fatalf("ListCompanyDocuments: %v", err)
	}
	return cteChaves(docs)
}

func cteChaves(docs []cte.CompanyDocument) []string {
	out := []string{}
	for _, d := range docs {
		out = append(out, string(d.ChaveAcesso))
	}
	return out
}

func TestCTeDocumentRoundTrip(t *testing.T) {
	f := newCTeFixture(t)
	doc := f.document("procte.xml", "hash-proc")
	if !f.applyDocument("mock", cnpjMock, doc, 7) {
		t.Fatal("first apply should insert")
	}
	if f.applyDocument("mock", cnpjMock, doc, 9) {
		t.Error("second apply of the same chave should not insert")
	}

	got := f.companyDocument("mock", cteKeyProc)
	if got.CompanyRole != cte.CompanyRoleTomador || got.VisibilityReason != cte.VisibilityReasonExactTomador {
		t.Errorf("(role, visibility) = (%s, %s), want (tomador, exact_tomador)", got.CompanyRole, got.VisibilityReason)
	}
	if !slices.Equal(got.Papeis, []cte.CompanyRole{cte.CompanyRoleTomador, cte.CompanyRoleDestinatario}) {
		t.Errorf("Papeis = %v, want [tomador destinatario]", got.Papeis)
	}
	if *got.FirstSeenNSU != 7 || *got.LastSeenNSU != 9 {
		t.Errorf("NSUs = (%d, %d), want (7, 9)", *got.FirstSeenNSU, *got.LastSeenNSU)
	}

	// Everything the store keeps comes back as parsed. The IE and UF of
	// the parties other than emitente and tomador are not stored.
	want := doc
	want.ID = got.ID
	for _, p := range []*cte.Party{&want.Remetente, &want.Destinatario, &want.Expedidor, &want.Recebedor} {
		p.IE, p.UF = "", ""
	}
	if !got.IssueDate.Equal(want.IssueDate) || got.AuthorizedAt == nil || !got.AuthorizedAt.Equal(*want.AuthorizedAt) {
		t.Errorf("times = (%s, %v), want (%s, %s)", got.IssueDate, got.AuthorizedAt, want.IssueDate, want.AuthorizedAt)
	}
	gotDoc := got.Document
	gotDoc.IssueDate, want.IssueDate = time.Time{}, time.Time{}
	gotDoc.AuthorizedAt, want.AuthorizedAt = nil, nil
	if !reflect.DeepEqual(gotDoc, want) {
		t.Errorf("stored document differs\n got: %+v\nwant: %+v", gotDoc, want)
	}
}

func TestCTeEventBeforeDocument(t *testing.T) {
	f := newCTeFixture(t)
	cancelamento := f.event("proceventocte-cancelamento.xml", "hash-canc")
	if !f.applyEvent(cancelamento) {
		t.Fatal("first event apply should insert")
	}
	if f.applyEvent(cancelamento) {
		t.Error("second event apply should not insert")
	}
	var linked int
	if err := f.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM cte_events WHERE cte_document_id IS NOT NULL`).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != 0 {
		t.Fatalf("events linked before the document arrived = %d", linked)
	}

	f.applyDocument("mock", cnpjMock, f.document("procte.xml", "hash-proc"), 1)
	got := f.companyDocument("mock", cteKeyProc)
	if got.Situacao != cte.SituacaoCancelada || got.EventCount != 1 {
		t.Errorf("(situação, events) = (%s, %d), want (cancelada, 1)", got.Situacao, got.EventCount)
	}
	if err := f.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM cte_events WHERE cte_document_id = ?`, got.ID).Scan(&linked); err != nil {
		t.Fatal(err)
	}
	if linked != 1 {
		t.Errorf("events linked after the document arrived = %d, want 1", linked)
	}
}

func TestCTeCancelamento(t *testing.T) {
	f := newCTeFixture(t)
	f.applyDocument("mock", cnpjMock, f.document("procte.xml", "hash-proc"), 1)
	f.applyDocument("remetente", cteCNPJRemetente, f.document("procte.xml", "hash-proc"), 3)
	if got := f.companyDocument("mock", cteKeyProc).Situacao; got != cte.SituacaoAutorizada {
		t.Fatalf("situação before the event = %s", got)
	}

	f.applyEvent(f.event("proceventocte-cancelamento.xml", "hash-canc"))
	for _, company := range []string{"mock", "remetente"} {
		if got := f.companyDocument(company, cteKeyProc).Situacao; got != cte.SituacaoCancelada {
			t.Errorf("%s situação = %s, want cancelada", company, got)
		}
	}

	// A later copy of the autorizada document does not bring it back.
	f.applyDocument("mock", cnpjMock, f.document("procte.xml", "hash-proc-2"), 5)
	if got := f.companyDocument("mock", cteKeyProc); got.Situacao != cte.SituacaoCancelada || got.RawHash != "hash-proc-2" {
		t.Errorf("(situação, raw hash) after a new copy = (%s, %s), want (cancelada, hash-proc-2)", got.Situacao, got.RawHash)
	}

	events := f.events(cteKeyProc)
	if len(events) != 1 || events[0].Type != cte.EventTypeCancelamento || events[0].Justificativa == "" || !events[0].Registered {
		t.Errorf("events = %+v, want the registered cancelamento", events)
	}
}

func TestCTeUnregisteredCancelamentoDoesNotCancel(t *testing.T) {
	f := newCTeFixture(t)
	f.applyDocument("mock", cnpjMock, f.document("procte.xml", "hash-proc"), 1)
	f.applyEvent(f.event("proceventocte-cancelamento.xml", "hash-canc", "<cStat>135</cStat>", "<cStat>631</cStat>"))
	if got := f.companyDocument("mock", cteKeyProc).Situacao; got != cte.SituacaoAutorizada {
		t.Errorf("situação = %s, want autorizada", got)
	}
}

func TestCTeDocumentSeenByTwoCompanies(t *testing.T) {
	f := newCTeFixture(t)
	f.applyDocument("mock", cnpjMock, f.document("procte-toma4.xml", "hash-toma4"), 1)
	f.applyDocument("terceiro", cteCNPJTerceiro, f.document("procte-toma4.xml", "hash-toma4"), 1)
	f.applyDocument("remetente", cteCNPJRemetente, f.document("procte-toma4.xml", "hash-toma4"), 1)

	tests := []struct {
		company    string
		role       cte.CompanyRole
		visibility cte.VisibilityReason
	}{
		{"mock", cte.CompanyRoleAutorizado, cte.VisibilityReasonExactAutorizado},
		{"terceiro", cte.CompanyRoleTomador, cte.VisibilityReasonExactTomador},
		{"remetente", cte.CompanyRoleRemetente, cte.VisibilityReasonExactRemetente},
	}
	for _, tt := range tests {
		got := f.companyDocument(tt.company, cteKeyToma4)
		if got.CompanyRole != tt.role || got.VisibilityReason != tt.visibility || !slices.Equal(got.Papeis, []cte.CompanyRole{tt.role}) {
			t.Errorf("%s = (%s, %s, %v), want (%s, %s, [%s])", tt.company, got.CompanyRole, got.VisibilityReason, got.Papeis, tt.role, tt.visibility, tt.role)
		}
	}

	var documents int
	if err := f.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM cte_documents`).Scan(&documents); err != nil {
		t.Fatal(err)
	}
	if documents != 1 {
		t.Errorf("cte_documents rows = %d, want one shared row", documents)
	}

	counts, err := f.repo.CountSummary(context.Background(), "terceiro", "")
	if err != nil {
		t.Fatal(err)
	}
	if counts.ByRole[cte.CompanyRoleTomador] != 1 || len(counts.ByRole) != 1 {
		t.Errorf("terceiro counts = %v, want one tomador", counts.ByRole)
	}
	exists, err := f.repo.CompanyDocumentExists(context.Background(), "remetente", cteKeyToma4)
	if err != nil || !exists {
		t.Errorf("CompanyDocumentExists(remetente) = %v, %v", exists, err)
	}
	if _, err := f.repo.CompanyDocumentByChave(context.Background(), "remetente", cteKeyProc); !errors.Is(err, cte.ErrDocumentNotFound) {
		t.Errorf("CompanyDocumentByChave of an unseen chave error = %v, want ErrDocumentNotFound", err)
	}
}

func TestCTeMaskedCopyKeepsTheFullDocument(t *testing.T) {
	const cnpjAutorizado = "45678901000175" // autXML of procte.xml
	masked := strings.Repeat("9", 44)
	maskedCopy := []string{cteNFeKeyA, masked, cteNFeKeyB, masked}

	tests := []struct {
		name  string
		order []string // the raw hash of each apply, in order
	}{
		{"full then masked", []string{"hash-full", "hash-masked"}},
		{"masked then full", []string{"hash-masked", "hash-full"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCTeFixture(t)
			mustExec(t, f.db, `
				INSERT INTO companies (id, cnpj, cnpj_root, name, environment, created_at, updated_at)
				VALUES ('autorizado', ?, ?, 'autorizado', 'producao', '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z')
			`, cnpjAutorizado, cnpjAutorizado[:8])
			for i, hash := range tt.order {
				if hash == "hash-full" {
					f.applyDocument("mock", cnpjMock, f.document("procte.xml", hash), int64(i+1))
				} else {
					f.applyDocument("autorizado", cnpjAutorizado, f.document("procte.xml", hash, maskedCopy...), int64(i+1))
				}
			}

			for _, company := range []string{"mock", "autorizado"} {
				got := f.companyDocument(company, cteKeyProc)
				if got.RawHash != "hash-full" || got.MaskedKeys || !slices.Equal(got.NFeChaves, []string{cteNFeKeyA, cteNFeKeyB}) {
					t.Errorf("%s sees (raw hash, masked, NF-e chaves) = (%s, %v, %v), want the full document", company, got.RawHash, got.MaskedKeys, got.NFeChaves)
				}
			}
			if got := f.companyDocument("autorizado", cteKeyProc); got.CompanyRole != cte.CompanyRoleAutorizado {
				t.Errorf("autorizado role = %s, want autorizado", got.CompanyRole)
			}
			if got := f.list("mock", cte.DocumentFilter{NFeChave: cteNFeKeyB}); !slices.Equal(got, []string{cteKeyProc}) {
				t.Errorf("mock list by NF-e chave = %v, want %s", got, cteKeyProc)
			}
		})
	}
}

func TestCTeListFilters(t *testing.T) {
	f := newCTeFixture(t)
	for i, name := range []string{"procte.xml", "procte-toma4.xml", "procte-v200-toma03.xml", "procteos.xml", "procgtve.xml"} {
		f.applyDocument("mock", cnpjMock, f.document(name, "hash-"+name), int64(i+1))
	}

	tests := []struct {
		name   string
		filter cte.DocumentFilter
		want   []string
	}{
		{"all, newest first", cte.DocumentFilter{}, []string{cteKeyGTVe, cteKeyOS, cteKeyV200, cteKeyToma4, cteKeyProc}},
		{"primary role", cte.DocumentFilter{Role: cte.CompanyRoleTomador}, []string{cteKeyGTVe, cteKeyOS, cteKeyV200, cteKeyProc}},
		{"secondary role remetente", cte.DocumentFilter{Role: cte.CompanyRoleRemetente}, []string{cteKeyV200}},
		{"secondary role destinatário", cte.DocumentFilter{Role: cte.CompanyRoleDestinatario}, []string{cteKeyGTVe, cteKeyProc}},
		{"autorizado", cte.DocumentFilter{Role: cte.CompanyRoleAutorizado}, []string{cteKeyToma4}},
		{"modelo 67", cte.DocumentFilter{Modelo: "67"}, []string{cteKeyOS}},
		{"modelo 64", cte.DocumentFilter{Modelo: "64"}, []string{cteKeyGTVe}},
		{"tomador", cte.DocumentFilter{TomadorCNPJ: "11.223.344/0001-86"}, []string{cteKeyToma4}},
		{"emitente", cte.DocumentFilter{EmitenteCNPJ: "12345678000195", Modelo: "57"}, []string{cteKeyV200, cteKeyToma4, cteKeyProc}},
		{"NF-e chave, second in the list", cte.DocumentFilter{NFeChave: cteNFeKeyB}, []string{cteKeyProc}},
		{"NF-e chave, first in the list", cte.DocumentFilter{NFeChave: cteNFeKeyA}, []string{cteKeyProc}},
		{"NF-e chave next to masked ones", cte.DocumentFilter{NFeChave: cteNFeKeyToma4}, []string{cteKeyToma4}},
		{"NF-e chave prefix never matches", cte.DocumentFilter{NFeChave: cteNFeKeyA[:43]}, []string{}},
		{"tpAmb homologação", cte.DocumentFilter{TpAmb: "2"}, []string{cteKeyV200}},
		{"competence and situação", cte.DocumentFilter{Competence: "2026-09", Situacao: cte.SituacaoAutorizada, Limit: 2}, []string{cteKeyGTVe, cteKeyOS}},
		{"chaves", cte.DocumentFilter{ChavesAcesso: []string{cteKeyOS, cteKeyProc}}, []string{cteKeyOS, cteKeyProc}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.list("mock", tt.filter); !slices.Equal(got, tt.want) {
				t.Errorf("list = %v, want %v", got, tt.want)
			}
		})
	}

	counts, err := f.repo.CountSummary(context.Background(), "mock", "1")
	if err != nil {
		t.Fatal(err)
	}
	if counts.ByRole[cte.CompanyRoleTomador] != 3 || counts.ByRole[cte.CompanyRoleAutorizado] != 1 {
		t.Errorf("tpAmb 1 counts = %v, want 3 tomador and 1 autorizado", counts.ByRole)
	}
}

func TestCTeExportMarks(t *testing.T) {
	ctx := context.Background()
	f := newCTeFixture(t)
	f.applyDocument("mock", cnpjMock, f.document("procte.xml", "hash-proc"), 1)
	f.applyDocument("mock", cnpjMock, f.document("procteos.xml", "hash-os"), 2)

	pending, err := f.repo.ListPendingExport(ctx, "mock", cte.DocumentFilter{}, cte.ExportKindXML)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Fatalf("pending before export = %v", cteChaves(pending))
	}
	if err := f.repo.MarkExported(ctx, "mock", cte.ExportKindXML, pending); err != nil {
		t.Fatal(err)
	}
	if pending, _ = f.repo.ListPendingExport(ctx, "mock", cte.DocumentFilter{}, cte.ExportKindXML); len(pending) != 0 {
		t.Errorf("pending after export = %v, want none", cteChaves(pending))
	}

	f.applyDocument("mock", cnpjMock, f.document("procteos.xml", "hash-os-2"), 3)
	pending, err = f.repo.ListPendingExport(ctx, "mock", cte.DocumentFilter{}, cte.ExportKindXML)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cteChaves(pending), []string{cteKeyOS}) {
		t.Errorf("pending after a new raw hash = %v, want the CT-e OS", cteChaves(pending))
	}
	if err := f.repo.MarkExported(ctx, "mock", cte.ExportKindXML, pending); err != nil {
		t.Fatal(err)
	}

	// A cancelamento stored after the export makes the document pending
	// again. The marks are moved back so the event is not stored in the
	// same second.
	mustExec(t, f.db, `UPDATE company_cte_export_marks SET exported_at = '2026-09-01T00:00:00Z'`)
	f.applyEvent(f.event("proceventocte-cancelamento.xml", "hash-canc"))
	pending, err = f.repo.ListPendingExport(ctx, "mock", cte.DocumentFilter{}, cte.ExportKindXML)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cteChaves(pending), []string{cteKeyProc}) {
		t.Fatalf("pending after a cancelamento = %v, want the CT-e", cteChaves(pending))
	}
	if err := f.repo.MarkExported(ctx, "mock", cte.ExportKindXML, pending); err != nil {
		t.Fatal(err)
	}
	if pending, _ = f.repo.ListPendingExport(ctx, "mock", cte.DocumentFilter{}, cte.ExportKindXML); len(pending) != 0 {
		t.Errorf("pending after exporting the cancelamento = %v, want none", cteChaves(pending))
	}
	if _, err := f.repo.ListPendingExport(ctx, "mock", cte.DocumentFilter{}, ""); err == nil {
		t.Error("ListPendingExport without a kind should fail")
	}
}

func TestCTeListEventsByChaves(t *testing.T) {
	f := newCTeFixture(t)
	f.applyEvent(f.event("proceventocte-cancelamento.xml", "hash-canc"))
	f.applyEvent(f.event("proceventocte-comprovante.xml", "hash-ce"))

	events, err := f.repo.ListEventsByChaves(context.Background(), []string{cteKeyToma4, cteKeyProc})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].ChaveAcesso != cteKeyProc || events[1].Type != cte.EventTypeComprovanteEntrega {
		t.Fatalf("events = %+v, want the cancelamento then the comprovante", events)
	}
	if events[0].COrgao != "35" || events[0].TpAmb != "1" || events[0].RegisteredAt == nil {
		t.Errorf("cancelamento = %+v", events[0])
	}
	if none, err := f.repo.ListEventsByChaves(context.Background(), nil); err != nil || none != nil {
		t.Errorf("ListEventsByChaves(nil) = %v, %v", none, err)
	}
}

func TestCTeResetCompany(t *testing.T) {
	ctx := context.Background()
	f := newCTeFixture(t)
	// mock and remetente both see cteKeyProc; only mock sees cteKeyOS.
	f.applyDocument("mock", cnpjMock, f.document("procte.xml", "hash-proc"), 1)
	f.applyDocument("remetente", cteCNPJRemetente, f.document("procte.xml", "hash-proc"), 1)
	f.applyDocument("mock", cnpjMock, f.document("procteos.xml", "hash-os"), 2)
	f.applyEvent(f.event("proceventocte-cancelamento.xml", "hash-canc"))
	f.applyEvent(f.event("proceventocte-cancelamento.xml", "hash-canc-os", cteKeyProc, cteKeyOS))
	docs, err := f.repo.ListCompanyDocuments(ctx, "mock", cte.DocumentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.MarkExported(ctx, "mock", cte.ExportKindXML, docs); err != nil {
		t.Fatal(err)
	}
	mustExec(t, f.db, `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		VALUES ('mock', 'cte', 'producao', ?, 42, '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z'),
			('mock', 'nfe', 'producao', ?, 7, '2026-09-01T00:00:00Z', '2026-09-01T00:00:00Z')
	`, cnpjMock, cnpjMock)
	cursors := func(source string) int {
		t.Helper()
		var n int
		if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_state WHERE company_id = 'mock' AND source = ?`, source).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	want := cte.ResetCounts{
		CompanyDocuments: 2,
		Documents:        1, // cteKeyProc stays for the remetente
		Events:           1, // the cancelamento of the CT-e OS
		ExportMarks:      2,
	}
	preview, err := f.repo.PreviewResetCompany(ctx, "mock")
	if err != nil {
		t.Fatal(err)
	}
	if preview != want {
		t.Errorf("PreviewResetCompany = %+v, want %+v", preview, want)
	}
	if n := cursors("cte"); n != 1 {
		t.Errorf("cte cursors after preview = %d, want 1", n)
	}
	if got := f.list("mock", cte.DocumentFilter{}); len(got) != 2 {
		t.Fatalf("documents after preview = %v, want both kept", got)
	}

	got, err := f.repo.ResetCompany(ctx, "mock")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("ResetCompany = %+v, want %+v", got, want)
	}
	if n := cursors("cte"); n != 0 {
		t.Errorf("cte cursors after reset = %d, want 0", n)
	}
	if n := cursors("nfe"); n != 1 {
		t.Errorf("nfe cursors after a CT-e reset = %d, want 1", n)
	}

	if docs := f.list("mock", cte.DocumentFilter{}); len(docs) != 0 {
		t.Errorf("mock documents after reset = %v, want none", docs)
	}
	shared := f.companyDocument("remetente", cteKeyProc)
	if shared.Situacao != cte.SituacaoCancelada || shared.EventCount != 1 {
		t.Errorf("shared CT-e = (%s, %d events), want (cancelada, 1)", shared.Situacao, shared.EventCount)
	}
	if events := f.events(cteKeyOS); len(events) != 0 {
		t.Errorf("events of the removed CT-e OS = %+v", events)
	}
	var marks int
	if err := f.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM company_cte_export_marks WHERE company_id = 'mock'`).Scan(&marks); err != nil {
		t.Fatal(err)
	}
	if marks != 0 {
		t.Errorf("export marks after reset = %d", marks)
	}

	// The CT-e comes back when the distribution delivers it again.
	f.applyDocument("mock", cnpjMock, f.document("procteos.xml", "hash-os"), 2)
	if got := f.companyDocument("mock", cteKeyOS); got.Situacao != cte.SituacaoAutorizada || got.EventCount != 0 {
		t.Errorf("CT-e OS delivered again = (%s, %d events)", got.Situacao, got.EventCount)
	}
}
