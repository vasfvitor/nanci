---
name: desktop-method
description: Use when exposing a backend capability to the nanci desktop app end to end (app service, DTO, Wails method, bindings, client, composable, store, page), or changing an existing one. Walks every layer with the file to edit, the pattern to copy and the test that pins it, using ExportNFeZIP as the worked example.
---

# Desktop method: one capability, every layer

Read AGENTS.md "Desktop Frontend Architecture" and "Testing Guidelines" first. Work bottom-up and keep one commit per layer pair where it builds (`app`, `wails`, `frontend`). The worked example is the NF-e XML ZIP export; open each cited file before copying its pattern.

## 1. Core: `internal/app`

- The service method holds the behaviour: lookup, validation, file writing, marks. Example: `NFeService.ExportXMLZip` in `internal/app/nfe_export.go` with `NFeExportInput` and `NFeExportResult` (embeds `ExportResult` from `internal/app/export.go`).
- Input and result types live here and use domain words: `Competence`, `Role`, `ChavesAcesso`, `IncludeResumos`. Document each field that is not obvious (formats like `"YYYY-MM"`, allowed values).
- Exports write through `writeViaTemp`/`writeFileAtomic`, then mark exported. `ExportedCount` is the number of documents written; with nothing to export, write no file and return `ExportedCount: 0` with an empty `OutPath`.
- Tests next to the code (`internal/app/nfe_test.go`, `TestNFeExportXMLZipLayoutAndIncrementalMarks`): archive layout, incremental marks, resumos skipped.
- Nothing Wails-specific here: no DTOs, no dialogs, no default file names.

## 2. DTO: `internal/desktop/desktopapi/dto.go`

- Every Wails input and result is a DTO here (`ExportNFeZIPInput`, `NFeExportResult`), never an `internal/nfe`, `internal/nfse` or `internal/app` struct.
- Enums cross as `string` in Portuguese (`Situacao`, `Manifestacao`, `CompanyRole` are `string(document.X)`); money as `int64` cents (`TotalValue: document.TotalValue.Cents()`); dates as `time.Time`, optional ones as `*time.Time` (`optionalTime`); slices never nil, so the frontend gets `[]`.
- A result built from more than a couple of fields gets a mapper function in `dto.go` (`NFeRows`/`nfeRow`, `NFeEventResultFrom`, `CTeResetResultFrom`) with a test in `desktopapi/dto_test.go` (`TestNFeRows` checks every field, cents, nil dates, enum strings, empty slices). Small results are built inline in `app.go` (`NFeExportResult{ExportResult: desktopapi.ExportResult(res.ExportResult), SkippedResumos: ...}`).

## 3. Wails method: `internal/desktop`

- `services.go`: add the core method to the service interface (`nfeService.ExportXMLZip`). `servicesFrom` wires the real `*app.App`; tests swap fakes.
- `app.go`: one thin method per action. Validate what the core cannot see (`if input.OutPath == "" { return ..., fmt.Errorf("caminho de saída não especificado") }`, enum parsing such as `dfe.ParseEnvironment` in `AddCompany`), map DTO to app input field by field, call `a.svc.nfe.ExportXMLZip(a.ctx, ...)`, return the zero DTO with the error unchanged, map the result.
- No path construction in Go beyond what the core does; the user picks the path through `SelectSaveFile`.
- Errors reach the frontend through `formatError` in `errors.go` as `{Message, Code}`. Codes today: `canceled`, `sefaz_blocked`, `sync_running`. A new code needs `errors.go`, `errors_test.go`, and `WailsErrorCode`/`asErrorCode` in `client.ts`.
- Tests: add the method to the fake in `fakes_test.go` (`f.record("ExportXMLZip", in)`, return a result field). Then in `app_export_test.go` (or `app_documents_test.go`, `app_company_test.go`, `app_sync_test.go` by area):
  - a mapping test (`TestExportNFeZIP`: `assertCalls` sees every input field, result equals the expected DTO);
  - an entry in the table tests: `exportRuns` feeds `TestExportsRequireOutPath` (no core call, zero result) and `TestExportsPassCoreErrors` (`errors.Is(err, errCore)`); document methods go in `TestDocumentMethodsPassCoreErrors`.

## 4. Bindings: `internal/desktop/frontend/wailsjs`

