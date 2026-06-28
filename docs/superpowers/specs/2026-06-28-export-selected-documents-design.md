# Design: Exportar Documentos Selecionados

## Resumo
Permitir que o usuário utilize checkboxes na tabela de documentos (`<q-table>`) para selecionar quais NFS-es deseja exportar (CSV, XLSX, ZIP XML, ZIP DANFSe). Ao realizar a exportação com itens selecionados, o sistema ignora a pergunta "Todos / Incremental" e exporta diretamente a seleção. Caso nenhum item esteja selecionado, o comportamento padrão permanece o mesmo.

## Componentes Envolvidos

### 1. Frontend (`internal/desktop/frontend/src/`)
* **`pages/DocumentsPage.vue`**:
  * Adicionar `selection="multiple"` e `v-model:selected="selected"` (um `ref<DocumentRow[]>([])`) no componente `<q-table>`.
  * Nas funções de ação `exportData(format)` e `exportDanfseZip()`:
    * Checar se `selected.value.length > 0`.
    * Se for verdadeiro, extrair `ChaveAcesso` dos selecionados e pular o modal. Chamar as APIs (`documentsApi.exportDocuments` e `documentsApi.exportDANFSeZIP`) passando a lista de chaves.
    * Se for falso, manter a lógica do modal para `Todos`/`Incremental`.
    * Limpar a seleção (`selected.value = []`) após a exportação com sucesso.
* **`types/desktop.ts`**:
  * Adicionar a propriedade opcional `ChavesAcesso?: string[]` na tipagem `ExportDocumentsInput`.
* **`composables/useDocuments.ts`** & **`platform/wails/client.ts`**:
  * Ajustar as assinaturas das funções de exportação para aceitarem o array de `ChavesAcesso`.

### 2. Backend API e DTOs (`internal/desktop/`)
* **`desktopapi/dto.go`**:
  * Adicionar `ChavesAcesso []string` na struct `ExportDocumentsInput`.
* **`app.go`**:
  * Repassar o array de `ChavesAcesso` para o `app.ExportInput` que será entregue para os métodos de exportação do core (`a.core.ExportCSV`, etc).

### 3. Core e Banco de Dados (`internal/app/`, `internal/nfse/`, `internal/store/`)
* **`app/export.go`**:
  * A struct `ExportInput` passa a ter o array `ChavesAcesso []string`.
  * Na função `bulkExport`, repassar essa lista de chaves para o `nfse.DocumentFilter`.
* **`nfse/repositories.go`**:
  * Adicionar `ChavesAcesso []string` à struct `DocumentFilter`.
* **`store/documents.go`**:
  * Na query SQL de `ListCompanyDocuments` e `ListPendingExportDocuments`, se a lista `filter.ChavesAcesso` tiver itens (tamanho > 0), adicionar um termo `AND d.chave_acesso IN (...)`. Se essa lista estiver presente, a condição padrão (como incremental ou limites) pode ser mantida ou ignorada dependendo de como a query se comporta de forma mais segura, mas o principal é adicionar a filtragem pela lista de chaves.
  
## Fluxo de Dados e Tratamento de Erros
* O payload cruzando o Wails (Go <-> JS) não deve ser gigantesco. Na prática, a seleção será para algumas dezenas/centenas de notas, o que é plenamente suportado (o array de strings é leve).
* Ao exportar notas específicas, a geração dos arquivos (arquivos na pasta .tmp e montagem de ZIP/planilha) reutilizará as funções já existentes perfeitamente, já que elas recebem uma lista limpa do banco de dados.

## Testes Requeridos
* Testes no repositório de documentos (`documents_test.go` ou `sync_integration_test.go`) validando que o filtro `ChavesAcesso` só retorna exatamente os documentos pedidos.
* Testes unitários no frontend (`useDocuments.test.ts` e `client.test.ts`) garantindo que as chaves de acesso são enviadas corretamente no payload do Wails.
