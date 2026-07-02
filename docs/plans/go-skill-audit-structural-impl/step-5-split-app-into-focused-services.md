# Step 5 — Split `app.App` Into Focused Services

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 5.
> Implements structural plan change 4.

## Goal

Break `app.App`'s god-object (24 exported methods + 7 helpers, survey Category 5) into focused services, each owning one concern. Default to **Option A**: keep `App` as a thin facade whose methods become one-line delegations. This preserves all 15 CLI call sites and 22 desktop `a.core.*` call sites verbatim.

## Service clusters (from survey)

| Service | File | Methods |
|---|---|---|
| `CompanyService` | `internal/app/company.go` | `AddCompany`, `ListCompanies`, `AssignCredentialToCompany`, `UpdateCompany` + helpers `resolveCredentialForCompany`, `companyByCNPJ` |
| `CredentialService` | `internal/app/credential.go` | `AddCredential`, `ListCredentials`, `UpdateCredentialPath`, `UpdateCredentialData` + `credentialByID` |
| `SyncService` | `internal/app/sync.go` (split from `pull.go`) | `Pull`, `ResetSyncState`, `Status` |
| `QueryService` | `internal/app/query.go` | `QueryNFSeEvents`, `TestConnection` + `queryGenericEndpoint`, `buildClientForQuery` |
| `DocumentService` | `internal/app/documents.go` (split from `list.go`/`events.go`) | `ListDocuments`, `MarkDocumentsViewed`, `ListEventsForDocument` |
| `ExportService` | `internal/app/export.go` | `ExportCSV`, `ExportXLSX`, `ExportZIP`, `ExportDANFSe`, `ExportDANFSeZIP`, `ExportXML`, `CountPendingExportDocuments` + `bulkExport`, `renderDANFSe` |

## Files touched

- `internal/app/bootstrap.go` — add six service-struct fields to `App`; keep `DataDir` and any logger used by facade methods.
- `internal/app/company.go`, `credential.go`, `query.go`, `list.go`/`events.go` → `documents.go`, `export.go`, `pull.go` → `sync.go` — convert from free methods on `*App` to methods on the new service structs; the bootstrap file wires them from subsets of `Dependencies`.
- `internal/app/bootstrap.go:64-98` (`func New`) — split the validator into per-service pieces; keep the single `New(deps Dependencies) (*App, error)` entrypoint invoked by `AppFactory` and desktop `startup()`.

## Implementation pattern (per service)

1. **Declare the service struct** with the dependency fields it needs (subset of `Dependencies`). Example for `CompanyService`:
   ```go
   type CompanyService struct {
       Log             *slog.Logger
       CompanyRepo     CompanyRepository
       CredentialRepo  CredentialRepository
       DataDir         string
   }
   func newCompanyService(deps Dependencies) CompanyService {
       return CompanyService{
           Log: deps.Log,
           CompanyRepo: deps.CompanyRepo,
           CredentialRepo: deps.CredentialRepo,
           DataDir: deps.DataDir,
       }
   }
   ```
2. **Move methods** from `func (a *App) AddCompany(...)` to `func (s CompanyService) AddCompany(...)`. Use pointer or value receivers consistently with the existing convention (current `*App` is a pointer; services can be values since they're cheap copies of pointers — match what `go vet`/lint accepts; `copyloopvar` won't complain).
3. **Update helper ownership.** `resolveCredentialForCompany`, `companyByCNPJ`, `credentialByID`, `bulkExport`, `renderDANFSe`, `queryGenericEndpoint`, `buildClientForQuery` become methods on (or package-level functions used by) the relevant service.
4. **Validate the service.** Add a small `func (s CompanyService) validate() error { … }` rejecting zero-valued critical deps; call them from `func New`.
5. **Facade delegation.** On `App`, add a `company CompanyService` field; rewrite `AddCompany` to `func (a *App) AddCompany(...) error { return a.company.AddCompany(...) }`. One line per method.

## Implementation steps

