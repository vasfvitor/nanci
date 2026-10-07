# Receita Federal e Serpro: Integra Contador, consultas pagas e fontes abertas

Levantado em 07/10/2026.

Responde: que dados federais a empresa consegue puxar por API com o próprio e-CNPJ, quanto custa e quais fontes gratuitas existem para CNPJ e certidões.

Todas as fontes foram consultadas em 07/10/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fornecedores, fóruns ou imprensa.

Sobre os preços da Loja Serpro:
- O WebFetch em loja.serpro.gov.br devolve 403.
- Por isso os preços vêm dos dados públicos dos produtos da loja, em https://loja.serpro.gov.br/ccstore/v1/products/<id>, no campo `x_formasDeContratacaoColuna1`. São as mesmas tabelas das páginas de produto.

Documentação do Integra Contador, abreviada como DOC abaixo: https://apicenter.estaleiro.serpro.gov.br/documentacao/api-integra-contador/ (páginas datadas de fevereiro a setembro de 2026).

## Integra Contador

### Uso pela própria empresa, sem contador

- DOC/pt/servicos_vs_procuracoes/: "Existem serviços que exigem a procuração eletrônica quando o autor do pedido de dados não é o próprio contribuinte."
- DOC/pt/integra_contador/: o `autorPedidoDados` "Pode ser o próprio Contratante, Procurador ou Contribuinte."
- Logo, a empresa pode ser ao mesmo tempo contratante, autor e contribuinte, com o próprio e-CNPJ A1. Nesse caso não precisa de procuração nem da etapa AUTENTICAPROCURADOR.
- Exceção: o EVENTOSATUALIZACAO aparece como "sim¹". O autor precisa de procuração válida dos contribuintes monitorados, por exemplo o código 00103 para o evento E0301.

### Chamadas

- POST em `https://gateway.apiserpro.serpro.gov.br/integra-contador/v1/{Apoiar|Consultar|Declarar|Emitir|Monitorar}`.
- Corpo: `{contratante{numero,tipo}, autorPedidoDados, contribuinte, pedidoDados{idSistema, idServico, versaoSistema, dados}}`. O campo `dados` é uma string JSON escapada.
- O `tipo` vale 1 para CPF e 2 para CNPJ. Os valores 3 e 4 são listas de PF e de PJ. O contratante tem de ser CNPJ.

### Autenticação

Fonte: DOC/pt/quick_start/

1. POST em `https://autenticacao.sapi.serpro.gov.br/authenticate`, com o .pfx do e-CNPJ do contratante como certificado de cliente (mTLS).
2. Cabeçalhos: `Authorization: Basic base64(consumerKey:consumerSecret)`, `Role-Type: TERCEIROS` e Content-Type form-urlencoded.
3. Corpo: `grant_type=client_credentials`.
4. A resposta traz `access_token` (Bearer) e `jwt_token`. Os dois cabeçalhos vão em toda chamada. Um 401 pede nova autenticação.

O CNPJ da requisição tem de bater com o NI do certificado. Se não bater, a resposta é 403 `AcessoNegado-ICGERENCIADOR-016` (DOC/pt/faq/).

### Catálogo de serviços

Fonte: DOC/pt/catalogo_de_servicos/. O tipo de caminho vem entre colchetes.

- **Integra-SN / PGDASD**
  - TRANSDECLARACAO11 [Declarar].
  - GERARDAS12 [Emitir]. Em 27/11/2024 entraram GERARDASCOBRANCA17, GERARDASPROCESSO18 e GERARDASAVULSO19 [Emitir].
  - CONSDECLARACAO13, CONSULTIMADECREC14, CONSDECREC15 e CONSEXTRATO16, que traz o extrato do DAS [Consultar].
- **DEFIS**
  - Transmitir [Declarar].
  - Listar, última declaração com recibo em PDF e declaração específica em PDF [Consultar].
- **REGIMEAPURACAO**
  - Opção por caixa ou competência [Declarar].
  - Consultar anos, opção e resolução [Consultar].
- **Integra-MEI**
  - PGMEI: DAS em PDF, código de barras do DAS e atualização de benefício [Emitir]; dívida ativa [Consultar].
  - CCMEI: emissão em PDF [Emitir]; dados e situação cadastral [Consultar].
  - DASN-SIMEI: envio [Declarar]; consulta [Consultar]; DAS de excesso [Emitir].
