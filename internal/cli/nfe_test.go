package cli

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
)

// Fixtures of internal/nfe/testdata. The company is their destinatário.
const (
	nfeTestCNPJ       = "70860312000150"
	nfeChaveProc      = "35260911222333000181550010000012341123456787" // resnfe.xml, procnfe.xml, proceventonfe-ciencia.xml
	nfeChaveCancelada = "35260911222333000181550010000012351234567894" // resnfe-cancelada.xml
	nfeChaveDenegada  = "35260911222333000181550010000012361345678900" // procnfe-denegada.xml
)

// refusingPasswords counts password requests and refuses them all. Nothing
// can be signed or sent to SEFAZ without the password, so zero requests
// means nothing left nanci.
type refusingPasswords struct {
	requests int
}

func (p *refusingPasswords) GetCertPassword(context.Context, app.CertPasswordRequest) ([]byte, error) {
	p.requests++
	return nil, errors.New("senha indisponível no teste")
}

type nfeTestRoot struct {
	t         *testing.T
	root      *cobra.Command
	out       *bytes.Buffer
	db        *sql.DB
	repo      *store.NFeRepository
	xml       files.XMLStore
	company   *nfse.Company
	passwords *refusingPasswords
}

// newNFeTestRoot builds a root over a real database holding one company,
// the destinatário of the NF-e fixtures.
func newNFeTestRoot(t *testing.T) *nfeTestRoot {
	t.Helper()
	ctx := context.Background()
	db := storetest.OpenTestDB(t)

	cred := &nfse.Credential{ID: "cred-1", Label: "Mock A1", CertPath: "mock.pfx"}
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
	passwords := &refusingPasswords{}
	application, err := app.New(app.Dependencies{
		Log:                slog.New(slog.DiscardHandler),
		CompanyStore:       company.NewStore(db),
		CredentialStore:    credential.NewStore(db),
		SyncRepo:           sync.NewStore(db),
		DocumentRepo:       store.NewDocumentRepository(db),
		NFeRepo:            repo,
		CTeRepo:            store.NewCTeRepository(db),
		XMLStore:           xmlStore,
		DataDir:            t.TempDir(),
		CredentialProvider: passwords,
	})
	if err != nil {
		t.Fatal(err)
	}

	root, out := newTestRootForApp(application)
	root.SetErr(&bytes.Buffer{})
	return &nfeTestRoot{t: t, root: root, out: out, db: db, repo: repo, xml: xmlStore, company: comp, passwords: passwords}
}

// run executes the command line and returns its error; stdout is in e.out.
func (e *nfeTestRoot) run(args ...string) error {
	e.t.Helper()
	e.out.Reset()
	e.root.SetArgs(args)
	return e.root.ExecuteContext(context.Background())
}

// seed stores the fixture as if the distribution had delivered it at nsu.
func (e *nfeTestRoot) seed(fixture string, nsu int64) {
	e.t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", fixture)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		e.t.Fatal(err)
	}
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
		doc.TpAmb = cmp.Or(doc.TpAmb, sefaz.TpAmbProducao) // a resumo takes the pull's, like the NF-e source does
		params := store.ApplyNFeDocumentParams{Document: doc, CompanyID: e.company.ID, CompanyCNPJ: e.company.CNPJ, NSU: nsu}
		if _, err := e.repo.ApplyDocumentTx(ctx, tx, params); err != nil {
			e.t.Fatal(err)
		}
	default:
		ev, err := nfe.ParseProcEventoNFe(data)
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
}

