# Migração para o Wails v3

Registro da migração do desktop (`internal/desktop`) do Wails v2.16.0 para o Wails v3.0.0, a primeira versão estável do v3, publicada em 26/10/2026. A migração foi refeita sobre a `main` em 29/10/2026, usando como receita o spike de 03/10/2026 na branch local `beta` (v3.0.0-beta.27). A `beta` não foi mesclada: a `main` já tinha reorganizado os arquivos que o spike mexeu.

| Peça | Versão |
| --- | --- |
| Módulo Go `github.com/wailsapp/wails/v3` | `v3.0.0` |
| Pacote npm `@wailsio/runtime` | `3.0.0` (versão exata) |
| CLI `wails3` | `v3.0.0` (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`, nunca `@latest`) |
| Go | 1.26 ou mais novo (exigência do v3) |

Os nomes de API e os comportamentos citados aqui foram conferidos no código-fonte da v3.0.0 no cache de módulos (`github.com/wailsapp/wails/v3@v3.0.0`), não no guia de migração. Os caminhos de arquivo sem prefixo são desse módulo.

## O que mudou em cada camada

### Go (`internal/desktop`)

- `main.go` troca `wails.Run` por `application.New` e uma única `WebviewWindow` sem moldura (`Frameless`), com fundo translúcido (`BackgroundTypeTranslucent` e cor RGBA zerada), Mica (`WindowsWindow.BackdropType`) e tema do sistema (`WindowsWindow.Theme: SystemDefault`), 1280×768. O menu nativo (Arquivo › Exportar Logs e Sair, Ajuda › Sobre) vai para `app.Menu.Set`, e o "Sobre" usa `app.Dialog.Info()` preso à janela.
- `App` é o único serviço (`application.NewService`). `ServiceStartup(ctx, options)` e `ServiceShutdown()` substituem `OnStartup` e `OnShutdown`. O `ctx` do `ServiceStartup` vale até o encerramento (o Wails o cancela logo antes do shutdown, `pkg/application/services.go`) e continua sendo o contexto de todas as chamadas ao núcleo. Uma falha na inicialização fica em `startupErr` e o `ServiceStartup` devolve `nil`: um erro ali faria o `App.Run` desistir antes de abrir a janela.
- `App` guarda `*application.App` (eventos, diálogos, saída) e a janela principal (pai dos diálogos). `main` preenche os dois antes do `Run`.
- Eventos: `app.Event.Emit(nome, dado)`, pelo `wailsEvents` de `app.go`. O logger do desktop recebe uma função de emissão (`App.emitLog`) em vez de um contexto. Os nomes não mudaram: `request-cert-password`, `backend-log`, `notify-success`, `notify-error` e `app:before-close`.
- Diálogos: `app.Dialog.OpenFileWithOptions` e `SaveFileWithOptions`, presos à janela principal. `dialogPath` e `dialogPaths` (`dialogs.go`) devolvem resultado vazio quando o usuário fecha o diálogo, como no v2 (ver a tabela de achados).
- Confirmação ao fechar durante uma sincronização (`closeguard.go`): o v2 tinha um único `OnBeforeClose`. No v3 são dois caminhos, e os dois chamam `App.beforeClose`:
  - `Options.ShouldQuit`, para `Application.Quit` (o X da barra de título do app chama `desktopRuntime.quit()`), o Ctrl+Q e o menu "Sair". `App.Quit` chama `destroy` na thread da interface, que consulta `shouldQuit` (`pkg/application/application_windows.go`).
  - Um hook em `events.Common.WindowClosing` na janela, para o fechamento pelo sistema (Alt+F4, barra de tarefas). O `WM_CLOSE` vira `Windows.WindowClosing`, mapeado para `Common.WindowClosing`, e um hook que chama `event.Cancel()` impede o listener interno de destruir a janela (`pkg/application/webview_window.go`, `HandleWindowEvent`). Fechar a última janela encerra o app com `PostQuitMessage` sem passar pelo `ShouldQuit` (`unregisterWindow`), por isso o hook é necessário.
- Logs do próprio Wails continuam em `wails.log` na pasta de logs do desktop, por um `slog` em nível warn. O v3 registra ali, em nível de erro, a mensagem de todo erro devolvido por um método exposto, inclusive cancelamentos (`pkg/application/transport_http.go`, `httpError`).
- Tema da janela: `SetWindowTheme("dark" | "light" | "system")` liga ou desliga o modo escuro do DWM pelo pacote público `pkg/w32` (`theme_windows.go`), o que muda a tonalidade do Mica atrás da janela. Em outros sistemas não faz nada. São 60 métodos expostos: os 59 do v2 mais este.
- Versão do WebView2 em Configurações › Sobre e no `diagnostico.txt` dos logs exportados: vem de `app.Env.Info().PlatformInfo["WebView2"]`, que o v3 preenche com o mesmo `GetAvailableCoreWebView2BrowserVersionString` que o desktop chamava (`pkg/application/application_windows.go`, `platformEnvironment`). O v3 traz o go-webview2 embutido em `internal/webview2`, então a dependência direta `github.com/wailsapp/go-webview2` e os arquivos `webview2_*.go` saíram. Fora do Windows o campo fica vazio, como antes.

### Como o código de erro chega ao frontend

No v2, o `ErrorFormatter` virava o valor inteiro da rejeição da promise. No v3:

1. `Options.MarshalError` chama `marshalError` (`errors.go`), que serializa o mesmo `desktopapi.ErrorPayload{Code, Message}` de antes. `Code` é `canceled`, `sefaz_blocked`, `sync_running`, `startup_failed` ou vazio.
2. O Wails monta um `CallError{message, cause, kind: "RuntimeError"}`. `message` é o texto do erro Go e `cause` é o JSON do passo 1 (`pkg/application/bindings.go`). A resposta HTTP sai com status 422.
3. O `@wailsio/runtime` rejeita a promise com um `RuntimeError` (subclasse de `Error`) e copia `cause` (`internal/runtime/desktop/@wailsio/runtime/src/runtime.ts`).
4. `normalizeError` em `src/platform/wails/client/base.ts` lê `code` e `message` de `error.cause`. Objetos simples com `message` continuam aceitos, e qualquer outro erro vira código vazio.

### Frontend (`internal/desktop/frontend`)

- `wailsjs/` foi apagado. `bindings/` é gerado por `wails3 generate bindings -clean=true -ts -i` (interfaces TypeScript, chamadas por ID numérico) e versionado. `wails3 build` e `wails3 dev` regeneram a pasta; com as flags do build de produção o conteúdo é o mesmo.
- `@wailsio/runtime` fica fixo em `3.0.0`. O `pnpm-workspace.yaml` exige 7 dias de publicação (`minimumReleaseAge: 10080`) e essa versão saiu em 26/10/2026, então `minimumReleaseAgeExclude` abre exceção só para `@wailsio/runtime@3.0.0`. Remova a exceção depois de 02/11/2026.
- `client/*.ts` só trocou o caminho do import; os mapeadores de DTO não mudaram.
- `events.ts`: no v3, o callback recebe um `WailsEvent`, e o dado emitido pelo backend está em `.data`.
- `runtime.ts` usa `Window.Minimise`, `Window.ToggleMaximise`, `Application.Quit`, `Window.Size`, `Window.SetSize`, `Window.IsMaximised` e `Screens.GetAll`; o tema vai para `SetWindowTheme`. Os controles da janela e o tema continuam "dispare e esqueça", como no v2. `windowGetSize` ainda devolve `{w, h}` (o v3 devolve `{width, height}`), em pixels lógicos. `windowSetSize` agora devolve uma promise e `useDisplayScale` espera cada chamada: no v3 cada chamada do runtime é uma requisição HTTP separada, e o reajuste de 1 px depende da ordem.
- ESLint: `bindings/**` fica fora do lint, e importar `bindings/` ou `@wailsio/runtime` só é permitido nos três wrappers e nos testes deles.
- Vitest: `src/test/wailsRuntime.setup.ts` substitui `@wailsio/runtime` em todos os testes. Importar o runtime de verdade instala listeners globais de mouse e faz chamadas HTTP ao backend.
- `scripts/generate-screenshots.ts`: o mock antigo usava os globais `window.go` e `window.runtime` do v2. Agora ele intercepta os `POST /wails/runtime` do runtime, traduz o ID numérico do método para o nome lendo `bindings/.../app.ts`, responde `Window.Size`, `Window.IsMaximised` e `Screens.GetAll`, e dispara eventos por `window._wails.dispatchWailsEvent`. No navegador, o runtime 3.0.0 escreve no console um aviso "Browser Environment Detected"; é esperado.
- O plugin Vite do v3 (`@wailsio/runtime/plugins/vite`) não foi adicionado. Ele só serve para eventos tipados registrados com `application.RegisterEvent`, que o app não usa.
- O CSS de arrasto `--wails-draggable: drag` / `no-drag` da barra de título continua valendo no v3.

### Build

- `wails.json` deu lugar a `build/config.yml` e aos Taskfiles do template Vue da v3.0.0 (`Taskfile.yml`, `build/Taskfile.yml`, `build/windows/Taskfile.yml`), reduzidos a Windows e pnpm. Executável: `bin/nanci-desktop.exe`.
- `build/windows/Taskfile.yml` acrescenta `EXTRA_LDFLAGS` ao `-ldflags` do build de produção. É por ali que o release passa os `-X .../buildinfo.Version`, `Commit` e `Date`, que no v2 iam no `wails build -ldflags`.
- O ícone versionado `build/windows/icon.ico` tem precedência, como no v2: o template regeraria o `.ico` a partir do `appicon.png` a cada build, e aqui `generate:icons` só roda quando o arquivo falta.
- O script NSIS foi para `build/windows/nsis/project.nsi` e manteve as personalizações do v2: compressão zlib, chave de desinstalação `NanciDesktop`, atalho na área de trabalho e nenhuma associação de arquivo ou protocolo. O `wails_tools.nsh` da v3.0.0 já sabe instalar por usuário (`WAILS_INSTALL_SCOPE=user` grava a chave de desinstalação em `HKCU`), então o Taskfile usa `INSTALL_SCOPE=user` por padrão e a macro própria de desinstalação do v2 saiu.
- `info.json`, o manifesto, `wails_tools.nsh` e os plists de `darwin` são gerados a partir de `config.yml` por `wails3 task common:update:build-assets`. O comando também escreve `build/ios` e `build/linux`, que este app não guarda. A versão em `config.yml` é `0.0.0`; o release grava a da tag antes de gerar.
- `config.yml` usa `description: "Nanci Desktop"` porque é a `FileDescription` do executável, o nome que o Gerenciador de Tarefas mostra, como no v2.
- `package.json` ganhou o script `build:dev`, que o `wails3 build DEV=true` (e o `wails3 dev`) chama.
- O executável de produção tem 24,9 MB (24.876.544 bytes), contra cerca de 18 MB no v2.

### CI e release

- No Linux, o v3 só compila com Cgo e os cabeçalhos do GTK 4 e do WebKitGTK 6.0 (`pkg/application/linux_cgo.go`: `#cgo linux pkg-config: gtk4 webkitgtk-6.0`; a pilha GTK 3 só com `-tags gtk3`, até a v3.1). O v2 compilava sem Cgo. O job Go do CI instala `libgtk-4-dev` e `libwebkitgtk-6.0-dev` (nomes do `wails3 doctor` para apt, `internal/doctor/packagemanager/apt.go`) e liga o Cgo nos passos do desktop. O módulo raiz continua sem Cgo.
- O job de bindings instala o `wails3` v3.0.0 e roda o gerador com `GOOS=windows`. No Linux sem Cgo o pacote `application` não compila, e o gerador termina com "0 Services" e só um aviso, em vez de falhar. Conferido aqui com `GOOS=linux CGO_ENABLED=0`.
- O job `build-desktop` do release grava a versão da tag (só a parte numérica, para os metadados do Windows) em `build/config.yml`, roda `wails3 task common:update:build-assets` e `wails3 task windows:package EXTRA_LDFLAGS="-X ..."`, com `GOOS=windows` pelo mesmo motivo do gerador. O instalador sai em `internal/desktop/bin/nanci-desktop-amd64-installer.exe`. O `mingw-w64` e o `jq` saíram: o executável do Windows é Go puro.

## Achados do spike na v3.0.0

| Achado do spike (beta.27) | Na v3.0.0 | O que ficou no código |
| --- | --- | --- |
| No Windows, fechar um diálogo de arquivo devolve o erro `cancelled by user` (o v2 devolvia caminho vazio) | Continua. `cfd.ErrorCancelled` em `internal/go-common-file-dialog/cfd/errors.go`, devolvido por `showCfdDialog` (`pkg/application/dialogs_windows.go`); nenhum erro exportado para comparar | `dialogPath`, e `dialogPaths` para a seleção múltipla de XMLs, comparando o texto |
| `app.KeyBinding.Add` grava o atalho sem normalizar, e `CmdOrCtrl+Q` nunca casa | Continua. `KeyBindingManager.Add` guarda a string crua; só `Options.KeyBindings` passa por `parseAccelerator` (`processKeyBindingOptions`) | Ctrl+Q em `Options.KeyBindings` |
| O runtime JS não tem chamadas de tema, e `WindowsWindow.Theme` só vale na criação da janela | Continua. `window.ts` não tem método de tema; `webview_window_windows.go` aplica o tema ao criar e, com `SystemDefault`, volta a seguir `SystemThemeChanged` | `SetWindowTheme` via `pkg/w32` |
| O proxy do modo dev força IPv4 para `localhost`, e o Vite padrão escuta só em `::1` | Continua. `internal/assetserver/build_dev.go` disca `tcp4`; `internal/commands/dev.go` define `FRONTEND_DEVSERVER_URL=http://localhost:<porta>`. O template da v3.0.0 já fixa `host: "127.0.0.1"` no Vite | `server.host: '127.0.0.1'` em `vite.config.ts`; conferido que o Vite responde em `127.0.0.1:9245` e recusa `[::1]:9245` |
| O erro de um método exposto chega como `CallError{message, cause, kind}`, status 422, e o runtime copia `cause` | Continua (`bindings.go`, `transport_http.go`, `runtime.ts`) | `marshalError` e `normalizeError` lendo `cause` |
| O v3 registra em `wails.log`, em nível de erro, todo erro devolvido por um método | Continua (`httpError`) | Nada; cancelamentos aparecem no `wails.log` |
| O callback de evento recebe um `WailsEvent` com o dado em `.data` | Continua (`events.ts` do runtime) | `onWailsEvent` desembrulha `.data` |
| Importar o runtime instala listeners de mouse e chama o backend | Continua (`index.ts` importa `drag.js`, `contextmenu.js`, `appregion.js`) | Mock global no Vitest |
| A verificação de prontidão do `wails3 dev` podia nunca passar no Windows ([#6209](https://github.com/wailsapp/wails/issues/6209)) | Corrigido: issue fechada em 19/10/2026 pelo PR #6305, antes da v3.0.0 | Nada; o `wails3 dev` abriu o app aqui |
| O Windows não desenha barra de menu numa janela sem moldura | Continua. `webview_window_windows.go` só cria o menu com `!options.Frameless` (e, no v3, só com `UseApplicationMenu` ou `Windows.Menu`) | Menu mantido para os outros sistemas; Ctrl+Q por `KeyBindings`; Exportar Logs também está em Configurações |
| Detecção de borda do runtime JS para janela sem moldura (`window._wails.setResizable`) poderia brigar com as faixas `.app-resize-edge` | Sem conflito pela leitura do código: `drag.ts` decide a borda pelas coordenadas do mouse em listeners de `window` (fase de captura), com a largura `SM_CXSIZEFRAME` vinda do Go (`application_windows.go`, `GetFlags`), qualquer que seja o elemento sob o cursor | Faixas mantidas; falta o teste manual |
| Matar o app à força deixa `msedgewebview2.exe` órfão, e a próxima execução sai com código 1 | Não reavaliado: nenhuma parada forçada nesta rodada. Fechar a janela encerrou tudo com código 0 e sem WebView2 restante | Nota mantida abaixo |
| O executável de produção cresce para cerca de 24 MB | Continua: 24,9 MB | Nada |

Achados novos desta rodada:

- O v3 no Linux exige Cgo e GTK 4 / WebKitGTK 6.0 (ver CI acima).
- O gerador de bindings não falha quando o pacote `application` não compila: termina com "0 Services" e um aviso.
- `wails3 update build-assets` escreve `build/ios` e `build/linux` além dos arquivos do Windows.
- O manifesto gerado pela v3.0.0 usa a versão com três partes (`version="0.0.0"`), onde o do v2 acrescentava `.0`. O executável abre normalmente com ela.

## Ciclo de desenvolvimento (`wails3 dev`)

```bash
cd internal/desktop
wails3 dev
```

O `wails3 dev` lê `build/config.yml`: compila o app com `wails3 build DEV=true` (o que regenera os bindings), sobe o Vite com `pnpm dev --port 9245 --strictPort` e abre o executável. Fechar a janela encerra tudo. Conferido em 29/10/2026 no Windows 10: a janela abriu em cerca de 40 s (com a compilação), o Vite respondeu em `127.0.0.1:9245`, e ao fechar a janela o `wails3 dev` saiu com código 0 sem deixar o Vite rodando.

Se o `wails3 dev` não abrir, rode as partes separadamente:

```bash
# terminal 1
cd internal/desktop/frontend
pnpm dev --port 9245 --strictPort

# terminal 2
cd internal/desktop
wails3 build DEV=true
FRONTEND_DEVSERVER_URL=http://127.0.0.1:9245 ./bin/nanci-desktop.exe
```

O app em modo dev lê `FRONTEND_DEVSERVER_URL`. Nesse modo não há recompilação automática do Go.

Para testar sem tocar nos dados reais, defina `NANCI_DATA_DIR` com uma pasta vazia antes de abrir o app.

### Processos do WebView2 órfãos

Matar o app à força pode deixar processos `msedgewebview2.exe` vivos, presos ao perfil `%APPDATA%\nanci-desktop.exe\EBWebView`. Enquanto eles existem, uma nova execução sai na hora com código 1, sem janela e sem nada no log (visto no spike com a beta.27). Fechar a janela normalmente encerra tudo com código 0. Se o app não abrir depois de uma parada forçada, procure `msedgewebview2.exe` com `nanci-desktop` na linha de comando e encerre esses processos.

## Build e instalador

- `wails3 build` gera `bin/nanci-desktop.exe`. Aberto com uma pasta de dados vazia, o executável criou o banco, mostrou a tela de boas-vindas (a lista de empresas veio do backend) e saiu com código 0 quando a janela recebeu `WM_CLOSE`. `wails.log` e `nanci-desktop.log` ficaram vazios.
- `wails3 task windows:build EXTRA_LDFLAGS="-X ..."` passa os `-X` ao `go build` (conferido na linha de comando impressa pelo Task e nas strings do executável).
- `wails3 task windows:package` gera o bootstrapper do WebView2 e chega ao `makensis` com `-DWAILS_INSTALL_SCOPE=user -DREQUEST_EXECUTION_LEVEL=user`. Parou ali porque o NSIS não está instalado nesta máquina. O instalador sairia em `bin/nanci-desktop-amd64-installer.exe`.
- O passo de build do release foi ensaiado localmente com a versão `0.0.0-dev` até o `makensis`. Com a versão `0.7.0` em `config.yml`, `update:build-assets` gravou `0.7.0` em `info.json`, no manifesto e no `INFO_PRODUCTVERSION` do `wails_tools.nsh`. O instalador em si não foi gerado: o job do release só roda em tags ou manualmente.

## Verificado automaticamente (29/10/2026)

| Comando | Resultado |
| --- | --- |
| `go build ./... && go vet ./... && go test ./...` na raiz | ok |
| `go build ./... && go vet ./... && go test ./...` em `internal/desktop` | ok |
| `golangci-lint run ./...` na raiz e em `internal/desktop` | 0 issues |
| `GOWORK=off go mod tidy` em `internal/desktop` | sem diferença |
| `govulncheck ./...` em `internal/desktop` | os mesmos achados da raiz (stdlib do Go 1.26.9 local e excelize); nada do Wails |
| `wails3 generate bindings -clean=true -ts -i`, duas vezes, e com as flags de produção | sem diferença |
| `pnpm run lint:check` | ok |
| `pnpm run test:unit` | 98 arquivos, 731 testes ok |
| `pnpm run build` (vue-tsc + Vite) | ok |
| `pnpm run screenshots documents` | ok; as imagens só diferem por anti-aliasing |
| `wails3 build` | ok |
| `bin/nanci-desktop.exe` | abriu; código 0 ao fechar a janela |
| `wails3 dev` | abriu o app com o Vite em `127.0.0.1`; código 0 ao fechar |
| `wails3 task windows:package` | parou no `makensis` ausente |
| `actionlint` em `ci.yml` e `release.yml` | ok |

Os jobs do CI no Linux (GTK 4 e WebKitGTK 6.0 com Cgo, gerador com `GOOS=windows`) não foram rodados nesta máquina.

## Checklist manual (dono)

- [ ] CI verde no push: jobs Go (com os pacotes GTK), bindings e frontend.
- [ ] Release manual (`workflow_dispatch`) gera o instalador; instalar numa máquina limpa: por usuário sem UAC, atalhos no menu Iniciar e na área de trabalho, versão certa em Propriedades › Detalhes do executável e em Configurações › Sobre; desinstalar remove a chave `NanciDesktop` em `HKCU`.
- [ ] `wails3 dev`: recarga do Vue ao salvar em `frontend/src` e recompilação ao salvar um `.go`.
- [ ] Os 60 métodos expostos respondem (passar pelas telas exercita a maioria).
- [ ] Os eventos chegam: `request-cert-password` (diálogo de senha), `backend-log` (console), `notify-success` e `notify-error` (Exportar Logs, onde houver menu), `app:before-close`.
- [ ] Códigos de erro: cancelar a senha do certificado mostra o tratamento de `canceled`; uma origem bloqueada mostra o de `sefaz_blocked`; dois pulls seguidos mostram o de `sync_running`.
- [ ] Fechar durante uma sincronização pelos três caminhos (X do app, Ctrl+Q, Alt+F4 ou barra de tarefas): a confirmação aparece; "Sair" fecha, "Continuar no Nanci" mantém a janela.
- [ ] Mica e transparência atrás da janela (Windows 11); tema do sistema na abertura; trocar o tema em Configurações muda a tonalidade do Mica; trocar o tema do Windows com o app aberto.
- [ ] Diálogos de arquivo: certificado, importar XMLs (seleção múltipla), salvar exportação e exportar logs. Cancelar cada um não mostra erro.
- [ ] Configurações › Sobre mostra a versão do WebView2, a escala e o número de monitores; "Reajustar janela" funciona; mudar a escala do monitor com o app aberto reajusta a janela.
- [ ] Redimensionar pelas duas laterais, arrastar pela barra de título, duplo clique maximiza e restaura, Windows Snap (Win+setas e arrastar para a borda).
- [ ] Smoke padrão: empresa, credencial, sync, listas de NFS-e/NF-e/CT-e, exportação, consulta direta, senha do certificado e console de logs.

## Faixas de redimensionamento

Mantenha as faixas `.app-resize-edge--left` e `.app-resize-edge--right` de `MainLayout.vue`. Elas evitam que um clique na borda caia numa barra de rolagem ou num link da gaveta, o que faria o WebView capturar o mouse e o redimensionamento falhar. No v3, o runtime JS decide a borda pela posição do mouse, independente do elemento sob o cursor, então as faixas não atrapalham a detecção (leitura de `drag.ts`; falta conferir na mão).

Experimento opcional: `WindowsWindow.NonClientRegionSupport: true` liga o suporte nativo do WebView2 a regiões não cliente, e a barra de título passaria a usar `--wails-non-client-region: caption` (com `minimize`, `maximize` e `close` nos botões). Isso poderia trazer o Snap Layouts ao botão de maximizar e dispensar as faixas, mas depende da versão do WebView2 Runtime instalada.

## Fontes (conferidas em 29/10/2026)

- Release v3.0.0 (26/10/2026; GTK 4 e WebKitGTK 6.0 por padrão no Linux, GTK 3 com `-tags gtk3` até a v3.1; Go 1.26 ou mais novo): https://github.com/wailsapp/wails/releases/tag/v3.0.0
- Publicação de `@wailsio/runtime@3.0.0` no npm: 2026-10-26T11:01:20Z (`npm view @wailsio/runtime time`).
- `minimumReleaseAgeExclude` aceita versões exatas (`nome@versão`): https://pnpm.io/settings/dependency-resolution#minimumreleaseageexclude
- Issue #6209 (prontidão do `wails3 dev` contra o Vite), fechada em 19/10/2026 pelo PR #6305: https://github.com/wailsapp/wails/issues/6209
- Código-fonte da v3.0.0: `go mod download github.com/wailsapp/wails/v3@v3.0.0`, arquivos citados acima.
