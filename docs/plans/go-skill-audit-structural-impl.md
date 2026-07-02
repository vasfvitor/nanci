# Go Skill Audit: Structural Refactor — Implementation Plan

## Index — Step-by-step guides

Each execution step below has its own implementation guide under `docs/plans/go-skill-audit-structural-impl/`. Read the step file *before* starting that step; the per-step guides carry concrete file lists, sample code, "do not" lists, and risk callouts the narrative below only references.

| # | Step | Guide | Outcome |
|---|------|-------|---------|
| 1 | `AppFactory` + command factories (root + `company`) | [`step-1-app-factory-and-command-factories.md`](go-skill-audit-structural-impl/step-1-app-factory-and-command-factories.md) | `CommandEnv`, `AppFactory`, `NewRootCommand`; `company` converted as the pattern |
| 2 | Migrate remaining subcommands to factories | [`step-2-migrate-remaining-subcommands.md`](go-skill-audit-structural-impl/step-2-migrate-remaining-subcommands.md) | No `init()`, no package-global `*cobra.Command`, every `RunE` reaches `app` via `env.AppFactory` |
| 3 | Consolidate `XMLStore` to the consumer side | [`step-3-consolidate-xmlstore.md`](go-skill-audit-structural-impl/step-3-consolidate-xmlstore.md) | `app.XMLStore` deleted; depguard rule relaxed with comment |
| 4 | Consolidate `SyncRepository` to the consumer side | [`step-4-consolidate-syncrepository.md`](go-skill-audit-structural-impl/step-4-consolidate-syncrepository.md) | `app.SyncRepository` embed deleted; new `SyncSnapshotStore` (3 methods) |
| 5 | Split `app.App` into focused services (Option A facade) | [`step-5-split-app-into-focused-services.md`](go-skill-audit-structural-impl/step-5-split-app-into-focused-services.md) | Six services behind `App` facade; `New(deps)` validates per service |
| 6 | Rename `internal/service/sync` → `internal/syncrun` | [`step-6-rename-service-sync-to-syncrun.md`](go-skill-audit-structural-impl/step-6-rename-service-sync-to-syncrun.md) | Layer-named directory gone; `AGENTS.md` updated |
| 7 | Harden CLI tests (durable harness) | [`step-7-harden-cli-tests.md`](go-skill-audit-structural-impl/step-7-harden-cli-tests.md) | `NANCI_DATA_DIR` workaround gone; matrix tests added |

The per-step guides are short on purpose. The "Goal", "Files touched", "Implementation steps", "Verification", "Do not", and "Risk callouts" sections in each step file are the load-bearing spec; this main file is the narrative + index.

## Status

Predecessor: `docs/plans/go-skill-audit-easy-wins.md` (executed 2026-07-01).
This plan is the second pass described by `go-skill-audit-structural.md`. Run it on the current `refact` branch; the easy-wins changes (Cobra `SilenceErrors`/`SilenceUsage`, `Execute(ctx) error`, stdout routing through `cmd.OutOrStdout()`, `ProcessingError` type, sentinel renames, `slices.SortFunc`, etc.) are already in place and are the foundation this plan builds on.

## Goal

Convert `internal/cli` from package-global Cobra side effects to a command-factory assembly with an injectable `AppFactory`, consolidate interface ownership to the consumer side, and reduce `app.App` from a 24-method god-object to a thin facade (or split services). The user-visible command tree and behavior stay stable.

Grounding facts (post-easy-wins survey):

- The only application factory today is `newApp()` in `internal/cli/root.go:47`; every `RunE` calls it directly (`company.go:37,76,106`, `credentials_cmd.go:27,48,83`, `export.go:43,72,101,130,156`, `init.go:13`, `list.go:23`, `pull.go:18`, `status.go:17`).
- `app.Dependencies` is already a struct literal with interface-typed fields (`bootstrap.go:50-61`); the desktop wires the same literal at `internal/desktop/app.go:172-185` with only `CredentialProvider.Fallback` swapped to `a.cred`. That single point of variance is the seam this plan widens.
- `internal/desktop/app.go:106` holds `core *app.App`; ~22 `a.core.*` call sites in desktop (`app.go:262-691`) must keep compiling during the split.
- Two interfaces are duplicated: `XMLStore` at `files/blobstore.go:13-16` and `app/repositories.go:47-50` (byte-identical), and `SyncRepository` at `syncservice.SyncRepository` (`service/sync/sync.go:27-36`, 8 methods) plus `app.SyncRepository` (`app/repositories.go:40-45`, embeds `syncservice.SyncRepository` + 3 methods: `LatestSyncSnapshot`, `ResetSyncState`, `HasSyncState`).
- `app.App` has 24 exported methods (survey Category 5) clustered into Company / Credential / Sync / Query / Documents / Export.
- CLI tests today (`internal/cli/root_test.go`, `company_test.go`) use `NANCI_DATA_DIR=t.TempDir()` as a *transitional* isolation workaround; `company_test.go:13-15` documents that this plan is expected to replace it with `AppFactory` injection.
- No `os/exec` CLI tests exist (easy-wins pass cleaned them up).

