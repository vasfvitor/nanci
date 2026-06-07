# Redesign para App Desktop

O objetivo deste plano e transformar a interface grafica do Nanci de um layout espacoso, parecido com portal web, para uma interface densa, rapida e produtiva, mais proxima de um aplicativo desktop nativo usado em rotinas operacionais e contabeis.

A direcao visual final deve priorizar leitura, comparacao e acao rapida: tabelas densas, barras de ferramentas compactas, navegacao lateral persistente, menos espaco ornamental e menos dependencia de cards para dados repetidos.

## Proposed Changes

### `MainLayout.vue`

- **Layout Model**: alterar o `q-layout` atual de `view="lHh Lpr lFf"` para um modelo em que a navegacao lateral ocupe a altura completa da janela. A opcao alvo e `view="hHh Lpr lFf"` ou equivalente validado no Quasar, preservando o comportamento correto do `q-page-container`.
- **Header compacto**: substituir o header atual por uma barra superior mais baixa e funcional, com titulo curto, botao de console em icone e controles alinhados como toolbar de aplicativo. Evitar texto grande ou composicao de landing page.
- **Header nativo Wails / barra arrastavel**: habilitar janela sem moldura no Wails apenas se os controles nativos forem implementados de forma completa. A barra customizada deve usar uma area com `--wails-draggable: drag`; botoes e campos interativos dentro dela devem usar `--wails-draggable: no-drag`.
- **Controles da janela**: se `Frameless` for ativado, adicionar botoes de minimizar, maximizar/restaurar e fechar usando as funcoes do runtime Wails ja disponiveis no frontend (`WindowMinimise`, `WindowToggleMaximise`, `Quit`/fechar conforme API disponivel). Os botoes devem ser icon-only, densos e com tooltip.
- **Sidebar**: tornar os itens de navegacao mais compactos (`dense`), com largura fixa menor, estados ativos claros e fundo discreto. A lista deve continuar simples: Empresas, Documentos e Credenciais.
- **Console**: manter o drawer de console, mas compactar sua toolbar e revisar a acao de limpar logs para usar confirmacao quando houver conteudo. O console pode continuar sendo um drawer overlay lateral.
- **Estilo global**: reduzir padding padrao de paginas e criar classes utilitarias locais para paginas de dados, por exemplo `desktop-page`, `desktop-page__toolbar` e `desktop-table`, em vez de repetir estilos ad hoc em cada tela.

### Funcionalidades do Wails (Native Feel)

- **Frameless Window**: configurar a janela sem moldura em `internal/desktop/main.go` somente junto com uma barra customizada funcional em Vue. A mudanca deve ser testada em janela normal, maximizada e redimensionada.
- **Windows Mica / Acrylic Backdrop**: avaliar suporte da versao atual do Wails antes de aplicar `windows.Options{ BackdropType: windows.Mica }`. O arquivo `internal/desktop/main.go` ja possui `windows.Options{ Theme: windows.SystemDefault }`; a alteracao deve ser incremental e compatibilizada com o background do Vue. Se Mica for usado, o app nao deve depender de transparencia forte para legibilidade.
- **Background e contraste**: se Mica/Acrylic estiver ativo, trocar `BackgroundColour` e fundos do layout para tons semitransparentes apenas onde fizer sentido. Tabelas e toolbars devem manter contraste suficiente para leitura prolongada.
- **Acoes Nativas**: o app ja usa `runtime.MessageDialog` no menu "Sobre". Expandir esse padrao para confirmacoes criticas, como sair durante sincronizacao, limpar logs com conteudo, ou erros que bloqueiam o fluxo. Notificacoes Quasar continuam adequadas para sucesso, avisos leves e feedback transiente.
- **Menu nativo**: manter o menu nativo existente em `internal/desktop/main.go` para Arquivo/Ajuda. Nao duplicar esses comandos dentro da UI principal, exceto quando fizer parte de uma toolbar operacional.

### `CompaniesPage.vue`

- **Tchau Cards**: substituir o grid atual de `q-card` por uma `q-table` plana, densa e com `flat`, `bordered`, `dense` e paginacao adequada.
- **Toolbar da pagina**: manter o titulo "Empresas" pequeno e adicionar o botao "Adicionar" na mesma linha. A barra deve ocupar pouca altura e ficar visualmente conectada a tabela.
- **Tudo na mesma linha**: as colunas principais serao:
  - Nome
  - CNPJ
  - Ambiente
  - Ultimo NSU
  - Credencial
  - Acoes
