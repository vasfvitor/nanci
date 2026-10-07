# Desenvolvimento

O Nanci tem duas frentes que compartilham o mesmo núcleo em Go (`internal/app`, `internal/store`, `internal/sync`):

1. **CLI** (`cmd/nanci`): Go puro.
2. **Desktop** (`internal/desktop`): [Wails v2](https://wails.io/) com Go no backend e Vue 3 + Quasar no frontend (`internal/desktop/frontend`). É um módulo Go separado.

Estrutura de pacotes e fluxo de dados em [ARCHITECTURE.md](ARCHITECTURE.md); padrões de código, commits e testes em [CONTRIBUTING.md](../CONTRIBUTING.md).

## Requisitos

| Ferramenta | Versão | Para quê |
|---|---|---|
| Go | a do `go.mod` (hoje 1.26) | tudo |
| Node.js | 24 (`internal/desktop/frontend/.node-version`; mínimo 22) | frontend |
| pnpm | 10 | frontend |
| Wails CLI | v2.16.0, a mesma do `internal/desktop/go.mod` | desktop |
| NSIS | 3 | só para gerar o instalador |
| OpenSSL | qualquer | só para recriar o certificado mock |

No Windows não é preciso compilador C: o SQLite é `modernc.org/sqlite` (Go puro) e o Wails v2 não usa Cgo no Windows. O release roda em Linux e usa `mingw-w64` só para a compilação cruzada.

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0
wails doctor
```

Ferramentas usadas pelo `make check`: `goimports`, `golangci-lint`, `govulncheck`, `gosec` e `gitleaks`.

## Go workspace

O desktop e o `third_party/go-pkcs12` são módulos separados. Para o editor e o `go build` resolverem os três juntos, crie um `go.work` na raiz (ele é ignorado pelo Git):

```bash
go work init . ./internal/desktop ./third_party/go-pkcs12
```

## Dados de desenvolvimento

Para não misturar dados de teste com os seus dados reais, crie o ambiente em `devdata/`:

```bash
make seeddev        # ou: go run ./cmd/seeddev
```

O comando é idempotente. Ele cria `devdata/nanci-dev.db` com as migrações aplicadas, copia o certificado mock para `devdata/certs/` e cadastra uma empresa e uma credencial de teste. O certificado mock fica em `internal/foundation/cert/testdata/` (senha `mockdata`, CNPJ `70860312000150`) e só serve até a primeira chamada a um serviço real. Para recriá-lo: `make mockcert` (requer OpenSSL).

Para rodar o CLI ou o desktop contra outra pasta de dados, defina `NANCI_DATA_DIR`. `NANCI_TRACE=1` liga o log de rastreamento, que inclui o corpo das respostas com identificadores mascarados.

## Rodar

### CLI

```bash
go build -o nanci.exe ./cmd/nanci
./nanci.exe --help
```

### Desktop

```bash
cd internal/desktop
wails dev
```

O Wails roda `pnpm install` e o Vite. Salvar arquivos em `frontend/src` recarrega a janela; mudanças em Go recompilam o backend. Depois de mudar um método exposto ao frontend, os bindings em `frontend/wailsjs/` são regenerados pelo `wails dev`; o resto do frontend usa só `src/platform/wails/` (veja [AGENTS.md](../AGENTS.md)).

Comandos do frontend, em `internal/desktop/frontend`:

```bash
pnpm run lint:check
pnpm run test:unit
pnpm run build
```

## Verificar antes do PR

```bash
make check    # fmt, vuln, lint, test e security
```

`make test` roda só `go test ./...`. Para mudanças no frontend, rode também os três comandos acima.

O CI falha se `go.mod`/`go.sum` ou `internal/desktop/frontend/wailsjs` estiverem desatualizados. Para regenerar: `GOWORK=off go mod tidy` na raiz e em `internal/desktop`, e `wails generate module` em `internal/desktop`.

## Gerar o instalador

```bash
cd internal/desktop
wails build -platform windows/amd64 -nsis -m
```

O instalador sai em `internal/desktop/build/bin/nanci-desktop-amd64-installer.exe`. Os releases são gerados pelo workflow `.github/workflows/release.yml` a partir de uma tag `vX.Y.Z`, com a versão gravada em `internal/foundation/buildinfo` e as notas tiradas da mensagem do commit da tag.

## Testes de certificado

Os testes de compatibilidade PKCS#12/BER (`third_party/go-pkcs12`, `internal/foundation/cert`) incluem a validação opcional de um certificado real, mantido fora do repositório. Sem ele, esses casos são pulados.

## Capturas de tela

As imagens de `docs/screenshots/` (usadas no README e no site) são geradas com dados fictícios: o Vite sobe o frontend com o backend simulado e o Playwright captura cada tela nos temas claro e escuro.

```bash
make screenshots    # ou, em internal/desktop/frontend: pnpm run screenshots
```

## Site de documentação

O site em [vasfvitor.github.io/nanci](https://vasfvitor.github.io/nanci/) é gerado com [Hugo](https://gohugo.io/) (extended) e o tema [Hextra](https://imfing.github.io/hextra/), a partir de `website/`. Ele é publicado pelo workflow `.github/workflows/website.yml` a cada push em `main` que mude `website/` ou `docs/screenshots/`.

Para ver localmente:

```bash
bash website/scripts/copy-screenshots.sh
cd website
hugo server
```

O site é a documentação de uso. Detalhes técnicos ficam em `docs/` e o site aponta para eles; não copie o conteúdo de um para o outro.
