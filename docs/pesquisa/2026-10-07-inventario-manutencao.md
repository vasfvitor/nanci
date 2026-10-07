# Inventário de manutenção

Levantado em 07/10/2026 no commit `a0ca132` de `main`, só com leitura (testes, lint e govulncheck rodados sem alterar nada). Responde: o que está pendente, desatualizado ou sem cobertura no repositório. O `docs/ROADMAP.md` ordena isto em fases.

Esforço: **[P]** menos de uma hora, **[M]** alguns dias, **[G]** semanas.

## Pendências registradas

- `docs/PENDENCIAS.md`: desacordo CT-e 610110 [G]; ligação CT-e ↔ NF-e nas telas [M] (dados em `cte_documents.nfe_chaves`, CLI `cte list --nfe`); importação de XML avulso [M/G]; documentos emitidos pela própria empresa [G] (SEFAZ não distribui ao emitente; NFS-e já vem com `prestada`); Ciência real nunca enviada (precisa de CNPJ que receba NF-e); vocabulário compartilhado em `internal/nfse` [G]; globs do `depguard` no Windows [P]; adiados por decisão: gancho de retry no `httpclient` e `DocumentTable`/store comum.
- Nenhum `TODO`/`FIXME`/`XXX`/`HACK` em código próprio; só dois comentários de upstream em `third_party/go-pkcs12/internal/rc2/`.

## Docs desatualizadas contra o código [P cada]

- `docs/NFSE_ADN.md:21-23` diz que o app consome `GET NFSe/{ChaveAcesso}`; o código só chama `NFSe/{chave}/Eventos` (`internal/app/query.go:49`).
- `internal/foundation/cert/pfx_test.go:88,137`: mensagem de skip manda rodar `go run gen/mock_cert.go`, que não existe mais (hoje `make mockcert`, em `cmd/mockcert`).
- `AGENTS.md` não lista `internal/cte`, `dfe`, `company`, `credential`, `danfse`, nem os stores `nfeDocuments`, `cteDocuments`, `diagnostics`, `preferences`, `documentListState`.
- `docs/ARCHITECTURE.md:3-21` não lista `internal/company`, `credential`, `danfse`, `files`, `store/sqlgen`, `cmd/mockcert`, `cmd/seeddev`. O resto está certo, inclusive a regra app→store e a ressalva do Windows no `depguard` (a nota em memória de que o código viola a regra está desatualizada: `rg` não acha importação de `app`, `store` ou `database/sql` nos pacotes de domínio).
- `README.md` foi reescrito em `65881e0` e está correto; lista só DANFSe porque DANFE/DACTE não existem (lacuna de funcionalidade, não de doc).

## GitHub (5 issues abertas, 0 PRs)

- **#4** fork do go-pkcs12: base v0.7.1, upstream v0.7.3; o fork também carrega `DecodeChainBytes` (#10), ver `third_party/go-pkcs12/README.nanci.md`. [M]
- **#10** senha na memória: feito em `main` (`[]byte` + `cert.ZeroBytes` em `internal/app/keyring.go:36-67`, `internal/cli/credential.go:41`, `internal/foundation/cert/pfx.go:66`). Residual que não zera: string do `keyring.Set` (`keyring.go:79`), `NANCI_CERT_PASSWORD`, string JS do Wails, chaves derivadas no fork. [P fechar]
- **#12** criptografia em repouso: `modernc.org/sqlite` não tem; blobs são arquivos 0600 (`internal/files/blobstore.go:42`). [G]
- **#14** links do site: páginas apontam para `github.com/.../blob/main`; possivelmente resolvido em `65881e0`, não verificado. [P]
- **#15** SignPath: `release.yml` sem assinatura; depende de aprovação externa. [M]

## Branches

Repositório faz squash-merge, então "ahead" não diz nada; comparado por conteúdo.

