# Go Skill Audit: Follow-up — Completing the Smaller Orchestration Surface

## Status

Predecessors:
- `docs/plans/go-skill-audit-easy-wins.md` (executed 2026-07-01)
- `docs/plans/go-skill-audit-structural.md` (parent)
- `docs/plans/go-skill-audit-structural-impl.md` (steps 1–7 executed 2026-07-01)

This is a **deferred** plan. The structural pass's only explicit folder move (`internal/service/sync` → `internal/syncrun`) is done, and step 5 (focused services) is partial. The remaining work finishes the "smaller orchestration surface" goal from `go-skill-audit-structural.md` §5. It is intentionally **not implemented in the current pass** — the user wants the scope captured here for a future commit/PR.

## Why this plan exists

`go-skill-audit-structural.md` §5 says:

> - Preferred end state for runtime-oriented code:
>   - CLI adapter package
>   - domain packages such as `nfse`, `adn`, `report`, `files`
>   - **a smaller orchestration surface instead of broad `app` plus `service` layering**

After the easy-wins + structural pass:

- `internal/service/sync` is renamed to `internal/syncrun` (the `service/` layer-name directory is gone).
- `internal/app.App` has 24 exported methods, but 4 of those (Company, Credential, SyncStatus, Query) now delegate to focused service structs (`CompanyService`, `CredentialService`, `SyncStatusService`, `QueryService`) on the App as fields.
- The remaining `*App` methods — `Pull`, `Status` (not yet split), `Export*` (7 methods), `ListDocuments`, `MarkDocumentsViewed`, `ListEventsForDocument`, `ResetSyncState`, plus helpers (`bulkExport`, `renderDANFSe`, `companyByCNPJ`/`credentialByID` re-removed in step 5) — still live as `*App` methods.

The step 5 plan (`docs/plans/go-skill-audit-structural-impl/step-5-split-app-into-focused-services.md`) called for six services; we shipped four. This plan captures the remaining two-and-a-half and the open question of package-level restructuring.

## Scope

Two scope buckets, listed in the recommended execution order. Bucket A finishes the partial step 5. Bucket B reconsiders the package-level split that the structural plan leaves open in its execution-order step 6.

## Bucket A — Finish the step 5 service extraction

### A1. DocumentService

Methods to extract from `*App` to `*DocumentService`:
- `ListDocuments(ctx, ListInput) ([]nfse.CompanyDocument, error)` — `internal/app/list.go:19`
- `MarkDocumentsViewed(ctx, ListInput) (int, error)` — `internal/app/list.go:45`
- `ListEventsForDocument(ctx, documentID) ([]EventView, error)` — `internal/app/events.go:18`

Service struct shape:
```go
type DocumentService struct {
    DocumentReader DocumentReader
}

func newDocumentService(d Dependencies) DocumentService {
    return DocumentService{DocumentReader: d.DocumentReader}
}
```

All three methods already call `a.DocumentReader` only. The service is the smallest of the four. The `ListInput` type stays in `internal/app` (it's the app's input DTO, even if it now lands on a service).

App facade additions to `internal/app/facade.go`:
```go
func (a *App) ListDocuments(ctx context.Context, input ListInput) ([]nfse.CompanyDocument, error) {
    return a.documents.ListDocuments(ctx, input)
}
func (a *App) MarkDocumentsViewed(ctx context.Context, input ListInput) (int, error) {
    return a.documents.MarkDocumentsViewed(ctx, input)
}
func (a *App) ListEventsForDocument(ctx context.Context, documentID string) ([]EventView, error) {
    return a.documents.ListEventsForDocument(ctx, documentID)
}
```

`internal/app/bootstrap.go` `New` adds `documents: newDocumentService(deps)` and `InitServicesForTest` adds the same.

The struct is value-typed like the other services; `App.documents` is the only field.

### A2. ExportService

Methods to extract from `*App` to `*ExportService`:
- `ExportCSV(ctx, ExportInput) (ExportResult, error)` — `internal/app/export.go:48`
- `ExportXLSX(ctx, ExportInput) (ExportResult, error)` — `internal/app/export.go:55`
- `ExportZIP(ctx, ExportInput) (ExportResult, error)` — `internal/app/export.go:62`
- `ExportDANFSeZIP(ctx, ExportInput) (ExportResult, error)` — `internal/app/export.go:69`
- `ExportDANFSe(ctx, ExportDANFSeInput) error` — `internal/app/export.go:189`
- `ExportXML(ctx, ExportXMLInput) error` — `internal/app/export.go:234`
- `CountPendingExportDocuments(ctx, ExportInput, kind string) (int, error)` — `internal/app/export.go:283`
- Helpers: `bulkExport`, `renderDANFSe`

