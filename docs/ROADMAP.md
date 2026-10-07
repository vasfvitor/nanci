# Roteiro de longo prazo

Levantado em 07/10/2026 a partir do código, das issues abertas, do `docs/PENDENCIAS.md` e de uma pesquisa sobre quais dados fiscais uma empresa consegue puxar por API hoje. O material bruto, com fontes e datas, está em `docs/pesquisa/` (inventário de manutenção, SEFAZ DFe, NFS-e e municípios, Serpro e federal, eSocial e reforma); confira a data antes de confiar num fato de lá. O `PENDENCIAS.md` continua sendo a lista curta de itens sem plano; este arquivo ordena o trabalho em fases e registra o que foi descartado e por quê. Ao concluir uma fase, apague-a daqui e registre as decisões no documento da área.

Esforço: **[P]** menos de uma hora, **[M]** alguns dias, **[G]** semanas.

## Fase 0: fundação (este mês)

Tudo aqui é barato e reduz o custo de todas as fases seguintes.

**Estado em 07/10/2026 (branch `chore/fase-0`, 21 commits):** feito o portão de CI (`ci.yml`, `release.yml` depende dele; primeira execução real só no PR), a faxina (branches locais, sobras do spike v3, `.gitattributes` com renormalização, `.gitignore`, docs desatualizadas, comentários nos métodos `sefaz` sem chamador, `Makefile`), as dependências Go (x/crypto 0.57, sqlite 1.60, cobra, goose, go-retry) e do frontend (menores + vitest 5, pinia 4, vue-router 5, @quasar/vite-plugin 2, @eslint/css 2), o aviso de 45 dias de NSU, o teste de reuso do `ultNSU`, a auditoria do CNPJ alfanumérico (corrigiu `nfse.ParseAccessKey`, que rejeitava chaves com letras), o backport de segurança do fork do go-pkcs12 (GO-2026-5052 e IV do PBES2, issue #4) e os links relativos das cópias Markdown do site (#14). Também caiu um bug antigo do `make seeddev`.

Ficou para o dono: publicar o branch e abrir o PR; colar os comentários prontos nas issues #4, #10, #12, #14 e #15 e apagar os cinco branches remotos (texto no scratchpad da sessão de 07/10); decidir o `histoire` (ninguém roda; `story:build` quebrou com o Quasar 2.34, stub de `screen.orientation` resolve, ou remover os 5 `*.story.vue`, config e scripts); TypeScript 7 (vue-tsc e typescript-eslint ainda não suportam); `@vue/test-utils` 2.5 (exige Node 24.15); o `prettier --check` já acusava 67 arquivos antes, `lint:check` não roda prettier; os pontos em aberto do CNPJ alfanumérico listados em `docs/NFE_SEFAZ.md`.

### Portão de CI [M]

Nenhum workflow roda `go test`, `golangci-lint`, `govulncheck` nem `pnpm lint:check`/`test:unit`/`build` em PR. O `release.yml` compila sem testar. Criar `ci.yml` em push/PR para `main` com dois jobs: Go (Ubuntu, módulo raiz e `internal/desktop` com `dist/` de mentira como o `codeql.yml` já faz) e frontend. Rodar o lint no Linux também faz a regra `depguard` valer, já que os globs `**/internal/{nfse,nfe,dfe,cte}/**/*.go` nunca casam no Windows. Fazer o `release.yml` depender desse workflow.

### Faxina [P cada]

- Apagar os branches locais já em `main`: `feat/cte`, `feat/document-ui-standard`, `chore/follow-ups`, `docs/pendencias`, `maintenance`, `maintenance-2026-08`, `backup-maintenance-2026-08`, `improve-log-and-keyring`, `logging`, `logs`, `eslint`. Remotos (`origin/eslint`, `origin/improve-log-and-keyring`, `origin/feat/export-selected-documents`, `origin/refact`, `origin/refact-b`) só com aprovação; o `eslint` carrega um `.golangci.yml` regredido e nunca deve ser aproveitado. Manter `beta` (spike Wails v3) até existir um RC do v3.
- Remover `internal/desktop/.task/` e `internal/desktop/build/darwin/icons.icns` (sobras do spike v3 de 03/10) e ignorar `.task/` e `.claude/` no `.gitignore`.
- `.gitattributes` diz `* text=LF`, valor que o git não reconhece (o certo é `* text=auto eol=lf`), então nada normaliza fim de linha hoje; corrigir e renormalizar (`git add --renormalize .`) num commit só disso, para não misturar churn de EOL com mudança de conteúdo. Padrão do repositório: LF.
- `golang.org/x/crypto` 0.55 → 0.56+ (GO-2026-6354 e GO-2026-6355 corrigidos; o govulncheck não os alcança hoje, mas o bump é grátis).
- `Makefile`: `.PHONY` sem `seeddev`, `mockcert`, `screenshots`; o `where … 2>NUL` só funciona no Windows.
- Docs desatualizadas: `docs/NFSE_ADN.md:21-23` diz que o app consome `GET NFSe/{chave}` (só chama `NFSe/{chave}/Eventos`); mensagem de skip em `internal/foundation/cert/pfx_test.go:88,137` aponta para `gen/mock_cert.go` (hoje é `make mockcert`); `AGENTS.md` e `docs/ARCHITECTURE.md` não listam `internal/cte`, `dfe`, `company`, `credential`, `danfse`, `files`, `cmd/mockcert`, `cmd/seeddev` nem os stores `nfeDocuments`, `cteDocuments`, `diagnostics`, `preferences`, `documentListState`.
- `internal/sefaz`: `ConsNSU`, `ConsChNFe` e `ConsCTeNSU` existem sem chamador. Não apagar: a Fase 2 usa os dois primeiros. Anotar no código.

### Issues [P/M]

- **#10** (senha na memória): fechar com comentário listando o residual que não dá para zerar (string do `keyring.Set`, `NANCI_CERT_PASSWORD`, string JS do Wails, chaves derivadas no fork do pkcs12).
- **#14** (links do site): rodar um verificador de links no HTML do Hugo e fechar ou corrigir.
- **#4** (fork do go-pkcs12): upstream está em v0.7.3; verificar se ganhou suporte a BER indefinido (`third_party/go-pkcs12/README.nanci.md`). O fork também carrega `DecodeChainBytes` (#10), então voltar ao upstream exige levar isso para o chamador. [M]
- **#15** (assinatura Windows): depende de aprovação externa (SignPath). Abrir o pedido; o passo no workflow é pequeno. [M]
- **#12** (criptografia em repouso): `modernc.org/sqlite` não tem SEE/SQLCipher e os blobs são arquivos simples. **Estacionado** (decisão de 07/10/2026): fica aberto, sem plano, até o driver mudar ou surgir um pedido concreto. Comentar na issue.

### Dependências [M]

- Go, menores: `modernc.org/sqlite` 1.49 → 1.60 (11 menores, rodar a suíte de migrações), `cobra` 1.10, `goose` 3.28, `sethvargo/go-retry` 0.3 → 0.5 (pré-1.0, conferir API), `x/term`.
- Frontend, menores primeiro: `quasar` 2.19 → 2.34 (conferir `pnpm run screenshots`), `vue`, `vite`, `eslint`, `vue-tsc`.
- Frontend, maiores, um PR cada, nesta ordem: `vitest` 5, `typescript` 7, `pinia` 4 (+ `@pinia/testing` 2), `vue-router` 5, `@quasar/vite-plugin` 2. `@types/node` fica em 24 (Node da CI).
- `histoire` está em `1.0.0-beta.1`; decidir se fica.

### Comportamentos do Ambiente Nacional que o app ainda não trata [M]

- **Regra dos 60 dias.** A SEFAZ só gera NSU para raízes de CNPJ que consultaram o `distNSU` nos últimos 60 dias; uma empresa parada por mais tempo perde o intervalo para sempre (NT 2014.002 v1.10, em produção desde 10/11/2021; confirmado só por relatos de terceiros, conferir o texto da NT). Como todo software que usa o mesmo CNPJ compartilha a mesma sequência de NSU, o ERP do contador e o nanci também podem colidir no 656. O app não avisa. Mostrar aviso na Empresa e no `company status` quando o último sync de NF-e/CT-e passar de **45 dias** (limiar aceito em 07/10/2026), e explicar na doc.
- **Reuso do `ultNSU`.** Conferir em `internal/sync/source_nfe.go` e `source_cte.go` que o cursor gravado é sempre o `ultNSU` devolvido, mesmo em lote vazio.
- **CNPJ alfanumérico em produção desde 10/08/2026** (NT 009 NFS-e). O dígito verificador já está feito; conferir o que ainda assume só dígitos: `ParseAccessKey` em `internal/nfse` e a regex da Consulta Direta.
- **DANFSe nacional.** A NT 008/2026 desliga a API de geração de DANFSe do ADN (prazo 03/08/2026). O app renderiza localmente com `go-danfse-v2`, então não é afetado; registrar na doc para ninguém propor usar a API.

## Fase 1: dívida estrutural (1 a 2 meses)

**Estado em 07/10/2026: concluída em `main`.** Vocabulário movido (`internal/syncstate`, `dfe`, `company`, `credential`; `internal/nfse` só NFS-e); módulo desktop de 24% para 67% de cobertura com `App` falando com interfaces estreitas (`internal/desktop/services.go`) e fakes em teste; os quatro módulos do frontend sem teste cobertos (+25 testes); logs em disco documentados em `privacidade.md`; CI com checks de drift (`go mod tidy`, `wails generate module`); skills de processo `lane` e `integrate` e de desenvolvimento `new-source`, `desktop-method`, `migration`, `cli-command`, `sefaz-call`, `website-docs`, `research-doc` em `.claude/skills/`, mais a seção "Working With Agents" no `AGENTS.md`. Derivar os skills do código achou e corrigiu: `--verbose`/`--trace` ignorados e prompt de senha no stdout (CLI), `schema.sql` divergente em duas tabelas (agora com teste de drift), quatro fatos errados no site, nome do banco de dev na doc. Achados dos testes do desktop que não foram corrigidos estão no `PENDENCIAS.md`.

- **Vocabulário compartilhado fora de `internal/nfse`** [G]: `Company`, `Credential`, `Environment`, `SyncSource`, `SyncStatus`, `SyncStopReason`, `SyncMode`, `SyncStartPolicy`, `SyncRun`, `SyncState` e `SyncSnapshot` têm 576 referências em 66 arquivos fora do pacote (`sync` 148, `company` 49, `credential` 24, `app` 24, `store` 19, `sefaz` 6). PR só de movimento, sem mudança de comportamento. Destino decidido em 07/10/2026:
  - `Company` → `internal/company`, que já é o pacote da empresa cadastrada (`Store` e `Manager`); passa a ter também o tipo. Idem `Credential` → `internal/credential`.
  - `Environment` (produção/homologação, o `tpAmb`) → `internal/dfe`, que é o vocabulário DF-e sem rede nem banco.
  - Os enums e registros de sincronização (`SyncSource`, `SyncStatus`, `SyncStopReason`, `SyncMode`, `SyncStartPolicy`, `SyncRun`, `SyncState`, `SyncSnapshot`) → novo pacote folha `internal/syncstate`. Não podem ir para `internal/sync` porque `sync` importa `store` e `store` precisa deles (ciclo). O nome diz o que é: o estado da sincronização, sem o loop.
  - Depois do movimento, `internal/nfse` fica só com a NFS-e, como `nfe` e `cte`.
  - **Feito em 07/10/2026**, em quatro commits de movimento; `CredentialID` e `GenerateID` foram para `internal/dfe` junto com `Environment`.
- **Testes do módulo desktop** [M]: `internal/desktop` está em 24% de cobertura; os métodos de `App` não têm teste. Introduzir um `core` falso (interface pequena por serviço) e testar validação de entrada, mapeamento de DTO e erros. Isso também é pré-requisito para a paridade de exportação.
- **Frontend sem teste** [P/M]: `composables/useCredentials.ts`, `composables/useSefazBlock.ts`, `stores/query.ts`, `platform/wails/runtime.ts`.
- **Logs em disco com CNPJ em claro**: mantém em claro (decisão de 07/10/2026), registrado em `website/content/docs/privacidade.md`.
- **Docs**: atualizar `ARCHITECTURE.md` e `AGENTS.md` depois do movimento de pacotes; `docs/specs/` tem um único spec de 06/2026, decidir se a pasta fica.

## Fase 2: funcionalidades já pendentes

Ordem por valor para quem usa, não por facilidade.

1. **Paridade de exportação (NF-e e CT-e com CSV, XLSX e PDF)** [G]. Plano aprovado em 07/10/2026 e interrompido antes do primeiro commit; o arquivo de plano está fora do repositório. Ao retomar, cortar: eventos no ZIP da NFS-e e `exported_events` na NFS-e (ninguém pediu), granularidade de 32 commits (fundir por lane) e testes de borda. Manter: `fiscal-renderer` v0.8.1 para DANFE/DACTE (só CT-e modelo 57 proper; OS, GTV-e e Simplificado ficam de fora e são contados), um tipo de linha por origem, sem camada central de exportação, migração 021 só para os novos `export_kind`, e o bug do toast "Nenhum documento para exportar" após exportar uma linha (`ExportedCount` 0 em `internal/desktop/app.go:448-478`).
2. **Importar XML avulso e buscar por chave** [M]. NF-e e CT-e recebidos por e-mail ou fora da janela de 90 dias / 3 meses. Reaproveitar `sefaz.ConsChNFe` para buscar o `procNFe` pela chave quando a empresa tem direito (manifestada), dentro do teto de 20/h compartilhado com `consNSU`. Para CT-e não existe consulta por chave na distribuição; só importação de arquivo.
3. **Ligação CT-e ↔ NF-e nas telas** [M]. Dados já em `cte_documents.nfe_chaves`; falta a navegação nas duas páginas.
4. **Prestação em desacordo (CT-e, 610110)** [G]. Só o tomador envia, até 45 dias, à SEFAZ autorizadora via `CTeRecepcaoEventoV4` (tabela UF → endpoint: SVRS para a maioria; SP, MT, MS, MG e PR próprios). Reaproveita `xmldsig.go`; precisa de tabela de auditoria e confirmação explícita como a Ciência.
5. **Recuperar intervalo perdido de NSU** [M]. Depois de um bloqueio 656 ou de uma pausa, oferecer `consNSU` para os NSUs faltantes (20/h). Depende do item 2 ter trazido a chamada para dentro do loop.

## Fase 3: mais dados fiscais por API

Universo levantado em 07/10/2026 (fontes oficiais quando existem; itens marcados *3P* vêm só de relatos de terceiros). Ordenado por valor para a empresa ou o contador × viabilidade para uma ferramenta desktop com certificado A1, em Go, gratuita.

| # | Fonte | O que traz | Acesso | Custo | Viabilidade | Valor |
|---|---|---|---|---|---|---|
| 1 | NF-e `consChNFe` / `consNSU` | procNFe por chave; NSU faltante | A1, mesmo serviço de hoje | grátis, 20/h | fácil | alto (import e recuperação) |
| 2 | **MDF-e** `MDFeDistribuicaoDFe` v1.00 | MDF-e em que a empresa é autXML, contratante ou dona do veículo | A1, SOAP SVRS | grátis | médio | médio (transportadoras e contratantes) |
| 3 | NFS-e ADN `GET /NFSe/{chave}` | a nota pela chave | A1, já documentado | grátis | fácil | médio (consulta direta completa) |
| 4 | NFS-e São Paulo `ConsultaNFeRecebidas` | notas tomadas em SP, 50 por página | A1 + XMLDSig SHA-1 | grátis | médio | depende: *verificar primeiro* se SP já aparece no ADN pelo compartilhamento (LC 214 art. 62; *3P* diz que sim) |
| 5 | `CadConsultaCadastro4` | cadastro de IE do fornecedor por CNPJ | A1; BA, GO, MS, MT, PE, PR, RS, SP e SVRS; SP/ES podem recusar certificado de outra UF | grátis | fácil | baixo a médio (validar contraparte) |
| 6 | CNPJ público (`minhareceita.org`, BrasilAPI) | razão social, UF, situação | sem certificado | grátis, limites por IP | fácil | médio (preencher cadastro da empresa) |
| 7 | **eSocial** download de eventos | XML dos eventos já enviados, inclusive por contador | A1 da empresa; 10 chamadas/dia, 50 eventos por chamada, nada nos dias 1 a 7 | grátis | médio | alto para quem tem folha, mas é folha/RH, não documento fiscal: **fora do escopo por ora** (07/10/2026) |
| 8 | Serpro Integra Contador | DCTFWeb, Simples/PGDAS, caixa postal, parcelamentos, pagamentos, DARF (Sicalc), CND | e-CNPJ mTLS + OAuth2; a própria empresa não precisa de procuração | pago por chamada: consulta R$0,24 a 0,06; sem faixa grátis | médio | alto para contador. **Sem plano agora** (07/10/2026); fica em aberto como módulo opcional "traga suas credenciais". Nada barato a preparar hoje: o mTLS do `httpclient` já serve, e o resto (OAuth2, envelope JSON) só faz sentido com o módulo |
| 9 | CBS apuração (`api.receitafederal.gov.br/apuracao-cbs/v2`) | débitos e créditos de CBS | OAuth gov.br (não é A1) + **webhook HTTPS público** | grátis, 4/dia | difícil para desktop | alto a partir de 2027; acompanhar |
| 10 | EFD-Reinf consulta de recibos | só recibo, id e status | A1 | grátis | fácil | baixo |
| 11 | GNRE lote | emissão de guia | e-CNPJ + cadastro no portal | grátis | médio | baixo (emissão, não coleta) |
| 12 | Serpro Consulta NF-e / CNPJ / CND | JSON limitado pela anuência; CND em PDF | API key | R$0,66 a 0,88 por consulta | fácil | baixo (não devolve o XML) |

Sem caminho (não planejar):

- **NF-e e CT-e emitidos pela própria empresa**: a distribuição nunca devolve ao emitente (`consChNFe` responde 641). Só via importação de XML ou do emissor.
- **NFC-e** para o comprador: não existe distribuição; desde 04/05/2026 compra com CNPJ exige NF-e 55. Em SP o emissor baixa as próprias pelo SAE-NFC-e.
- **NF3e, NFCom, BP-e, NFAg, NFGas**: sem serviço de distribuição na SVRS.
- **SPED (EFD, ECD, ECF)**: só pelo ReceitanetBX, sem API.
- **IBS (Comitê Gestor)**: nenhuma API encontrada.
- **CNDT e CRF/FGTS**: sem API oficial.
- **Open Finance**: exige autorização do Banco Central para receber dados.

Fatos do ecossistema que mudam prioridades:

- LC 214/2025 art. 62: desde 01/01/2026 todo município usa o emissor nacional ou compartilha com o ADN; gov.br diz que os 5.571 entes aderiram. Entre os 10 maiores PIBs, Rio, Maricá, BH, Manaus, Curitiba e Porto Alegre estão no emissor nacional; São Paulo e Brasília têm sistema próprio compartilhando; Osasco e Guarulhos sem confirmação.
- CGSN 191/2026: ME e EPP obrigadas ao emissor nacional a partir de 01/11/2026. Reduz ainda mais o caso para APIs municipais.
- Módulo de Apuração Nacional (ISS, guia DNA) em produção restrita desde 14/04/2026, sem spec de API.
- Nenhuma biblioteca Go para DistDFe, CT-e, MDF-e, GNRE ou Integra Contador; referências MIT/Apache existem em Dart, Python e TypeScript.

Sequência proposta: **1 → 3 → 6** são pequenas e cabem na Fase 2. **2 (MDF-e)** é a quarta origem e é o gatilho que o `PENDENCIAS.md` já previa para reconsiderar um `DocumentTable` e um store comum; fazer depois da paridade de exportação, para não carregar quatro páginas de exportação diferente. **4 e 5** quando houver um caso real (uma nota de SP que não apareça no ADN; um fornecedor a validar). **7 e 8** fora do plano por decisão (acima). **9** só quando houver um caminho sem webhook público ou um relay opcional.

## Decisões tomadas em 07/10/2026

1. Vocabulário compartilhado: `Company` → `internal/company`, `Credential` → `internal/credential`, `Environment` → `internal/dfe`, estado de sync → novo `internal/syncstate`.
2. #12 criptografia em repouso: estacionado, sem plano.
3. APIs pagas (Serpro): sem plano agora; possível módulo opcional no futuro; nada a preparar hoje.
4. eSocial: fora do escopo por ora (é folha/RH, não documento fiscal).
5. Logs em disco: mantêm o CNPJ em claro; só o pacote exportado mascara.
6. Aviso dos 60 dias de NSU: limiar de 45 dias.
