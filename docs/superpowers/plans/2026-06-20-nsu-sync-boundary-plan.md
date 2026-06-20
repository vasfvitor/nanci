# NSU Sync Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the NSU sync boundary logic to process documents in batch, stop correctly on NENHUM_DOCUMENTO_LOCALIZADO, and rename LastCheckedNSU to LastProcessedNSU.

**Architecture:** We will rename the tracking variable across the Go backend and Vue frontend to LastProcessedNSU for semantic clarity. Then we will rewrite the `Sync` method in `sync.go` to loop over batches, advance `advanceNSU` based on successfully saved documents instead of a sequential counter, and break out of the loop cleanly when the API returns no documents or an error occurs.

**Tech Stack:** Go, SQLite, Vue 3, TypeScript.

## Global Constraints
- Process the batch in ascending NSU order.
- `unique(company_id, nsu)` guarantees idempotency in the DB layer. If a document already exists, treat it as a success and advance LastProcessedNSU.
- Document saving and pointer advancing (`LastProcessedNSU`) must happen inside the same transaction or equivalent atomic operation.
- Ensure all tests pass (`make test`).
- Keep Vue Composition API patterns.

---

### Task 1: Rename LastCheckedNSU to LastProcessedNSU in Domain and Store

**Files:**
- Modify: `internal/nfse/model.go`
- Modify: `internal/nfse/repositories.go`
- Modify: `internal/store/sync.go`

**Interfaces:**
- Produces: Updated struct definitions with `LastProcessedNSU` instead of `LastCheckedNSU`.

- [ ] **Step 1: Update Domain Models**
Modify `internal/nfse/model.go` to change `LastCheckedNSU` to `LastProcessedNSU` in `SyncState` and `SyncRun`.

- [ ] **Step 2: Update Repository Interfaces**
Modify `internal/nfse/repositories.go` to change `LastCheckedNSU` to `LastProcessedNSU` in `GetOrCreateSyncStateParams` and `PersistSyncProgressParams`.

- [ ] **Step 3: Update SQLite Store**
Modify `internal/store/sync.go` to map the renamed fields when inserting/updating sync runs and state. The DB column `last_nsu` remains unchanged. Update the queries and struct mappings.

- [ ] **Step 4: Run Go tests**
Run: `make test`
Expected: Failures in app, service, and cli packages due to the name change. We will fix these in the next tasks.

---

### Task 2: Fix App and CLI Packages

**Files:**
- Modify: `internal/app/company.go`
- Modify: `internal/app/company_test.go`
- Modify: `internal/app/pull.go`
- Modify: `internal/app/status.go`
- Modify: `internal/cli/pull.go`
- Modify: `internal/cli/status.go`

**Interfaces:**
- Consumes: Updated structs from Task 1.

- [ ] **Step 1: Fix App layer**
Update `LastCheckedNSU` to `LastProcessedNSU` in `internal/app/company.go`, `internal/app/pull.go`, `internal/app/status.go`, and `internal/app/company_test.go`.

- [ ] **Step 2: Fix CLI layer**
Update `LastCheckedNSU` to `LastProcessedNSU` in `internal/cli/pull.go` and `internal/cli/status.go`.

- [ ] **Step 3: Run Go tests**
Run: `make test`
Expected: Failures remaining only in `internal/service/sync`.

---

### Task 3: Rewrite Sync Loop (Batch Optimization)

**Files:**
- Modify: `internal/service/sync/sync.go`
- Modify: `internal/store/sync.go` (if atomic transaction method needs to be added)

**Interfaces:**
- Consumes: Updated domain models.

- [ ] **Step 1: Update State Tracking**
In `internal/service/sync/sync.go`, rename `lastCheckedNSU` to `lastProcessedNSU` inside `syncRuntimeState`.
Update lines loading from and saving to DB to use `LastProcessedNSU`. Remove empty count properties.