- **Integra-DCTFWeb**
  - DCTFWEB [Emitir]: gerar guia, guia MAED, guia com abatimento, aplicar vinculação, editar valor suspenso e GERARGUIAANDAMENTO313 (28/03/2025).
  - DCTFWEB [Consultar]: recibo, declaração completa, XML, relatórios de crédito e débito, notificação MAED.
  - TRANSDECLARACAO310 [Declarar].
  - MIT, desde 28/03/2025: encerrar apuração [Declarar]; situação do encerramento [Apoiar]; consultar apuração e listar por ano ou mês [Consultar].
- **Integra-Procurações:** OBTERPROCURACAO41 [Consultar].
- **Integra-Sicalc**
  - CONSOLIDARGERARDARF51, em PDF, e GERARDARFCODBARRA53, com código de barras [Emitir].
  - CONSOLIDAR54, que consolida sem emitir [Consultar].
  - CONSULTAAPOIORECEITAS52 [Apoiar, gratuito].
- **Integra-CaixaPostal**
  - Lista e detalhe de mensagens [Consultar].
  - Indicador de mensagem nova [Monitorar, gratuito].
  - DTE: situação da adesão [Consultar].
- **Integra-Pagamento (PAGTOWEB)**
  - PAGAMENTOS71, consulta de pagamentos, e CONTACONSDOCARRPG73, contagem [Consultar].
  - COMPARRECADACAO72, comprovante de arrecadação [Emitir].
- **Integra-Sitfis**
  - SOLICITARPROTOCOLO91 [Apoiar], depois RELATORIOSITFIS92 [Emitir].
  - Devolve o Relatório de Situação Fiscal da RFB e da PGFN, previsto na Portaria 1.751/2014.
  - É assíncrono: responde 202 com `tempoEspera`.
  - É relatório, não certidão.
- **Integra-Parcelamento** (desde 11/11/2024)
  - Modalidades: PARCSN, PARCSN-ESP, PERTSN, RELPSN, PARCMEI, PARCMEI-ESP, PERTMEI e RELPMEI.
  - Cada uma tem pedidos, plano, detalhe de pagamento e parcelas para impressão [Consultar], e emissão do DAS [Emitir].
  - PARC-PAEX e PARC-SIPADE têm extrato em PDF ou JSON e documento de arrecadação. Sem data de inclusão.
- **Integra-Redesim / PNRCONTADOR** (02/12/2025): vínculos, pedido e situação de renúncia, comprovante.
- **Integra-e-Processo** (10/11/2025): processos por interessado, lista e obtenção de documentos, comunicados e intimações.
- **Gerenciador**
  - AUTENTICAPROCURADOR ENVIOXMLASSINADO81 [Apoiar].
  - EVENTOSATUALIZACAO, assíncrono, com os eventos de "última atualização" de PF e PJ: SOLICEVENTOSPF131, SOLICEVENTOSPJ132, OBTEREVENTOSPF133 e OBTEREVENTOSPJ134 [Monitorar, gratuito].

### Códigos de procuração

Só se aplicam quando o autor não é o contribuinte.

| Serviço | Código |
|---|---|
| PGDAS-D e DEFIS | 00146 |
| DCTFWeb | 00103 |
| Caixa Postal | 00006 |
| Pagamentos | 00004 |
| Sitfis | 00002 |
| Regime de apuração | 00060 |
| DTE | 00050 |
| e-Processo | 00051 |
| Consultas DASN-SIMEI | 00229 |
| PARCSN | 00076 / 00188 |
| PARCSN-ESP | 00125 |
| PERTSN | 00149 / 10011 |
| RELPSN | 00210 / 10036 |
| PARCMEI | 00134 |
| PARCMEI-ESP | 00133 |
| PERTMEI | 00152 / 10012 |
| RELPMEI | 00209 / 10035 |

PGMEI, CCMEI, Sicalc, Procurações e PNRCONTADOR não exigem procuração.

### AUTENTICAPROCURADOR

Fonte: DOC/pt/solucoes/integra-contador-gerenciador/autenticaprocurador/
- Serve só para o contratante que age em nome de um procurador, por exemplo uma software house em nome de um escritório de contabilidade.
- O procurador assina em XMLDSig um XML de "Termo de Autorização". O padrão é o mesmo de NF-e e eSocial, com e-CPF ou e-CNPJ ICP-Brasil A1 ou A3, e o XML vai em base64.
- O token devolvido vai no cabeçalho `autenticar_procurador_token` e vale até a meia-noite do dia seguinte, no horário de Brasília.
- Um 304 indica que o token em cache continua válido.

### Contratação