## Implementation Changes

### 1. Replace package-global Cobra state with command factories

Target end state: `internal/cli` builds the command tree from a `NewRootCommand(env CommandEnv) *cobra.Command` function; no `init()` registration, no package-global `*cobra.Command` vars, no package-global flag vars.

- Introduce `internal/cli/env.go` (or extend `root.go`) with:
  ```go
  type CommandEnv struct {
      Stdin          io.Reader
      Stdout         io.Writer
      Stderr         io.Writer
      AppFactory     AppFactory
      Verbose        *bool
      Trace          *bool
  }
  type AppFactory func(ctx context.Context) (*app.App, func(), error)
  ```
  `AppFactory` replaces the direct `newApp()` calls. `Verbose`/`Trace` become pointers so the factory can route `--verbose`/`--trace` into the logger without mutating package globals.
- Replace each `var xxCmd = &cobra.Command{...}` + `init(){ rootCmd.AddCommand(xxCmd); … }` pair with a constructor:
  - `func newRootCommand(env CommandEnv) *cobra.Command`
  - `func newCompanyCommand(env CommandEnv) *cobra.Command`
  - `func newCredentialCommand(env CommandEnv) *cobra.Command`
  - `func newExportCommand(env CommandEnv) *cobra.Command`
  - `func newListCommand(env CommandEnv) *cobra.Command`
  - `func newPullCommand(env CommandEnv) *cobra.Command`
  - `func newStatusCommand(env CommandEnv) *cobra.Command`
  - `func newInitCommand(env CommandEnv) *cobra.Command`
  The root builder wires subcommands via `rootCmd.AddCommand(newCompanyCommand(env))` etc. Each leaf command closures over `env.AppFactory` for its `RunE`; flags move from package globals into closure-local `var` declarations inside the constructor.
- Keep `Execute(ctx context.Context) error` (already correct from easy-wins) but change its body to build a fresh `rootCmd` per call from `NewRootCommand(currentEnv())`. Cobra accumulates arg/IO state on a command instance; per-call construction makes repeated `Execute` calls in tests safe without `resetRoot` cleanup.
- Lift package globals `verbose`/`trace` (`root.go:19-21`) into the `CommandEnv` so flag parsing flips pointers rather than package state.
- Preserve the user-visible command names, flags, and short descriptions exactly.
- Risk callouts: the package-global `rootCmd` is referenced by `Execute` and by every `init()` chain today; remove all those references atomically in one commit so nothing half-converts.

### 2. Lift runtime assembly (newApp) into an injectable factory module

- Create `internal/cli/runtime.go` exporting `func prodAppFactory(verbose, trace bool, stdin io.Reader, stdout, stderr io.Writer) AppFactory`. This wraps today's `newApp()` body (`root.go:47-96`) without changing its wiring; it closes over the IO streams and the verbose/trace booleans instead of touching `os.Stdin`/`os.Stdout`/`os.Stderr` and the package globals directly.
- Keep the existing desktop / CLI split: `internal/desktop/app.go:172-185` continues to construct `app.Dependencies` directly (it owns a different `CredentialProvider`); leave it alone in this step. The factory is CLI-local for now — the structural plan explicitly scopes desktop out except where app-constructor changes force a small update.
- Replace the 15 `newApp()` call sites in `RunE`s with `app, cleanup, err := env.AppFactory(cmd.Context())`.
- Validation: every behavior verified by the easy-wins CLI tests (`root_test.go`, `company_test.go`) must still pass with the factory injected.

### 3. Consolidate interface ownership to consumer side

