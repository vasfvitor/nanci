package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
)

// runWithoutApp executes the command line over a root whose AppFactory
// fails, and returns how many times the factory ran and the error.
func runWithoutApp(t *testing.T, args ...string) (int, error) {
	t.Helper()
	calls := 0
	v, tr := false, false
	root := NewRootCommand(CommandEnv{
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
		Stdout: &bytes.Buffer{},
		AppFactory: func(context.Context) (*app.App, func(), error) {
			calls++
			return nil, nil, errors.New("AppFactory must not be called")
		},
		Verbose: &v,
		Trace:   &tr,
	})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return calls, err
}

// outputFields splits the output into lines, each into its fields.
func outputFields(out string) [][]string {
	var lines [][]string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		lines = append(lines, strings.Fields(line))
	}
	return lines
}

func TestNFeImport_MixedBatch(t *testing.T) {
	env := newNFeTestRoot(t)
	testdata := filepath.Join("..", "nfe", "testdata")
	procnfe := filepath.Join(testdata, "procnfe.xml")
	resumo := filepath.Join(testdata, "resnfe-cancelada.xml")
	invalid := filepath.Join(testdata, "invalid.xml")

	if err := env.run("nfe", "import", "-c", nfeTestCNPJ, procnfe, resumo, invalid); err != nil {
		t.Fatalf("import: %v\n%s", err, env.out.String())
	}
	lines := outputFields(env.out.String())
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4:\n%s", len(lines), env.out.String())
	}
	if want := []string{"importado", "completa", nfeChaveProc}; !slices.Equal(lines[0], want) {
		t.Errorf("line 1 = %v, want %v", lines[0], want)
	}
	if want := []string{"importado", "resumo", nfeChaveCancelada}; !slices.Equal(lines[1], want) {
		t.Errorf("line 2 = %v, want %v", lines[1], want)
	}
	if got := lines[2]; len(got) < 4 || !slices.Equal(got[:4], []string{"rejeitado", "-", "-", invalid + ":"}) {
		t.Errorf("line 3 = %v, want rejeitado - - %s: <motivo>", got, invalid)
	}
	if got, want := strings.Join(lines[3], " "), "Importados: 2 | Atualizados: 0 | Rejeitados: 1"; got != want {
		t.Errorf("counts = %q, want %q", got, want)
	}

	if err := env.run("nfe", "import", "-c", nfeTestCNPJ, procnfe); err != nil {
		t.Fatalf("second import: %v", err)
	}
	if got := outputFields(env.out.String()); !slices.Equal(got[0], []string{"atualizado", "completa", nfeChaveProc}) {
		t.Errorf("second import = %v, want atualizado", got[0])
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
}

func TestNFeImport_AllRejectedFails(t *testing.T) {
	env := newNFeTestRoot(t)
	err := env.run("nfe", "import", "-c", nfeTestCNPJ, filepath.Join("..", "nfe", "testdata", "invalid.xml"))
	if err == nil || err.Error() != "nenhum arquivo importado" {
		t.Errorf("err = %v, want nenhum arquivo importado", err)
	}
	if !strings.Contains(env.out.String(), "Importados: 0 | Atualizados: 0 | Rejeitados: 1") {
		t.Errorf("output lacks the counts:\n%s", env.out.String())
	}
}

func TestImport_BadPathSkipsAppFactory(t *testing.T) {
	existing := filepath.Join("..", "nfe", "testdata", "procnfe.xml")
	missing := filepath.Join(t.TempDir(), "nao-existe.xml")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"nfe missing", []string{"nfe", "import", "-c", nfeTestCNPJ, existing, missing}, "arquivo não encontrado: " + missing},
		{"cte missing", []string{"cte", "import", "-c", nfeTestCNPJ, missing}, "arquivo não encontrado: " + missing},
		{"folder", []string{"nfe", "import", "-c", nfeTestCNPJ, t.TempDir()}, "é uma pasta"},
		{"no files", []string{"cte", "import", "-c", nfeTestCNPJ}, "requires at least 1 arg"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls, err := runWithoutApp(t, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
			if calls != 0 {
				t.Errorf("AppFactory called %d times, want 0", calls)
			}
		})
	}
}

func TestCTeImport_MixedBatch(t *testing.T) {
	env := newCTeTestRoot(t)
	testdata := filepath.Join("..", "cte", "testdata")
	procte := filepath.Join(testdata, "procte.xml")
	cancelamento := filepath.Join(testdata, "proceventocte-cancelamento.xml")
	invalid := filepath.Join(testdata, "invalid.xml")

	if err := env.run("cte", "import", "-c", nfeTestCNPJ, procte, cancelamento, invalid); err != nil {
		t.Fatalf("import: %v\n%s", err, env.out.String())
	}
	lines := outputFields(env.out.String())
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4:\n%s", len(lines), env.out.String())
	}
	if want := []string{"importado", "completa", cteChaveProc}; !slices.Equal(lines[0], want) {
		t.Errorf("line 1 = %v, want %v", lines[0], want)
	}
	if want := []string{"importado", "evento", cteChaveProc}; !slices.Equal(lines[1], want) {
		t.Errorf("line 2 = %v, want %v", lines[1], want)
	}
	if got := lines[2]; len(got) < 1 || got[0] != "rejeitado" {
		t.Errorf("line 3 = %v, want rejeitado", got)
	}
	if got, want := strings.Join(lines[3], " "), "Importados: 2 | Atualizados: 0 | Rejeitados: 1"; got != want {
		t.Errorf("counts = %q, want %q", got, want)
	}
}

func TestCTeImport_AllRejectedFails(t *testing.T) {
	env := newCTeTestRoot(t)
	err := env.run("cte", "import", "-c", nfeTestCNPJ, filepath.Join("..", "cte", "testdata", "invalid.xml"))
	if err == nil || err.Error() != "nenhum arquivo importado" {
		t.Errorf("err = %v, want nenhum arquivo importado", err)
	}
}
