# Step 1 — AppFactory and Command Factories

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 1.
> Covers structural plan changes 1 and 2 (command factories + `AppFactory`).

## Goal

Introduce `CommandEnv` and `AppFactory`; convert the root command and one representative subcommand (`company`) to constructor-style assembly so the pattern is proven before scaling. The old package-global path keeps working temporarily via the prod factory wired into `NewRootCommand`.

## Files touched

- `internal/cli/root.go` — add `CommandEnv`, `AppFactory`, `NewRootCommand(env)`, change `Execute` to build a fresh root per call.
- `internal/cli/runtime.go` (new) — `prodAppFactory` wrapping the current `newApp()` body.
- `internal/cli/env.go` (new, optional) — holds `CommandEnv` if you don't put it in `root.go`.
- `internal/cli/company.go` — convert `companyCmd`, `companyAddCmd`, `companyListCmd`, `companyAssignCredentialCmd` to constructors; fold flag vars into the constructor closures; delete the `init()` block.
- `internal/cli/root_test.go` — adjust `Execute` callers if the signature/behavior shifted (should still take `ctx` and return `error`).
- `internal/cli/company_test.go` — update `Execute` callers; keep `NANCI_DATA_DIR` workaround for now.

## Implementation steps

1. **Define the seam.** In `internal/cli/root.go` (or a new `env.go`):
   ```go
   type CommandEnv struct {
       Stdin      io.Reader
       Stdout     io.Writer
       Stderr     io.Writer
       AppFactory AppFactory
       Verbose    *bool
       Trace      *bool
   }
   type AppFactory func(ctx context.Context) (*app.App, func(), error)
   ```
   `Verbose`/`Trace` are pointers so flag parsing flips them without package globals.

2. **Lift `newApp()` into `prodAppFactory`.** Move the body of `newApp()` (`root.go:47-96`) into `internal/cli/runtime.go` as:
   ```go
   func prodAppFactory(verbose, trace bool, stdin io.Reader, stdout, stderr io.Writer) AppFactory {
       return func(ctx context.Context) (*app.App, func(), error) {
           // existing body, with os.Stdin/Stdout/Stderr replaced by the params
       }
   }
   ```
   Do not change the `app.Dependencies` wiring yet.

3. **Add `NewRootCommand(env CommandEnv) *cobra.Command`.** Copy today's `rootCmd` literal, set `SilenceUsage: true, SilenceErrors: true` (already there from easy-wins), register `--verbose`/`--trace` against `env.Verbose`/`env.Trace`. Leave subcommand wiring for step 2 except for `company` (this step).

4. **Convert `company`.** In `internal/cli/company.go`:
   - Delete the package-global flag vars (`companyCNPJ`, `companyName`, ... `assignCredentialID`).
   - Replace `var companyCmd = &cobra.Command{...}` + `init()` with `func newCompanyCommand(env CommandEnv) *cobra.Command { ... }`.
   - Inside the constructor, declare `cnpj, name, cert, env, ...` as locals and bind flags against them via `cmd.Flags().StringVarP(&cnpj, "cnpj", "c", "", ...)`. Capture them in the `RunE` closure.
   - `RunE` calls `app, cleanup, err := env.AppFactory(cmd.Context())` instead of `newApp()`.
   - `rootCmd.AddCommand(newCompanyCommand(env))` inside `NewRootCommand`.

5. **Change `Execute`.** Build a fresh root per call:
   ```go
   func Execute(ctx context.Context) error {
       v, tr := false, false
       root := NewRootCommand(CommandEnv{
           Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
           AppFactory: prodAppFactory(&v, &tr, os.Stdin, os.Stdout, os.Stderr),
           Verbose: &v, Trace: &tr,
       })
       return root.ExecuteContext(ctx)
   }
   ```
   Note: a temporary shim may be needed because the *other* subcommands still register via package globals. Either (a) temporarily keep their `init()` blocks adding themselves to a `rootCmd` you no longer expose, or (b) wait until step 2 to remove `rootCmd`. Pick (b): have `NewRootCommand` add `company` only, and also add the still-package-global subcommands by reading their `xxxCmd` vars during the transition. Delete those vars in step 2.

## Verification

- `go build ./cmd/nanci ./internal/cli` succeeds.
- `go test ./internal/cli/` — all 6 existing tests pass.
- `nanci --help` still prints the same header.
- `nanci company list` against a `NANCI_DATA_DIR=t.TempDir()` dir still prints `Nenhuma empresa cadastrada.` to stdout.
- `nanci company add` with an invalid CNPJ still surfaces the error via `Execute`'s return, no usage text.

## Do not

- Do not touch `internal/desktop/app.go` in this step.
- Do not change `app.Dependencies` or any interface in this step — that's steps 3 and 4.
- Do not delete `withTempDataDir` in `company_test.go` yet; that lives until step 7.

## Risk callouts

- The transitional `rootCmd` + `init()` shim during steps 1–2 means a transient state where some subcommands are factory-built and others are package-global. Keep this window to one commit; do not split across PRs mid-migration.
- `Execute` must produce a fresh `*cobra.Command` per call. Hunt any caller that cached the old `rootCmd`.