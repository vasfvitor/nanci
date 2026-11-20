package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// Chaves of internal/cte/testdata with the mock company as tomador (procte,
// procteos), as autorizado only (procte-toma4) and of a GTV-e (procgtve).
const (
	desacordoChave      = "35260912345678000195570010000001011123456784"
	desacordoChaveOS    = "35260912345678000195670010000001041456789014"
	desacordoChaveToma4 = "35260912345678000195570010000001021234567891"
	desacordoChaveGTVe  = "35260912345678000195640010000001051567890127"
	desacordoObs        = "Carga entregue avariada no destino"
)

func newDesacordoEnv(t *testing.T, fixtures ...string) *nfeTestEnv {
	t.Helper()
	env := newNFeTestEnv(t)
	for i, fixture := range fixtures {
		env.seedCTe(env.company.ID, env.company.CNPJ, fixture, int64(i+1))
	}
	env.app.CTe.now = func() time.Time { return mustTime(t, "2026-10-01T12:00:00-03:00") }
	return env
}

func (e *nfeTestEnv) desacordoInput(chave string) CTeDesacordoInput {
	return CTeDesacordoInput{CNPJ: e.company.CNPJ, ChaveAcesso: chave, Observacao: desacordoObs}
}

func (e *nfeTestEnv) desacordoStatuses() []string {
	e.t.Helper()
	rows, err := e.db.QueryContext(context.Background(), `SELECT status FROM cte_desacordos ORDER BY created_at, rowid`)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var statuses []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			e.t.Fatal(err)
		}
		statuses = append(statuses, s)
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return statuses
}

func TestPlanDesacordo(t *testing.T) {
	env := newDesacordoEnv(t, "procte.xml", "procteos.xml")
	plan, err := env.app.CTe.PlanDesacordo(context.Background(), env.desacordoInput(desacordoChave))
	if err != nil {
		t.Fatalf("PlanDesacordo: %v", err)
	}
	wantDue := mustTime(t, "2026-10-17T08:30:41-03:00")
	if plan.BlockReason != "" || plan.NSeqEvento != 1 || plan.TpAmb != "1" || plan.Observacao != desacordoObs ||
		!plan.Due.Equal(wantDue) || plan.DaysLeft != 16 || plan.CompanyName != env.company.Name {
		t.Errorf("plan = %+v (Due %s)", plan, plan.Due)
	}

	plan, err = env.app.CTe.PlanDesacordo(context.Background(), env.desacordoInput(desacordoChaveOS))
	if err != nil || plan.BlockReason != "" {
		t.Errorf("CT-e OS: plan = %+v, err = %v", plan, err)
	}
	if len(env.passwords.requests) != 0 {
		t.Errorf("password requests = %d, want 0", len(env.passwords.requests))
	}
}

func TestPlanDesacordo_Blocks(t *testing.T) {
	env := newDesacordoEnv(t, "procte-toma4.xml", "procgtve.xml")
	ctx := context.Background()
	tests := []struct{ chave, want string }{
		{desacordoChaveToma4, "não é a tomadora"},
		{desacordoChaveGTVe, "modelo 64"},
	}
	for _, tt := range tests {
		plan, err := env.app.CTe.PlanDesacordo(ctx, env.desacordoInput(tt.chave))
		if err != nil || !strings.Contains(plan.BlockReason, tt.want) {
			t.Errorf("%s: BlockReason = %q, err = %v; want %q", tt.chave, plan.BlockReason, err, tt.want)
		}
	}

	cancelled := newDesacordoEnv(t, "procte.xml", "proceventocte-cancelamento.xml")
	plan, err := cancelled.app.CTe.PlanDesacordo(ctx, cancelled.desacordoInput(desacordoChave))
	if err != nil || !strings.Contains(plan.BlockReason, "cancelada") {
		t.Errorf("cancelled CT-e: BlockReason = %q, err = %v", plan.BlockReason, err)
	}
}

func TestPlanDesacordo_InvalidInput(t *testing.T) {
	env := newDesacordoEnv(t, "procte.xml")
	ctx := context.Background()

	in := env.desacordoInput(desacordoChave)
	in.Observacao = "curta"
	_, err := env.app.CTe.PlanDesacordo(ctx, in)
	if !errors.Is(err, ErrInvalidObservacao) || ErrorCode(err) != ErrorCodeInvalidInput {
		t.Errorf("short observação: err = %v, code %q", err, ErrorCode(err))
	}

	_, err = env.app.CTe.PlanDesacordo(ctx, env.desacordoInput(desacordoChaveOS)) // not seeded
	if ErrorCode(err) != ErrorCodeNotFound {
		t.Errorf("unknown CT-e: err = %v, code %q", err, ErrorCode(err))
	}
}

