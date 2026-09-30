# Pendências

Itens anotados durante as entregas de NF-e (v0.4.0) e CT-e (v0.5.0), em 24/09/2026, ainda sem plano. Ordem aproximada de valor. Ao resolver um item, remova-o daqui e registre a decisão no documento da área.

## Funcionalidade

- **Prestação do serviço em desacordo (CT-e, evento 610110).** Só o tomador envia, em até 45 dias da autorização, com observação de 15 a 255 caracteres, à SEFAZ autorizadora do CT-e (não ao Ambiente Nacional). Precisa de: tabela UF → serviço de recepção de eventos por ambiente (SVRS para a maioria; SP, MT, MS, MG e PR com endpoint próprio), assinatura do `eventoCTe` reaproveitando `internal/sefaz/xmldsig.go` com o namespace do CT-e, tabela de auditoria dos envios (espelho de `nfe_manifestacoes`), confirmação explícita como na ciência da NF-e. Ver `docs/CTE_SEFAZ.md`, "Próximos passos".
- **Ligação CT-e ↔ NF-e nas telas.** As chaves de NF-e transportadas já ficam em `cte_documents.nfe_chaves` e servem de filtro (`cte list --nfe`, campo na página CT-e). Falta mostrar, na NF-e, o CT-e do frete, e, no CT-e, as notas que ele transporta, com navegação entre as duas páginas.
- **Importação de XML avulso** (NF-e e CT-e recebidos por e-mail ou fora da janela de 90 dias / 3 meses da distribuição).
- **Notas emitidas pela própria empresa.** A distribuição não devolve ao emitente os próprios documentos; seria preciso outra fonte.

## Comportamento conhecido

- **Exportação incremental e eventos no mesmo segundo.** A marca de exportação guarda só o hash do documento e a data; um evento gravado no mesmo segundo da marca (ou durante a exportação) fica de fora do próximo `--incremental`. Solução: guardar a contagem de eventos (ou o hash do último) na marca, com migração para `company_nfe_export_marks` e `company_cte_export_marks`.
- **Bloqueio da SEFAZ compartilhado entre ambientes.** `company_sync_sources.blocked_until` e o orçamento por hora valem por empresa e origem, em qualquer ambiente. Como o ambiente agora pode ser trocado a qualquer momento, um bloqueio recebido em produção adia o próximo pull em homologação. Resolver chaveando `company_sync_sources` e `sync_requests` também por ambiente.
- **Parse de CT-e real ainda não exercitado.** A empresa usada nos testes não tinha CT-e na fila; os parsers foram validados com fixtures fictícias e o XSD 4.00. Ao aparecer o primeiro CT-e real, conferir a linha na tela e os avisos de leitura (`parse_warnings`). Os códigos de denegação (110, 205, 301, 302, 303) foram assumidos iguais aos da NF-e; conferir no MOC CT-e 4.00.
- **Ciência da Operação real ainda não enviada.** O fluxo de manifestação da NF-e foi testado só com o servidor de teste; um envio real exige um CNPJ que receba NF-e. A assinatura foi conferida byte a byte contra um evento real aceito pela SEFAZ.

## Interface

- **Captura de tela de Configurações** mostra um erro porque o mock do script de screenshots não implementa `GetBuildInfo`.

## Código

- **Vocabulário compartilhado ainda em `internal/nfse`.** `Company`, `Credential`, `Environment`, `SyncSource`, `SyncStatus`, `SyncStopReason`, `SyncMode`, `SyncStartPolicy` e os parâmetros de estado de sync (cerca de 790 referências em 73 arquivos) são usados por `sync`, `store`, `app` e `sefaz`, que não são específicos de NFS-e. A parte mínima (`Money`, `CompanyID`, `ErrInvalidEnum`, chave de acesso, helpers de parse) já foi para `internal/dfe`. O resto pede um PR próprio, só de movimento, com um nome de pacote decidido antes.
- **`depguard` "app não importa store" não dispara no Windows e `main` já viola a regra** (`internal/app/{bootstrap,export,list,nfe,cte}.go` importam `internal/store`). Decidir: ou a camada `app` passa a depender de interfaces e o `store` é injetado, ou a regra sai de `.golangci.yml` e de `docs/ARCHITECTURE.md`.
- **Serviços de NF-e e CT-e com corpos quase iguais** em `Status` e `TestConnection` (`internal/app`). Um helper compartilhado evitaria a cópia, ao custo de mexer nos DTOs e nos testes.
- **Gancho de retry por requisição em `httpclient`.** Hoje o transporte só tem `MaxRetries` global; a distribuição da SEFAZ nunca reenvia e o evento reenvia uma vez, resolvido no chamador. Só vale se aparecer um terceiro caso.
- **Teste de `uf.ts`** no frontend (baixa prioridade).