- `cd internal/desktop && wails generate module` (copy `frontend/dist` in first, see the `lane` skill). Expect `wailsjs/go/main/App.js` (`window['go']['main']['App']['ExportNFeZIP'](arg1)`), `App.d.ts` (`ExportNFeZIP(arg1:desktopapi.ExportNFeZIPInput):Promise<desktopapi.NFeExportResult>`) and `wailsjs/go/models.ts` (one class per new DTO). Nothing else should change.
- Same commit: add the method to the `vi.mock('../../../wailsjs/go/main/App', ...)` list in `src/platform/wails/client.test.ts` and to the `window.go.main.App` mock in `scripts/generate-screenshots.ts` (`ExportNFeZIP: async () => ({ OutPath: ..., ExportedCount: 4, SkippedResumos: 3 })`). The build does not type-check that script.
- CI job `bindings` in `.github/workflows/ci.yml` regenerates and fails on any drift in `frontend/wailsjs`.

## 5. Client: `src/types/desktop.ts` and `src/platform/wails/client.ts`

- Types mirror the DTO with typed unions where the DTO has strings: `ExportNFeZIPInput.Role: NFeRole | ''`, `NFeExportResult = ExportResult & { SkippedResumos: number }`. Pages, stores and composables import these, never `wailsjs`.
- The method goes in `desktopClient` and wraps the call in `callWails` (error normalization). Exports take `Omit<ExportNFeZIPInput, 'OutPath'> & { BaseName?: string; OutPath?: string }`, build the default name (`nfe_${CNPJ}_${fileTimestamp()}.zip`), call `desktopClient.selectSaveFile(title, defaultName, '*.zip')`, return `null` when the dialog is cancelled, fill optional arrays (`ChavesAcesso || []`), and map the result.
- Result mappers read `unknown` through `asString`, `asNumber`, `asBoolean`, `asDate`, `asNullableNumber`, `asStringArray`, and `asEnum` (unknown values become `''`). Example: `mapExportResult`, `mapNFeExportResult`. Exported mappers are tested directly.
- `client.test.ts`: a success test (dialog title and default-name regex, exact Wails input, mapped result with defaults) and a cancel test (`SelectSaveFile` resolves `''`, result `null`, Wails method not called). See "exports NF-e XML and ZIP to the path chosen in the save dialog".

## 6. Composable and store: `src/composables`, `src/stores`

- Feature logic goes in the source's composable (`useNFeDocuments`, `useDocuments`, `useCTeDocuments`). Return functions and `storeToRefs` refs, nothing generated.
- State that outlives the route lives in the source store (`nfeDocuments`, `documents`, `cteDocuments`); the shared list fields (`rows`, `selected`, `exporting`, `markingViewed`, `resettingCNPJ`, `searchGate`) come from `documentListState()`. A new cross-route flag goes there or in a focused setup store.
- Mark in-flight state in the store before awaiting and clear it in `finally`. Exports use `useExportGuard(exporting, () => store.listInput.CNPJ)`: `runExport` returns `null` without a company or while another export runs. Syncs use `companySync.runSync`; resets set `resettingCNPJ`.
- Guards return `null` for "did not run" (`exportZIP([])`), so the page can tell it from a result.
- Backend events go through `onWailsEvent` from `src/platform/wails/events.ts`; keep the returned `Unsubscribe` and call it on unmount or stop (`layouts/MainLayout.vue`, `stores/console.ts`).
- Tests (`useNFeDocuments.test.ts`, `vi.mock('@/platform/wails/client')`): the request built from the store, the guards, failure clearing the flag, and the mandatory remount test: start the call on a pending promise (`deferred()` from `@/test/fixtures`), create a second instance, assert the flag is visible and a second call resolves `null` without a new client call, resolve, assert the flag clears ("keeps an export visible to a second instance while it is pending").

## 7. Page and components: `src/pages`, `src/components`, `src/utils`