func TestRegisterDesacordo(t *testing.T) {
	env := newDesacordoEnv(t, "procte.xml")
	fake := useFakeSEFAZ(t)
	ctx := context.Background()

	outcome, err := env.app.CTe.RegisterDesacordo(ctx, env.desacordoInput(desacordoChave))
	if err != nil {
		t.Fatalf("RegisterDesacordo: %v", err)
	}
	if outcome.Status != cte.DesacordoStatusRegistrada || outcome.CStat != "135" || outcome.Protocolo != "135260000000123" || outcome.RegisteredAt == nil {
		t.Errorf("outcome = %+v", outcome)
	}
	if len(fake.desacordos) != 1 {
		t.Fatalf("eventos sent = %d, want 1", len(fake.desacordos))
	}
	sent := fake.desacordos[0]
	if sent.ChaveAcesso != desacordoChave || sent.CNPJ != env.company.CNPJ || sent.NSeqEvento != 1 || sent.XObs != desacordoObs {
		t.Errorf("sent = %+v", sent)
	}
	if len(env.passwords.requests) != 1 {
		t.Errorf("password requests = %d, want 1", len(env.passwords.requests))
	}

	events, err := env.app.CTe.CTeRepo.ListEventsByChave(ctx, desacordoChave)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != cte.EventTypePrestacaoDesacordo || !events[0].Registered ||
		events[0].Observacao != desacordoObs || events[0].Protocolo != "135260000000123" || events[0].RawHash == "" {
		t.Fatalf("events = %+v, want the registered desacordo", events)
	}
	if got := env.desacordoStatuses(); len(got) != 1 || got[0] != cte.DesacordoStatusRegistrada {
		t.Errorf("cte_desacordos = %v", got)
	}

	plan, err := env.app.CTe.PlanDesacordo(ctx, env.desacordoInput(desacordoChave))
	if err != nil || !strings.Contains(plan.BlockReason, "135260000000123") {
		t.Errorf("after the send: BlockReason = %q, err = %v", plan.BlockReason, err)
	}
	if _, err := env.app.CTe.RegisterDesacordo(ctx, env.desacordoInput(desacordoChave)); err == nil || len(fake.desacordos) != 1 {
		t.Errorf("second send: err = %v, sent %d; want refused before sending", err, len(fake.desacordos))
	}
}

func TestRegisterDesacordo_Rejected(t *testing.T) {
	env := newDesacordoEnv(t, "procte.xml")
	fake := useFakeSEFAZ(t)
	fake.desacordoCStat = sefaz.CStatDesacordoForaDoPrazo
	ctx := context.Background()

	outcome, err := env.app.CTe.RegisterDesacordo(ctx, env.desacordoInput(desacordoChave))
	if err != nil {
		t.Fatalf("RegisterDesacordo: %v", err)
	}
	if outcome.Status != cte.DesacordoStatusRejeitada || outcome.CStat != "787" || outcome.Protocolo != "" {
		t.Errorf("outcome = %+v", outcome)
	}
	events, err := env.app.CTe.CTeRepo.ListEventsByChave(ctx, desacordoChave)
	if err != nil || len(events) != 0 {
		t.Errorf("events = %+v, %v; want none for a rejection", events, err)
	}
	if got := env.desacordoStatuses(); len(got) != 1 || got[0] != cte.DesacordoStatusRejeitada {
		t.Errorf("cte_desacordos = %v", got)
	}
}

func TestRegisterDesacordo_NoAnswer(t *testing.T) {
	env := newDesacordoEnv(t, "procte.xml")
	fake := useFakeSEFAZ(t)
	fake.desacordoErr = errors.New("conexão recusada")

	outcome, err := env.app.CTe.RegisterDesacordo(context.Background(), env.desacordoInput(desacordoChave))
	if err == nil || outcome.Status != NFeOutcomeNaoEnviada || !strings.Contains(outcome.XMotivo, "conexão recusada") {
		t.Errorf("outcome = %+v, err = %v", outcome, err)
	}
	if got := env.desacordoStatuses(); len(got) != 1 || got[0] != cte.DesacordoStatusErro {
		t.Errorf("cte_desacordos = %v", got)
	}
}

func TestRegisterDesacordo_BlockedAsksNoPassword(t *testing.T) {
	env := newDesacordoEnv(t, "procte-toma4.xml")
	fake := useFakeSEFAZ(t)
	if _, err := env.app.CTe.RegisterDesacordo(context.Background(), env.desacordoInput(desacordoChaveToma4)); err == nil {
		t.Fatal("RegisterDesacordo = nil error, want the block reason")
	}
	if len(env.passwords.requests) != 0 || len(fake.desacordos) != 0 {
		t.Errorf("password requests %d, sent %d; want 0 and 0", len(env.passwords.requests), len(fake.desacordos))
	}
}
