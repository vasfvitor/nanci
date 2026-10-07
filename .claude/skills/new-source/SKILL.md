---
name: new-source
description: Use when adding a new document source to nanci (a fourth DF-e such as MDF-e), or when a feature spans all sources. Ordered checklist rebuilt from how NF-e and CT-e were added, with the file to copy at each step, every place that enumerates sources, the decisions not to reopen and the gotchas NF-e taught.
---

# New source: one DF-e, every layer

CT-e (PR #20, `6f4f1ad`) is the template: it reused the loop, the budget, `httpclient` and `CertificateLoader` unchanged and added only its own `Source`, tables and page (docs/ARCHITECTURE.md, "Origens de Sincronização"). Copy CT-e; copy NF-e only for what CT-e lacks (resumos, manifestação, signed events). Read docs/CTE_SEFAZ.md and docs/NFE_SEFAZ.md first. Below, `<x>` is the source key (`mdfe`), `<X>` its Go/TS prefix (`MDFe`).

## Places that enumerate sources

Run these before and after; every hit needs the new source.

- `rg "'nfse', 'nfe', 'cte'"`: the `source` CHECK of `sync_runs`, `sync_state`, `company_sync_sources`, `sync_requests` (`internal/store/schema.sql`; a CHECK change is a table rebuild, see the `migration` skill) and `documentTables` in `frontend/src/stores/preferences.ts`.
- `rg "case syncstate.SyncSource"`: `sourceLabel` (`internal/sync/errors.go`), `requestsPerHour` (`internal/sync/dist.go`), `checkSource` and `newSource` (`internal/sync/manager.go`).
- `rg "SyncSourceCTe"`: also `SyncSource.Valid()` (`internal/syncstate/sync.go`) and the `DistIdle` condition in `dist.go`.
- `rg -F "'nfse' | 'nfe' | 'cte'" internal/desktop/frontend/src`: `SyncSource` (`types/desktop.ts`), `DocumentSource` (`utils/documentSources.ts`, plus its `DOCUMENT_SOURCES` noun, gender and article), `DocumentTable` (`stores/preferences.ts`).
- Not caught by a grep: the `domain` depguard file list in `.golangci.yml`, `utils/stateLegends.ts` (one `<X>_LEGEND`), `router/index.ts`, the nav entry in `components/AppLeftDrawer.vue`, the key element list `identifierElement` in `internal/foundation/redact/redact.go` (`chCTe` today).

A feature that spans all sources: run the same greps, then change each per-source file (`nfe*.go`, `cte*.go`, `documents.go`) in place. There is no shared layer to put it in.

## Checklist, in order

Domain (`internal/<x>`, copy `internal/cte`; no `store`, `app` or `database/sql` imports):
1. `types.go`: `Situacao`, `CompanyRole`, `VisibilityReason`, `EventType` with `Parse*`/`Valid()` returning `dfe.ErrInvalidEnum`; Portuguese values (`autorizada`, `tomador`).
2. `schema.go`: `ClassifySchema` on the docZip schema name; unknown returns `SchemaUnknown`.
3. `parser_*.go`: stream with `foundation/xmlwalk` (`Walk`, `HasAnySuffix`, `AttrValue`) and match by path suffix so older layouts parse (`cte/parser_procte.go`). Money via `dfe.Money`, keys via `dfe.ParseAccessKey`. Unknown fields go to `ParseWarnings`, not errors. Fixtures in `testdata/`.
4. `participation.go`: `ClassifyParticipation(doc, companyCNPJ)`, roles in priority order, `same_root_only`/`unknown` fallbacks.
5. `merge.go`: `MergeDocument` and `SituacaoFromEvents`; situação only worsens (`moreSevere`).
6. `repositories.go`: `DocumentFilter`, `Counts`, `ResetCounts`, `ExportKindXML`, `ErrDocumentNotFound`.

Sync identity:
7. `syncstate.SyncSource<X>` and every hit of the greps above; a migration that rebuilds the four sync tables with the new CHECK value.

Store:
8. Migration `0NN_<x>.sql` (copy `015_cte.sql` and `017_cte_viewed_at.sql`): `<x>_documents` (one row per chave, `tp_amb`, `raw_hash`, `parse_warnings`), `company_<x>_documents` (role, visibility, `viewed_at`), `<x>_events`, `company_<x>_export_marks` with `exported_events` (as `018`). Mirror in `schema.sql`; test in `migrations_test.go`.
9. `internal/store/queries/<x>.sql` (copy `cte.sql`: `Upsert*`, `HasCompany<X>Document`, `Has<X>Event`, `Link<X>EventsToDocument`, `Mark<X>Exported`), regenerate `sqlgen`.
10. `internal/store/<x>.go` (copy `store/cte.go`): `ApplyDocumentTx`/`ApplyEventTx` on the loop's `*sql.Tx`, `CompanyDocumentExists`, hand-written `listCompanyDocuments` + `build<X>FilterSQL` (pending export = no mark, hash changed, or event count changed), `MarkViewed`, `MarkExported`, `CountSummary`, `ResetCompany`/`PreviewResetCompany` (calls `ResetSyncStateTx`). Tests in `store/<x>_test.go` with `storetest.OpenTestDB`.

Protocol and sync:
11. `internal/sefaz` (if SEFAZ SOAP): endpoint consts and an `Endpoints` field (`endpoints.go`), a `distService` var and `Dist<X>NSU` (`cte_distribuicao.go`), `CheckTLS<X>` (`tls_check.go`), `retdist-<x>-137/138/656.xml` fixtures.
12. `internal/sync/source_<x>.go` (copy `source_cte.go`): `Kind`, `Policy` (delay var plus `requestsPerHour`), `Fetch` through `distBatch`, `ProcessItem` returning `*ProcessingError` on decode/parse failures, `checkTpAmb` for every item, events of an unknown chave dropped via `commitSkip` unless the company authored them, `processUnsupported` for unknown schemas. Add `<X>Repo` to `sync.Manager`. Tests: `source_cte_test.go` cases.

App and entry points:
13. `internal/app/<x>.go`, `<x>_export.go`, `<x>_reset.go` (copy `cte*.go`): `<X>Service` with `Pull`, `Status` (`loadSefazSourceStatus`), `ListDocuments`, `MarkViewed`, `ListEvents`, `TestConnection` (`testSefazConnection`), `ExportXMLZip`/`ExportXML` (`writeViaTemp`, then `MarkExported`), `PreviewReset`/`Reset` under `SyncManager.ReserveSource`. ZIP layout in `internal/report/<x>_zip.go` (`eventZipEntry`).
14. Wiring: `<X>Repo` in `app.Dependencies`, the nil check and `New` in `internal/app/bootstrap.go`; `store.New<X>Repository(db)` in `internal/cli/runtime.go` and in `startup` of `internal/desktop/app.go`.
15. CLI `internal/cli/<x>.go` + `<x>_pull|status|list|export|reset|testar.go` (copy `cte*.go`), registered in `root_factory.go`; `--cnpj` persistent, `reset` simulates without `--confirmar`. Tests in `cli/<x>_test.go`.

Desktop (one method at a time with the `desktop-method` skill):
16. `desktopapi/dto.go`: inputs, `<X>Row`, `<X>Event`, status, pull and reset DTOs plus mappers (`CTeRows`, `CTeEvents`, `CTeResetResultFrom`) and tests in `dto_test.go`.
17. `services.go` interface + `servicesFrom`; `app.go` methods `Pull<X>`, `Status<X>`, `List<X>`, `Mark<X>Viewed`, `List<X>Events`, `Test<X>Connection`, `Export<X>XML`, `Export<X>ZIP`, `PreviewReset<X>`, `Reset<X>`; fake in `fakes_test.go`, tests in `app_*_test.go`.
18. `wails generate module`; `client.ts` (imports, enum lists, `map<X>*`, `desktopClient` methods), `types/desktop.ts`, the `vi.mock` list in `client.test.ts`, mocks in `scripts/generate-screenshots.ts`.

Frontend (copy the CT-e files):
19. `stores/<x>Documents.ts` on `documentListState`, filter + `listInput` computed; `composables/use<X>Loaders.ts` (`useDocumentLoaders`) and `use<X>Documents.ts` (`useMarkViewed`, `useExportGuard`, `useSefazBlock`, `useRowTextFilter`, `useTablePagination('<x>')`, `companySync.runSync`); `utils/<x>Display.ts`.
20. `pages/<X>Page.vue`: `DocumentPageHeader`, `DocumentFilterBar`, `DocumentTableTop`, `StateLegend` with `<X>_LEGEND`, a `q-table` with class `document-table`, shared cells (`ChaveCell`, `PartyCell`, `NumeroCell`, `StateBadges`, `RowActionsMenu source="<x>"`), `useDocumentListActions`, an events dialog on `EventsDialogFrame`. Tests next to each file; `pnpm run screenshots`.

Docs:
21. `docs/<X>_SEFAZ.md` (sections as CTE_SEFAZ.md), ARCHITECTURE.md (package list, "Origens"), AGENTS.md package list, README "O que faz" table, `website/content/docs/<x>.md` + `_index.md` table, `website/content/docs/cli.md` commands, hosts in `docs/CERTIFICATES.md` and `website/content/docs/privacidade.md`.

## Decisions already made, do not reopen

- Per-source everything: `NFeService`/`CTeService`, `nanci nfe`/`nanci cte` trees, `NFePage`/`CTePage`, one row type per source.
- No central export layer (docs/ROADMAP.md, Fase 2 item 1). Each service writes its own ZIP and marks.
- No shared `DocumentTable` component or store factory (docs/PENDENCIAS.md, 30/09/2026). The shared parts already exist: `useMarkViewed`, `useExportGuard`, `companySync.runSync`, `documentListState`. With a fourth source, evaluate before copying a fourth time: whether the four stores differ only in filter shape and `listInput` (then a factory pays), and whether the four tables can be built from shared cells without one slot per column (otherwise `DocumentTable` is just `q-table` again).
- Portuguese fiscal identifiers everywhere (AGENTS.md): `ChaveAcesso`, `Situacao`, `tomador`, `<x>_manifestacoes`.
- Wails methods return `desktopapi` DTOs, never domain structs.
- Export marks carry `exported_events`; incremental export re-sends a document when its hash or event count changes, always with all its events.
- 20 requests/hour per company, source and environment is an app choice for CT-e (no published limit; `requestsPerHour` comment). A source without a published limit takes 20 too.

## What NF-e taught

- tpAmb per document and event, decided only in `sync.checkTpAmb`: the XML wins with a warning, a resumo or invalid value takes the pull's. Lists, status and export filter by the company's current environment (`environmentTpAmb` in `app/nfe.go`).
- Resumo vs completa: `nfe.MergeDocument` never replaces a completa with a resumo; CT-e's masked `autXML` copy never replaces the full one. Decide the merge rule of the new source in its domain `merge.go`.
- Environment lock removed: sync state, block, initial sync and budget are keyed by environment (`019`) and every row has `tp_amb` (`016`). Do not add a lock back.
- 3 strikes: `maxItemAttempts = 3` (`sync/loop.go`, `sync_state.failed_nsu`) skips an item that fails to decode or parse three runs in a row, keeping its XML. Only a `*ProcessingError` counts.
- ultNSU: `distBatch` takes `max(cursor, ultNSU)`, so a 656 never moves the cursor back; distribution is never retried by the transport (every send counts against the budget).
- 60-day rule: `DistIdle` warns at `DistIdleWarningDays` (45) only for sources listed in its condition; the rule itself is known from third-party reports. Decide whether the new source joins it and say so in its doc.

## Verify

The blocks in the `lane` skill for Go root, Desktop Go, Bindings, Frontend and Store, all of them.