Service struct shape:
```go
type ExportService struct {
    Log              *slog.Logger
    DocumentReader   DocumentReader
    DocumentTracker  DocumentTracker
    XMLStore         files.XMLStore
    DANFSeRenderer   danfse.Renderer
    DataDir          string
}

func newExportService(d Dependencies) ExportService {
    return ExportService{
        Log: d.Log, DocumentReader: d.DocumentReader, DocumentTracker: d.DocumentTracker,
        XMLStore: d.XMLStore, DANFSeRenderer: d.DANFSeRenderer, DataDir: d.DataDir,
    }
}
```

`renderDANFSe` reads from the DANFSe renderer only. `bulkExport` needs `DocumentTracker` and a `*nfse.CompanyDocument`-from-document iteration. The struct is the heaviest of the four to convert because it has the most methods and the most cross-cutting deps.

App facade additions:
```go
func (a *App) ExportCSV(ctx context.Context, input ExportInput) (ExportResult, error) {
    return a.exports.ExportCSV(ctx, input)
}
// ...same for XLSX, ZIP, DANFSeZIP, DANFSe, XML, CountPendingExportDocuments
```

`New` adds `exports: newExportService(deps)`. `InitServicesForTest` adds the same.

The DTOs `ExportInput`, `ExportDANFSeInput`, `ExportXMLInput`, `ExportResult` stay in `internal/app` (they are the app's contract surface, not the service's). The service methods take them as parameters.

### A3. SyncService (around `Pull`)

The current `*App.Pull(ctx, PullInput) (PullResult, error)` is the most complex method in the file: it loads the cert, builds the ADN client, runs the orchestrator, captures progress, and writes the snapshot back. Extract it (and its helpers `parsePullMode`, `validateConsultationCompatibility`, `resolveEnvironmentURL`, the existing `newSyncRunner` factory) into a `SyncService` that owns the sync use case.

Service struct shape:
```go
type SyncService struct {
    Log                *slog.Logger
    CompanyRepo        CompanyRepository
    CredentialRepo     CredentialRepository
    CredentialProvider CredentialProvider
    SyncRepo           *store.SyncRepository
    XMLStore           files.XMLStore
}
```

`Pull` becomes a method on `SyncService`. The `newSyncRunner` factory lives in this file. The package-level helpers `parsePullMode`, `validateConsultationCompatibility`, `resolveEnvironmentURL` move with the service.

`ResetSyncState` is currently a separate method on `*App` (`internal/app/reset_sync.go:15`). It belongs to the same use case (sync lifecycle). Two options:
- (A) Put it on `SyncService` too. Con: increases the service's surface.
- (B) Keep it on `*App` as a single delegating method. Pro: matches the existing `ResetSyncInput`/`Status` patterns.

**Default: (A).** `SyncService` becomes the "sync lifecycle" service with `Pull` and `ResetSyncState`. The existing `SyncStatusService` already covers `Status`; that stays separate because it is a read-only summary, not a use case.

App facade:
```go
func (a *App) Pull(ctx context.Context, input PullInput) (PullResult, error) {
    return a.sync.Pull(ctx, input)
}
func (a *App) ResetSyncState(ctx context.Context, input ResetSyncInput) error {
    return a.sync.ResetSyncState(ctx, input)
}
```

`New` adds `sync: newSyncService(deps)`. `InitServicesForTest` adds the same.

### A4. Verification after Bucket A

- `go build ./...` clean
- `go test ./...` passes (the pre-existing `TestAppIntegration_SyncPreferencesFlow` timezone failure is unrelated)
- `golangci-lint` produces the same issue list as the post-step-7 baseline; the only `contextcheck` warning in `internal/cli/init.go:13` is unchanged
- `go run ./cmd/nanci --help` and `nanci company list` work end-to-end
- `go test -race ./internal/cli/...` (the harness has more concurrent surface now; race detector helps)
- Optional: a wails dev smoke of the password prompt, sync, export paths

The expected end state of Bucket A is **six focused services** behind the `App` facade. `*App` itself becomes a 12-line struct with field accessors and one-line facade methods. `internal/app` reads as a "wiring + facade" package, not a "bag of methods."

## Bucket B — Reassess the package boundary

This is the open question from the structural plan's Execution Order step 6:

> 6. Reassess whether `app.App` should remain a single facade or be split.

The plan's preferred end state is "a smaller orchestration surface instead of broad app plus service layering." Bucket A reduces the surface within `internal/app`. Bucket B reconsiders whether `internal/app` itself should be split.

The survey and plan do not commit to a specific shape. Three candidate directions, in increasing order of churn:

### B1. Status quo (recommended default)

Keep `internal/app` as one package with focused service structs. This is the end state of Bucket A. The "service" directory is gone (renamed to `syncrun`), and `app` is no longer a god-object. The structural plan's "smaller orchestration surface" is satisfied without moving code between packages.

Acceptance: the `app` package's imports shrink, but its directory stays at `internal/app/`. No new packages created.

### B2. Split by domain

