# Step 7 — Harden CLI Tests (Durable Harness)

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 7.
> Implements structural plan change 6.

## Goal

Replace the transitional `NANCI_DATA_DIR=t.TempDir()` workaround in `internal/cli/company_test.go` with a shared in-memory harness that injects `AppFactory` fakes. Expand coverage across the five categories the structural plan calls for: argument validation, flag interactions, success output, runtime failures, interactive password prompts.

## Files touched

- `internal/cli/harness_test.go` (new) — shared `newTestRoot(t *testing.T, deps app.Dependencies) *cobra.Command` helper.
- `internal/cli/root_test.go` — keep the three existing tests; if `resetRoot` becomes redundant (because `NewRootCommand` builds a fresh root per `Execute` call from step 1), remove `resetRoot` to reduce noise.
- `internal/cli/company_test.go` — replace `withTempDataDir` with the harness invoking `AppFactory` returning `app.New(app.Dependencies{...})` wired with fakes. Each test then drives the relevant table-driven matrix.
- `internal/cli/credentials_test.go` (new) — argument validation for `credential add`, `credential list`, `credential update-path` required flags.
- `internal/cli/export_test.go` (new) — success output for each format end (xlsx/csv/zip/danfse/danfse-zip), plus argument validation for shared persistent `--cnpj` flag.
- `internal/cli/pull_test.go` (new, optional) — runtime failures (sync returning `*ProcessingError` surfaces once via `Execute`; assert stderr silence and stdout silence). Flag interactions with `--last-12-months` style overrides belong on `company_test.go`'s `company add`.
- `internal/cli/query_test.go` (new, optional) — interactive password-prompt path with a fake `CredentialProvider` returning a canned password; assert no `os.Stdin`/`os.Stderr` access and that the canned password flows to the client.

## Harness design (draft)

```go
package cli

import (
    "bytes"
    "context"
    "io"
    "testing"

    "github.com/vasfvitor/nanci/internal/app"
)

type testEnv struct {
    root   *cobra.Command
    stdout *bytes.Buffer
    stderr *bytes.Buffer
}

// newTestRoot wires a CommandEnv pointing at an in-memory app constructed from
// the given Dependencies. Tests can pass real fakes (XMLStore, *Repository
// stubs, slog.NewTextHandler(io.Discard, nil)) and avoid any filesystem touch.
func newTestRoot(t *testing.T, deps app.Dependencies) testEnv {
    t.Helper()
    a, cleanup, err := app.New(deps)              // validation runs
    if err != nil {
        t.Fatalf("app.New: %v", err)
    }
    t.Cleanup(cleanup)

    var v, tr bool
    out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
    factory := func(context.Context) (*app.App, func(), error) {
        return a, func() {}, nil
    }
    root := NewRootCommand(CommandEnv{
        Stdin:      bytes.NewReader(nil),
        Stdout:     out,
        Stderr:     errOut,
        AppFactory: factory,
        Verbose:    &v,
        Trace:      &tr,
    })
    return testEnv{root: root, stdout: out, stderr: errOut}
}

// runArgs is a thin helper that sets args, runs Execute, and returns the error
// so table-driven tests can assert on it directly.
func (e testEnv) runArgs(args ...string) error {
    e.root.SetArgs(args)
    return e.root.ExecuteContext(context.Background())
}
```

## Test matrix to add

| File | Test | Category | Description |
|---|---|---|---|
| `company_test.go` | `TestCompanyAdd_Last12MonthsOverridesPolicy` | flag interactions | `--last-12-months` overrides `--sync-start-policy=since_date` |
| `company_test.go` | `TestCompanyAdd_InvalidEnv_ReturnsError` | argument validation | unknown `--env` → `nfse.ErrInvalidEnum` wrapped |
| `company_test.go` | `TestCompanyList_Empty_PrintsPlaceholder` | success output | existing test, ported from `NANCI_DATA_DIR` to the harness |
| `credentials_test.go` | `TestCredentialAdd_RequiredFlags` | argument validation | missing `--cert` → error, no usage |
| `credentials_test.go` | `TestCredentialList_Empty_PrintsPlaceholder` | success output | empty fake `CredentialRepo` → `Nenhuma credencial cadastrada.` |
| `export_test.go` | `TestExportXLSX_WritesSuccessHeader` | success output | fake `ExportService` returns 1 doc; assert `Arquivo xlsx gerado com sucesso: <path>` to stdout |
| `export_test.go` | `TestExportCmd_RequiresCNPJ` | argument validation | missing persistent `--cnpj` → error |
| `pull_test.go` | `TestPull_FactoryReturnsProcessingError_Propagates` | runtime failures | fake `SyncService` returns `&ProcessingError{...}`; assert `Execute` returns it and stderr is silent (Cobra boundary) |
| `query_test.go` | `TestQueryFake_FakeCredentialProvider_ReturnsPassword` | interactive password | inject `CredentialProvider` returning `"fakepass"`; assert client receives it; assert no stdin/stderr pollution |
| `root_test.go` (existing 3 tests) | unchanged | boundary | keep as-is or simplify if `resetRoot` is no longer needed |