Apply skill guidance: interfaces live where they are *consumed*, implementors stay concrete.

- **`XMLStore` duplication** (the trivial one):
  - Delete `app.XMLStore` at `app/repositories.go:47-50`.
  - Update `app.Dependencies.XMLStore` and `app.App.XMLStore` fields (`bootstrap.go:42,57`) to type `files.XMLStore`.
  - Add `import "github.com/vasfvitor/nanci/internal/files"` to `internal/app/repositories.go` and `internal/app/bootstrap.go`.
  - `app` already transitively depends on `files` for the `BlobStore` constructor today (`cli/root.go:84`); formally accepting the consumer interface cuts the duplication without new coupling.
  - Risk: `internal/app` is currently depguarded against importing `internal/files` for non-test production code (`.golangci.yml:60` `app-core` rule denies `internal/files`). Resolve by either (a) widening the depguard rule to allow only `files.XMLStore` — depguard cannot express per-symbol allowlists, so the realistic choice is (b) re-exporting `XMLStore` from `files` and accepting that `app`'s categorical "no files" rule is too strict; relax it for the interface type. Pick (b) and document the carve-out in the depguard rule comment. Include `internal/files` test files are already exempt by the `!**/*_test.go` matcher.
- **`SyncRepository` duplication** (the intentional one):
  - Keep `syncservice.SyncRepository` (`service/sync/sync.go:27-36`) as the consumer interface for the orchestration code.
  - Delete `app.SyncRepository` (`app/repositories.go:40-45`) and replace with a new consumer-owned interface declared in `internal/app` that **only** lists the 3 app-side methods (`LatestSyncSnapshot`, `ResetSyncState`, `HasSyncState`); callers that need both the orchestration and the app-side operations continue to wire `*store.SyncRepository` — a single concrete that satisfies both interfaces structurally. This decouples `app` from `syncservice` (no embedding import) while preserving the structural identity of `*store.SyncRepository`.
  - Update `app.Dependencies.SyncRepo` and `app.App.SyncRepo` field types accordingly.
- Audit `internal/app/repositories.go` for any other interface that is byte-identical to its implementor's exported interface; none were found in the survey beyond the two above.
- Tests: `internal/app/*_test.go` constructs fakes for `SyncRepository` and `XMLStore`; any fake implementing the deleted interface must be updated to implement the surviving consumer interface. `company_test.go:478` returns `files.ErrBlobNotFound` from a fake `XMLStore` — confirm that fake still satisfies the new (now `files.XMLStore`) field type.

### 4. Simplify application/runtime composition

- Split the 24 `*App` methods into focused services, each owning one concern from the survey's clusters:
  - `internal/app/company.go` — `CompanyService` (AddCompany, ListCompanies, AssignCredentialToCompany, UpdateCompany + `resolveCredentialForCompany`, `companyByCNPJ`)
  - `internal/app/credential.go` — `CredentialService` (AddCredential, ListCredentials, UpdateCredentialPath, UpdateCredentialData + `credentialByID`)
  - `internal/app/sync.go` — `SyncService` (Pull, ResetSyncState, Status) — note: the existing `internal/service/sync/sync.go` `syncservice.SyncService` is the low-level orchestrator; this new `app.SyncService` is the higher-level use case that injects `syncservice.newSyncRunner`.
  - `internal/app/query.go` — `QueryService` (QueryNFSeEvents, TestConnection, queryGenericEndpoint, buildClientForQuery)
  - `internal/app/documents.go` — `DocumentService` (ListDocuments, MarkDocumentsViewed, ListEventsForDocument)
  - `internal/app/export.go` — `ExportService` (ExportCSV, ExportXLSX, ExportZIP, ExportDANFSe, ExportDANFSeZIP, ExportXML, CountPendingExportDocuments, bulkExport, renderDANFSe) — heaviest single surface, ~9 methods.
- Each service is constructed from a subset of `app.Dependencies` and exposes only its concern. `app.App` becomes either:
  - **Option A (preferred): thin facade** — `App` keeps all fields but the 24 methods become one-liners delegating to the corresponding service (e.g. `func (a *App) AddCompany(...) error { return a.company.AddCompany(...) }`). Backward compatible: every CLI/desktop call site keeps compiling unchanged. Costs one indirection per call; acceptable.
  - **Option B: full split** — delete `App`, expose the services directly on `Dependencies`, change all 15 CLI + 22 desktop `application.Pull` / `a.core.Pull` call sites to `application.Sync.Pull` / `a.core.Sync.Pull`. Larger blast radius; only take this if Option A's indirection proves to obscure the structure during implementation.
  - Default to **A**. Convert to B only if a follow-up review shows the facade is routinely bypassed or feels misleading.
