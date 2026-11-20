# Desenvolvimento

O Nanci tem duas frentes que compartilham o mesmo núcleo em Go (`internal/app`, `internal/store`, `internal/sync`):

1. **CLI** (`cmd/nanci`): Go puro.
2. **Desktop** (`internal/desktop`): [Wails v3](https://v3.wails.io/) com Go no backend e Vue 3 + Quasar no frontend (`internal/desktop/frontend`). É um módulo Go separado.

Estrutura de pacotes e fluxo de dados em [ARCHITECTURE.md](ARCHITECTURE.md); padrões de código, commits e testes em [CONTRIBUTING.md](../CONTRIBUTING.md).

## Requisitos

| Ferramenta | Versão | Para quê |
|---|---|---|
| Go | a do `go.mod` (hoje 1.26) | tudo |
| Node.js | 24 (`internal/desktop/frontend/.node-version`; mínimo 22) | frontend |
| pnpm | 10 | frontend |
| Wails CLI (`wails3`) | v3.1.0, a mesma do `internal/desktop/go.mod` | desktop |
| golangci-lint | a do CI (`version:` em `.github/workflows/ci.yml`, hoje v2.12.2) | `scripts/verify.sh` |
| NSIS | 3 | só para gerar o instalador |
| OpenSSL | qualquer | só para recriar o certificado mock |

No Windows não é preciso compilador C: o SQLite é `modernc.org/sqlite` (Go puro) e o Wails v3 não usa Cgo no Windows. O release roda em Linux e compila o executável do Windows sem Cgo; só precisa do NSIS. No Linux, compilar o módulo do desktop (inclusive para os testes) exige Cgo e os cabeçalhos do GTK 4 e do WebKitGTK 6.0 (`libgtk-4-dev` e `libwebkitgtk-6.0-dev` no Ubuntu), como faz o CI.

Instale o CLI na versão exata, nunca `@latest`:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
wails3 doctor
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

O comando é idempotente. Ele cria `devdata/nanci-v1.db` com as migrações aplicadas, copia o certificado mock para `devdata/certs/` e cadastra uma empresa e uma credencial de teste. O certificado mock fica em `internal/foundation/cert/testdata/` (senha `mockdata`, CNPJ `70860312000150`) e só serve até a primeira chamada a um serviço real. Para recriá-lo: `make mockcert` (requer OpenSSL).

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
wails3 dev
```

O `wails3 dev` segue `build/config.yml` e os Taskfiles: compila o app em modo dev (o que regenera os bindings em `frontend/bindings/`), sobe o Vite em `127.0.0.1:9245` e abre a janela. Salvar arquivos em `frontend/src` recarrega a janela; mudanças em Go recompilam o backend. O frontend usa só `src/platform/wails/` para falar com o Wails (veja [AGENTS.md](../AGENTS.md)). Se o `wails3 dev` não abrir, rode as partes separadas: `pnpm dev --port 9245 --strictPort` em `frontend/` e, em `internal/desktop`, `wails3 build DEV=true` seguido de `FRONTEND_DEVSERVER_URL=http://127.0.0.1:9245 ./bin/nanci-desktop.exe`. Assim o Go não recompila sozinho.

Comandos do frontend, em `internal/desktop/frontend`:

```bash
pnpm run lint:check
pnpm run test:unit
pnpm run build
```

`lint:check` roda o ESLint e o `prettier --check` em `src` e `scripts`; `pnpm run lint` corrige os dois.

## Verificar antes do PR

```bash
bash scripts/verify.sh    # ou: make verify
```

O `verify.sh` roda na sua máquina o que o CI confere, com as versões das ferramentas lidas do `ci.yml`: lint, testes e build do frontend; testes e `golangci-lint` nos dois módulos Go (que incluem os testes de arquitetura); `go.mod` arrumado; bindings do Wails, código do sqlc e a cópia dos skills em `.claude/skills` em dia; e o `govulncheck`. Roda todas as etapas mesmo depois de uma falha, termina com a lista do que falhou e avisa quando um arquivo de regra (skills, lint, testes de arquitetura, migrations) mudou desde a `main`. Leva uns 2 minutos.

`make check` (fmt, vuln, lint, test e security) também formata o código e roda o `gosec` e o `gitleaks`; `make test` roda só `go test ./...`.

O CI falha se `go.mod`/`go.sum` ou `internal/desktop/frontend/bindings` estiverem desatualizados. Para regenerar: `GOWORK=off go mod tidy` na raiz e em `internal/desktop`, e `wails3 generate bindings -clean=true -ts -i` em `internal/desktop`.

## Gerar o instalador

```bash
cd internal/desktop
wails3 build                    # só o executável: bin/nanci-desktop.exe
wails3 task windows:package     # executável e instalador NSIS
```

O instalador sai em `internal/desktop/bin/nanci-desktop-amd64-installer.exe` e instala por usuário, sem pedir permissão de administrador. Os releases são gerados pelo workflow `.github/workflows/release.yml` a partir de uma tag `vX.Y.Z`: ele grava a versão em `build/config.yml` (de onde saem os metadados do executável e do instalador) e em `internal/foundation/buildinfo` (pela variável `EXTRA_LDFLAGS` do Taskfile), e tira as notas da mensagem do commit da tag.

## Chave de atualização

O app só instala atualizações assinadas pela chave Ed25519 fixada em `internal/desktop/updater_key.go`, e o release falha sem ela ou sem o segredo que corresponde a ela. Para criar ou trocar o par (num fork, por exemplo):

1. Fora do repositório, `wails3 updater genkey -out nanci-updater.key`. Copie a linha impressa abaixo de "Or inline as base64 of the raw key" (44 caracteres, termina em `=`) para `updaterPublicKey`; `TestCommittedUpdaterKeyIsValid` confere o formato.
2. `gh secret set UPDATER_SIGNING_KEY < nanci-updater.key` (o PEM inteiro).
3. Guarde o `.key` num cofre de senhas e apague a cópia local. Quem tem a chave publica atualizações para todos os usuários. Se ela se perder ou for trocada, cada usuário precisa reinstalar pelo instalador uma versão com a chave nova.

Para testar uma atualização sem release: dois builds `-tags production` com `-ldflags "-X main.updaterPublicKey=<chave descartável> -X main.updateManifestURL=http://127.0.0.1:<porta>/nanci-update.json"`, e o manifesto do segundo gerado por `wails3 updater manifest -key`. Dar outro nome ao exe faz o WebView2 usar outra pasta de dados (`%APPDATA%\<nome do exe>`).

## Atualizar o Wails

O módulo Go (`internal/desktop/go.mod`), o `@wailsio/runtime` (versão exata em `package.json`), o `wails3` (`go install ...@vX.Y.Z`, nunca `@latest`) e o `WAILS_VERSION` de `ci.yml` e `release.yml` mudam juntos. O `pnpm-workspace.yaml` exige 7 dias de publicação (`minimumReleaseAge: 10080`): escolha uma versão publicada há mais de 7 dias ou abra em `minimumReleaseAgeExclude` uma exceção `@wailsio/runtime@X.Y.Z`, com data para remover ([pnpm](https://pnpm.io/settings/dependency-resolution#minimumreleaseageexclude), conferido em 29/10/2026).

Os testes não cobrem a janela. Depois de atualizar, confira na mão: fechar durante uma sincronização (X, Ctrl+Q, Alt+F4), tema fixo contra o tema do Windows, cancelar um diálogo de arquivo, abrir uma segunda instância, redimensionar pelas laterais e usar o Snap.

## Problemas comuns do desktop

Matar o app à força pode deixar processos `msedgewebview2.exe` presos ao perfil `%APPDATA%\nanci-desktop.exe\EBWebView`. Enquanto eles existem, o app sai na hora com código 1, sem janela e sem log (visto na v3.0.0-beta.27). Encerre os `msedgewebview2.exe` que têm `nanci-desktop` na linha de comando.

Uma atualização automática que falha deixa o log do auxiliar em `%TEMP%\wails-update-<pid do app>.log`. O exe substituído fica como `.old.<n>` na pasta de instalação até a próxima atualização.

## Testes de certificado

Os testes de compatibilidade PKCS#12/BER (`third_party/go-pkcs12`, `internal/foundation/cert`) incluem a validação opcional de um certificado real, mantido fora do repositório. Sem ele, esses casos são pulados.

## Capturas de tela

As imagens de `docs/screenshots/` (usadas no README e no site) são geradas com dados fictícios: o Vite sobe o frontend com o backend simulado e o Playwright captura cada tela nos temas claro e escuro.

```bash
make screenshots    # ou, em internal/desktop/frontend: pnpm run screenshots
```

## Site de documentação

O site em [nanci.abelhaninja.de](https://nanci.abelhaninja.de/) é gerado com [Hugo](https://gohugo.io/) (extended) e o tema [Hextra](https://imfing.github.io/hextra/), a partir de `website/`. Ele é publicado pelo workflow `.github/workflows/website.yml` a cada push em `main` que mude `website/` ou `docs/screenshots/`.

Para ver localmente:

```bash
bash website/scripts/copy-screenshots.sh
cd website
hugo server
```

O site é a documentação de uso. Detalhes técnicos ficam em `docs/` e o site aponta para eles; não copie o conteúdo de um para o outro.

Além do HTML, o build gera saídas para buscadores e LLMs: cada página da documentação tem uma cópia em Markdown (`docs/<página>.md`, `layouts/page.markdown.md`); a raiz tem `llms.txt`, uma linha por página a partir do `description` (`layouts/llms.txt`), e `llms-full.txt`, o Markdown de todas as páginas na ordem do site com links absolutos (`layouts/llms-full.txt`); uma página sai das duas listas com `llms: false`. `robots.txt` aponta o `sitemap.xml`. As tags Open Graph e Twitter usam `static/img/og-default.png` (1200×630), ou o primeiro screenshot da página na documentação, e cada página leva JSON-LD (`layouts/_partials/jsonld.html`): `SoftwareApplication` na home, `TechArticle` na documentação e `FAQPage` numa página feita de `{{< faq >}}`.