Move each focused service to its own package and delete `internal/app`. The result is a "thin orchestrator" pattern where each use case lives next to its inputs/outputs:

- `internal/company` — `CompanyService`, `AddCompanyInput`, `UpdateCompanyInput`, `AssignCredentialInput`
- `internal/credential` — `CredentialService`, `AddCredentialInput`, `UpdateCredentialPathInput`, `UpdateCredentialDataInput`
- `internal/document` — `DocumentService`, `ListInput`
- `internal/export` — `ExportService`, `ExportInput`, `ExportDANFSeInput`, `ExportXMLInput`, `ExportResult`
- `internal/pull` — `SyncService`, `PullInput`, `PullResult`, `ResetSyncInput`
- `internal/query` — `QueryService`, `QueryNFSeInput`, `ConnectionTestResult`
- `internal/status` — `SyncStatusService`, `StatusResult`
- `internal/runtime` (or keep `internal/app`) — `Dependencies`, `New`, `CommandEnv` (or its equivalent) — the wiring only

This is the most thorough. The trade-off is import churn: every CLI call site changes from `application.Pull(...)` to `application.sync.Pull(...)` (or via a thin runtime facade). The `internal/desktop` package is similarly affected (22 `a.core.*` sites).

The structural plan does not explicitly call for B2. The plan says "**a smaller orchestration surface**" — the word "surface" is ambiguous between "package surface" and "API surface."

**Recommendation: defer B2 unless a follow-up review shows the facade is bypassed.**

### B3. Hybrid: keep the facade, split the internals

Keep `internal/app` as a thin wiring/facade package (current `App` shape) but move the heavy `Export*` and `Pull` methods out into their own packages:

- `internal/app/company.go`, `credential.go`, `query.go`, `status.go`, `list.go`, `events.go`, `reset_sync.go` — keep here, with their existing `*App` methods (or service structs in Bucket A)
- `internal/app/export.go` → `internal/export/` with `ExportService`
- `internal/app/pull.go` → `internal/pull/` with `SyncService`

CLI and desktop keep calling `application.ExportXLSX(...)` and `application.Pull(...)` because the `App` facade retains those one-liners; only the implementation files move. This is the lowest-churn way to honor "smaller orchestration surface" while keeping the `app` package's API stable.

Acceptance: `internal/app` shrinks from ~10 files to ~7; `internal/export/` and `internal/pull/` are new packages with one service struct each.

**This is the most pragmatic extension of Bucket A. It is the option I would pick if B is in scope.**

## Recommended execution

1. **Bucket A only** in the next commit. Ship A1, A2, A3 in order, running `go build && go test ./...` between each. Update `internal/app/facade.go` and `internal/app/bootstrap.go` per step. Commit each service as a separate commit (`app: extract DocumentService`, etc.) for easy review.
2. **Hold on Bucket B** until a follow-up review surfaces a concrete reason: god-object regression, circular import between services, an exporter or sync contributor asking for a tighter scope. The structural plan's open-ended "reassess" supports deferring this.

## Test plan for the future implementation

- After each service extract: `go build ./... && go test ./internal/app/... -skip 'TestAppIntegration_SyncPreferencesFlow' && go test ./internal/cli/...`
- After all three services: full `go test ./...` and `make security` (the export and sync services touch the `XMLStore`, `DANFSeRenderer`, and `CredentialProvider` seams that the security script flags)
- A wails dev smoke covering add company → add credential → sync → export xlsx/zip/danfse → list documents → mark viewed (the surfaces the new service structs own)
- A focused `go test -race ./internal/cli/...` run; the in-memory harness now exercises more service paths and the race detector is cheap insurance

## Acceptance criteria for Bucket A

- `*App` has 12-line struct body (or close) with 7 service fields: `company`, `credential`, `documents`, `exports`, `query`, `status`, `sync`
- `internal/app/facade.go` has 24 one-liner delegations: every `*App` method delegates to a service struct
- All `*App` business methods are no longer defined on `*App`; only the facade methods remain
- The `CLI` and `desktop` call sites still work unchanged
- `App` no longer imports `adn`, `cert`, `danfse`, `nfse.NewMoneyFromCents` paths; the service files import them instead

## Assumptions

- The deferred work is captured in code, not just this document, so a future implementer can pick it up without re-surveying
- The pre-existing `TestAppIntegration_SyncPreferencesFlow` timezone test stays a known failure
- The structural plan's broader "smaller orchestration surface" goal is satisfied by Bucket A + the deferred B1 status quo; B2/B3 are not pursued unless future contributors flag a need
- The desktop `WailsCredentialProvider` refactor and the in-memory harness cleanup in step 7 (which is done) are not touched by this follow-up

## Estimated size

Bucket A is roughly comparable to step 5's existing extract (4 services in step 5 took ~200 lines of facade + 4 service files; the remaining 3 are similar in surface). Bucket B is open-ended and would warrant a separate plan if pursued.