- Validate dependencies on construction: each focused service rejects its own zero-value dependencies (e.g. `ExportService` requires `XMLStore`, `DANFSeRenderer`, `DocumentReader`, `DocumentTracker`). The existing `app.New` validator in `bootstrap.go:64-84` becomes the union of the per-service validators; keep the single `app.New(deps Dependencies) (*App, error)` entrypoint as the seam both CLI `AppFactory` and desktop `startup()` call.
- `application.DataDir` field access in `cli/init.go:20` is the only place the CLI reaches into App state; preserve a `DataDir() string` accessor on `App` (cheap delegation to the kept `DataDir` field).
- Risk callout: ~22 desktop sites read `a.core.<Method>`. Under Option A all of these keep working without desktop edits. Under Option B each becomes `a.core.<Service>.<Method>` and the desktop PR spans `internal/desktop/app.go` plus the generated `wailsjs` bindings — defer B until a dedicated desktop PR to keep blast radius manageable.

### 5. Revisit package boundaries with a domain-first bias

- Keep `internal/` for the compiler boundary — do not flatten.
- Rename `internal/service/sync/` → `internal/syncrun/` (or `internal/pull/`) to drop the `service/` layer-name directory segment. Package name `syncservice` → `syncrun`. Update imports in `internal/app/pull.go:14`, the `newSyncRunner` factory (`pull.go:21`), and `internal/service/sync/*_test.go`. The package's consumer-owned `SyncRepository` interface stays in `syncrun`; the duplication fix in step 3 already removed the `app.SyncRepository` re-declaration.
  - Alternative: leave it as `internal/service/sync` and accept the layer-naming, since `AGENTS.md` already acknowledges the path. Default to the rename because the structural plan explicitly asks for the move; the AGENTS.md note can be updated in the same commit.
- No other packages need moving. Do not touch `internal/desktop/frontend` or `wailsjs` (per structural plan scope).
- Pair the move with import cleanup across `internal/app`, the reflectively-targeted `*_test.go`, and any `go.sum` reference churn from the module-path shift (there is none — module path is unchanged, only directory moves).

### 6. Build a durable CLI test harness

- Add `internal/cli/harness_test.go` (or `harness.go` exported to `*_test.go`) with a `newTestRoot(t *testing.T, deps app.Dependencies) *cobra.Command` helper that:
  - Constructs a real `app.New(deps)` with in-memory fakes (already-seen patterns: `company_test.go:478` fake `XMLStore`, the `*Repository` fakes in `internal/app/*_test.go`).
  - Builds a `CommandEnv` with `bytes.Buffer` stdout/stderr and the fakes wired.
  - Returns a fresh `*cobra.Command` per call so tests do not share cobra state.
