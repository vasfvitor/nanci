# Step 3 — Consolidate `XMLStore` to the Consumer Side

> Parent: `docs/plans/go-skill-audit-structural-impl.md` — Execution Order step 3.
> Implements part of structural plan change 3 (consolidate interface ownership).

## Goal

Delete `app.XMLStore` and use `files.XMLStore` directly. There is one duplication today: `files/blobstore.go:13-16` and `app/repositories.go:47-50` declare byte-identical interfaces. `files.XMLStore` wins because the consumer package owns interfaces (skill guidance).

## Files touched

- `internal/app/repositories.go:47-50` — delete the `XMLStore` declaration.
- `internal/app/bootstrap.go:43,57` — change field type `XMLStore` → `files.XMLStore`; add `import "github.com/vasfvitor/nanci/internal/files"`.
- `internal/app/repositories.go` — also import `files`.
- `.golangci.yml` — relax the `app-core` depguard rule that denies `internal/files` for `internal/app` production code.
- Test fakes: find any `internal/app/*_test.go` fake that implemented the deleted `app.XMLStore` and switch it to `files.XMLStore`. Per the survey, `company_test.go:478` returns `files.ErrBlobNotFound` from a fake — verify that fake implements `files.XMLStore` (it almost certainly already does, mechanically; just update the type annotation if present).

## Implementation steps

1. **Relax depguard first.** Edit `.golangci.yml` depguard `app-core` rule:
   ```yaml
       app-core:
         files: ["**/internal/app/**/*.go", "!**/*_test.go"]
         deny:
           - pkg: "github.com/vasfvitor/nanci/internal/store"
             desc: "Core business logic (app) cannot depend on the database layer."
           # Deleted: internal/files deny rule — the app-local XMLStore dup is gone;
           # app now consumes files.XMLStore directly. See docs/plans/go-skill-audit-structural-impl/step-3-consolidate-xmlstore.md.
           - pkg: "database/sql"
             desc: "Core business logic (app) cannot depend on SQL."
   ```
   Keeping the new-comment explains *why* the carve-out exists for future contributors.

2. **Add the import to `internal/app`.** Both `repositories.go` and `bootstrap.go` get `"github.com/vasfvitor/nanci/internal/files"`.

3. **Delete `app.XMLStore`.** Remove the 4-line declaration at `app/repositories.go:47-50`.

4. **Retype the two fields.** `internal/app/bootstrap.go`:
   - Line 43 in `type App struct`: `XMLStore           files.XMLStore`
   - Line 57 in `type Dependencies struct`: `XMLStore           files.XMLStore`
   Confirm exact field alignment with `gofmt`/`goimports`.

5. **Update test fakes.** `rg -n 'app\.XMLStore'` across the repo; expected hits are zero or one (the fake in `company_test.go`). If the fake declares `var _ app.XMLStore = fakeXML{}`, switch to `var _ files.XMLStore = fakeXML{}`.

6. **Run `go build ./internal/app/...`.** Confirm no compile breakage elsewhere.

## Verification

- `rg -n 'XMLStore interface'` returns **one** hit only: `internal/files/blobstore.go:13`.
- `rg -n 'app\.XMLStore'` returns **zero** hits.
- `make lint` — no depguard error about `app` importing `files`.
- `go test ./...` — all packages pass.
- Manual sanity: `nanci pull` against a real `NANCI_DATA_DIR` still writes blobs to `<dataDir>/blobs/`; read one back successfully.

## Do not

- Do not change anything about `files.BlobStore` — only the interface declaration moves.
- Do not change the desktop wiring (`desktop/app.go:179` constructs `files.NewBlobStore(dataDir)` and slots it into `app.Dependencies.XMLStore`; nothing to update because the field was interface-typed and still is).
- Do not touch `syncservice`'s `fileWriter files.XMLStore` field (`service/sync/sync.go:41`) — it already imports `files.XMLStore`. No duplication there per the survey.

## Risk callouts

- The depguard relaxation is the load-bearing change: with it documented in `.golangci.yml`, future PRs that *do* expand `internal/app`'s dependency on `internal/files` beyond `XMLStore` won't be flagged by the linter. Mitigate by pairing the rule change with a precise comment and a follow-up issue proposing to enforce the carve-out via a custom lint rule or a smaller internal package `files/xmlstore` exposing only the interface (deferred).
- If any other app package happens to reach for the deleted `app.XMLStore` name (unlikely, survey said no), the build will catch it during step 3's verification run.