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
- `App` é o único serviço (`application.NewService`). `ServiceStartup(ctx, options)` e `ServiceShutdown()` substituem `OnStartup` e `OnShutdown`. O `ctx` do `ServiceStartup` vale até o encerramento (o Wails o cancela logo antes do shutdown, `pkg/application/services.go`) e limita todas as chamadas ao núcleo; os métodos longos também param com a própria chamada (ver [Cancelamento por chamada](#cancelamento-por-chamada)). O Wails chama o `ServiceShutdown` logo depois de cancelar esse `ctx`, e uma sincronização que está parando ainda grava como a execução terminou e a espera imposta pela SEFAZ. Por isso o `ServiceShutdown` espera as operações contadas pelo guarda de `closeguard.go` (ver [Atualização automática](#atualização-automática)) terminarem, por até 10 s, antes de fechar o banco. Uma falha na inicialização fica em `startupErr` e o `ServiceStartup` devolve `nil`: um erro ali faria o `App.Run` desistir antes de abrir a janela.
- `App` guarda `*application.App` (eventos, diálogos, saída) e a janela principal (pai dos diálogos). `main` preenche os dois antes do `Run`.
- Eventos: `app.Event.Emit(nome, dado)`, pelo `wailsEvents` de `app.go`. O logger do desktop recebe uma função de emissão (`App.emitLog`) em vez de um contexto. Os nomes não mudaram: `request-cert-password`, `backend-log`, `notify-success`, `notify-error` e `app:before-close`. `events.go` registra os cinco com `application.RegisterEvent[T]` e constantes para os nomes (o gerador só enxerga argumentos constantes em `init`). O Wails cancela um emit cujo dado não é exatamente o tipo registrado (ponteiro não casa com valor, `int32` não casa com `int`) e passa o erro ao tratador de erros do app, sem avisar quem emitiu (`pkg/application/events.go`, `validateCustomEvent`; `event_manager.go`, `Emit`); `events_test.go` passa o dado de cada emissor pela mesma checagem.
- Diálogos: `app.Dialog.OpenFileWithOptions` e `SaveFileWithOptions`, presos à janela principal. `dialogPath` e `dialogPaths` (`dialogs.go`) devolvem resultado vazio quando o usuário fecha o diálogo, como no v2 (ver a tabela de achados).
- Confirmação ao fechar durante uma sincronização (`closeguard.go`): o v2 tinha um único `OnBeforeClose`. No v3 são dois caminhos, e os dois chamam `App.beforeClose`:
  - `Options.ShouldQuit`, para `Application.Quit` (o X da barra de título do app chama `desktopRuntime.quit()`), o Ctrl+Q e o menu "Sair". `App.Quit` chama `destroy` na thread da interface, que consulta `shouldQuit` (`pkg/application/application_windows.go`).
  - Um hook em `events.Common.WindowClosing` na janela, para o fechamento pelo sistema (Alt+F4, barra de tarefas). O `WM_CLOSE` vira `Windows.WindowClosing`, mapeado para `Common.WindowClosing`, e um hook que chama `event.Cancel()` impede o listener interno de destruir a janela (`pkg/application/webview_window.go`, `HandleWindowEvent`). Fechar a última janela encerra o app com `PostQuitMessage` sem passar pelo `ShouldQuit` (`unregisterWindow`), por isso o hook é necessário.
- Logs do próprio Wails continuam em `wails.log` na pasta de logs do desktop, por um `slog` em nível warn. O v3 registra ali, em nível de erro, a mensagem de todo erro devolvido por um método exposto, inclusive cancelamentos (`pkg/application/transport_http.go`, `httpError`).
- Tema da janela: `SetWindowTheme("dark" | "light" | "system")` liga ou desliga o modo escuro do DWM pelo pacote público `pkg/w32` (`theme_windows.go`), o que muda a tonalidade do Mica atrás da janela. Em outros sistemas não faz nada. São 60 métodos expostos: os 59 do v2 mais este.
- Versão do WebView2 em Configurações › Sobre e no `diagnostico.txt` dos logs exportados: vem de `app.Env.Info().PlatformInfo["WebView2"]`, que o v3 preenche com o mesmo `GetAvailableCoreWebView2BrowserVersionString` que o desktop chamava (`pkg/application/application_windows.go`, `platformEnvironment`). O v3 traz o go-webview2 embutido em `internal/webview2`, então a dependência direta `github.com/wailsapp/go-webview2` e os arquivos `webview2_*.go` saíram. Fora do Windows o campo fica vazio, como antes.

### Como o código de erro chega ao frontend

No v2, o `ErrorFormatter` virava o valor inteiro da rejeição da promise. No v3:

1. `Options.MarshalError` chama `marshalError` (`errors.go`), que serializa o mesmo `desktopapi.ErrorPayload{Code, Message}` de antes. `Code` é `canceled`, `sefaz_blocked`, `sync_running`, `startup_failed`, `update_unavailable`, `update_installing` ou vazio.
2. O Wails monta um `CallError{message, cause, kind: "RuntimeError"}`. `message` é o texto do erro Go e `cause` é o JSON do passo 1 (`pkg/application/bindings.go`). A resposta HTTP sai com status 422.
3. O `@wailsio/runtime` rejeita a promise com um `RuntimeError` (subclasse de `Error`) e copia `cause` (`internal/runtime/desktop/@wailsio/runtime/src/runtime.ts`).
4. `normalizeError` em `src/platform/wails/client/base.ts` lê `code` e `message` de `error.cause`. Objetos simples com `message` continuam aceitos, e qualquer outro erro vira código vazio.

### Frontend (`internal/desktop/frontend`)

- `wailsjs/` foi apagado. `bindings/` é gerado por `wails3 generate bindings -clean=true -ts -i` (interfaces TypeScript, chamadas por ID numérico) e versionado. `wails3 build` e `wails3 dev` regeneram a pasta; com as flags do build de produção o conteúdo é o mesmo.
- `@wailsio/runtime` fica fixo em `3.0.0`. O `pnpm-workspace.yaml` exige 7 dias de publicação (`minimumReleaseAge: 10080`) e essa versão saiu em 26/10/2026, então `minimumReleaseAgeExclude` abre exceção só para `@wailsio/runtime@3.0.0`. Remova a exceção depois de 02/11/2026.
- Os mapeadores de `client/*.ts` recebem o modelo gerado em `bindings/.../desktopapi/models.ts`, então renomear ou remover um campo de um DTO em Go quebra o `vue-tsc` no mapeador (conferido renomeando `UF` e `LastSyncAt` de `CompanySummary`). Os helpers `as*` de `base.ts` ficaram só para os campos que o tipo gerado admite nulos (ponteiros e slices) e para restringir um texto a uma união de valores conhecidos (`asEnum`); os testes que alimentam um mapeador com dados parciais fazem o cast explícito, e os testes das telas montam linhas com `makeNFeRow`, `makeCTeRow` e `makeDocumentRow` de `src/test/fixtures.ts`.
- `events.ts`: no v3, o callback recebe um `WailsEvent`, e o dado emitido pelo backend está em `.data`. O gerador escreve o tipo do dado de cada evento registrado em `bindings/github.com/wailsapp/wails/v3/internal/eventdata.d.ts`, que aumenta `Events.CustomEvents` do `@wailsio/runtime` e entra no `vue-tsc` pelo `include` de `bindings/**/*.ts`. `onWailsEvent('nome', cb)` entrega o dado tipado; `DesktopEvents` troca o tipo gerado pelo de `@/types/desktop` quando ele existe (`CertPasswordRequest`) e quebra o build se o tipo gerado deixar de caber nele.
- `runtime.ts` usa `Window.Minimise`, `Window.ToggleMaximise`, `Application.Quit`, `Window.Size`, `Window.SetSize`, `Window.IsMaximised` e `Screens.GetAll`; o tema vai para `SetWindowTheme`. Os controles da janela e o tema continuam "dispare e esqueça", como no v2. `windowGetSize` ainda devolve `{w, h}` (o v3 devolve `{width, height}`), em pixels lógicos. `windowSetSize` agora devolve uma promise e `useDisplayScale` espera cada chamada: no v3 cada chamada do runtime é uma requisição HTTP separada, e o reajuste de 1 px depende da ordem.
- ESLint: `bindings/**` fica fora do lint, e importar `bindings/` ou `@wailsio/runtime` só é permitido nos três wrappers e nos testes deles.
- Vitest: `src/test/wailsRuntime.setup.ts` substitui `@wailsio/runtime` em todos os testes. Importar o runtime de verdade instala listeners globais de mouse e faz chamadas HTTP ao backend.
- `scripts/generate-screenshots.ts`: o mock antigo usava os globais `window.go` e `window.runtime` do v2. Agora ele intercepta os `POST /wails/runtime` do runtime, traduz o ID numérico do método para o nome lendo `bindings/.../app.ts`, responde `Window.Size`, `Window.IsMaximised` e `Screens.GetAll`, e dispara eventos por `window._wails.dispatchWailsEvent`. No navegador, o runtime 3.0.0 escreve no console um aviso "Browser Environment Detected"; é esperado.
- O plugin Vite do v3 (`@wailsio/runtime/plugins/vite`) não foi adicionado. Ele injeta no runtime o `eventcreate.ts` gerado, que constrói os dados dos eventos quando os bindings são classes; com interfaces (`-i`) esse arquivo não tem construtores, e os tipos vêm só do `eventdata.d.ts`.
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

## Tema fixo, escala do monitor e instância única

Conferido no código da v3.0.0 em 30/10/2026.

- **Tema fixo contra o tema do Windows.** Com `WindowsWindow.Theme: SystemDefault`, o Wails registra um listener de `events.Windows.SystemThemeChanged` que põe a janela no tema novo do sistema (`pkg/application/webview_window_windows.go:650-654`), desfazendo o claro ou escuro escolhido em Configurações. O `App` guarda o último modo pedido por `SetWindowTheme` (`theme.go`), e `main.go` registra um hook no mesmo evento (`App.keepFixedTheme`) que cancela o evento quando o modo é `dark` ou `light`. Reaplicar o tema depois do Wails não teria ordem garantida: `EventManager.handleApplicationEvent` (`pkg/application/event_manager.go`) roda os hooks em sequência e depois cada listener na sua própria goroutine, então a ordem das chamadas a `InvokeAsync` dos listeners não é definida. Um evento cancelado num hook não chega a nenhum listener. O cancelamento também cala o repasse para `Common.ThemeChanged` (`events_common_windows.go`), que o app não escuta. Com `system`, o hook não faz nada e o Wails segue o Windows.
- **Escala do monitor.** Na hospedagem em janela, o WebView2 acompanha sozinho a escala do monitor (`ShouldDetectMonitorScaleChanges` ligado), e o Wails trata `WM_DPICHANGED`. O empurrão automático de 1 px do `useDisplayScale` saiu; o botão **Reajustar janela** fica uma versão como diagnóstico. Detalhes e linhas do código em `docs/pesquisa/2026-10-17-webview2-dpi.md`.
- **Instância única.** `Options.SingleInstance` (`pkg/application/single_instance.go` e `single_instance_windows.go`) cria um mutex nomeado na sessão do Windows. Uma segunda execução manda os argumentos à primeira por `WM_COPYDATA` e sai com `os.Exit` dentro de `application.New` ([#6052](https://github.com/wailsapp/wails/issues/6052), ainda aberta em 30/10/2026), sem rodar os `defer` do `main`. A primeira recebe o aviso em `OnSecondInstanceLaunch`, numa goroutine, e `focusMainWindow` restaura a janela se estiver minimizada, mostra e foca, pela thread da interface (`InvokeAsync`).
  - O `UniqueID` vem do build: `com.wails.nanci-desktop` com `-tags production` (`instance_production.go`; o `wails3 build` e o release usam essa tag, `build/windows/Taskfile.yml`) e `com.wails.nanci-desktop.dev` no resto (`instance_dev.go`), para o `wails3 dev` abrir com o app instalado aberto.
  - O `wails.log` é aberto antes do `application.New`, então a segunda execução também o abre. Isso não estraga o log da primeira: `newRotatingFileWriter` abre com `O_APPEND` e não trunca nem rotaciona, e a rotação só acontece numa escrita. A segunda execução só escreve ali se não conseguir avisar a primeira; as outras mensagens dela no início são de nível info, abaixo do warn do logger. Mesmo nesse caso, o Windows não deixa renomear o `wails.log` aberto pela primeira (o Go abre arquivos sem `FILE_SHARE_DELETE`); só os backups `.1` a `.3` andariam uma posição.
  - Conferido com `wails3 build` e uma pasta de dados vazia: com a primeira janela minimizada, a segunda execução saiu em 41 ms com código 0, e a primeira voltou à frente, restaurada. Um executável de `wails3 build DEV=true` abriu ao lado do de produção, e uma segunda execução dele foi para a instância dev. Fechar as janelas com `WM_CLOSE` encerrou com código 0, sem `msedgewebview2.exe` restante, e os logs ficaram vazios.

## Cancelamento por chamada

Conferido no código da v3.0.0 em 30/10/2026.

- **O contexto da chamada.** Um método exposto cujo primeiro parâmetro é `context.Context` recebe um contexto próprio a cada chamada (`pkg/application/bindings.go`, `needsContext`). O gerador omite esse parâmetro, então a assinatura em TypeScript e o ID do método não mudam; os bindings regenerados saíram sem diferença. `processCallMethod` (`pkg/application/messageprocessor_call.go`) cria o contexto com `context.WithCancel(context.WithoutCancel(...))`: ele termina quando o frontend cancela a promise (`CancellablePromise.cancel()` ou `cancelOn(signal)`, que chamam o `CancelCall` do runtime), quando a janela fecha (`CancelWindowCalls`, chamado em `webview_window.go`) ou quando o método retorna. O encerramento do app não o cancela.
- **Quais métodos.** `Pull`, `PullNFe`, `PullCTe`, `QueryNFSeEvents`, `FetchNFeByChave`, `TestConnection`, `TestCTeConnection`, `ImportNFeXML`, `ImportCTeXML` e as nove exportações. `App.callContext` (`app.go`) passa ao núcleo um contexto que termina com a chamada ou com o `ctx` do `ServiceStartup`, o que vier primeiro, e mantém os valores da chamada; sem isso, fechar o app não pararia uma sincronização. Ciência, manifestação e as redefinições continuam só com o contexto do app: cancelar o envio de um evento no meio poderia deixá-lo registrado na SEFAZ e não no banco.
- **O erro.** `formatError` dá o código `canceled` a `context.Canceled`, como a `ErrOperationCanceled` do diálogo de senha. No JS, o runtime rejeita a promise na hora com um `CancelError` (`name` igual a `"CancelError"`, `cause` com o motivo do abort; `cancellable.ts`), enquanto o Go segue até checar o contexto. `normalizeError` (`client/base.ts`) converte esse erro em `WailsClientError` com o código `canceled`. Quando o método Go termina depois, a rejeição dele chega a uma promise já cancelada, e o runtime a reporta como rejeição não tratada (`CancelledRejectionError`) no console do WebView. É esperado; o usuário não vê nada.
- **O que a interface oferece.** `callWails` recebe um `AbortSignal` opcional e liga a chamada a ele com `cancelOn`, então stores e composables não dependem dos tipos do runtime. O store `companySync` guarda o `AbortController` de cada sincronização em andamento (`runCancellableSync`, por empresa e origem), e o store `query` o da consulta direta; os dois sobrevivem à troca de tela. O cabeçalho das páginas de NFS-e, NF-e e CT-e mostra **Cancelar** ao lado de uma sincronização em andamento, e a Consulta Direta mostra **Cancelar** enquanto consulta. O cancelamento aparece como aviso, não como erro. Importação de XML, busca por chave, testes de conexão e exportações não têm botão: param ao fechar a janela.
- **Onde cada operação para.**
  - Sincronização: o laço checa o contexto antes de cada lote e na espera entre lotes, e a requisição HTTP em andamento é abortada. Cada item é gravado na mesma transação que o seu NSU (`ApplyWithProgress`), então o cursor fica no último item gravado e a sincronização seguinte continua dali (`TestSyncServiceResumesAfterCancelMidBatch` em `internal/sync/loop_test.go`). A execução termina como `interrupted` com `context_canceled`, gravada com um contexto próprio de 5 s (`finishRun`), inclusive quando o cancelamento cai no meio da gravação de um item. O NSU de um lote vazio, ou o `ultNSU` além do último item da NF-e, só é gravado depois do lote; um cancelamento nesse intervalo faz a próxima sincronização repetir uma consulta. A espera que a SEFAZ impõe numa resposta (`WaitUntil`, como a hora depois de `ultNSU` igual a `maxNSU`) é gravada antes dos itens e fora do contexto da chamada, então um cancelamento no meio do lote não a perde (`TestSourceLoopCancelMidBatchStillRecordsWait`). O bloqueio do cStat 656 na busca por chave segue a mesma regra. O que continua possível: cancelar com a requisição em andamento descarta a resposta, e a espera que ela traria não é gravada. A requisição não fica imune ao cancelamento de propósito, porque o cliente HTTP repete tentativas que falham e continuaria consultando a SEFAZ depois do cancelamento.
  - Consulta direta, busca por chave e testes de conexão: a requisição ou a conexão TLS em andamento é abortada.
  - Importação de XML: para antes do próximo arquivo; os já gravados ficam.
  - Exportações: não checam o contexto entre documentos. O gerador do arquivo (CSV, XLSX, ZIP, PDFs) vai até o fim; só as leituras do banco antes dele e a marcação de exportados depois veem o contexto. Fechar a janela no meio pode deixar o arquivo gravado sem a marcação, e a exportação incremental o repete.
- **Senha do certificado.** Se o contexto termina com o diálogo de senha aberto, `GetCertPassword` devolve `context.Canceled`, e a chamada termina com o código `canceled`. O diálogo é modal, então isso só acontece ao fechar a janela.
- **Depois do cancelamento.** Até o Go checar o contexto, a origem continua reservada e o guarda de fechamento (`closeguard.go`) continua contando a sincronização. Uma nova sincronização nesse intervalo recebe `sync_running`.

## Atualização automática

Conferido no código da v3.0.0 em 30/10/2026. Os caminhos sem prefixo são de `pkg/updater`.

- **Onde procura.** `CheckForUpdate` (`internal/desktop/updater.go`) chama `app.Updater.Check` com o provider `endpoint` e a URL `https://github.com/vasfvitor/nanci/releases/latest/download/nanci-update.json`. O GitHub redireciona `releases/latest/download/<asset>` para o asset do release mais recente que não é rascunho nem pré-release, então só versões estáveis contam. O provider acrescenta `platform`, `arch` e `version` à URL, e o redirecionamento ignora esses parâmetros. Um release sem o manifesto, como a v0.5.1, responde 404, e o provider lê isso como "sem atualização" (`providers/endpoint/endpoint.go`, `Check`). Os dois comportamentos foram conferidos com `curl` em 30/10/2026.
- **O que o usuário vê.** O `MainLayout` faz uma checagem por sessão, em segundo plano (store `updates`, `checkOnce`). Se houver versão nova, aparece o aviso "Nova versão X disponível.", com **Atualizar e reiniciar** e **Depois**, que fica na tela até o usuário escolher. Em Configurações › Sobre, a linha "Atualizações" mostra o estado (procurando, disponível, baixando com a porcentagem, instalando, erro, versão mais recente) e o botão **Procurar atualizações**. Quando há versão nova, o botão vira **Atualizar e reiniciar**. Nenhuma janela de atualização do Wails é aberta: `Check` e `DownloadAndInstall` não abrem janela, só `CheckAndInstall` abre (`updater.go:361-405`).
- **Download e verificação.** `InstallUpdate` chama `DownloadAndInstall` (`updater.go:278-359`). O exe é baixado para uma pasta `wails-update-*` ao lado do exe instalado, para que a troca seja uma renomeação no mesmo volume (`download.go:273-279`), e o SHA-512 é calculado durante o download. Depois o digest e a assinatura são conferidos (`verify.go:80-119`). Com `Config.PublicKey` definido, a assinatura é obrigatória: um release sem assinatura é recusado (`verify.go:87`), e a única chave aceita é a fixada no build, nunca uma indicada pelo manifesto.
- **Troca e reinício.** `Restart` (`updater.go:421-492`) inicia uma cópia do próprio exe em modo auxiliar (variáveis `WAILS_UPDATER_HELPER*`, `helper.go:28-35`) e espera ela avisar que está pronta. Em seguida chama `App.Quit`. O auxiliar espera o app terminar por até 30 s (`helper.go:152`) e faz um backup `.bak`. Depois move o exe em uso para `.old.<n>` e põe o novo no lugar: o Windows deixa renomear um executável ainda mapeado, mas não apagá-lo (`helper_windows.go:132-163`). Por fim valida o resultado e inicia a versão nova. Qualquer falha devolve o original e o inicia (os códigos de saída estão em `helper.go:110-128`). O `.old.<n>` fica na pasta até a próxima atualização, que o apaga (`sweepRenameAsides`). O auxiliar grava o log em `%TEMP%\wails-update-<pid do app>.log`.
- **Instância única.** `application.New` chama `updater.HandleHelperMode()` antes de tudo (`pkg/application/application.go:47`), e a instância única só é checada depois (`:204`). O auxiliar, portanto, faz a troca e sai com `os.Exit` sem passar pelo mutex. Antes do `application.New`, `main` só abre o `wails.log` para acrescentar, o mesmo caso da segunda execução descrito em [Tema fixo, escala do monitor e instância única](#tema-fixo-escala-do-monitor-e-instância-única). O auxiliar só inicia a versão nova depois que o processo antigo terminou (`helper.go:152-167`), e nesse ponto o Windows já fechou o mutex nomeado. Se o app antigo não sair em 30 s, o auxiliar não troca nada e reabre o original (código 17). Esse original encontra a instância antiga ainda aberta e só a traz para a frente.
- **Operações em andamento.** `App.Quit` passa pelo `ShouldQuit` e pelo `beforeClose`. Com uma sincronização em andamento, o `beforeClose` seguraria o fechamento para perguntar ao usuário, e o auxiliar desistiria depois de 30 s; uma exportação, uma consulta, uma redefinição ou um evento enviado à SEFAZ seria cortado no meio pelo reinício. Por isso o guarda de `closeguard.go` conta, além das sincronizações (importações de XML e busca por chave incluídas), todos os métodos longos: as nove exportações, `QueryNFSeEvents`, os dois testes de conexão, `PlanNFeCiencia`, `RegisterNFeCiencia`, `RegisterNFeManifestacao`, `ResetSyncState`, `ResetNFe`, `PreviewResetCTe` e `ResetCTe`. `InstallUpdate` primeiro marca a instalação e só então confere o contador, sob a mesma trava em que cada operação confere a marca e se conta, então uma operação e uma atualização nunca começam juntas (`TestInstallUpdateAndOperationNeverBothStart`). Com uma sincronização em andamento, a recusa vem com `sync_running`; com outra operação, com a mensagem "aguarde a operação em andamento terminar para atualizar", que Sobre mostra como erro, com o botão ainda disponível. Da marca até o reinício, toda operação contada é recusada com `update_installing` ("o Nanci está instalando uma atualização e vai reiniciar"), e nada mais pode segurar o `ShouldQuit`. Uma falha no download, na verificação ou no reinício tira a marca. O frontend também desliga a ação enquanto houver qualquer sincronização (`companySync.anySyncing`) e mostra o motivo em Sobre.
- **Quando fica desligada.** O updater não é configurado (`setupUpdater` devolve `errUpdatesDisabled`) quando falta a chave em `updater_key.go` ou quando a versão não é `X.Y.Z`, o que inclui `0.0.0-dev`. Nesse caso `CheckForUpdate` devolve `enabled: false` sem fazer requisição, Sobre mostra "indisponíveis nesta versão" e esconde o botão, e `InstallUpdate` responde com o código `update_unavailable`.
- **Programas e Recursos.** O instalador grava `DisplayVersion` uma única vez em `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\NanciDesktop` (`build/windows/nsis/wails_tools.nsh`, macro `wails.writeUninstaller`). A atualização troca só o exe, então um build de release grava a própria versão ali ao abrir (`internal/desktop/uninstall.go`). Isso só acontece se a chave existir e se o `DisplayIcon` dela for o exe em execução, de modo que uma cópia avulsa em outra pasta não mexe na entrada.
- **Pasta de instalação.** O instalador é por usuário e instala em `%LOCALAPPDATA%\Programs\Nanci Desktop` (`project.nsi`, `WAILS_INSTALL_SCOPE=user`), uma pasta que o usuário pode gravar sem UAC. Numa instalação por máquina, em Program Files, o updater não conseguiria criar a pasta de staging, e o download falharia com erro.
- **Por que não o provider `github`.** O provider `github` só preenche um digest, lido de um arquivo de checksums, e nunca uma assinatura (`providers/github/github.go:167-179`). Com a chave fixada, o updater recusaria todos os releases (`verify.go:87`). As ferramentas de publicação do Wails (`wails3 updater genkey`, `sign`, `manifest` e `verify`, em `internal/commands/updater_tool.go`) seguem o protocolo do provider `endpoint`, e o manifesto continua hospedado no próprio release do GitHub.
- **O que é assinado.** `wails3 updater manifest -key` calcula o SHA-512 do exe e assina esse digest com Ed25519ph (`updater_tool.go`, `signUpdaterArtifact`). O manifesto leva o digest e a assinatura em base64 (`digestAlgo: sha512`, `signatureAlgo: ed25519ph`) e o tamanho do arquivo. O artefato é o exe cru, não um zip: assim o digest cobre exatamente o arquivo instalado. Um zip seria extraído e apagado antes da troca (`extract.go:66-123`), e o digest deixaria de cobrir o que vai para o disco (`validate.go:18-31`).
- **Artefatos do release.** Além do instalador, `release.yml` publica `nanci-desktop-windows-amd64-vX.Y.Z.exe` (o exe que o instalador empacota) e `nanci-update.json`, cuja URL do exe aponta para a tag. `nanci-checksums.txt` cobre os dois exes. Uma tag para no job `prepare` quando `updater_key.go` não tem chave ou quando falta o segredo `UPDATER_SIGNING_KEY`. Depois de assinar, `wails3 updater verify` confere o manifesto com a chave fixada no app, então um segredo que não corresponde a ela também derruba o release. O `wails3` desses passos é o v3.0.0, do mesmo cache do build.
- **A primeira versão.** As versões até a v0.6.0 não têm o updater. Quem usa uma delas instala a primeira versão com a chave pelo instalador; daí em diante, as atualizações chegam pelo app.

### Passos do dono (uma vez)

1. Gerar o par de chaves fora do repositório:

   ```
   wails3 updater genkey -out nanci-updater.key
   ```

   O comando grava `nanci-updater.key` (chave privada Ed25519 em PEM PKCS#8) e `nanci-updater.key.pub`. Ele também imprime a chave pública em base64 dos 32 bytes da chave crua, abaixo de "Or inline as base64 of the raw key": são 44 caracteres terminados em `=`.
2. Colar essa linha em `internal/desktop/updater_key.go`, como `var updaterPublicKey = "<base64>"`, e fazer o commit. Em `internal/desktop`, `go test ./...` roda `TestCommittedUpdaterKeyIsValid`, que confere o formato.
3. Criar o segredo do repositório com o conteúdo inteiro do arquivo PEM:

   ```
   gh secret set UPDATER_SIGNING_KEY < nanci-updater.key
   ```

   Também dá para criar pelo GitHub, em Settings › Secrets and variables › Actions. O `wails3` aceita ainda o base64 da chave de 64 bytes ou da semente de 32 bytes (`loadUpdaterPrivateKey`), mas o PEM é o que o `genkey` grava.
4. Guardar `nanci-updater.key` num cofre de senhas e apagar a cópia local. Quem tem a chave privada pode publicar atualizações para todos os usuários. Se a chave se perder, os apps instalados não aceitam atualizações assinadas com outra, e trocar a chave obriga os usuários a reinstalar pelo instalador uma versão com a chave nova.
5. Para conferir o par antes de uma tag, opcionalmente: `wails3 updater manifest -version 0.0.1 -key nanci-updater.key -output m.json <um arquivo qualquer>-windows-amd64.exe` e depois `wails3 updater verify -manifest m.json -publickey <base64 do passo 1>`.

### Teste local (30/10/2026)

O teste usou dois builds de produção (`-tags production`, `.syso` do Windows) da mesma árvore: A, com versão `0.0.1`, e B, com `0.0.2`. Os dois recebiam uma chave descartável e a URL de um servidor local por `-ldflags "-X main.updaterPublicKey=... -X main.updateManifestURL=http://127.0.0.1:18765/nanci-update.json"`. Os dois valores são `var` justamente para isso, e nada do teste foi para o repositório. O manifesto de B saiu de `wails3 updater manifest -key`, exatamente como no release. O exe se chamava `nanci-e2e.exe`, para usar uma pasta de dados do WebView2 separada (`%APPDATA%\<nome do exe>`), e a pasta de dados era vazia (`NANCI_DATA_DIR`). O aviso foi acionado por UI Automation, que enxerga o conteúdo do WebView2.

- A abriu, buscou o manifesto (`version=0.0.1` na query) e mostrou "Nova versão 0.0.2 disponível.". Depois do clique em **Atualizar e reiniciar**, A saiu em 867 ms. O auxiliar (filho de A) registrou "swap succeeded on attempt 1" e iniciou B, filho dele. B ficou aberto, sem esbarrar na instância única, e o hash do exe instalado passou a ser o de B. Na checagem dele, B pediu o manifesto com `version=0.0.2`, e Sobre mostrou "Versão 0.0.2", o commit de B e "versão mais recente". O banco na pasta de dados continuou lá, aberto por B. Na pasta de instalação ficaram o exe e um `.old.<n>`.
- Recusas: com o manifesto assinado por outra chave, o aviso de erro foi "updater: ed25519ph signature did not verify". Com um byte alterado no exe servido, foi "digest mismatch". Com o manifesto sem assinatura, foi "a public key is configured, so the release must carry a signature". Nos três casos A continuou aberto, com o exe intacto e sem pasta `wails-update-*` restante. `wails3 updater verify` também rejeitou o manifesto assinado com a outra chave (código 1).
- Todas as janelas foram fechadas com `CloseMainWindow`, sem `msedgewebview2.exe` restante. As chaves descartáveis foram apagadas.

Não foi testado: um release real no GitHub (redirecionamento, upload, segredo), a troca dentro de `%LOCALAPPDATA%\Programs\Nanci Desktop` (o teste usou uma pasta na `%TEMP%` do mesmo perfil e do mesmo volume) e a gravação real do `DisplayVersion`, para não mexer na instalação desta máquina.

## Achados do spike na v3.0.0

| Achado do spike (beta.27) | Na v3.0.0 | O que ficou no código |
| --- | --- | --- |
| No Windows, fechar um diálogo de arquivo devolve o erro `cancelled by user` (o v2 devolvia caminho vazio) | Continua. `cfd.ErrorCancelled` em `internal/go-common-file-dialog/cfd/errors.go`, devolvido por `showCfdDialog` (`pkg/application/dialogs_windows.go`); nenhum erro exportado para comparar | `dialogPath`, e `dialogPaths` para a seleção múltipla de XMLs, comparando o texto |
| `app.KeyBinding.Add` grava o atalho sem normalizar, e `CmdOrCtrl+Q` nunca casa | Continua. `KeyBindingManager.Add` guarda a string crua; só `Options.KeyBindings` passa por `parseAccelerator` (`processKeyBindingOptions`) | Ctrl+Q em `Options.KeyBindings` |
| O runtime JS não tem chamadas de tema, e `WindowsWindow.Theme` só vale na criação da janela | Continua. `window.ts` não tem método de tema; `webview_window_windows.go` aplica o tema ao criar e, com `SystemDefault`, volta a seguir `SystemThemeChanged` | `SetWindowTheme` via `pkg/w32`, e um hook que cancela `SystemThemeChanged` quando o tema está fixo |
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
- [ ] **Cancelar** durante uma sincronização de NFS-e, NF-e e CT-e (também depois de trocar de tela e voltar) e durante uma consulta direta: aviso de cancelamento, sem erro; a sincronização seguinte continua do NSU onde a cancelada parou.
- [ ] Fechar durante uma sincronização pelos três caminhos (X do app, Ctrl+Q, Alt+F4 ou barra de tarefas): a confirmação aparece; "Sair" fecha, "Continuar no Nanci" mantém a janela.
- [ ] Mica e transparência atrás da janela (Windows 11); tema do sistema na abertura; trocar o tema em Configurações muda a tonalidade do Mica; trocar o tema do Windows com o app aberto (com tema fixo em Configurações, a janela não muda).
- [ ] Diálogos de arquivo: certificado, importar XMLs (seleção múltipla), salvar exportação e exportar logs. Cancelar cada um não mostra erro.
- [ ] Configurações › Sobre mostra a versão do WebView2, a escala e o número de monitores; "Reajustar janela" funciona; mudar a escala do monitor com o app aberto não deixa o texto borrado.
- [ ] Abrir o Nanci instalado com ele já aberto (inclusive minimizado) traz a janela existente para a frente, sem segunda janela.
- [ ] Redimensionar pelas duas laterais, arrastar pela barra de título, duplo clique maximiza e restaura, Windows Snap (Win+setas e arrastar para a borda).
- [ ] Smoke padrão: empresa, credencial, sync, listas de NFS-e/NF-e/CT-e, exportação, consulta direta, senha do certificado e console de logs.
- [ ] Atualização automática, depois dos [passos do dono](#passos-do-dono-uma-vez) e de duas tags com a chave: a versão anterior instalada mostra o aviso na abertura; **Atualizar e reiniciar** reabre na versão nova; Sobre e Programas e Recursos mostram a versão nova; com uma sincronização em andamento, a ação fica desligada e explica o motivo.

## Faixas de redimensionamento

Mantenha as faixas `.app-resize-edge--left` e `.app-resize-edge--right` de `MainLayout.vue`. Elas evitam que um clique na borda caia numa barra de rolagem ou num link da gaveta, o que faria o WebView capturar o mouse e o redimensionamento falhar. No v3, o runtime JS decide a borda pela posição do mouse, independente do elemento sob o cursor, então as faixas não atrapalham a detecção (leitura de `drag.ts`; falta conferir na mão).

Experimento opcional: `WindowsWindow.NonClientRegionSupport: true` liga o suporte nativo do WebView2 a regiões não cliente, e a barra de título passaria a usar `--wails-non-client-region: caption` (com `minimize`, `maximize` e `close` nos botões). Isso poderia trazer o Snap Layouts ao botão de maximizar e dispensar as faixas, mas depende da versão do WebView2 Runtime instalada.

## Fontes (conferidas em 29/10/2026)

- Release v3.0.0 (26/10/2026; GTK 4 e WebKitGTK 6.0 por padrão no Linux, GTK 3 com `-tags gtk3` até a v3.1; Go 1.26 ou mais novo): https://github.com/wailsapp/wails/releases/tag/v3.0.0
- Publicação de `@wailsio/runtime@3.0.0` no npm: 2026-10-26T11:01:20Z (`npm view @wailsio/runtime time`).
- `minimumReleaseAgeExclude` aceita versões exatas (`nome@versão`): https://pnpm.io/settings/dependency-resolution#minimumreleaseageexclude
- Issue #6209 (prontidão do `wails3 dev` contra o Vite), fechada em 19/10/2026 pelo PR #6305: https://github.com/wailsapp/wails/issues/6209
- Código-fonte da v3.0.0: `go mod download github.com/wailsapp/wails/v3@v3.0.0`, arquivos citados acima.
- Link para o asset do release mais recente (`/releases/latest/download/<asset>`): https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases (conferido em 30/10/2026).
- "The latest release is the most recent non-prerelease, non-draft release": https://docs.github.com/en/rest/releases/releases#get-the-latest-release (conferido em 30/10/2026).
- Issue #6052 (`SingleInstance` chama `os.Exit` dentro de `application.New`), aberta em 29/08/2026 e ainda aberta em 30/10/2026: https://github.com/wailsapp/wails/issues/6052