// manifestacaoCount is how many manifestação events nanci recorded as sent.
func (e *nfeTestRoot) manifestacaoCount() int {
	e.t.Helper()
	var count int
	if err := e.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM nfe_manifestacoes`).Scan(&count); err != nil {
		e.t.Fatal(err)
	}
	return count
}

func TestNFeCiencia_ChaveAndTodosResumosAreExclusive(t *testing.T) {
	env := newNFeTestRoot(t)

	tests := []struct {
		name string
		args []string
	}{
		{"both", []string{"nfe", "ciencia", "-c", nfeTestCNPJ, "--chave", nfeChaveProc, "--todos-resumos"}},
		{"neither", []string{"nfe", "ciencia", "-c", nfeTestCNPJ}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := env.run(tc.args...)
			if err == nil || !strings.Contains(err.Error(), "--todos-resumos") {
				t.Errorf("Execute = %v, want the --chave/--todos-resumos error", err)
			}
		})
	}
}

func TestNFeManifestar_ValidatesJustificativa(t *testing.T) {
	env := newNFeTestRoot(t)

	tests := []struct {
		name string
		args []string
	}{
		{"nao_realizada without justificativa", []string{"--tipo", "nao_realizada"}},
		{"nao_realizada with a short justificativa", []string{"--tipo", "nao_realizada", "--justificativa", "curta"}},
		{"confirmacao with justificativa", []string{"--tipo", "confirmacao", "--justificativa", "mercadoria não recebida no prazo"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"nfe", "manifestar", "-c", nfeTestCNPJ, "--chave", nfeChaveProc}, tc.args...)
			err := env.run(args...)
			if err == nil || !strings.Contains(err.Error(), "justificativa") {
				t.Errorf("Execute = %v, want a justificativa error", err)
			}
		})
	}
}

func TestNFeManifestacaoDryRunSendsNothing(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("resnfe.xml", 1)

	if err := env.run("nfe", "ciencia", "-c", nfeTestCNPJ, "--todos-resumos"); err != nil {
		t.Fatalf("ciencia: %v", err)
	}
	got := env.out.String()
	for _, want := range []string{nfeChaveProc, "1 lote(s)", "Nada foi enviado. Use --confirmar"} {
		if !strings.Contains(got, want) {
			t.Errorf("ciencia output lacks %q:\n%s", want, got)
		}
	}

	// The hyphenated nao-realizada stays accepted as an alias.
	err := env.run("nfe", "manifestar", "-c", nfeTestCNPJ, "--chave", nfeChaveProc,
		"--tipo", "nao-realizada", "--justificativa", "mercadoria devolvida ao emitente")
	if err != nil {
		t.Fatalf("manifestar: %v", err)
	}
	got = env.out.String()
	for _, want := range []string{"Operação não realizada (210240)", nfeChaveProc, "Prazo da manifestação conclusiva:", "Nada foi enviado. Use --confirmar"} {
		if !strings.Contains(got, want) {
			t.Errorf("manifestar output lacks %q:\n%s", want, got)
		}
	}

	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
	if n := env.manifestacaoCount(); n != 0 {
		t.Errorf("manifestações recorded = %d, want 0", n)
	}
}

func TestNFeManifestarDryRunRefusesSecondConclusive(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("procnfe.xml", 1)
	registeredAt := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	err := env.repo.RecordManifestacoes(context.Background(), []nfe.ManifestacaoRecord{{
		CompanyID:    env.company.ID,
		CompanyCNPJ:  env.company.CNPJ,
		IDLote:       "1",
		TpAmb:        "1",
		ChaveAcesso:  nfeChaveProc,
		TpEvento:     nfe.TpEventoConfirmacao,
		NSeqEvento:   1,
		EventAt:      &registeredAt,
		Status:       nfe.ManifestacaoStatusRegistrada,
		CStat:        "135",
		Protocolo:    "891260000000099",
		RegisteredAt: &registeredAt,
	}})
	if err != nil {
		t.Fatal(err)
	}
	recorded := env.manifestacaoCount()

	for _, args := range [][]string{
		{"--tipo", "confirmacao"},
		{"--tipo", "desconhecimento"},
		{"--tipo", "nao_realizada", "--justificativa", "mercadoria devolvida ao emitente"},
	} {
		err := env.run(append([]string{"nfe", "manifestar", "-c", nfeTestCNPJ, "--chave", nfeChaveProc}, args...)...)
		const want = "erro: NF-e já possui manifestação conclusiva (Confirmada)"
		if err == nil || err.Error() != want {
			t.Errorf("manifestar %v = %v, want %q", args, err, want)
		}
		if strings.Contains(env.out.String(), "Nada foi enviado") {
			t.Errorf("manifestar %v printed a simulation:\n%s", args, env.out.String())
		}
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
	if n := env.manifestacaoCount(); n != recorded {
		t.Errorf("manifestações recorded = %d, want %d", n, recorded)
	}
}

func TestNFeList_PrintsColumns(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("resnfe-cancelada.xml", 1)
	env.seed("procnfe.xml", 2)
	env.seed("proceventonfe-ciencia.xml", 3)
	env.seed("procnfe-denegada.xml", 4)

	if err := env.run("nfe", "list", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(env.out.String()), "\n")
	header := strings.Join(strings.Fields(lines[0]), " ")
	if want := "EMISSÃO CHAVE DE ACESSO PAPEL COMPLETUDE SITUAÇÃO EMITENTE NOME EMITENTE VALOR (R$) MANIFESTAÇÃO"; header != want {
		t.Errorf("header = %q, want %q", header, want)
	}

	// Rows come newest first; the columns after the chave are papel, completude,
	// situação and, last, manifestação.
	wantRows := []struct {
		chave, papel, completude, situacao, manifestacao string
	}{
		{nfeChaveDenegada, "destinatario", "completa", "denegada", "nenhuma"},
		{nfeChaveProc, "destinatario", "completa", "autorizada", "ciencia"},
		{nfeChaveCancelada, "destinatario", "resumo", "cancelada", "nenhuma"},
	}
	rows := lines[2 : len(lines)-2]
	if len(rows) != len(wantRows) {
		t.Fatalf("rows = %d, want %d:\n%s", len(rows), len(wantRows), env.out.String())
	}
	for i, want := range wantRows {
		fields := strings.Fields(rows[i])
		got := []string{fields[1], fields[2], fields[3], fields[4], fields[len(fields)-1]}
		if strings.Join(got, " ") != strings.Join([]string{want.chave, want.papel, want.completude, want.situacao, want.manifestacao}, " ") {
			t.Errorf("row %d = %q, want %+v", i, rows[i], want)
		}
	}
	if last := lines[len(lines)-1]; last != "Total de 3 nota(s) listada(s)." {
		t.Errorf("footer = %q", last)
	}

	env.out.Reset()
	if err := env.run("nfe", "list", "-c", nfeTestCNPJ, "--completude", "resumo", "-p", "destinatario"); err != nil {
		t.Fatalf("list --completude -p: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, nfeChaveCancelada) || strings.Contains(got, nfeChaveProc) || !strings.Contains(got, "Total de 1 nota(s) listada(s).") {
		t.Errorf("list --completude resumo -p destinatario:\n%s", got)
	}
}

func TestNFeOutcomesError(t *testing.T) {
	registrada := app.NFeEventOutcome{Status: nfe.ManifestacaoStatusRegistrada}
	jaRegistrada := app.NFeEventOutcome{Status: nfe.ManifestacaoStatusJaRegistrada}
	rejeitada := app.NFeEventOutcome{Status: nfe.ManifestacaoStatusRejeitada}
	naoEnviada := app.NFeEventOutcome{Status: app.NFeOutcomeNaoEnviada}

	tests := []struct {
		name        string
		outcomes    []app.NFeEventOutcome
		interrupted string
		wantErr     bool
	}{
		{"all registered", []app.NFeEventOutcome{registrada, jaRegistrada}, "", false},
		{"one rejected", []app.NFeEventOutcome{registrada, rejeitada}, "", true},
		{"one not sent", []app.NFeEventOutcome{naoEnviada}, "", true},
		{"interrupted", []app.NFeEventOutcome{registrada}, "timeout", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := nfeOutcomesError(tc.outcomes, tc.interrupted); (err != nil) != tc.wantErr {
				t.Errorf("nfeOutcomesError = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNFePull_BlockedPrintsNextAllowedAt(t *testing.T) {
	env := newNFeTestRoot(t)
	until := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	if err := sync.NewStore(env.db).SetBlockedUntil(context.Background(), env.company.ID, nfse.SyncSourceNFe, until, nfse.SyncStopReasonCaughtUp); err != nil {
		t.Fatal(err)
	}

	err := env.run("nfe", "pull", "-c", nfeTestCNPJ)
	if !errors.Is(err, app.ErrSourceBlocked) {
		t.Fatalf("Execute = %v, want ErrSourceBlocked", err)
	}
	want := "Próxima consulta permitida após: " + until.Local().Format("2006-01-02 15:04") + "\n"
	if got := env.out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
}

func TestPendingAlert(t *testing.T) {
	tests := []struct {
		name string
		p    app.NFePendingManifestacao
		want string
	}{
		{"expired", app.NFePendingManifestacao{NFeDocument: app.NFeDocument{TacitlyConfirmed: true}, CienciaOverdue: true}, "confirmada tacitamente (prazo expirado)"},
		{"ciência overdue", app.NFePendingManifestacao{CienciaOverdue: true}, "ciência atrasada"},
		{"on time", app.NFePendingManifestacao{}, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pendingAlert(tt.p); got != tt.want {
				t.Errorf("pendingAlert = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNFeReset_DryRunThenConfirm(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("procnfe.xml", 1)
	env.seed("proceventonfe-ciencia.xml", 2)
	const now = "2026-09-20T10:00:00Z"
	_, err := env.db.ExecContext(context.Background(), `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		VALUES (?, 'nfe', 'producao', ?, 2, ?, ?)
	`, string(env.company.ID), env.company.CNPJ, now, now)
	if err != nil {
		t.Fatal(err)
	}

	// The environment switches freely; each one lists only its own notes.
	if err := env.run("company", "update", "-c", nfeTestCNPJ, "--env", "producao_restrita"); err != nil {
		t.Fatalf("company update --env producao_restrita: %v", err)
	}
	if err := env.run("nfe", "list", "-c", nfeTestCNPJ); err != nil || strings.Contains(env.out.String(), nfeChaveProc) {
		t.Fatalf("homologação lists the produção note: %v\n%s", err, env.out.String())
	}
	if err := env.run("company", "update", "-c", nfeTestCNPJ, "--env", "producao"); err != nil {
		t.Fatalf("company update --env producao: %v", err)
	}

	if err := env.run("nfe", "reset", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("reset dry-run: %v", err)
	}
	got := env.out.String()
	for _, want := range []string{"Notas da empresa (seriam removidos): 1", "Eventos (seriam removidos): 1", "Nada foi alterado. Use --confirmar"} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output lacks %q:\n%s", want, got)
		}
	}
	if err := env.run("nfe", "list", "-c", nfeTestCNPJ); err != nil || !strings.Contains(env.out.String(), nfeChaveProc) {
		t.Fatalf("the dry-run removed the note: %v\n%s", err, env.out.String())
	}

	if err := env.run("nfe", "reset", "-c", nfeTestCNPJ, "--confirmar"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "Notas da empresa (removidos): 1") {
		t.Errorf("reset output:\n%s", got)
	}
	docs, err := env.repo.ListCompanyDocuments(context.Background(), env.company.ID, nfe.DocumentFilter{})
	if err != nil || len(docs) != 0 {
		t.Errorf("documents after the reset = %d, %v; want none", len(docs), err)
	}
}
