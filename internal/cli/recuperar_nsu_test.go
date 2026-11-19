package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func (e *nfeTestRoot) recordGap(source syncstate.SyncSource, first, last int64) {
	e.t.Helper()
	gaps := []sync.NSURange{{First: first, Last: last, Cause: sync.NSUGapConsumoIndevido}}
	if err := sync.NewStore(e.db).RecordNSUGaps(context.Background(), e.company.ID, source, e.company.Environment, gaps); err != nil {
		e.t.Fatal(err)
	}
}

func TestNFeRecuperarNSU_AsksTheMissingNSUsAndStatusShowsThem(t *testing.T) {
	env, fake := newNFeFetchTestRoot(t, retDistDFeInt(t, 138, "Documento localizado", "procnfe.xml"))
	env.recordGap(syncstate.SyncSourceNFe, 41, 42)

	if err := env.run("nfe", "status", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("status: %v", err)
	}
	if want := "NSUs faltando: 2. Rode `nanci nfe recuperar-nsu --cnpj " + nfeTestCNPJ + "`"; !strings.Contains(env.out.String(), want) {
		t.Errorf("status output lacks %q:\n%s", want, env.out.String())
	}

	if err := env.run("nfe", "recuperar-nsu", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("recuperar-nsu: %v\n%s", err, env.out.String())
	}
	want := "NSUs consultados: 2 (2 com documento, 0 sem documento)\n" +
		"NSUs ainda faltando: 0\n" +
		"Consultas na última hora: 2/20\n"
	if got := env.out.String(); got != want {
		t.Errorf("output =\n%s\nwant\n%s", got, want)
	}
	if fake.requests != 2 {
		t.Errorf("SEFAZ requests = %d, want 2", fake.requests)
	}

	if err := env.run("nfe", "status", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("status: %v", err)
	}
	if strings.Contains(env.out.String(), "NSUs faltando") {
		t.Errorf("status still shows missing NSUs:\n%s", env.out.String())
	}
}

func TestNFeRecuperarNSU_StopsAtTheLimit(t *testing.T) {
	env, fake := newNFeFetchTestRoot(t, retDistDFeInt(t, 137, "Nenhum documento localizado", ""))
	env.recordGap(syncstate.SyncSourceNFe, 41, 45)

	if err := env.run("nfe", "recuperar-nsu", "-c", nfeTestCNPJ, "--limite", "2"); err != nil {
		t.Fatalf("recuperar-nsu: %v\n%s", err, env.out.String())
	}
	out := env.out.String()
	for _, want := range []string{
		"NSUs consultados: 2 (0 com documento, 2 sem documento)",
		"NSUs ainda faltando: 3",
		"Parou no limite desta execução. Rode `nanci nfe recuperar-nsu --cnpj " + nfeTestCNPJ + "` de novo para continuar.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if fake.requests != 2 {
		t.Errorf("SEFAZ requests = %d, want 2", fake.requests)
	}
}

func TestNFeRecuperarNSU_NothingMissingAsksNoPassword(t *testing.T) {
	env := newNFeTestRoot(t)

	if err := env.run("nfe", "recuperar-nsu", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("recuperar-nsu: %v", err)
	}
	if got := env.out.String(); got != "Nenhum NSU faltando.\n" {
		t.Errorf("output = %q", got)
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
}

func TestCTeRecuperarNSU_RejectsANegativeLimitWithoutTheApp(t *testing.T) {
	calls, err := runWithoutApp(t, "cte", "recuperar-nsu", "-c", nfeTestCNPJ, "--limite", "-1")
	if err == nil || !strings.Contains(err.Error(), "--limite") {
		t.Errorf("err = %v, want the negative limit error", err)
	}
	if calls != 0 {
		t.Errorf("AppFactory called %d times, want 0", calls)
	}
}
