# Design: Correção da Borda de NSU e Lógica de Avanço no Sync

## 1. Contexto e Problema
O sincronizador atual do Nanci falha ao registrar corretamente a borda de consulta da API do ADN:
- Ele avança o ponteiro (`LastCheckedNSU`) no banco de dados mesmo quando a requisição para um NSU retorna `NENHUM_DOCUMENTO_LOCALIZADO`.
- Isso causa um "salto" irrecuperável. Se um documento for disponibilizado depois pela infraestrutura nacional para aquele NSU, o Nanci já terá registrado que o consultou, ignorando-o em sincronizações futuras.
- O loop de sincronização atualmente realiza um avanço ingênuo do tipo `advanceNSU++` até estourar um limite de `emptyLimit` de requisições sequenciais vazias, em vez de aproveitar o retorno em lote da API e parar quando o fim real da fila é atestado.

## 2. Nomenclatura e Semântica
- O conceito de *último NSU checado* passará a ser **último NSU processado com sucesso** (`LastProcessedNSU`).
- O código será refatorado para utilizar a nomenclatura `LastProcessedNSU` ao invés de `LastCheckedNSU` (structs, variáveis, CLI e Desktop TypeScript definitions).
- No banco de dados, o campo `last_nsu` na tabela `companies` permanecerá com este nome (para evitar migrações de DB invasivas), mas sua semântica documentada será "último NSU cuja persistência e extração mínima foram concluídas com sucesso".

## 3. Lógica do Loop de Avanço (Batch Optimization)
- **Cálculo da próxima requisição**: A consulta sempre começa de `advanceNSU = LastProcessedNSU + 1`.
- **Tratamento da Resposta (Unitária ou Lote)**:
  - A resposta da camada ADN será normalizada internamente como uma lista de documentos.
  - Se a API retornar um único documento, a lista terá 1 item. Se retornar múltiplos, todos serão tratados como lote. Se não houver documentos, a lista estará vazia e/ou o status será `NENHUM_DOCUMENTO_LOCALIZADO`.
  - O Nanci deve ordenar explicitamente essa lista por ordem crescente de `NSU` antes do processamento.
  - O processamento percorrerá documento a documento.
- **Transação e Avanço**:
  - Cada documento será processado e persistido (com idempotência garantida na camada de banco usando `unique(company_id, nsu)` e/ou `unique(company_id, chave_acesso, tipo_documento)`).
  - O XML bruto e o avanço do ponteiro devem ser persistidos de forma atômica sempre que possível.
  - O `LastProcessedNSU` só avançará até o NSU do documento cuja persistência foi concluída com sucesso.
  - Assim que um documento falhar, o loop processual é quebrado imediatamente (retornando erro). O avanço será registrado apenas até o último documento salvo, garantindo uma retomada segura a partir desse ponto.

## 4. Fim da Sincronização e Lacunas
- **Retorno Vazio (`NENHUM_DOCUMENTO_LOCALIZADO`)**:
  - O sync será finalizado com sucesso.
  - O `LastProcessedNSU` não é atualizado.
  - Na próxima vez que o Nanci sincronizar, ele recomeçará do mesmo `LastProcessedNSU + 1`.
- **Remoção do `consecutiveEmptyLimit`**:
  - Como a regra oficial atesta o fim da fila (`NENHUM_DOCUMENTO_LOCALIZADO`), não precisamos mais ficar testando os NSUs de `1` a `100` "no escuro". O limite e o contador de requisições sequenciais vazias serão removidos.

## 5. Resumo do Fluxo de Decisão

```text
LastProcessedNSU = último NSU cujo documento/evento foi salvo e processado com sucesso.

Ao iniciar:
advanceNSU = LastProcessedNSU + 1

Ao receber lote com documentos:
1. Ordenar por NSU crescente.
2. Processar e salvar (a idempotência já está protegida pelo unique(company_id, nsu) / uuid).
3. Atualizar LastProcessedNSU a cada persistência bem-sucedida ou em um checkpoint agregado, garantindo que nunca ultrapasse uma falha.
4. Próxima consulta usa o LastProcessedNSU + 1 atualizado.

Ao receber NENHUM_DOCUMENTO_LOCALIZADO:
1. Não atualizar LastProcessedNSU.
2. Finalizar sync com sucesso.
3. Próxima execução tentará o mesmo advanceNSU novamente.

Ao falhar no meio do processamento de um lote:
1. Não avançar o ponteiro além do último NSU salvo com sucesso.
2. Retornar erro do Sync (irá parar a run).
3. A próxima execução reprocessará da borda segura preservada.
```