- Replace the `NANCI_DATA_DIR=t.TempDir()` workaround in `company_test.go:13-15` with `AppFactory` injection. Real SQLite may still be used for repository-touching tests by injecting a `*sql.DB` backed by an in-memory or `t.TempDir()` DB, but the harness lets most tests avoid the filesystem entirely.
- Add table-driven tests covering, per the structural plan's Category 6:
  - argument validation (already partially covered by `TestCompanyList_MissingRequiredFlag`; expand to each subcommand's required flags).
  - flag interactions (e.g. `--last-12-months` overrides `--sync-start-policy` in `company add`).
  - success output (assert captured stdout bytes for `company list`, `status`, `pull`, each `export` subcommand).
  - runtime failures (assert a `RunE`-returned error surfaces once without usage).
  - interactive password-prompt paths: inject a fake `CredentialProvider` returning a canned password; assert no real `os.Stdin`/`os.Stderr` access.
- Delete the `withTempDataDir` helper (`company_test.go:13-26`) once all `company_test.go` cases are migrated to the harness.
- Keep the `resetRoot`/`tempCommand` helpers in `root_test.go` if still useful, or migrate them into the harness. Per-call `NewRootCommand` should make `resetRoot` redundant.

## Execution Order

1. **Step 1 — `AppFactory` and command factories (steps 1+2 together):** introduce `CommandEnv`/`AppFactory`, build `NewRootCommand(env)` and one converted subcommand (`company`) end-to-end as the pattern. Keep the old package-global path working temporarily by having `Execute` build from env with the prod factory. Run the easy-wins CLI tests.
2. **Step 2 — migrate remaining subcommands** to factories: credentials, export, list, pull, status, init. Drop the package globals and the `init()` registrations. Delete the transitional `withTempDataDir` once `AppFactory` is injectable in tests.
3. **Step 3 — consolidate `XMLStore`:** relax depguard, delete `app.XMLStore`, point `app.Dependencies.XMLStore` at `files.XMLStore`. Update test fakes.
4. **Step 4 — consolidate `SyncRepository`:** delete `app.SyncRepository`, declare the app-side-only interface in `internal/app`, update `app.Dependencies.SyncRepo`. Update `syncservice` and test fakes.
5. **Step 5 — split `app.App` into focused services (Option A):** introduce `CompanyService`, `CredentialService`, `SyncService`, `QueryService`, `DocumentService`, `ExportService` behind the `App` facade. Validate per-service. Run `go test ./...`.
6. **Step 6 — rename `internal/service/sync` → `internal/syncrun`:** move files, update imports, update `AGENTS.md` line referencing `internal/service/sync`.
7. **Step 7 — harden CLI tests:** build the shared harness; convert existing tests; add the table-driven coverage matrix.

Each step ends with `make fmt && make lint && go test ./...` plus a desktop smoke (when step 5 lands) to confirm no Wails binding breakage.

## Test Plan

- After every step: `go test ./internal/cli ./internal/app ./internal/service/sync ./internal/files` then `go test ./...`.
- After step 5: `wails dev` smoke in `internal/desktop` (add company, add credential, sync, list documents, export, direct query, password prompt, logs console — the AGENTS.md smoke list).
- After step 6: confirm desktop `go vet` still passes inside `internal/desktop`'s separate module (frontend/dist embed is a pre-existing constraint; a placeholder `frontend/dist/.gitkeep` lets the Go-level vet run).
- Run `make security` after steps 3 and 5 (steps touching storage/auth seams).

## Acceptance Criteria

- No Cobra command registration relies on package `init()` side effects or package-global command instances. A fresh root command is constructed per `Execute` call from `NewRootCommand(env CommandEnv)`.
- The CLI layer owns Cobra concerns only; business operations are dispatched through an injected `AppFactory` (production) or fakes (tests). No `internal/app`, `internal/syncrun`, or domain package imports `cobra`.
- `XMLStore` exists in one place only (`internal/files`), owned by the consumer side via direct import.
- `SyncRepository` exists in two places, each listing only what its consumer needs: `syncservice.SyncRepository` (8 orchestration methods) and an app-local interface with 3 app-side methods. `app.SyncRepository` as a superset embedding is gone.
- `app.App` is a thin facade over six focused services; new feature work lands in the service that owns the concern, not on `App` directly.
- CLI test coverage includes argument validation, flag interactions, success output, runtime failures, and interactive password paths, executing in memory against injected fakes.
- All existing easy-wins behavior preserved: `Execute(ctx) error` contract, `SilenceUsage`/`SilenceErrors`, stdout routing through `cmd.OutOrStdout()`, `ProcessingError` returned by `syncservice` and rendered once at a boundary.

## Assumptions

- Option A (facade split) is the default; Option B (full split) is deferred to a follow-up PR if a review shows the facade is bypassed.
- `internal/desktop/frontend` and `wailsjs` generated bindings are out of scope; step 5 deliberately keeps `a.core.<Method>` call sites working so the generated Wails surface is unchanged.
- The depguard rule relaxing `internal/app` to import `internal/files` for `XMLStore` is explicitly documented in `.golangci.yml` so future contributors understand the carve-out.
- The pre-existing `desktop` test `TestFormatExportErrorAddsRecoveryHintOnlyForMissingFiles` (which fails on HEAD because `formatExportError` is a stub returning the error unchanged at `internal/desktop/app.go:550`) is not in scope; track separately.
- `internal/desktop` cannot run `go test` without a built `frontend/dist` (pre-existing embed gate from `main.go:24`); verification inside that module uses a placeholder `frontend/dist/.gitkeep` and only covers `go vet`/`go build`, not the runtime Wails flows.