Fontes: DOC/pt/como_contratar/ e a loja.
- Empresa de qualquer porte pode contratar. O pré-requisito é ter e-CNPJ.
- Passos: clicar em "Quero contratar" e assinar o contrato com o e-CNPJ. As chaves aparecem em cliente.serpro.gov.br em cerca de 10 minutos.
- Cobrança pós-paga mensal, sem taxa de adesão e com cancelamento a qualquer momento.
- Não há faixa gratuita em produção. Existe um Swagger de demonstração com dados fictícios (DOC/pt/como_usar_a_demonstracao_da_api/).

### Cobrança por status

Fontes: DOC/pt/codigos_retorno/ e DOC/pt/faq/
- São cobrados os status HTTP 200, 202 e 403. Uma chamada negada por falta de procuração também custa.
- Não são cobrados 204, 304, 400, 401, 404, 429, 500, 503 e 504.
- Os caminhos /Apoiar e /Monitorar são gratuitos.

### Limites

- O 429 existe "em alguns serviços".
- Os únicos números publicados são do EVENTOSATUALIZACAO (DOC/pt/solucoes/integra-contador-gerenciador/eventosatualizacao/limites/, de 25/02/2026):
  - 1.000 requisições por dia para eventos de PF e 1.000 por dia para PJ;
  - até 1.000 contribuintes distintos por requisição;
  - resultado guardado por 20 minutos e apagado depois da primeira leitura com sucesso.

### Tabela de preços

Fonte: produto `integracontador` da loja.
- A faixa é definida pelo volume mensal de cada categoria.
- O preço unitário da faixa vale para todas as requisições do mês.

| Faixa | Consulta: req/mês | Consulta (R$) | Emissão: req/mês | Emissão (R$) | Declaração: req/mês | Declaração (R$) |
|---|---|---|---|---|---|---|
| 1 | 1 a 300 | 0,24 | 1 a 500 | 0,32 | 1 a 100 | 0,40 |
| 2 | 301 a 1.000 | 0,21 | 501 a 5.000 | 0,29 | 101 a 500 | 0,36 |
| 3 | 1.001 a 3.000 | 0,18 | 5.001 a 10.000 | 0,26 | 501 a 1.000 | 0,32 |
| 4 | 3.001 a 7.000 | 0,16 | 10.001 a 15.000 | 0,22 | 1.001 a 3.000 | 0,28 |
| 5 | 7.001 a 15.000 | 0,14 | 15.001 a 25.000 | 0,19 | 3.001 a 5.000 | 0,24 |
| 6 | 15.001 a 23.000 | 0,11 | 25.001 a 35.000 | 0,16 | 5.001 a 8.000 | 0,20 |
| 7 | 23.001 a 30.000 | 0,09 | 35.001 a 50.000 | 0,12 | 8.001 a 10.000 | 0,16 |
| 8 | acima de 30.000 | 0,06 | acima de 50.000 | 0,08 | acima de 10.000 | 0,12 |

- A tabela parece igual desde pelo menos maio de 2024.
  - O exemplo da SCI de 29/05/2024 bate exatamente: R$ 96 para 50 empresas do Simples, com 100 requisições de cada tipo na faixa 1 (relato de terceiros): https://www.blog.sci.com.br/post/an%C3%A1lise-de-custos-do-integra-contador-o-que-toda-empresa-cont%C3%A1bil-deve-saber
  - A ajuda da Domínio cita os mesmos preços da faixa 1, 0,40, 0,32 e 0,24 (relato de terceiros): https://suporte.dominioatendimento.com/central/faces/solucao.html?codigo=10776
- Para uma empresa que usa só para si, o custo fica entre centavos e poucos reais por mês.

### CNPJ alfanumérico

Fonte: DOC/pt/faq/
- É aceito para contratante, autor e contribuinte, com validação do dígito verificador.
- O `numero` deve ser tratado como string em maiúsculas, no padrão `[A-Z0-9]`.
- Só é atribuído a inscrições novas, a partir de julho de 2026 (IN RFB 2.119/2022).
- A Fazenda informou que o primeiro foi gerado em 31/07/2026: https://www.gov.br/fazenda/pt-br/assuntos/noticias/2026/julho/receita-federal-gera-o-primeiro-cnpj-em-formato-alfanumerico/

## Outras APIs do Serpro

Todas são pós-pagas e exigem e-CNPJ para contratar. Fonte: dados de produto da loja.

### Consulta CNPJ (produto consultacnpj)

