# Exportar Documentos Selecionados Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Permitir a seleção e exportação de documentos específicos diretamente da tabela, ignorando o modal de Todos/Novos.

**Architecture:** Adicionaremos `ChavesAcesso []string` às structs do backend (Wails DTO e filtro do repositório). A query do SQLite aplicará `IN (...)` na chave de acesso. No frontend (Vue 3 + Quasar), a `<q-table>` armazenará os itens selecionados e os repassará via `desktopClient`.

**Tech Stack:** Go, Wails, Vue 3, Quasar, SQLite

## Global Constraints

- Go: use `make test`, `make lint`
- Frontend: `pnpm run test:unit`, `pnpm run lint:check`
- Não alterar/importar `wailsjs/go/models` no frontend, alterar `desktop.ts`.

---

### Task 1: DTOs e Modelos do Backend

**Files:**
- Modify: `internal/desktop/desktopapi/dto.go`
- Modify: `internal/nfse/repositories.go`
- Modify: `internal/app/export.go`

**Interfaces:**
- Produces: `ExportDocumentsInput` com `ChavesAcesso []string`, `DocumentFilter` com `ChavesAcesso []string`, `ExportInput` com `ChavesAcesso []string`

- [ ] **Step 1: Write the failing test**

*(Opcional aqui, mas vamos pular um teste para as structs já que são apenas definições de dados, vamos direto para a implementação)*

- [ ] **Step 2: Write minimal implementation**

No arquivo `internal/desktop/desktopapi/dto.go`, atualize a struct:
```go
type ExportDocumentsInput struct {
	CNPJ        string
	Competence  string
	Direction   string
	Format      string
	OutPath     string
	Incremental bool
	ChavesAcesso []string
}
```

No arquivo `internal/nfse/repositories.go`, atualize a struct:
```go
type DocumentFilter struct {
	Competence   string
	Direction    string
	Status       string
	FromNSU      *int64
	ToNSU        *int64
	Limit        *int
	OnlyUnread   bool
	IssueDateGTE *time.Time
	ChavesAcesso []string
}
```

No arquivo `internal/app/export.go`, atualize a struct:
```go
type ExportInput struct {
	CNPJ        string
	Competence  string
	Direction   string
	OutPath     string
	Incremental bool
	ChavesAcesso []string
}
```

E no mesmo arquivo `export.go`, na função `bulkExport`, atualize o trecho do `filter`:
```go
	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
		ChavesAcesso: input.ChavesAcesso,
	}
```

- [ ] **Step 3: Atualizar mapeamento no `app.go`**

Modifique `internal/desktop/app.go` nos métodos `ExportDANFSeZIP` e `ExportDocuments`:
```go
	exportInput := app.ExportInput{
		CNPJ:        input.CNPJ,
		Competence:  input.Competence,
		Direction:   input.Direction,
		OutPath:     input.OutPath,
		Incremental: input.Incremental,
		ChavesAcesso: input.ChavesAcesso,
	}
```

- [ ] **Step 4: Run test to verify it compiles**

Run: `go build ./...`
Expected: PASS sem erros de compilação.

- [ ] **Step 5: Commit**

```bash
git add internal/desktop/desktopapi/dto.go internal/nfse/repositories.go internal/app/export.go internal/desktop/app.go
git commit -m "feat(backend): add ChavesAcesso to export and filter structs"
```

---

### Task 2: Lógica de Banco de Dados (ListCompanyDocuments e ListPendingExportDocuments)

**Files:**
- Modify: `internal/store/documents.go`

**Interfaces:**
- Consumes: `DocumentFilter.ChavesAcesso`
- Produces: Queries do banco que aplicam `chave_acesso IN (...)`

- [ ] **Step 1: Write the failing test**
*(Testes de DB em `documents_test.go` se aplicável, mas podemos focar na lógica para garantir o filtro)*

- [ ] **Step 2: Write minimal implementation**

No `internal/store/documents.go`, em **ambas** as funções `ListCompanyDocuments` e `ListPendingExportDocuments`, logo após as checagens atuais de `filter`:

```go
	if len(filter.ChavesAcesso) > 0 {
		placeholders := make([]string, len(filter.ChavesAcesso))
		for i, chave := range filter.ChavesAcesso {
			placeholders[i] = "?"
			args = append(args, chave)
		}
		query += fmt.Sprintf(" AND d.chave_acesso IN (%s)", strings.Join(placeholders, ","))
	}
```

- [ ] **Step 3: Run test to verify it passes**

