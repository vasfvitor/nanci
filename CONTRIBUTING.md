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
| `internal/testutil/fixtures` | Factories e helpers de teste. |
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

- Testes ficam ao lado do código (`*_test.go`). Use `testdata/` para arquivos de um pacote e `internal/testutil/fixtures` para helpers reutilizáveis.
- Quando o teste precisar gravar arquivos ou criar banco, use `t.TempDir()`.
- Mudanças em parser, armazenamento ou caminhos de arquivo pedem teste de regressão.
- Rode `make check` antes do PR. Para rodar só os testes com o detector de corrida: `go test -race ./...`.
- No frontend (`internal/desktop/frontend`): `pnpm run lint:check`, `pnpm run test:unit` e `pnpm run build`.

## Commits e PRs

Commits no formato `escopo: mudança`, no imperativo e curtos:

```text
frontend: fix company dialog spacing
cli: validate --competencia input
nfe: handle cStat 656 on distribution
```

Escopos: `frontend`, `cli`, `app`, `store`, `sync`, `nfse`, `adn`, `nfe`, `cte`, `sefaz`, `wails`, `docs`, ou a combinação de dois quando a mudança atravessa camadas (`frontend,cli`).

No PR, explique a mudança de comportamento, liste os comandos de verificação e inclua capturas de tela para mudanças visuais. Aponte qualquer impacto em schema, migração ou tratamento de certificado.