- **Credencial inline**: renderizar um `q-select dense borderless` ou `outlined dense` dentro da celula de Credencial, ligado a `selectedCredentials[company.CNPJ]`, mantendo `AssignCredentialToCompany` no `@update:model-value`.
- **Acoes por icone**: mover Sincronizar e Editar para a coluna Acoes com botoes icon-only (`sync`, `edit`) e tooltips. O botao de sincronizar deve preservar estado de loading por empresa.
- **Estado vazio**: substituir o bloco grande com icone por um estado vazio compacto dentro/abaixo da tabela, com acao primaria para adicionar empresa.
- **Dados existentes**: preservar `reloadData`, `assignCredential`, `syncCompany`, dialogs de adicionar/editar e notificacoes atuais.

### `CredentialsPage.vue`

- **Data Table**: substituir os cards de credenciais por `q-table` densa, plana e voltada a administracao.
- **Toolbar da pagina**: titulo compacto "Credenciais" e botao "Adicionar" na mesma linha.
- **Colunas**:
  - Apelido
  - Proprietario
  - Caminho do Certificado
  - Ambiente
  - Status de Inspecao
  - Acoes
- **Caminho do certificado**: renderizar o caminho com truncamento visual e tooltip com o caminho completo, para evitar que uma linha longa destrua a largura da tabela.
- **Status de inspecao**: usar badge discreto para "Inspecionada" / "Pendente", com base em `InspectedAt`.
- **Acoes por icone**: usar botoes icon-only para editar e trocar arquivo (`edit`, `folder_open`) com tooltips.
- **Estado vazio**: usar estado compacto com acao para adicionar credencial.
- **Dados existentes**: preservar `ListCredentials`, `SelectCertificate`, `UpdateCredentialPath`, `AddCredentialDialog` e `EditCredentialDialog`.

### `DocumentsPage.vue`

- **Filtros sem card**: substituir o `q-card` de filtros por uma `q-toolbar` ou linha flex compacta grudada ao topo da tabela.
- **Fluxo vertical**: a pagina deve usar layout em coluna com altura util completa: toolbar de filtros no topo e `q-table` ocupando o restante.
- **Filtros compactos**: manter Empresa, Competencia e Direcao, mas com campos densos e larguras controladas. Em telas estreitas, permitir quebra limpa sem sobrepor controles.
- **Acoes**: manter Buscar e Exportar na toolbar. Exportar continua como dropdown com CSV, XLSX e ZIP.
- **Tabela**: manter as colunas existentes, badges de Direcao, Visibilidade e Status, mas ativar densidade e altura calculada para reduzir rolagem da pagina inteira.
- **Paginacao/linhas**: revisar `rows-per-page-options` para valores produtivos, por exemplo 25, 50, 100 e 0 quando "todos" for aceitavel.
- **Exportacao**: manter o fluxo atual com `SelectExportDirectory` e funcoes `ExportCSV`, `ExportXLSX`, `ExportZIP`. Uma melhoria futura pode mover a montagem de caminho para o backend, mas isso fica fora deste redesign.

## User Review Required

> [!WARNING]
> **Mudanca drastica de identidade visual**
> Esta mudanca substitui as telas de gerenciamento baseadas em cards por tabelas densas. Isso melhora leitura, comparacao e produtividade quando houver muitas empresas, credenciais e documentos, mas o visual ficara mais proximo de software operacional desktop e menos parecido com um portal web.

> [!WARNING]
> **Frameless exige acabamento completo**
> Ativar janela sem moldura remove controles nativos do sistema. Antes de concluir essa parte, a UI precisa ter minimizar, maximizar/restaurar, fechar, area arrastavel, areas `no-drag` em botoes, comportamento correto em janela maximizada e contraste adequado.

> [!TIP]
> O uso de tabelas densas, toolbars limpas, menu lateral full-height e confirmacoes nativas onde fazem sentido deve deixar o Nanci com cara de ferramenta profissional de rotina fiscal.

Decisao recomendada: aprovar a transicao completa de cards para tabelas nas telas de gerencia, mas implementar `Frameless`/Mica como uma etapa separada dentro do mesmo redesign, para evitar misturar mudanca visual de alto impacto com comportamento de janela nativa.