Run: `make test`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/store/documents.go
git commit -m "feat(store): filter documents by ChavesAcesso"
```

---

### Task 3: Atualizar Tipos e Wails Client no Frontend

**Files:**
- Modify: `internal/desktop/frontend/src/types/desktop.ts`
- Modify: `internal/desktop/frontend/src/composables/useDocuments.ts`
- Modify: `internal/desktop/frontend/src/platform/wails/client.ts`
- Modify: `internal/desktop/frontend/src/composables/useDocuments.test.ts`
- Modify: `internal/desktop/frontend/src/platform/wails/client.test.ts`

**Interfaces:**
- Consumes: A assinatura atualizada de `ExportDocumentsInput` do Wails.
- Produces: `exportDocuments` e `exportDANFSeZIP` no frontend suportando `ChavesAcesso?: string[]`.

- [ ] **Step 1: Write minimal implementation**

No `internal/desktop/frontend/src/types/desktop.ts`:
```typescript
export type ExportDocumentsInput = {
  CNPJ: string
  Competence: string
  Direction: string
  Format: ExportFormat
  OutPath: string
  Incremental: boolean
  ChavesAcesso?: string[]
}
```

No `internal/desktop/frontend/src/platform/wails/client.ts`:
(Perto de onde repassamos o input para Wails)
Para `exportDocuments`:
```typescript
    try {
      const res = await window.go.main.App.ExportDocuments({
        CNPJ: input.CNPJ,
        Competence: input.Competence,
        Direction: input.Direction,
        Format: input.Format,
        OutPath: finalPath,
        Incremental: input.Incremental,
        ChavesAcesso: input.ChavesAcesso || []
      })
      // ...
```
E repita para `exportDANFSeZIP`. (Lembre de passar `ChavesAcesso: input.ChavesAcesso || []`).

No `internal/desktop/frontend/src/composables/useDocuments.ts`:
```typescript
  async function exportDocuments(format: ExportFormat, incremental: boolean = false, outPath: string = '', chavesAcesso: string[] = []) {
    // ...
      return await desktopClient.exportDocuments({
        CNPJ: session.company.CNPJ,
        Competence: filters.competence,
        Direction: filters.direction,
        Format: format,
        OutPath: finalPath,
        Incremental: incremental,
        ChavesAcesso: chavesAcesso
      })
  }

  async function exportDANFSeZIP(incremental: boolean = false, outPath: string = '', chavesAcesso: string[] = []) {
    // ...
      return await desktopClient.exportDANFSeZIP({
        CNPJ: session.company.CNPJ,
        Competence: filters.competence,
        Direction: filters.direction,
        OutPath: finalPath,
        Incremental: incremental,
        ChavesAcesso: chavesAcesso
      })
  }
```

- [ ] **Step 2: Update Tests**

Atualize `useDocuments.test.ts` e `client.test.ts` para mockar corretamente as chamadas (ignorando erros de tipagem com o parâmetro extra onde apropriado, ou atualizando o mock).
No `client.test.ts` garanta que `ChavesAcesso: []` seja esperado.

- [ ] **Step 3: Run test to verify it passes**

Run: `cd internal/desktop/frontend && pnpm run test:unit`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/desktop/frontend/src/types/desktop.ts internal/desktop/frontend/src/composables/useDocuments.ts internal/desktop/frontend/src/platform/wails/client.ts internal/desktop/frontend/src/composables/useDocuments.test.ts internal/desktop/frontend/src/platform/wails/client.test.ts
git commit -m "feat(frontend): support ChavesAcesso in export client and composable"
```

---

### Task 4: Atualizar Interface em DocumentsPage.vue

**Files:**
- Modify: `internal/desktop/frontend/src/pages/DocumentsPage.vue`

**Interfaces:**
- Consumes: `<q-table>` com suporte a `selection="multiple"`, actions de exportação
- Produces: Selecionar linhas e disparar `exportDocuments` sem popup quando há seleção.

- [ ] **Step 1: Adicionar Seleção na Tabela**

No `DocumentsPage.vue`:
Adicionar na tag `<script setup>`:
```typescript
const selected = ref<DocumentRow[]>([])
```

Na tag `<q-table>`:
Adicionar as props:
```html
      selection="multiple"
      v-model:selected="selected"
```

- [ ] **Step 2: Atualizar Lógica de Exportação**

Alterar `exportData`:
```typescript
async function exportData(format: ExportFormat) {
  if (selected.value.length > 0) {
    const chaves = selected.value.map(d => d.ChaveAcesso).filter(Boolean) as string[]
    try {
      const result = await documentsApi.exportDocuments(format, false, '', chaves)
      notifyExportSuccess(`Arquivo ${format.toUpperCase()}`, result)
      selected.value = [] // clear selection
    } catch (error) {
      notifyError('Erro ao exportar', error)
    }
    return
  }

  // Lógica anterior de Dialog mantida... (Todos / Incremental)
  $q.dialog({
    title: 'Exportar Documentos',
    message: 'Todos os documentos ou apenas os que não foram exportados:',
  //...
      const result = await documentsApi.exportDocuments(format, incremental)
```

E alterar `exportDanfseZip` da mesma forma:
```typescript
async function exportDanfseZip() {
  if (selected.value.length > 0) {
    const chaves = selected.value.map(d => d.ChaveAcesso).filter(Boolean) as string[]
    try {
      const result = await documentsApi.exportDANFSeZIP(false, '', chaves)
      notifyExportSuccess('ZIP de DANFSes', result)
      selected.value = []
    } catch (error) {
      notifyError('Erro ao exportar ZIP de DANFSes', error)
    }
    return
  }
  
  // Dialog original mantido
  //...
```

- [ ] **Step 3: Run UI test/linter**

Run: `cd internal/desktop/frontend && pnpm run lint:check`
Expected: PASS sem erros. (Execute o `build` ou `dev` e verifique na UI manualmente)

- [ ] **Step 4: Commit**

```bash
git add internal/desktop/frontend/src/pages/DocumentsPage.vue
git commit -m "feat(frontend): allow exporting specific selected documents from table"
```