- [ ] **Step 2: Refactor `processNSU` signature**
Make `processNSU` return `(processedCount int, err error)`.
Inside `processNSU`, sort `resp.Docs` by `NSU` ascending:
```go
sort.Slice(resp.Docs, func(i, j int) bool {
    return resp.Docs[i].NSU < resp.Docs[j].NSU
})
```
Loop through `resp.Docs`. For each document:
- If `NSU < advanceNSU`, ignore/log it as stale or duplicate to avoid regressions.
- If a conflict occurs for `unique(company_id, nsu)`, treat it as a success.
- Ensure the persistence of the document and updating `runState.lastProcessedNSU = env.NSU` happens atomically (e.g., in a single transaction).
If any error occurs during processing, return immediately. Return the number of documents successfully handled, including idempotent successes, excluding stale ignored documents.

- [ ] **Step 3: Refactor `Sync` Loop**
Remove `consecutiveEmpty` limit logic and the sequential `advanceNSU++` loop.
Replace with a cleanly defined `for` loop:
```go
advanceNSU := state.LastProcessedNSU + 1
for {
	processedCount, err := s.processNSU(ctx, company, syncRun.ID, advanceNSU, &runState, progress)
	if err != nil {
		finalStatus, stopReason, errorCode, errorMsg = classifySyncError(err)
		// Ensure finishRun is registered as a defer closure that reads finalStatus at execution time,
		// e.g. defer func() { _ = s.finishRun(...) }() rather than capturing eager arguments.
		return err
	}

	if processedCount == 0 {
		break
	}

	advanceNSU = runState.lastProcessedNSU + 1
}
```

- [ ] **Step 4: Run Go tests**
Run: `make test`
Expected: Failures in `sync_test.go` due to behavior changes.

---

### Task 4: Update Sync Tests

**Files:**
- Modify: `internal/service/sync/sync_test.go`

**Interfaces:**
- Consumes: New sync loop behavior.

- [ ] **Step 1: Update Test Mocks and Assertions**
Rename `LastCheckedNSU` to `LastProcessedNSU` in all test cases.
Adjust mock assertions and add specific test cases for:
1. Retorno vazio não atualiza `LastProcessedNSU`.
2. Retorno vazio encerra após uma chamada, sem `emptyLimit`.
3. Lote fora de ordem é ordenado antes do processamento.
4. Lote com múltiplos documentos avança até o último salvo.
5. Falha no meio do lote não avança além do último salvo.
6. Documento já existente é tratado como sucesso idempotente.
7. Próxima execução começa em `LastProcessedNSU + 1`.

- [ ] **Step 2: Run Go tests**
Run: `make test`
Expected: PASS

---

### Task 5: Update Frontend Types and Final Checks

**Files:**
- Modify: `internal/desktop/frontend/src/types/desktop.ts`
- Modify: `internal/desktop/frontend/src/pages/CompaniesPage.vue`
- Modify: `internal/desktop/frontend/src/composables/useCompanies.test.ts`
- Modify: `internal/desktop/frontend/scripts/generate-screenshots.ts`

**Interfaces:**
- Consumes: Wails generated models matching the new Go struct names.

- [ ] **Step 1: Regenerate Wails Models**
Run: `wails generate module` from `internal/desktop`.

- [ ] **Step 2: Update TypeScript definitions**
In `src/types/desktop.ts`, change `LastCheckedNSU` to `LastProcessedNSU`.
Update `CompaniesPage.vue` string template referencing `result.LastCheckedNSU`.
Update test payloads in `useCompanies.test.ts`.
Update mocked data in `generate-screenshots.ts`.

- [ ] **Step 3: Run grep search for leftovers**
Run the following searches from the project root:
```bash
rg "LastCheckedNSU|lastCheckedNSU|checked NSU|checked_nsu|Last Checked"
rg "LastProcessedNSU|lastProcessedNSU"
```
Fix any remaining occurrences of the old name so the output is clean.

- [ ] **Step 4: Run Frontend Lint and Tests**
Run: `cd internal/desktop/frontend && pnpm run lint:check && pnpm run test:unit`
Expected: PASS

- [ ] **Step 5: Prepare Commit Candidate**
Run:
```bash
git status
git diff --stat
```
Verify the changes are correct before proceeding with execution or opening a PR.