1. **Pick one service first** — `CompanyService` — and convert the four `company.go` methods + two helpers. Wire it into `App` as a facade-tested route. Run `go build ./...` and all tests list pass before scaling.
2. **Convert `CredentialService`** next (four methods, simplest dependencies).
3. **Convert `DocumentService`** (three methods; create the new `documents.go`; keep `events.go` for the time being per the existing split, or fold into `documents.go`).
4. **Convert `QueryService`** (two methods + two helpers; depends on `adn.Client` construction — confirm whether that belongs in the service or stays in the existing `buildClientForQuery` helper).
5. **Convert `SyncService`** — `Pull`, `ResetSyncState`, `Status`. The `Status` method reads multiple stores (`SyncRepo`, `DocumentReader`); carry the union of those fields on the service.
6. **Convert `ExportService`** last (nine methods, largest surface). Keep `bulkExport` and `renderDANFSe` as unexported helpers on the service.
7. **Rewrite `New(deps)`** to call the six `newXService(deps)` constructors in sequence, validate each, and assemble `App{...}` with all fields.

## Verification

- `go build ./...` clean.
- `go test ./internal/app/...` and the easy-wins test set; especially the integration tests construct `app.New(app.Dependencies{...})` directly — all must keep passing unchanged.
- `go test ./...` — full repo.
- `wails dev` smoke in `internal/desktop`:
  - Add company / Add credential (CompanyService + CredentialService)
  - Sync one document (SyncService)
  - List documents + view one (DocumentService)
  - Export XLSX / CSV / ZIP / DANFSe / DANFSe ZIP (ExportService)
  - Direct query + test connection (QueryService)
  - Logs console still works
- `make lint` — no new issues vs easy-wins baseline.
- `make security` — passes after touching storage/auth seams (`ExportService` reads XMLStore; `SyncService`, `QueryService` use cert). The plan's test section flagged steps 3 **and** 5 for `make security`; run it.

## Do not

- **Do not** migrate to Option B (delete `App`) in this PR. Step 5 ships Option A. Defer Option B until a code-review finding shows the facade is bypassed or misleading.
- **Do not** change the desktop's `a.core.<Method>` call sites (`internal/desktop/app.go:262-691`) — facade compatibility is the whole point of Option A.
- **Do not** change `app.Dependencies` shape — production wiring (`internal/cli/runtime.go` post-step-1; `internal/desktop/app.go:172-185`) stays intact. `New` consumes `Dependencies` and produces services internally.
- **Do not** add functional-options constructors — the existing `Dependencies` struct is already the right shape per the skill (rejected F.O. for new code matches the easy-wins audit conclusion).
- **Do not** split into more than six services on first pass. Two-cluster refinements (e.g. separating `SyncService` sync use case from `SyncService` status query) are easy follow-ups after the surface is mapped.

## Risk callouts

- **`Pull` carries the most dependencies.** Survey shows it uses `SyncRepo`, `CredentialRepo`, `CompanyRepo`, `CredentialProvider`, `DANFSeRenderer` is *not* used by pull but is used by export — don't over-couple the service. Keep `SyncService`'s deps narrow: company repo + credential repo + credential provider + `newSyncRunner` factory + log.
- **`buildClientForQuery` in `query.go:55`** constructs an `adn.Client` (TLS-loaded cert). It belongs on `QueryService`; but the CLI `credentials_cmd.go` doesn't touch it. Keep the gap clean — `QueryService` owns `adn.Client` construction for both query and (transitively, via `pull`) sync. If duplication appears, lift `adn.Client`-construction into a `client.go` helper shared by `SyncService` and `QueryService`.
- **Test fakes break if interface types changed.** Step 5 only moves method definitions onto services; the *tests* still call `application.AddCompany` (now `application.company.AddCompany` via the facade). No fake signatures change.
- **Desktop binding surface.** Wails generated bindings (`internal/desktop/frontend/wailsjs/go/main/App.d.ts`) reference methods on `*App`; Option A keeps `*App` method names unchanged, so generated front-end code remains accurate. Do not regenerate bindings in this step; they will produce a no-op diff.
- **One PR or many?** Six service conversions are well-defined committable units; a sensible reviewer-experience shape is one commit per service + a final commit that rewrites `New` and the facade. Keep the tree compiling between each.