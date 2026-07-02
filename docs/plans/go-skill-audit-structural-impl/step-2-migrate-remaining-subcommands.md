# Step 2 — Migrate Remaining Subcommands to Factories

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 2.

## Goal

Complete the conversion started in step 1. Delete every remaining package-global `*cobra.Command` var and every `init()` registration. After this step, `internal/cli` builds the entire command tree from `NewRootCommand(env)` and every `RunE` reaches the app via `env.AppFactory(ctx)`.

## Files touched

- `internal/cli/credentials_cmd.go` — 4 commands + 3 flag vars + `init()` block (`:100-114`).
- `internal/cli/export.go` — 6 commands + 6 flag vars + `init()` block (`:181-204`) + `printExportResult` helper (already takes `io.Writer` from easy-wins; leave the signature).
- `internal/cli/list.go` — 1 command + 3 flag vars + `init()` block (`:72-78`). The tabwriter writes to `cmd.OutOrStdout()`; preserve that.
- `internal/cli/pull.go` — 1 command + 1 flag var + `init()` block (`:52-56`).
- `internal/cli/status.go` — 1 command + 1 flag var + `init()` block (`:55-59`).
- `internal/cli/init.go` — 1 command, no flags, `init()` at `:25-27`. `application.DataDir` field read at `:20` must keep working (preserve the `DataDir` accessor on `App`; step 5 may thin it).
- `internal/cli/root.go` — drop the package-global `rootCmd` var and the `init()` flag-registration block. `Execute` becomes: build from `NewRootCommand(prodEnv())`, run, return error.

## Implementation pattern (apply per file)

For each command file:

1. Delete the file-level `var ( ... )` flag block.
2. Delete the file-level `var xxCmd = &cobra.Command{...}` declaration(s) — each one becomes a function.
3. Replace `func init() { rootCmd.AddCommand(...); cmd.Flags()... }` with a single `func newXCommand(env CommandEnv) *cobra.Command` that builds and returns the command. Inside it:
   - Declare local flag vars.
   - Register flags on the local `cmd` via `cmd.Flags().StringVarP(&local, ...)`.
   - The `RunE` closure captures `env` and the flag locals; it calls `env.AppFactory(cmd.Context())`.
4. Register the new command in `NewRootCommand(env)` (`internal/cli/root.go`): `root.AddCommand(newXCommand(env))`.

Special cases:

- **`export.go`** has 6 leaf commands and 1 parent (`exportCmd`). Build the parent from `newExportCommand(env)`; the parent's `PersistentFlags` (`exportCNPJ`, `exportCompetence`, `exportDirection`, `exportIncremental`) must move into the parent constructor and stay persistent so the leaf subcommands inherit them. The `exportOut` and `exportChave` flags are per-leaf — keep them on whichever leaf command currently declares them (`exportXlsxCmd.Flags()` etc.). The `MarkPersistentFlagRequired("cnpj")` call moves inside the parent constructor after flags are defined.
- **`init.go`** reads `application.DataDir` directly. Keep that working; after step 5 you might replace the struct-field access with an accessor, but for step 2 preserve it as-is. The `application.Log.Info("Ambiente inicializado com sucesso!", ...)` call at `init.go:19` is the only logging-on-stdout side-effect path — fine to keep through this step.
- **`root.go`'s `init()`** sets `rootCmd.PersistentFlags().BoolVarP(&verbose, ...)`. Move these registrations to `NewRootCommand`, binding against `*env.Verbose` / `*env.Trace`.

## Verification

- `rg -n '^var .*= &cobra.Command\{|^func init\(\)' internal/cli/` must return **nothing**. Zero `init()` functions, zero package-global `*cobra.Command` literals.
- `rg -n '^func New\w+Command\(env CommandEnv\)' internal/cli/` should return ~7 functions (root + company + credential + export + list + pull + status + init = 8; root is `NewRootCommand`).
- `go build ./...` clean.
- `go test ./internal/cli/` — all existing tests pass unchanged.

## Do not

- Do not change `printExportResult`'s signature (`io.Writer` first arg) — that already routes via `cmd.OutOrStdout()` from easy-wins.
- Do not touch desktop.
- Do not yet delete `withTempDataDir` — that waits for step 7 when fakes take over.

## Risk callouts

- `exportCmd`'s persistent flags (the `--cnpj`/`--competencia` triplet) being inherited by leaf subcommands depends on the leaves being added to the parent via `parent.AddCommand(leaf)`, not to root directly. Verify the tree wiring in `NewRootCommand` honors this.
- The default value of `exportOut` differs per leaf (`export.xlsx`, `export.csv`, `export.zip`, `danfse.pdf`, `danfses.zip`). Each leaf owns its own `exportOut` local — do not share the local across leaves or defaults will collide.