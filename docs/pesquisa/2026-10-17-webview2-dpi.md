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

## WebView2Feedback, issue 58

Fonte: repositório de feedback do WebView2 mantido pela Microsoft, https://github.com/MicrosoftEdge/WebViewFeedback/issues/58 ("Dynamic DPI support may not use the right monitor").

- Aberta em 17/10/2019. Ao arrastar uma janela per-monitor v2 de um monitor de 200% para um de 100%, o WebView redimensionava, mas continuava com a escala do monitor antigo até a maior parte dele passar para o novo. O diagnóstico do autor da issue (consulta de DPI pela janela filha do processo do Edge) é relato de terceiros.
- Resposta da Microsoft (David Risney, mesma data): bug confirmado. Em 02/2021, Brad Peters (Microsoft) anunciou uma API para o app controlar a escala, que virou `RasterizationScale`.
- Fechada como concluída em 19/02/2022, sem comentário dizendo qual mudança resolveu.

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

- `useDisplayScale` (`internal/desktop/frontend/src/composables/useDisplayScale.ts`), iniciado pelo `MainLayout`, observa `devicePixelRatio` com `matchMedia("(resolution: Xdppx)")`, refeito a cada mudança, e confere de novo em `focus` e `visibilitychange`. Se a razão mudou, espera um frame, aumenta a janela em 1 px pelo `WindowSetSize` e volta ao tamanho no frame seguinte, o que força um `WM_SIZE` e um `chromium.Resize()`. Janela maximizada não é tocada (redimensionar tiraria a maximização); só fica a linha no console.
- No console, uma linha por mudança: "Escala do monitor mudou de X para Y; janela reajustada". E uma foto da janela (escala, área visível, tela, tamanho pelo `WindowGetSize`) no início e a cada foco ou mudança de visibilidade, só quando algum valor mudou.
- Configurações → Sobre o Nanci mostra a versão do WebView2, a escala e o número de monitores, e tem o botão **Reajustar janela**, o mesmo empurrão de 1 px a pedido. Serve para o usuário testar se um relayout resolve mesmo quando a escala não mudou.
- O pacote de diagnóstico (`ExportLogs`) inclui `diagnostico.txt` com versão, commit, sistema e versão do WebView2, lida com `webviewloader.GetAvailableCoreWebView2BrowserVersionString("")`, a mesma função que o Wails usa (`internal/system/system_windows.go:32`).
- O empurrão passa por tamanhos lógicos: o Wails converte pela escala da janela na ida e na volta, e com escala fracionária a janela pode perder 1 px físico por empurrão.

## Hipóteses em aberto

- **Mudança de escala (não confirmada para este relato).** Como a janela não muda de monitor, nada indica que a escala mude quando o borrado aparece. As linhas de console e o botão **Reajustar janela** dizem se um relayout resolve; se resolver sem mudança de escala, a causa é outra perda de rasterização do compositor, não o DPI.
- **Mica e fundo transparente (a testar).** O app usa `BackdropType: windows.Mica` e `BackgroundColour` com alfa 0 (`internal/desktop/main.go`). A Mica esmaece quando a janela perde o foco, e o texto sobre fundo transparente tende a sair com antialiasing em tons de cinza em vez de ClearType, o que pode parecer borrado justamente quando o foco vai para outro programa ou um diálogo abre. Teste: build com `BackdropType: windows.None` e fundo opaco, e comparar com o foco dentro e fora da janela. Não testado.

## Não encontrado

- Texto da Microsoft que descreva texto borrado no WebView2 depois de diálogos ou da perda de foco sem mudança de DPI.
- Release do JUCE que traga o commit `06855ed`.
- Nenhuma chamada a `PutRasterizationScale` no Wails v2.16.0 nem no go-webview2 v1.0.22. O rastreador de issues do Wails não foi consultado nesta rodada.