- Três níveis:
  - Básica: situação, endereço, CNAE, natureza jurídica e telefone.
  - QSA: acrescenta o quadro societário e as opções de MEI e Simples.
  - Empresa: acrescenta o CPF ou CNPJ dos sócios.
- Preço por consulta, nos níveis Básica, QSA e Empresa:

| Faixa | Volume mensal | Básica (R$) | QSA (R$) | Empresa (R$) |
|---|---|---|---|---|
| 1 | 1 a 999 | 0,6591 | 0,8683 | 1,1717 |
| 2 | 1.000 a 9.999 | 0,5649 | 0,7428 | 1,0148 |
| 16 | 30 milhões ou mais | 0,025 | 0,067 | 0,108 |

### Consulta CPF (produto consultacpf)

R$ 0,6591 até 999 por mês, caindo a R$ 0,017 acima de 30 milhões.

### Consulta NF-e (produto consultanfe)

Documentação:
- https://apicenter.estaleiro.serpro.gov.br/documentacao/consulta-nfe/pt/faq/
- https://apicenter.estaleiro.serpro.gov.br/documentacao/consulta-nfe/pt/leiautes_formatos/

Características:
- Só modelo 55, consultado pela chave na base do SPED. Atende empresas públicas e privadas.
- Devolve JSON no formato do schema da NF-e, só com os campos "autorizados na anuência". Há uma planilha no FAQ.
- Eventos:
  - cancelamento, CC-e, manifestação e outros vêm completos;
  - passagem, MDF-e e CT-e vêm resumidos.
- Preço: R$ 0,6591 por consulta até 999 por mês, caindo a R$ 0,027 acima de 30 milhões. O monitoramento push custa o mesmo, por nota, por 30 dias.
- Chave e CNPJ devem ser tratados como string, por causa do CNPJ alfanumérico.

### Outras

- **Consulta CND:** ver Certidões abaixo.
- **Consulta Dívida Ativa (PGFN):** de R$ 0,6591 a R$ 0,0314.
- **Consulta Faturamento:** exige token de autorização do titular. As 5 primeiras do mês são gratuitas, e depois cada uma custa R$ 3,6617.
- **Datavalid:** validação de identidade e biometria. Fora do escopo.

## Fontes gratuitas de CNPJ

### Dados abertos da Receita: a hospedagem mudou

- O endereço antigo `https://arquivos.receitafederal.gov.br/dados/cnpj/dados_abertos_cnpj/` devolve 404. O `dadosabertos.rfb.gov.br` não conecta.
- O novo host é um compartilhamento público Nextcloud ("SERPRO+"): https://arquivos.receitafederal.gov.br/index.php/s/YggdBLfdninEJX9
  - Também é acessível por WebDAV em `https://arquivos.receitafederal.gov.br/public.php/webdav/`, com o token como usuário de basic auth.
- As pastas são mensais. A mais recente hoje é 2026-09.
- Os dados de regime tributário ficam em outro compartilhamento, de token MPPfFit7g7zdA8C.
- Verificado com um PROPFIND ao vivo e no código do minha-receita: https://codeberg.org/cuducos/minha-receita/src/branch/main/download/webdav.go

### APIs públicas

- **minha-receita**
  - Escrito em Go, com licença MIT (LICENSE © 2021 Eduardo Vicente Gonçalves) e auto-hospedável.
  - Mudou do GitHub, hoje arquivado com README apontando para o Codeberg, para https://codeberg.org/cuducos/minha-receita. Última atualização em 01/10/2026.
  - Instância pública: minhareceita.org.
- **BrasilAPI**
  - O `/api/cnpj/v1/{cnpj}` só repassa para `https://minhareceita.org/{cnpj}`: https://github.com/BrasilAPI/BrasilAPI/blob/main/services/cnpj.js
  - Licença MIT, escrita em JavaScript. Aceita o padrão alfanumérico.
- **publica.cnpj.ws**
  - Limite de 3 requisições por minuto por IP.
  - Mais de 360 respostas 429 em uma hora bloqueiam o IP por 1 hora, e o bloqueio se renova se as tentativas continuarem.
  - Alguns campos são omitidos a pedido das empresas.
  - Fonte: https://docs.cnpj.ws/referencia-de-api/api-publica/limitacoes
- **CNPJá open**
  - Endereço: `https://open.cnpja.com/office/{cnpj}`, que respondeu 200.
  - Limite de 5 por minuto por IP sem login, segundo os termos de uso: https://cnpja.com/subscription/tos
  - Não verificado: a página de referência oficial devolveu 429.

## Certidões

