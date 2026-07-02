# Step 6 — Rename `internal/service/sync` → `internal/syncrun`

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 6.
> Implements structural plan change 5.

## Goal

Drop the architecture-layer-named directory segment `internal/service/`. Move the package one level up: `internal/service/sync/` → `internal/syncrun/`. Package is renamed from `syncservice` to `syncrun` to match.

The `service/` directory layer name is a smell per the skill ("Packages do not import each other sideways; cycles are a sign of wrong boundaries, `main` is the wiring point"). The AGENTS.md line that acknowledges `internal/service/sync` gets updated in the same commit.

## Files touched

- `internal/service/sync/*.go` → `internal/syncrun/*.go` (move).
- `internal/service/sync/*_test.go` → `internal/syncrun/*_test.go` (move).
- Package line: `package syncservice` → `package syncrun` in every `.go` file under the moved directory (per survey, that's `sync.go`, `sync_test.go`, plus any others).
- `internal/app/pull.go:14` — import path `syncservice "github.com/vasfvitor/nanci/internal/service/sync"` → `"github.com/vasfvitor/nanci/internal/syncrun"`. Update the references at `pull.go:21` (`newSyncRunner = func(...) { return syncservice.NewSyncService(...) }`) to use `syncrun.NewSyncService`.
- Any test imports; `rg -n 'internal/service/sync'`.
- `AGENTS.md` — update the "Project Structure" line that names `internal/service/sync`.

## Implementation steps

1. **Verify consumers** before moving. `rg -n 'internal/service/sync\|syncservice' --glob '*.go'` and confirm only `internal/app/pull.go:14,24` plus internal self-references need updating. Per the survey, the desktop does not import `syncservice` directly.
2. **Move the directory** with `git mv internal/service internal_syncrun_tmp && git mv internal_syncrun_tmp/sync internal/syncrun && rmdir internal_syncrun_tmp` (or any idiomatic path that preserves history). Avoid plain `mv` — `git mv` keeps the rename tracked for `git log --follow`.
3. **Rename the package** in every `.go` file in `internal/syncrun/`: `package syncservice` → `package syncrun`. Use `replaceAll` since it's the package clause, not a stray reference.
4. **Update imports.** Edit `internal/app/pull.go`:
   ```go
   import (
       …
       "github.com/vasfvitor/nanci/internal/syncrun"
   )
   ```
   Update the `newSyncRunner` factory:
   ```go
   var newSyncRunner = func(repo SyncSnapshotStore, client *adn.Client, xmlStore files.XMLStore, log *slog.Logger) syncRunner {
       return syncrun.NewSyncService(/* … */)
   }
   ```
   (Adjust the `NewSyncService` arg types if step 4 changed `SyncRepository` to `SyncSnapshotStore`; the actual repository wire is the concrete passed in.)
5. **Hunt internal references.** Inside `internal/syncrun/sync.go` (the moved package), self-references like `syncservice.SyncRepository` (in the interface declaration at line 27) become `syncrun.SyncRepository`. Test files (`sync_test.go`) reference the same names — apply `replaceAll` per file.
6. **Update `AGENTS.md`.** The line referring to "sync orchestration in `internal/service/sync`" becomes "`internal/syncrun`".
7. **Run `go build ./...`**, fix any remaining imports, repeat until clean.

## Verification

- `rg -n 'internal/service/sync\|syncservice'` returns **zero** hits anywhere outside `third_party/` and `docs/`.
- `internal/service/` directory does not exist; `internal/syncrun/` does.
- `go build ./... && go test ./...` clean. In particular `go test ./internal/syncrun ./internal/app ./internal/cli` green.
- Smoke: `nanci pull` against a real data dir still invokes the orchestrator (validated by tests + a manual run).
- Edit `docs/plans/go-skill-audit-structural.md` if it still references `internal/service/sync` — keep alignment with this plan; the structural plan can stay as a narrative but point readers to `docs/plans/go-skill-audit-structural-impl/` for the actual rename.
- `make lint` clean (no depguard changes — depguard denies `internal/store` from `internal/app`, not anything about `service/`).

## Do not

- **Do not** leave the package named `syncservice` in `internal/syncrun/` — the window of inconsistency is the worst outcome; rename and import-rename in a single atomic commit.
- **Do not** rename `*store.SyncRepository` or the `SyncRepository`/`SyncSnapshotStore` interface types — only the package name (`syncservice` → `syncrun`) and directory change.
- **Do not** update `internal/desktop/...` — the desktop does not import the orchestrator directly; surveys confirm zero desktop imports of `syncservice`.
- **Do not** mark this as "trivial mechanical" — git subject lines should be `refactor: rename internal/service/sync to internal/syncrun` so future `git log` readers understand it's a directory rename, not a content change.

## Risk callouts

- Existing PRs/reviewers may cite AGENTS.md quoting `internal/service/sync`; the commit's AGENTS.md update is the load-bearing doc reference.
- Git history may be dead-ended for files that don't survive `git mv` cleanly when the import alias is touched. Aim for a single commit that contains: (a) directory move, (b) package rename, (c) import updates, (d) AGENTS.md update. Any split between these four pieces will make one of the intermediate states uncompilable.
- `go mod`'s module path is unchanged (`github.com/vasfvitor/nanci`); only directory paths under it move, so `go.sum` is unaffected.
- If the desktop Gu bound **does** turn out to reference `syncservice` (re-grep at execution time; the original survey said no), update it in the same commit — no desktop behavioral change is intended by a rename.