- `feat/export-parity`: idêntico a `main`, criado em 07/10/2026.
- Já em `main`, apagar: `feat/cte` (PR #20), `feat/document-ui-standard` (PR #21), `chore/follow-ups` (PR #19), `docs/pendencias`, `maintenance`, `maintenance-2026-08`, `backup-maintenance-2026-08` (três cópias dos mesmos 9 commits), `improve-log-and-keyring`, `logging`, `logs`, `eslint`.
- Remotos (`origin/eslint`, `origin/improve-log-and-keyring`, `origin/feat/export-selected-documents`, `origin/refact`, `origin/refact-b`): conteúdo já portado segundo a triagem de 24/08/2026; `eslint` carrega um `.golangci.yml` regredido; apagar só com aprovação.
- `beta`: spike Wails v3 (8 commits, não enviado). Manter até existir RC do v3.
- Sobras não rastreadas: `internal/desktop/.task/` (cache do Taskfile do spike v3, 03/10) e `internal/desktop/build/darwin/icons.icns` (mesmo spike). `.claude/` na raiz não está no `.gitignore`.

## Dependências

- Go 1.26.6 nos dois módulos e instalado. `replace`: raiz e desktop apontam `go-pkcs12` para `third_party/go-pkcs12`; desktop aponta `nanci` para `../../`. Wails v2.16.0 (último v2), igual ao pin do `release.yml`.
- Atualizações diretas: `goose` 3.27.1→3.28.0; `sethvargo/go-retry` 0.3.0→0.5.0 (pré-1.0); `cobra` 1.9.1→1.10.2; `x/term` 0.45→0.46; `modernc.org/sqlite` 1.49.1→1.60.1 com `libc` 1.72→1.77 [M]; `x/crypto` (indireto) 0.55→0.57, com vulnerabilidades (abaixo). Em dia: `uuid`, `go-danfse-v2` 0.1.0, `excelize` 2.11.0, `go-keyring` 0.2.8. Sem salto de major em dependência direta.
- Frontend (`pnpm outdated`): Node 24.14 local e 24 na CI; pnpm 10.33; `minimumReleaseAge` 7 dias. Majors [M cada]: `pinia` 3.0.4→4.0.3 (+ `@pinia/testing` 1→2), `vue-router` 4.6.4→5.3.1, `typescript` 6.0.3→7.0.2, `vitest` 4.1.10→5.0.3, `@quasar/vite-plugin` 1.12→2.0.2, `@eslint/css` 1→2, `@types/node` 24→26 (manter 24). Menores: `vue` 3.5.34→3.5.43, `quasar` 2.19.3→2.34.0 (conferir screenshots), `vite` 8.0.12→8.3.1, `eslint` 10.4→10.11, `vue-tsc` 3.2.8→3.3.11, `happy-dom`, `playwright`, `prettier`. `histoire` em `1.0.0-beta.1`.

## CI/CD e release

- `codeql.yml`: push/PR em `main` e semanal; actions, go, js-ts; compila os dois módulos para windows/amd64 com `dist/` de mentira.
- `release.yml`: tag `v*` ou manual; valida semver, compila no Ubuntu com Wails 2.16.0, mingw e NSIS; checksums, atestação de proveniência, release. Só o instalador é publicado, não o CLI. Sem assinatura (#15). **Não roda teste, lint nem checagens do frontend.** A chave de cache dos binários Go fixa `wails-2.16.0`.
- `website.yml`: Hugo 0.163.2 extended, Go "1.26", deploy no Pages.
- **Lacuna [M]**: nenhum workflow roda `go test`, `golangci-lint`, `govulncheck`, `pnpm lint:check`/`test:unit`/`build` em PR. Um job de lint no Linux faria o `depguard` valer.
- `Makefile`: alvos `fmt`, `lint`, `vuln`, `test`, `security` (gosec + gitleaks), `check`, `seeddev`, `mockcert`, `screenshots`. `.PHONY` sem os três últimos; `where … 2>NUL` só no Windows; gosec e gitleaks não estão instalados localmente, então `make security`/`check` falham aqui; `go.work` ignorado, logo num clone limpo `make test`/`lint`/`vuln` só cobrem o módulo raiz.
- `internal/desktop/wails.json`: `productVersion` 0.0.0-dev, remendado pela CI com `jq`.

## Arquitetura e lint

- `depguard` regra `domain` (`.golangci.yml`) usa globs `**/internal/{nfse,nfe,dfe,cte}/**/*.go`, que não casam no Windows. O código obedece mesmo assim; golangci-lint 2.12.2 reporta 0 problemas.
- Vocabulário compartilhado em `internal/nfse`: `Company` (`company.go:10`), `Credential` (`:32`), `Environment` (`types.go:45`), `SyncSource` (`sync.go:14`), `SyncStartPolicy` (`types.go:308`) e demais enums de sync. 576 referências `nfse.X` em 66 arquivos fora do pacote: `sync` 148, `company` 49, `credential` 24, `app` 24, `store` 19, `sefaz` 6, `testutil` 4, `desktop` 2, `cli` 2. `internal/sefaz/endpoints.go:14` importa `nfse` só por `Environment`.
- `internal/dfe`: `accesskey.go`, `money.go`, `parse.go`, `types.go` (chave de acesso, `Money`, `CompanyID`, `ErrInvalidEnum`, helpers de parse); `accesskey` e `parse` têm teste.
- Sem chamador em produção: `sefaz.ConsNSU` (`distribuicao.go:99`), `ConsChNFe` (`:108`), `ConsCTeNSU` (`cte_distribuicao.go:39`).

## Testes e qualidade

- Cobertura Go (tudo passa): `adn` 85,0; `app` 65,6; `cli` 56,7; `company` 50,3; `credential` 75,3; `cte` 91,2; `dfe` 79,7; `files` 83,3; `cert` 80,6; `cnpj` 100; `httpclient` 90,6; `redact` 100; `uf` 100; `nfe` 89,8; `nfse` 76,8; `report` 82,4; `sefaz` 92,0; `store` 75,4; `sync` 81,7; **`desktop` 24,4**; `desktopapi` 71,9.
- Sem teste: `internal/danfse`, `foundation/buildinfo`, `foundation/logger` (0%); `foundation/xmlwalk` sem cobertura própria (exercitado por `nfe` e `cte`); `store/seed`, `store/sqlgen` (gerado), `storetest`, `testutil/fixtures`, `cmd/*`.
- Frontend: 81 arquivos, 536 testes passando (vitest 4.1.10). Sem teste: `composables/useCredentials.ts`, `composables/useSefazBlock.ts`, `stores/query.ts`, `platform/wails/runtime.ts`. Nenhum `it.skip`/`describe.skip`/`.only`.
- Skips Go, todos por ambiente: `files/blobstore_test.go:133` (Windows); `app/keyring_test.go:21` e `cert/pfx_test.go:88,137,163` (sem mock cert); `cert/ber_test.go:136` e `pfx_test.go:210` (sem `NANCI_TEST_PFX_PATH`); `sefaz/xmlsec_test.go:18` (sem `xmlsec1`, build tag).

## Segurança e configuração

- govulncheck nos dois módulos: 0 alcançáveis. Em `golang.org/x/crypto` v0.55.0: GO-2026-6355 e GO-2026-6354 (corrigidos em v0.56.0) e GO-2026-5932 (sem correção).
- gosec via golangci-lint: 0 problemas; `G404` excluído globalmente; `G401`/`G505` excluídos em `internal/sefaz` (perfil NF-e exige RSA-SHA1). gosec e gitleaks avulsos não rodados (não instalados).
- Senha no keyring do SO, serviço `nanci_certs`, chave = id da credencial (`internal/app/keyring.go:14`), validada contra o PFX antes de salvar; sem senha válida, CLI pergunta no terminal e desktop por diálogo Wails; alternativa `NANCI_CERT_PASSWORD` ou `.env.local` ignorado. `*.pfx`/`*.p12` ignorados, exceto o mock de teste.
- ZIP de logs exportado mascara CNPJ e chave (`internal/desktop/logsanitize.go:23-28`); **logs em disco mantêm CNPJ em claro** (`internal/desktop/app.go:632`).
- SQLite: DSN só com WAL e `foreign_keys` (`internal/store/db.go:32`); nada criptografado em repouso.

## Origens de dados hoje

- **NFS-e (ADN)**: `https://adn.nfse.gov.br/contribuintes` e produção restrita (`internal/adn/client.go:20-21`), REST/JSON por mTLS A1. Sync por `GET DFe/{NSU}?cnpjConsulta=` (`internal/adn/endpoints.go:49`), lote gzip+base64 com documentos e eventos; papéis tomada, prestada e intermediario (as emitidas vêm). 500 ms entre requisições (`internal/sync/source_nfse.go:21`), sem teto por hora; lote vazio encerra. Política inicial all/since_date/from_now (`internal/nfse/types.go:308`). Consulta direta só `NFSe/{chave}/Eventos`. DANFSe renderizado localmente.
- **NF-e (NFeDistribuicaoDFe, AN)**: www1/hom1 para distribuição, www/hom1 para `NFeRecepcaoEvento4` (`internal/sefaz/endpoints.go:20-25`); SOAP 1.2, TLS 1.2 com renegociação; `cUFAutor` da empresa. Só `distNSU` (`internal/sync/source_nfe.go:62`): até 50 por chamada, resNFe, procNFe, resEvento, procEventoNFe; XML completo só após Ciência ou manifestação conclusiva; eventos de chave desconhecida são descartados salvo se a empresa assinou. Limites: 20/h por empresa, origem e ambiente (`sync_requests`); 2 s entre páginas; em 137 ou `ultNSU == maxNSU` espera 1 h; em 656 bloqueia 1 h; janela ~90 dias; distribuição nunca reenvia; item que falha 3× no mesmo NSU é pulado. Envia 210200/210210/210220/210240 via `EnviarEventos` (`internal/app/nfe_manifest.go:427`), Ciência em lotes de até 20; prazos em `internal/nfe/manifestacao.go`. Não busca: notas emitidas pela empresa; `consChNFe`/`consNSU` (existem, sem chamador); fora da janela; NFC-e, NFCom, NF3e, CF-e SAT; DANFE.
- **CT-e (CTeDistribuicaoDFe)**: www1.cte/hom1.cte (`endpoints.go:22-23`), mesmo SOAP com namespace cte. Só `distNSU` (`internal/sync/source_cte.go:62`): procCTe, procCTeOS, procGTVe, procCTeSimp, procEventoCTe completos, sem resumo. Mesmas regras da NF-e com orçamento próprio de 20/h (escolha do app; a NT não publica limite); janela ~3 meses. Não busca nem envia: CT-e emitidos pela empresa (só eventos de MDF-e sobre eles); MDF-e; 610110 e seu cancelamento; `consNSU` (`ConsCTeNSU` sem chamador); sem consulta por chave; DACTE.
