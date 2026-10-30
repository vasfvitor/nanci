# WebView2: janela borrada com dois monitores

Levantado em 17/10/2026 no commit `c4cb1fa` (Wails v2.16.0, go-webview2 v1.0.22).

Responde: por que a janela do Nanci pode ficar levemente borrada num notebook com monitor externo, o que o Wails faz com a escala do monitor e o que o app passou a fazer.

Todas as fontes foram consultadas em 17/10/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fóruns ou de código de outros projetos.

## O relato

- Windows 11, tela do notebook mais um monitor HDMI, provavelmente com escalas diferentes (por exemplo 125% e 100%).
- A janela fica "como numa resolução menor" depois de abrir ou fechar um diálogo, ou quando o foco vai para outro programa no outro monitor.
- A janela **não** muda de monitor quando o borrado aparece (correção do dono do relato em 17/10/2026).
- O frontend não usa `transform`, `filter` nem `zoom` no CSS. O manifesto declara per-monitor v2.

## WebView2: RasterizationScale e ShouldDetectMonitorScaleChanges

Fonte: Microsoft Learn, referência Win32 `ICoreWebView2Controller3`, SDK 1.0.4258.31, página datada de 21/09/2026: https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2controller3 (a mesma página em japonês: https://learn.microsoft.com/ja-jp/microsoft-edge/webview2/reference/win32/icorewebview2controller3). Foi lida a versão em inglês.

- `RasterizationScale` é a escala DPI do monitor combinada com a escala de texto do usuário. Segundo a página, o valor "should be updated when the DPI scale of the app's top level window changes (i.e. monitor DPI scale changes or window changes monitor) or when the text scale factor of the system changes".
- Com `ShouldDetectMonitorScaleChanges` verdadeiro, o WebView acompanha a escala do monitor, atualiza `RasterizationScale` e dispara `RasterizationScaleChanged`. Com falso, "the app must update the RasterizationScale property itself", e o evento nunca dispara.
- Em `COREWEBVIEW2_BOUNDS_MODE_USE_RAW_PIXELS`, os bounds são pixels de tela; mudar a escala de rasterização não muda o tamanho em pixels.
- O exemplo da página trata `WM_DPICHANGED` atualizando a escala do WebView e depois chamando `SetWindowPos` com o retângulo sugerido.

## JUCE: o mesmo sintoma em outro host (relato de terceiros)

Fonte: fórum do JUCE, https://forum.juce.com/t/bug-webbrowsercomponent-webview2-backend-ignores-dpi-change/58923

- Relato de 23/11/2023: ao mover a janela entre monitores de escalas diferentes, o WebView2 não acompanhava os bounds do pai até um redimensionamento manual. (relato de terceiros)
- Correção no JUCE em 05/12/2023, commit `06855ed05da4e1638674904f9b595fe7d6c049ae`, "WebBrowserComponent: Windows: Trigger resized event on DPI change when using WebView2": dispara um resize na mudança de DPI. (relato de terceiros)

## O que o Wails 2.16 faz

Lido no cache de módulos (`github.com/wailsapp/wails/v2@v2.16.0` e `github.com/wailsapp/go-webview2@v1.0.22`):

- `internal/frontend/desktop/windows/window.go:226-227`: em `WM_MOVE` e `WM_MOVING`, chama `NotifyParentWindowPositionChanged`.
- `internal/frontend/desktop/windows/window.go:240-248`: em `WM_DPICHANGED`, só aplica o retângulo sugerido com `SetWindowPos` (`SWP_NOZORDER|SWP_NOACTIVATE`). Não mexe na escala do WebView.
- `internal/frontend/desktop/windows/frontend.go:186-217`: em todo `WM_SIZE` que não seja minimizar, chama `chromium.Resize()`, que passa os bounds novos ao controller. O Nanci não define `ResizeDebounceMS`, então o resize é imediato.
- go-webview2 `pkg/edge/chromium.go:341-351`: ao criar o controller, define `BoundsMode` como `USE_RAW_PIXELS` e `ShouldDetectMonitorScaleChanges` como falso ("Disable monitor scale changes since we're using raw pixels").
- Nem o Wails v2.16.0 nem o go-webview2 v1.0.22 chamam `PutRasterizationScale` (busca por `RasterizationScale` no código Go dos dois módulos).

Pelo texto da Microsoft acima, com a detecção desligada o host precisa atualizar a escala, e ninguém atualiza. Inferência, não testada: depois de uma mudança real de escala (janela levada a um monitor de outra escala, ou escala alterada nas Configurações do Windows), o WebView continua rasterizando na escala do início, e `window.devicePixelRatio` na página também fica no valor antigo. Nesse caso o detector do frontend descrito abaixo não vê a mudança.

## O que o Nanci faz

Situação no commit `c4cb1fa`, ainda no Wails v2. Com o Wails v3 o empurrão automático saiu; ver a seção seguinte.

- `useDisplayScale` (`internal/desktop/frontend/src/composables/useDisplayScale.ts`), iniciado pelo `MainLayout`, observa `devicePixelRatio` com `matchMedia("(resolution: Xdppx)")`, refeito a cada mudança, e confere de novo em `focus` e `visibilitychange`. Se a razão mudou, espera um frame, aumenta a janela em 1 px pelo `WindowSetSize` e volta ao tamanho no frame seguinte, o que força um `WM_SIZE` e um `chromium.Resize()`. Janela maximizada não é tocada (redimensionar tiraria a maximização); só fica a linha no console.
- No console, uma linha por mudança: "Escala do monitor mudou de X para Y; janela reajustada". E uma foto da janela (escala, área visível, tela, tamanho pelo `WindowGetSize`) no início e a cada foco ou mudança de visibilidade, só quando algum valor mudou.
- Configurações → Sobre o Nanci mostra a versão do WebView2, a escala e o número de monitores, e tem o botão **Reajustar janela**, o mesmo empurrão de 1 px a pedido. Serve para o usuário testar se um relayout resolve mesmo quando a escala não mudou.
- O pacote de diagnóstico (`ExportLogs`) inclui `diagnostico.txt` com versão, commit, sistema e versão do WebView2, lida com `webviewloader.GetAvailableCoreWebView2BrowserVersionString("")`, a mesma função que o Wails usa (`internal/system/system_windows.go:32`).
- O empurrão passa por tamanhos lógicos: o Wails converte pela escala da janela na ida e na volta, e com escala fracionária a janela pode perder 1 px físico por empurrão.

## Wails v3.0.0

Lido em 30/10/2026 no cache de módulos (`github.com/wailsapp/wails/v3@v3.0.0`), versão para a qual o desktop migrou. O go-webview2 agora vem embutido no módulo, em `internal/webview2`.

- `internal/webview2/pkg/edge/chromium.go:625-649`: ao criar o controller na hospedagem em janela (a padrão), o v3 mantém `BoundsMode` em `USE_RAW_PIXELS`, mas deixa `ShouldDetectMonitorScaleChanges` no padrão do WebView2, ligado. Segundo o comentário no código, com a escala escrita pelo host, cruzar monitores de escalas diferentes podia derrubar o processo da GPU do WebView2.
- `pkg/application/webview_window_windows.go:2603-2660` (`setupChromium`): liga a detecção de forma explícita com `PutShouldDetectMonitorScaleChanges(true)`, salvo com `Windows.UseVisualHosting`, e guarda em `monitorScaleDetectionOn` o valor que o controller devolve. O Nanci não usa `UseVisualHosting`.
- `pkg/application/webview_window_windows.go:2041-2110`: em `WM_DPICHANGED`, aplica o retângulo sugerido com `SetWindowPos` (com `SWP_FRAMECHANGED` numa janela sem moldura, como a do Nanci) e, fora do minimizado, chama `resyncWebviewRasterizationScale`. Com a detecção ligada essa função não faz nada (`:2238-2248`): o WebView2 é o único a escrever a escala. Quando ela escreve, o código chama `chromium.Resize()` em seguida.

Ou seja, a causa provável apontada acima para o v2 (detecção desligada e nenhuma chamada a `PutRasterizationScale`) não existe no v3: o próprio WebView2 acompanha a escala do monitor.

O que mudou no Nanci com o v3:

- O empurrão automático saiu. `useDisplayScale` não observa mais `devicePixelRatio`, `focus` nem `visibilitychange`, e as linhas "Escala do monitor mudou" e "Janela: escala ..." deixaram de ir para o console.
- O botão **Reajustar janela** de Configurações → Sobre fica por mais uma versão, como diagnóstico: se o texto ficar borrado no v3, ele diz se um relayout resolve. A escala e o número de monitores continuam na mesma seção.

## Hipóteses em aberto

- **Mudança de escala (não confirmada para este relato).** Como a janela não muda de monitor, nada indica que a escala mude quando o borrado aparece. O botão **Reajustar janela** diz se um relayout resolve; se resolver sem mudança de escala, a causa é outra perda de rasterização do compositor, não o DPI.
- **Mica e fundo transparente (a testar).** O app usa `BackdropType: windows.Mica` e `BackgroundColour` com alfa 0 (`internal/desktop/main.go`). A Mica esmaece quando a janela perde o foco, e o texto sobre fundo transparente tende a sair com antialiasing em tons de cinza em vez de ClearType, o que pode parecer borrado justamente quando o foco vai para outro programa ou um diálogo abre. Teste: build com `BackdropType: windows.None` e fundo opaco, e comparar com o foco dentro e fora da janela. Não testado.

## Não encontrado

- Texto da Microsoft que descreva texto borrado no WebView2 depois de diálogos ou da perda de foco sem mudança de DPI.
- Release do JUCE que traga o commit `06855ed`.
- Nenhuma chamada a `PutRasterizationScale` no Wails v2.16.0 nem no go-webview2 v1.0.22. O rastreador de issues do Wails não foi consultado nesta rodada.
