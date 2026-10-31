# Contribuindo

Obrigado pelo interesse. Para preparar o ambiente, rodar e compilar o projeto, veja [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Este arquivo trata de onde as coisas ficam e das regras do código.

> [!CAUTION]
> Nunca anexe certificados (`.pfx`, `.p12`), senhas, XMLs fiscais reais ou o banco SQLite em issues, PRs ou commits. Problemas de segurança vão por e-mail, conforme o [SECURITY.md](SECURITY.md).

## Estrutura do projeto

| Pasta | Conteúdo |
|---|---|
| `cmd/nanci` | Entrada do CLI. |
| `cmd/seeddev`, `cmd/mockcert` | Ambiente local em `devdata/` e certificado mock. |
| `internal/cli` | Comandos Cobra. |
| `internal/app` | Casos de uso e montagem das dependências. |
| `internal/company`, `internal/credential` | Cadastro de empresas e de credenciais (certificados). |
| `internal/sync` | Loop de sincronização por NSU, comum a NFS-e, NF-e e CT-e. |
| `internal/store` | SQLite, repositórios (sqlc) e migrações em `migrations_v2/`. |
| `internal/nfse`, `internal/adn` | Domínio da NFS-e e cliente do ADN. |
| `internal/nfe`, `internal/cte`, `internal/dfe` | Domínio da NF-e, do CT-e e o vocabulário comum aos dois. |
| `internal/sefaz` | Cliente SOAP da SEFAZ (distribuição e eventos) e assinatura XMLDSig. |
| `internal/report`, `internal/danfse` | Exportações (XLSX, CSV, ZIP) e geração do DANFSe em PDF. |
| `internal/files` | Armazenamento dos XMLs originais em `blobs/`. |
| `internal/foundation` | Utilitários: CNPJ, caminhos, certificado, cliente HTTP mTLS, mascaramento de logs. |
| `internal/desktop` | App Wails: backend Go na raiz, frontend Vue 3 em `frontend/`. |
| `website` | Site de documentação (Hugo). |
| `docs` | Documentação técnica. |

Detalhes em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Código

- Formate antes de commitar: `make fmt` (roda `goimports -local github.com/vasfvitor/nanci` e `gofmt -s`).
- Nomes exportados em `CamelCase`, internos em `camelCase`, pacotes curtos e em minúsculas.
- Termos fiscais ficam em português em todas as camadas (Go, DTOs, TypeScript, colunas): `ChaveAcesso`, `CNPJ`, `Competence`, `tomada`/`prestada`, `Manifestacao`. Não traduza.
- No frontend, páginas e componentes não importam os bindings gerados do Wails; use `desktopClient` e `src/platform/wails/`.

As regras completas, inclusive as do frontend, estão em [AGENTS.md](AGENTS.md).

## Testes

- Testes ficam ao lado do código (`*_test.go`). Use `testdata/` para arquivos de um pacote e `internal/store/storetest` para o banco de teste migrado e as fábricas de empresa e credencial.
- Quando o teste precisar gravar arquivos ou criar banco, use `t.TempDir()`.
- Mudanças em parser, armazenamento ou caminhos de arquivo pedem teste de regressão.
- Rode `bash scripts/verify.sh` (ou `make verify`) antes de cada commit: ele confere Go e frontend como o CI ([docs/DEVELOPMENT.md](docs/DEVELOPMENT.md#verificar-antes-do-pr)). `make check` também formata e roda `gosec` e `gitleaks`. Para rodar só os testes com o detector de corrida: `go test -race ./...`.

## Commits e PRs

Commits no formato `escopo: mudança`, no imperativo e curtos:

```text
frontend: fix company dialog spacing
cli: validate --competencia input
nfe: handle cStat 656 on distribution
```

Escopos: `frontend`, `cli`, `app`, `store`, `sync`, `nfse`, `adn`, `nfe`, `cte`, `sefaz`, `wails`, `docs`, ou a combinação de dois quando a mudança atravessa camadas (`frontend,cli`).

No PR, explique a mudança de comportamento, liste os comandos de verificação e inclua capturas de tela para mudanças visuais. Aponte qualquer impacto em schema, migração ou tratamento de certificado.

## Trabalhar num fork com um agente de IA

Para quem vai mudar o Nanci pedindo a um agente (Claude Code, por exemplo), mesmo sem ser desenvolvedor.

### Preparar uma vez

1. Instale as ferramentas da tabela de [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md#requisitos). O `scripts/verify.sh` mostra o comando de instalação de cada uma que faltar ou estiver na versão errada.
2. Clone o seu fork e aponte para o repositório original, para receber as atualizações:

   ```bash
   git remote add upstream https://github.com/vasfvitor/nanci.git
   ```

3. Crie os dados de teste e rode a verificação uma vez, para saber que o ambiente está certo:

   ```bash
   go run ./cmd/seeddev
   bash scripts/verify.sh
   ```

### Pedir uma mudança

Peça em português, do jeito que você explicaria a uma pessoa. O agente segue o skill [`nova-funcionalidade`](.claude/skills/nova-funcionalidade/SKILL.md): mostra um plano em linguagem simples e espera o seu ok, trabalha numa branch local com um commit por passo, roda o `scripts/verify.sh` e, no fim, explica o que mudou e como testar. Não precisa de PR: a branch entra na sua `main` por fast-forward, o que mantém o histórico linear.

O skill tem uma lista de coisas que o agente não faz sem perguntar, entre elas enviar eventos à SEFAZ, usar produção com um CNPJ real e editar uma migration existente. Se ele pedir para fazer uma delas, leia o motivo com calma antes de aceitar.

Duas verificações automáticas seguram a arquitetura mesmo que ninguém revise o código:

- `TestOnlyDesktopAPITypesReachTheFrontend` ([internal/desktop/architecture_test.go](internal/desktop/architecture_test.go)) falha se um tipo interno, em vez de um DTO de `desktopapi`, passar a chegar à tela por um método ou por um evento;
- `TestMigrationsAreFrozen` ([internal/store/migrations_frozen_test.go](internal/store/migrations_frozen_test.go)) falha se uma migration já existente mudar.

### Receber as atualizações do original

```bash
git fetch upstream
git switch main
git rebase upstream/main
```

Os seus commits são reaplicados por cima dos novos do original. Se aparecer conflito, peça ao agente para explicar antes de resolver. Se o seu fork também está no GitHub, a `main` reescrita só sobe com `git push --force-with-lease origin main`, e só no seu fork.

### CI no GitHub (opcional)

A verificação principal é a local, com o `scripts/verify.sh`. Para ter o mesmo no GitHub, habilite o Actions na aba **Actions** do fork, porque num fork ele vem desligado ([GitHub Docs](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows), conferido em 30/10/2026). Deixe ligado o **CI** e desabilite **Website**, que publica o site do original, e **Release**, que falha sem o segredo `UPDATER_SIGNING_KEY` ([.github/workflows/release.yml](.github/workflows/release.yml)): clique no workflow e em **Disable workflow** ([GitHub Docs](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/disable-and-enable-workflows)).

### Se você for publicar versões suas

O release de um fork já procura atualização nos releases do próprio fork: o `release.yml` grava no exe a URL do repositório que gera o release ("Onde procura" em [docs/WAILS_V3.md](docs/WAILS_V3.md)). A chave não muda sozinha. Para publicar, gere o seu par de chaves, troque `updaterPublicKey` em [internal/desktop/updater_key.go](internal/desktop/updater_key.go) e cadastre a chave privada no segredo `UPDATER_SIGNING_KEY`, como em "Passos do dono" de [docs/WAILS_V3.md](docs/WAILS_V3.md). Sem isso, o release falha na conferência da assinatura (`wails3 updater verify` em [.github/workflows/release.yml](.github/workflows/release.yml)) e nada é publicado. Builds feitos fora do `release.yml` ficam sem a URL e não procuram atualização.
