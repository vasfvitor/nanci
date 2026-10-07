---
name: migration
description: Use when changing the SQLite schema of nanci (new table, column, index, CHECK value, export kind, sync source, or a data fix). Covers the goose migration, schema.sql, sqlc regeneration, the hand-written SQL that sqlc does not see, the migration test pattern and the docs list.
---

# Migration: change the SQLite schema

## How the schema is applied

- Migrations are goose SQL files in `internal/store/migrations_v2/`, embedded by `//go:embed migrations_v2/*.sql` in `internal/store/db.go`. `store.OpenDB(ctx, path, true)` runs `provider.Up` on every CLI run (`internal/cli/runtime.go`) and desktop startup (`internal/desktop/app.go`), so a release migrates the user's database on first launch.
- The connection has `_pragma=foreign_keys(1)` and WAL (`db.go`). goose runs each file in a transaction.
- The latest is `020_nfse_chave_without_prefix.sql`; the next is `021`. Check `ls internal/store/migrations_v2` and open branches first: docs/ROADMAP.md (Fase 2, item 1) already reserves 021 for the new export kinds.

## Rules

1. Never edit a migration that has been applied anywhere, even only on your dev database. goose records the version and never reruns it. Editing `015_cte.sql` after it ran left a dev database without `cte_documents.masked_keys`, and the first real CT-e failed until it was fixed by hand. A new number, always.
2. One file per change, `NNN_snake_case.sql`, with `-- +goose Up` and `-- +goose Down`. Comments in English say why (see 013, 018, 019, 020).
3. Down undoes Up. Every schema change since 006 has a real Down (001 and 003 have no Down section, 002 and 005 a no-op one; do not copy them). A data-only migration whose input cannot be rebuilt says so in a comment instead of pretending (`020`: "Nothing to undo: ... which rows had the prefix is not kept").
4. SQLite cannot add, drop or change a CHECK or a primary key on an existing table: rebuild it. Adding a new column that carries a CHECK works with `ALTER TABLE ... ADD COLUMN` (`016`).
5. Keep `internal/store/schema.sql` in sync in the same commit (below). Regenerate `sqlgen` in the same commit.
6. Before running a new migration against a database you care about: close the app and copy `%LOCALAPPDATA%\nanci\nanci-v1.db` together with its `-wal`/`-shm` files (path from `paths.DataDir` + `app.RuntimeDBPath`; website/content/docs/privacidade.md says to copy the whole folder). Better: `make seeddev`, then point the CLI at it with `NANCI_DATA_DIR=devdata`.

## Patterns to copy

- Add a column + backfill: `018_export_marks_event_count.sql`. `ADD COLUMN ... NOT NULL DEFAULT 0`, then one `UPDATE ... SET col = (SELECT ...)` per table; Down drops the columns in reverse order.
- Rebuild for a CHECK or key change: `013_manifestacoes_and_sync_requests_check.sql` and `019_sync_sources_by_environment.sql`. `CREATE TABLE x_new` with the full new definition, `INSERT INTO x_new SELECT ...` (019 joins `companies` to fill the new `environment`), `DROP INDEX`, `DROP TABLE x`, `ALTER TABLE x_new RENAME TO x`, recreate the indexes. Down rebuilds the old shape the same way and says what it loses.
- Data migration: `020_nfse_chave_without_prefix.sql`. Stage the affected ids in a scratch table (`nfse_prefixed_020`), update from it, relink dependent rows, recompute derived columns with the same rule as the Go code (`recomputeDocumentStatus` in `internal/sync/store.go`), drop the scratch table.
- Rename: `013` (`ALTER TABLE ... RENAME TO`, drop and recreate the index under the new name).
- Every rebuild so far (006, 007, 013, 019) was of a table that no other table `REFERENCES`. Rebuilding a parent (`companies`, `documents`, `nfe_documents`, `cte_documents`) with foreign keys on is untried here; read SQLite's "ALTER TABLE" 12-step procedure and test it with child rows seeded.

## CHECK constraints that enumerate values