## Implementation steps

1. **Build the harness** in `harness_test.go`. Verify it builds against the post-step-1 `CommandEnv`/`NewRootCommand`.
2. **Port `company_test.go`** to the harness. Delete `withTempDataDir` (`company_test.go:13-26`) and `NANCI_DATA_DIR` usage. Confirm the three existing tests still pass (now in-process).
3. **Add the matrix tests** one table at a time. Each test must run without touching the filesystem and without mutating package globals (the harness builds a fresh root per call).
4. **Simplify `root_test.go`** if `resetRoot` is no longer needed. The `tempCommand` helper for registering throwaway subcommands can stay for boundary tests.
5. **Run all tests** with `-race` at least once: `go test -race ./internal/cli/...`.

## Verification

- `rg -n 'NANCI_DATA_DIR' internal/cli/` — **zero** hits. The transitional workaround is gone.
- `rg -n 't\.Setenv' internal/cli/` — only legitimate env cases (if any); the workspace workaround no longer.
- `go test -race ./internal/cli/...` — all tests pass.
- `go test ./...` — full repo still green.
- `make lint` — no test lint complaints (e.g. `testifylint`, `paralleltest` if enabled). If `paralleltest` is enabled, ensure table tests use `t.Run(tt.name, ...)` so each case parallélizes; not strictly required by AGENTS but matches `make lint`.
- Manual smoke: at minimum, `nanci company list` and `nanci company add --cnpj 11222333000181 --name "Test" --cert /tmp/test.pfx` against a real data dir still works (the harness in-process tests are the stable regression net; this is the boundary smoke).

## Do not

- **Do not** keep `NANCI_DATA_DIR=t.TempDir()` as a fallback. Step 7's purpose is to remove it; tests that genuinely need a real DB should use an in-memory SQLite via injected `*store` constructed against an `:memory:` conn (modernc.org/sqlite supports `":memory:"` DSN). If that fails, fall back to `t.TempDir()` scoped to `*store` *only* and keep the harness agnostic.
- **Do not** introduce `ginkgo` or any BDD framework. The skill is explicit: `go test`-vanilla + table-driven is the project convention.
- **Do not** write `os/exec` tests invoking the compiled binary. Easy-wins removed them; structural plan forbids reintroducing.
- **Do not** add generated Wails frontend bindings test coverage — those belong under `internal/desktop/frontend/src/__tests__/` per AGENTS.md's frontend note.

## Risk callouts

- The `*store` repository fakes need a real `*sql.DB`. The cleanest test seam is to introduce a tiny `store.NewRepositories(db *sql.DB)` constructor (not required by step 7; flag for a future refactor) that lets tests pass an `:memory:` SQLite. For step 7, prefer constructing real `*store` repos against `t.TempDir()` DBs *behind* the harness; the harness gets `app.New(app.Dependencies{...})` pre-filled, so tests express intent cleanly without the env var.
- `export_test.go`'s success-output tests need a fake `XMLStore` returning canned blobs and a fake `DANFSeRenderer` for the danfse-* formats. Match the pattern at `internal/app/company_test.go:478` (`return nil, files.ErrBlobNotFound`-returning fake) — that fake predates step 7 and is the established idiom.
- `pull_test.go` needs to inject a `SyncService` or `syncRunner` fake. The private `newSyncRunner` factory (`internal/app/pull.go:21`) is package-private — tests in `package cli_test` (black-box) cannot patch it. Either (a) drop pull's deep test in favor of an `app_test`-level test already covering `Pull`, or (b) move the harness to white-box `package cli` (current) and reach the seam via a package-private injection function. Prefer (a): keep the CLI test matrix surface-light, defer back-end `Pull` to `internal/app/*_test.go`.