- The page wires the composable, dialogs and notifications only. `NFePage.vue`: `useDocumentListActions({ source: 'nfe', ..., exportList: exportZIP })` opens `ExportDialog`; the page's `exportZIP` calls `nfe.exportZIP`, then `notifyExported(result, 'XML')`, and `notifyError('Erro ao exportar XMLs', error)` on failure. Extra outcomes get their own Portuguese notice (`SkippedResumos` via `notifyInfo`).
- `useNotify` wording: `notifyExported` says "N XMLs exportados para <caminho>." or "Nenhum documento para exportar."; errors read "Erro ao <ação>: <causa>".
- New dialogs take typed `defineProps`/`defineEmits` and return the choice (`ExportDialog` emits `ok: [choice: ExportChoice]`); the page or composable makes the backend call.
- Reuse the shared pieces before writing markup: `DocumentPageHeader`, `DocumentFilterBar`, `DocumentFilterSelect`, `DocumentTableTop`, `StateLegend`, `StateBadges`, `AbbrBadge`, `ChaveCell`, `NumeroCell`, `PartyCell`, `RowActionsMenu`/`RowMenuItem`, `DocumentDetailRow`, `DetailList`, `ExportDialog`, `EventsDialogFrame`, `ParseWarnings`; columns from `utils/documentColumns.ts`.
- Formatting in `utils/formatters.ts` (`formatCpfCnpj`, `formatCurrencyCents`, `formatDate`, `formatDateTime`, `formatNFeNumber`); labels in `utils/nfseDisplay.ts`, `nfeDisplay.ts`, `cteDisplay.ts`, `sefazDisplay.ts`; source nouns and gender agreement in `utils/documentSources.ts` (`DOCUMENT_SOURCES`, `agree`).
- Dark mode: `$q.dark.isActive` class bindings or Quasar CSS variables, no hex colors, no `dark:` selectors.
- Tests: the component's `*.test.ts` for props/emits; the page test (`NFePage.test.ts`, "exports the filtered rows as a ZIP with the choice of the dialog") for the dialog-to-client wiring and the notification text.
- Screenshots: `pnpm run screenshots nfe` (routes as arguments, leading slash optional). Specs with `fitsWidth: true` fail when `.document-table` scrolls sideways; at the 1280px window the table area is about 946px, so a new column must fit that budget.

## 8. CLI parity: `internal/cli`

A user-facing capability usually ships with its CLI command in the same change: NF-e export landed in desktop and in `internal/cli/nfe_export.go` (`nfe export zip`, flags `--competencia`, `--papel`, `--chave`, `--incluir-resumos`, `--incremental`, `--out`, output via `printExportResult`) in the same PR (#18, `25315c2`). CLI tests sit next to the command (`internal/cli/cte_test.go`, `TestCTeExportZip_Layout`).

## Decisions already made

- One composable, store and page per source (NFS-e `useDocuments`/`documents`/`DocumentsPage`, NF-e `useNFeDocuments`/`nfeDocuments`/`NFePage`, CT-e `useCTeDocuments`/`cteDocuments`/`CTePage`); shared behaviour goes in `documentListState`, `useExportGuard`, `useMarkViewed`, `useDocumentListActions` and the shared components above, not in a central export layer.
- Only `src/platform/wails/client.ts`, `events.ts` and `runtime.ts` import `wailsjs`. No `window.go`, no `@ts-expect-error` Wails calls, no raw `EventsOff`, no `v-html`.
- Wails methods return `desktopapi` DTOs; the default file name is suggested in `client.ts`, the path is chosen by the user, the file is written by the core.
- Portuguese fiscal vocabulary stays in every layer: `ChaveAcesso`, `ChavesAcesso`, `Competence`, `Manifestacao`, `tomada`/`prestada`, `destinatario`, `tomador`.
- Known debt, do not copy: single-document exports (`ExportXML`, `ExportDANFSe`, `ExportNFeXML`, `ExportCTeXML` in `app.go`) return `ExportedCount: 0`, so the page shows "Nenhum documento para exportar." after a successful export (ROADMAP item). New single exports return `ExportedCount: 1`.

## Checklist

- [ ] `internal/app` method, input/result types, tests
- [ ] `desktopapi` DTOs (+ mapper and `dto_test.go` when not trivial)
- [ ] `services.go` interface, `app.go` method, fake in `fakes_test.go`, mapping test, table entries (out path, core errors)
- [ ] `wails generate module`; diff shows only the new method and DTOs
- [ ] `client.test.ts` `vi.mock` list and `scripts/generate-screenshots.ts` mock updated in the same commit
- [ ] `types/desktop.ts` type, `desktopClient` method with mapper, success and cancel tests
- [ ] composable function with guard, store flag set before await and cleared in `finally`, second-instance test
- [ ] page wiring with `useNotify`, dialog typed props/emits, page test
- [ ] CLI command and test when user-facing
- [ ] Go: `go build ./... && go test ./...` at the root and in `internal/desktop`; `golangci-lint run ./...`
- [ ] Frontend: `pnpm run lint:check && pnpm run test:unit && pnpm run build`; `pnpm run screenshots <route>` when a page changed
- [ ] Manual smoke test in `wails dev` of the flow touched