- **CND federal (RFB e PGFN)**
  - Existe API oficial paga, a Serpro Consulta CND (produto consultacnd).
    - Verifica se há certidão válida ou pede uma nova.
    - Devolve código de controle, tipo (Negativa ou Positiva com efeitos de Negativa), validade, data e hora de emissão e o PDF.
    - Atende PF, PJ e imóvel rural.
  - Preço por requisição: R$ 0,8788 até 5.000 por mês e R$ 0,8265 de 5.001 a 10.000. Cai a R$ 0,6173 de 100.001 a 500.000 e a R$ 0,2511 acima de 10 milhões.
  - Não faz parte do Integra Contador. O Sitfis de lá é só o relatório de situação.
  - O caminho gratuito é o portal da RFB, sem API. Em julho de 2025 a Receita unificou emissão e consulta de certidões (relato de terceiros): https://coad.com.br/home/noticias-detalhe/132818/receita-federal-lanca-novo-servico-digital-de-emissao-e-consulta-de-certidao-negativa
- **CNDT (TST, cndt-certidao.tst.jus.br)**
  - Só portal web.
  - Existem raspadores de terceiros (relato de terceiros):
    - https://infosimples.com/consultas/tribunal-tst-validacao-cndt/
    - https://www.directd.com.br/apis/tst-cndt-debitos-trabalhistas
- **CRF do FGTS (consulta-crf.caixa.gov.br)**
  - Sem API pública. O formulário tem captcha.
  - A Circular Caixa 392 prevê convênio para órgãos consultarem o sistema do FGTS direto (relato de terceiros): https://www.normaslegais.com.br/legislacao/circularcef392.htm
  - Fora isso, só terceiros (relato de terceiros): https://infosimples.com/consultas/caixa-regularidade/
- **CND estadual e municipal**
  - Só portais.
  - Existem raspadores de terceiros, por exemplo (relato de terceiros): https://infosimples.com/consultas/sefaz-ap-certidao-debitos/

## DARF

- O caminho oficial é o Integra-Sicalc (DOC/pt/catalogo_de_servicos/):
  - CONSOLIDARGERARDARF51 devolve o PDF e GERARDARFCODBARRA53 o código de barras. Os dois são cobrados como Emissão, R$ 0,32 na faixa 1.
  - CONSOLIDAR54 calcula os acréscimos sem emitir e é cobrado como Consulta.
  - CONSULTAAPOIORECEITAS52 consulta códigos de receita, em Apoiar, gratuito.
- Não exige procuração.
- O DAE doméstico está fora do escopo.

## Clientes existentes

- Não há cliente Go para o Integra Contador.
  - A busca no GitHub com language:go e no pkg.go.dev não achou nada.
  - O único resultado Go para "serpro" é `bitbucket.org/foxsuporte/lib-serpro-go/serpro-authorizer`, sem licença.
- Clientes em outras linguagens:

| Repositório | Linguagem | Licença | Observação |
|---|---|---|---|
| github.com/MarlonSantosDev/serpro_integra_contador_api | Dart | MIT | o mais completo, atualizado em 07/2026 |
| github.com/GabrielHCamargo/SDK-Integra-Contador | Python | Apache-2.0 | |
| github.com/danilorc/integra-contador-opensource | JavaScript | MIT | |
| github.com/lucas-bogos/xsign | TypeScript | MIT | assina e valida o XML do Termo de Autorização |
| github.com/AndreP05/API-serpro, github.com/Renata5207418/PGDAS-IntegraContador | Python, TypeScript | sem licença | não podem ser usados |

- Os exemplos oficiais ficam em DOC/pt/modelos/: autenticação na loja em .NET e Python, e assinador em .NET e PHP.
- MIT e Apache-2.0 são compatíveis com a GPL-3.0 do nanci.
- Um cliente Go é pequeno: um POST de autenticação com mTLS e, depois, um POST com envelope JSON genérico por caminho.

## Não encontrado

- Se o EVENTOSATUALIZACAO dispensa a procuração quando a empresa monitora a si mesma.
- Limite geral de requisições do Integra Contador, fora o EVENTOSATUALIZACAO.
- Se a Serpro Consulta NF-e devolve o XML original assinado. Pela documentação, parece que não.
- Limite publicado da BrasilAPI e do minhareceita.org.
- Limite gratuito de 3 por minuto do ReceitaWS. As páginas só renderizam com JavaScript.
- API oficial de CNDT, CRF do FGTS e CND estadual ou municipal.
- API pública do Sicalc web fora do Integra Contador.
- Cliente Go com licença para qualquer API do Serpro.
