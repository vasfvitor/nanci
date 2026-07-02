# Step 4 — Consolidate `SyncRepository` to the Consumer Side

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 4.
> Implements part of structural plan change 3.

## Goal

End the `SyncRepository` duplication. Today:
- `syncservice.SyncRepository` (`internal/service/sync/sync.go:27-36`) — 8 methods used by the orchestrator: `GetOrCreateState`, `StartRun`, `PersistProgress`, `MarkInitialSyncCompleted`, `FinishRun`, `ApplyDocumentAndProgress`, `CompanyDocumentExistsByAccessKey`, `ApplyEventAndProgress`.
- `app.SyncRepository` (`internal/app/repositories.go:40-45`) — *embeds* `syncservice.SyncRepository` and adds `LatestSyncSnapshot`, `ResetSyncState`, `HasSyncState`.

Keep `syncservice.SyncRepository` (consumer over the orchestrator). Replace `app.SyncRepository` with a small app-local interface listing **only** the 3 app-side methods. The concrete `*store.SyncRepository` structurally satisfies both; callers wiring app and orchestrator use the same concrete.

## Files touched

- `internal/app/repositories.go:40-45` — delete the `SyncRepository` declaration; insert a new 3-method interface:
  ```go
  type SyncSnapshotStore interface {
      LatestSyncSnapshot(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, cnpj string) (*nfse.SyncSnapshot, error)
      ResetSyncState(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, cnpj string) error
      HasSyncState(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, cnpj string) (bool, error)
  }
  ```
  (Verify parameter types from the existing `app.SyncRepository` declarations before adopting this signature verbatim.)
- `internal/app/bootstrap.go:39,53` — change `SyncRepo` field type on both `App` and `Dependencies` from `SyncRepository` to `SyncSnapshotStore`.
- `internal/app/pull.go:21` — the `newSyncRunner` factory currently reads `SyncRepository` (the 11-method superset) so it can pass the repo subset to `syncservice.NewSyncService(syncservice.SyncRepository, …)`. After split, `newSyncRunner` takes the wider type needed by the orchestrator. Pick one of:
  - **(a) Inject the wider concrete.** Change `newSyncRunner`'s parameter to `*store.SyncRepository` (concrete) and pass it directly to `syncservice.NewSyncService`. Simple; loses the seam for the orchestrator caller.
  - **(b) Keep the orchestrator-side consumer interface.** Take `syncservice.SyncRepository` plus the app-side `SyncSnapshotStore` separately in `newSyncRunner` — but then both arguments resolve to the same struct at call sites.
  - **Default: (a).** `app.App.SyncRepo` is the *app-side* seam (`SyncSnapshotStore`); `newSyncRunner` accepts the same concrete `*store.SyncRepository` (or `syncservice.SyncRepository`) to ferry the wider 8-method surface to `syncservice`. Keep `app.App.SyncRepo = SyncSnapshotStore` while `*store.SyncRepository` flows in both as the same instance. Confirm both fields are wired in `internal/cli/runtime.go` and `internal/desktop/app.go` from the same `store.NewSyncRepository(db)` result.
- `internal/desktop/app.go:178` — already constructs `store.NewSyncRepository(db)` once and slots it into `app.Dependencies.SyncRepo`. No code change if the field type narrowing is satisfied structurally (it is, because `*store.SyncRepository` satisfies `SyncSnapshotStore` directly).
- Test fakes — `rg -n 'app\.SyncRepository'`. Expected hits:
  - Any factory fake in `internal/app/*_test.go` (e.g. the fake passed through `newSyncRunner` in `pull_internal_test.go:53`).
  - `internal/app/app_integration_test.go` — likely a fake implementing the superset. Trim each fake to the narrower interface the wiring now expects; concrete code still calls the methods that remain.
- `internal/cli/runtime.go` and the desktop wiring — confirm `*store.SyncRepository` flows into both `SyncRepo` (now `SyncSnapshotStore`) and any `syncservice.SyncRepository`-typed factory argument.

## Implementation steps

1. Generate the new interface signature from the actual method declarations at `app/repositories.go:42-44`. Use `go doc` or read the file; do not trust this doc verbatim.
2. Replace the declaration in `repositories.go`; rename the type (suggest `SyncSnapshotStore` to disambiguate from the deleted `SyncRepository`). Update both `bootstrap.go` fields.
3. Update `newSyncRunner` per the chosen (a)/(b) decision above.
4. Grep every fake: `rg -n 'app\.SyncRepository\|SyncRepository\b' internal/app/` and convert any local fake type to either `syncservice.SyncRepository` or `SyncSnapshotStore` (or both, depending on what it's wired into).
5. Update `internal/cli/runtime.go` (post-step-1) — if `app.Dependencies.SyncRepo = store.NewSyncRepository(db)` slips into a type error, declare the `*store.SyncRepository` once and reuse it for `newSyncRunner` too.
6. `go build ./... && go vet ./...` clean.

## Verification

- `rg -n 'type SyncRepository' internal/app/` — **zero** hits; the duplication with `syncservice` is gone.
- `rg -n 'syncservice\.SyncRepository' internal/app/` — at most one hit, inside `pull.go` if you chose option (b) for `newSyncRunner`. Zero hits for option (a).
- `go test ./internal/app ./internal/service/sync ./internal/cli ./internal/desktop` — all pass (desktop via `frontend/dist/.gitkeep` placeholder for build-level verification only).
- `make lint` — no `interfacebloat` or `dupword` complaints; no new issues vs the easy-wins baseline.
- Smoke: `nanci pull` against a real data dir still loads the latest sync snapshot, persists progress, and finishes the run; `nanci status` still prints `Última sincronização` (which reads `LatestSyncSnapshot`).

## Do not

- Do not delete `syncservice.SyncRepository` — it remains the consumer interface for the orchestrator code.
- Do not rename `*store.SyncRepository` — the concrete name is fine; only the duplicate interface declarations go.
- Do not combine `SyncSnapshotStore` methods into an existing interface like `DocumentTracker`; SyncRepo concerns are a different cluster (survey Category 5 cluster: "Sync/Pull").

## Risk callouts

- **Type widening in tests:** any test that constructed a fake implementing only the 3 app-side methods will now fail to satisfy `syncservice.NewSyncService` (which expects 8). Inject the concrete `*store.SyncRepository` against a real in-memory SQLite for those tests instead of a hand-rolled fake; only the narrower methods truly matter for `app.App`-side tests.
- **API surface impact:** callers external to `internal/app` who imported `app.SyncRepository` for mocking lose it. This is `internal/` — no external consumers; safe.
- Reviewers may push back on the new interface name `SyncSnapshotStore` if a "Repository" suffix is the local idiom. Match the surrounding convention; if `*Repository` is the local name and there's no other interface-style seam in `internal/app/repositories.go`, the rename is fine — `repositories.go` itself uses interface conventions like `CompanyRepository`, `DocumentReader` so a behavior-name (`SyncSnapshotStore`) reads cleanly.