- Sync source `'nfse', 'nfe', 'cte'` (`rg "'nfse', 'nfe', 'cte'" internal/store`): `sync_runs`, `sync_state`, `company_sync_sources`, `sync_requests`. A new source rebuilds all four; see the `new-source` skill.
- Export kind: `company_document_export_marks` (NFS-e: `'xml', 'csv', 'xlsx', 'danfse'`), `company_nfe_export_marks` and `company_cte_export_marks` (`'xml'`). The Go side is `nfe.ExportKindXML`, `cte.ExportKindXML` and string literals in `internal/app/export.go`. A new kind is a rebuild of that marks table.
- Enums of the domain packages (`situacao`, `company_role`, `visibility_reason`, event `type`, `tp_amb`) mirror `Valid()` in `internal/nfse`, `internal/nfe`, `internal/cte`. Change both together. `cte_documents.tp_cte`, `tp_serv` and `modal` have no CHECK on purpose (layouts disagree; docs/CTE_SEFAZ.md).

## schema.sql and sqlc

- `internal/store/schema.sql` is the current schema written as plain `CREATE TABLE`/`CREATE INDEX`, edited in place, never appended as ALTERs. It is the `schema:` input of `sqlc.yaml`. `TestSchemaSQLMatchesMigrations` (`internal/store/schema_drift_test.go`) compares it against a migrated database, column order included: an added column goes last, where `ADD COLUMN` puts it.
- Convention (commits `119dd2d` for 018, `641ce45` for 019): an added column goes last in its `CREATE TABLE`, where `ADD COLUMN` puts it; a rebuilt table copies the migration's new `CREATE TABLE`; indexes follow their table.
- Regenerate from the repo root: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate` (on Windows with TDM-GCC prefix `CGO_ENABLED=0`, or the link fails). Commit `internal/store/sqlgen/` with it; never edit it by hand. `rename` in `sqlc.yaml` fixes inflections (`nfe_manifestaco` -> `NfeManifestacao`).
- Queries live in `internal/store/queries/*.sql` (`companies`, `credentials`, `nfe`, `cte`, `sync`).

## SQL that sqlc does not check

A renamed or new column must also be fixed in hand-written SQL; only tests catch it:

- NFS-e: reads, lists, viewed and export marks in `internal/store/documents.go` (`DocumentRepository`). Its writes are sqlc (`queries/sync.sql`, called from `internal/sync/store.go`).
- NF-e and CT-e: list, filter and pending-export queries in `internal/store/nfe.go` and `cte.go` (`listCompanyDocuments`, `buildNFeFilterSQL`/`buildCTeFilterSQL`, the `*CompanyDocumentColumns` lists and `scan*` functions) and the reset functions.
- Sync: raw queries in `internal/sync/store.go` (`SourceState`, `SetBlockedUntil`, `RecordRequest`).

`rg -n "<column>" internal --type go` after the change.

## Test

Add `TestMigrationNNN<What>` to `internal/store/migrations_test.go` (copy `TestMigration018CountsExportedEvents` or `TestMigration020StripsTheNFSPrefixOfNFSeChaves`):

1. `db, provider := migrateTo(t, NNN-1)` (helper at the bottom of the file).
2. Seed with `mustExec` and raw `INSERT`s written for the N-1 schema; repositories expect the latest schema, so do not use them here. Seed the edge cases the Up must handle (020: collision, invalid id, unlinked event).
3. `provider.UpTo(ctx, NNN)`, assert rows and columns (`pragma_table_info`, `sqlite_master`).
4. `provider.DownTo(ctx, NNN-1)`, assert the old shape is back, or that kept data survived.

Repository tests use `storetest.OpenTestDB(t)` (`internal/store/storetest`), which opens a temp database with every migration applied.

Run: `go test ./internal/store -run Migration -v`, then `go test ./...` (`sync`, `app`, `cli`, `company` and `credential` tests open migrated databases too).

## Docs

- docs/NFE_SEFAZ.md, "Modelo de dados": the list "Migrações `007` a `0NN`" gets the new number in its heading and one bullet `` `0NN`: <what changes and what happens to existing rows> `` in Portuguese, linking to the area doc when the migration belongs to another source (`015` -> CTE_SEFAZ.md, `020` -> NFSE_ADN.md).
- The area doc of the table describes the column (docs/CTE_SEFAZ.md "Modelo de dados", docs/NFSE_ADN.md).
- The PR or report names the schema impact explicitly (AGENTS.md, "Commit & Pull Request Guidelines"). Commit scope `store`.
