# Pendências

Itens anotados durante as entregas de NF-e (v0.4.0) e CT-e (v0.5.0), em 24/09/2026, ainda sem plano. Ordem aproximada de valor. Ao resolver um item, remova-o daqui e registre a decisão no documento da área.

## Funcionalidade

- **Prestação do serviço em desacordo (CT-e, evento 610110).** Só o tomador envia, em até 45 dias da autorização, com observação de 15 a 255 caracteres, à SEFAZ autorizadora do CT-e (não ao Ambiente Nacional). Precisa de: tabela UF → serviço de recepção de eventos por ambiente (SVRS para a maioria; SP, MT, MS, MG e PR com endpoint próprio), assinatura do `eventoCTe` reaproveitando `internal/sefaz/xmldsig.go` com o namespace do CT-e, tabela de auditoria dos envios (espelho de `nfe_manifestacoes`), confirmação explícita como na ciência da NF-e. Ver `docs/CTE_SEFAZ.md`, "Próximos passos".
- **Ligação CT-e ↔ NF-e nas telas.** As chaves de NF-e transportadas já ficam em `cte_documents.nfe_chaves` e servem de filtro (`cte list --nfe`, campo na página CT-e). Falta mostrar, na NF-e, o CT-e do frete, e, no CT-e, as notas que ele transporta, com navegação entre as duas páginas.
- **Importação de XML avulso** (NF-e e CT-e recebidos por e-mail ou fora da janela de 90 dias / 3 meses da distribuição).
- **Notas emitidas pela própria empresa.** A distribuição não devolve ao emitente os próprios documentos; seria preciso outra fonte.

## Comportamento conhecido

- **Ciência da Operação real ainda não enviada.** O fluxo de manifestação da NF-e foi testado só com o servidor de teste; um envio real exige um CNPJ que receba NF-e. A assinatura foi conferida byte a byte contra um evento real aceito pela SEFAZ.

## Código

- **Achados dos testes do módulo desktop (07/10/2026), sem correção ainda:** `UpdateCompany` com `SyncStartPolicy` vazio manda `from_now` sem data (o `AddCompany` manda com a data de hoje); latente porque o `EditCompanyDialog` sempre envia a política. `startup` que falha (env, logger, banco) deixa os serviços nulos e todo método ligado ao Wails entra em pânico. `formatExportError` (dica "XML não encontrado, resetar NSU") nunca é chamado fora de teste. `ListNFeInput` não tem `Limit` (o CT-e tem). `desktopapi.PullResult` não expõe `MaxNSU`, `CompletasSaved`, `ResumosSaved`, `NextAllowedAt`. Os diálogos (`SelectCertificate`, `SelectExportDirectory`, `SelectSaveFile`, `ExportLogs`) e `startup`/`openDir` seguem sem teste, chamando o runtime direto.
- **`depguard` não casa caminhos no Windows.** A regra `domain` roda, mas o glob `**/internal/dfe/*.go` não casa com caminhos de barra invertida (`files: ["$all"]` casa). Conferir em WSL ou trocar os globs; até lá a regra só vale fora do Windows.
- **Gancho de retry por requisição em `httpclient`.** Hoje o transporte só tem `MaxRetries` global; a distribuição da SEFAZ nunca reenvia e o evento reenvia uma vez, resolvido no chamador. Só vale se aparecer um terceiro caso.
- **Não fazer (decidido no padrão de UI de 30/09/2026):** um `DocumentTable` único para as três páginas e uma fábrica de store comum às três fontes. Os corpos das tabelas são diferentes demais (spinner e chip de prazo na NF-e, papéis secundários no CT-e, descrição do serviço na NFS-e); o componente viraria a `q-table` com um slot por coluna. O ganho do store comum já veio com `useMarkViewed`, a guarda de exportação, `runSync` e `documentListState`, que só junta o estado de lista que os três stores repetiam. Reconsiderar só se surgir uma quarta fonte, como MDF-e.
