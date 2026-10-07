# eSocial, EFD-Reinf, SPED e Reforma Tributária

Levantado em 07/10/2026.

Responde: o que a empresa consegue ler, por API oficial e com o próprio certificado, de eSocial, EFD-Reinf, DCTFWeb, MIT, Simples, SPED e das novas plataformas da Reforma Tributária.

Todas as fontes foram consultadas em 07/10/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fornecedores, fóruns ou imprensa.

## eSocial

Fonte: Manual de Orientação do Desenvolvedor eSocial, versão 1.16: https://www.gov.br/esocial/pt-br/documentacao-tecnica/manuais/996775-manualorientacaodesenvolvedoresocialv1-16.pdf

### Envio e consulta de lote (SOAP)

- Envio: https://webservices.envio.esocial.gov.br/servicos/empregador/enviarloteeventos/WsEnviarLoteEventos.svc
- Consulta: https://webservices.consulta.esocial.gov.br/servicos/empregador/consultarloteeventos/WsConsultarLoteEventos.svc
- Cada lote tem no máximo 50 eventos.
- O resultado fica disponível por 30 dias. Depois disso, para obter o recibo original, reenvia-se o evento original ou usam-se os serviços de download abaixo.

### Download de eventos já enviados

Serviço de identificadores, WsConsultarIdentificadoresEventos.svc:
- Produção: https://webservices.download.esocial.gov.br/servicos/empregador/dwlcirurgico/WsConsultarIdentificadoresEventos.svc
- Produção restrita: o mesmo caminho no host webservices.producaorestrita.esocial.gov.br.
- Operações:
  - ConsultarIdentificadoresEventosEmpregador: tipo de evento "S-XXXX" e período, no formato AAAA-MM ou AAAA.
  - ConsultarIdentificadoresEventosTabela: tipo de evento, chave opcional (por exemplo codRubr=1;ideTabRubr=1) e dtIni e dtFim opcionais.
  - ConsultarIdentificadoresEventosTrabalhador: CPF do trabalhador e dtIni e dtFim, com intervalo máximo de 31 dias. Acima disso, erro 410.
- Cada chamada devolve até 50 pares de Id e nrRec. A paginação usa o dhUltimoEvtRetornado como próximo dtIni.

Serviço de download, WsSolicitarDownloadEventos.svc, no mesmo host e no mesmo caminho dwlcirurgico:
- SolicitarDownloadEventosPorId e SolicitarDownloadEventosPorNrRecibo aceitam de 1 a 50 itens.
- Devolvem o XML do evento junto com o recibo.

### Regras dos dois serviços

- Exigem certificado de cliente. O certificado da conexão não precisa de vínculo formal com o empregador.
- O pedido tem de ser assinado por uma destas opções; fora delas, erros 407 e 411:
  - e-CNPJ com a mesma raiz;
  - e-CPF do representante legal;
  - procurador cadastrado na procuração da RFB ou da CEF.
- Não aceitam pedidos do dia 1 ao dia 7 de cada mês.
- Só um pedido por vez por empregador.
- No máximo 10 chamadas por dia por empregador, somando os dois serviços.
- Só ficam disponíveis os eventos recebidos há mais de 1 hora.

### O que isso significa para o nanci

- Dá para baixar eventos enviados por qualquer pessoa, inclusive pelo contador ou por outro software, porque a consulta é por empregador e não por remetente.
- Dividindo as chamadas entre listar e baixar, o volume fica em torno de 250 eventos por dia.
  - Serve para carga incremental ou pontual.
  - Não serve para carga inicial de folha grande.

### Download pelo portal

- Pelo portal, sem API, o pedido cobre até 35 dias e gera arquivos ZIP guardados por 7 dias.
- Fonte de 2020 (relato de terceiros): https://cfc.org.br/noticias/esocial-download-para-facilitar-a-vida-do-empregador/

### Mudanças 2025-2026

- O leiaute vigente é o S-1.3, com a NT 06/2026 revisada e a NT 07/2026.
- Houve troca do padrão de certificado do servidor: produção restrita em 12/01/2026 e produção em 24/06/2026: https://www.gov.br/esocial/pt-br/noticias/prorrogacao-da-atualizacao-de-certificado-do-esocial-para-um-novo-padrao-de-seguranca
- Desde 01/05/2026, o FGTS devido em sentença trabalhista passa pelo FGTS Digital, com as bases informadas no evento S-2500.
- O "novo eSocial Simplificado" de 2026 se refere aos módulos de doméstico, MEI e segurado especial: https://www.gov.br/esocial/pt-br/noticias/governo-anuncia-novo-esocial-simplificado

## EFD-Reinf

Fonte: Manual de Orientação do Desenvolvedor EFD-Reinf, versão 2.8: https://www.gov.br/sped/pt-br/assuntos/escrituracoes-digitais/efd-reinf/manuais-e-documentos-tecnicos/manualorientacaodesenvolvedor-reinf-v2-8.pdf/@@display-file/file

### Endpoints

API REST com certificado de cliente.
- **Envio:** POST https://reinf.receita.economia.gov.br/recepcao/lotes, com limite de 54 MB.
- **Resultado do lote:** GET https://reinf.receita.economia.gov.br/consulta/lotes/{numeroProtocolo}.
  - Devolve o resultado e o recibo de cada evento.
  - HTTP 429 indica excesso de requisições.
- **Consulta de recibo:** GET https://reinf.receita.economia.gov.br/consulta/reciboevento/{endpoint}
  - Cobre R1000, R1050, R1070, R2010 a R2099 (com R2055), R3010, R4010 (com e sem CPF), R4020, R4040, R4080 e R4099.
  - Exemplo: /R2099/{tpInsc}/{nrInsc}/{perApur}, usando a raiz do CNPJ com 8 dígitos.
  - Devolve a hora de recepção, o número do recibo, o Id do evento, a situação (ativo, retificado ou excluído) e o canal de entrada (web service, portal ou API).
- **Produção restrita:** host pre-reinf. O Swagger só existe lá: https://pre-reinf.receita.economia.gov.br/consulta/swagger/index.html

### Quem pode chamar

- e-CNPJ com a mesma raiz.
- e-CPF do representante legal.
- Procurador com procuração da RFB outorgada no CNPJ da matriz.

### Limitações para leitura

- Não há serviço que baixe o XML de um evento, nem consulta por período que devolva conteúdo.
- Os totalizadores R-9001, R-9005, R-9011 e R-9015 só voltam no resultado do lote.
- Para recuperá-los depois, é preciso uma das duas opções:
  - ter o número do protocolo;
  - reenviar o evento idêntico, com o mesmo Id e o mesmo hash. A resposta é o código MS0022 com o recibo original.
- Resultado: dá para listar o que existe, inclusive eventos enviados por outros, mas não recuperar o conteúdo.

## DCTFWeb e MIT

- Não há web service oficial gratuito de leitura.
- O único caminho automatizado é o Integra Contador do Serpro, pago e por contrato. Detalhes em `2026-10-07-federal-serpro.md`.
- Serviços de 2025 no Integra Contador: encerrar a apuração do MIT, consultar a situação do encerramento e consultar apurações, inclusive por ano ou mês.
  - Fonte: https://serpro.gov.br/menu/noticias/noticias-2025/integra-contador-novas-funcionalidades
  - A página não carregou; o conteúdo foi visto pela busca.
- O MIT substituiu o antigo PGD DCTF a partir da competência de janeiro de 2025 (IN RFB 2.237/2024).

## Simples Nacional

- Não há API oficial gratuita para PGDAS-D, DEFIS ou DAS.
- Os caminhos são:
  - o Integra Contador, pago;
  - o portal do Simples, com código de acesso ou certificado;
  - o e-CAC.
- Fonte: https://www.gov.br/pt-br/servicos/declarar-apuracoes-mensais-do-simples-nacional
- A Consulta Optantes, em consopt.www8.receita.fazenda.gov.br, é pública mas tem captcha, então não dá para automatizar (relato de terceiros).
- As datas de opção e de exclusão do Simples e do MEI também estão nos dados abertos do CNPJ.

## SPED: EFD ICMS/IPI, EFD Contribuições, ECD e ECF

- Não há API.
- Os arquivos transmitidos só voltam pelo programa desktop ReceitanetBX, com o certificado da empresa ou com perfil de procurador: https://www.gov.br/receitafederal/pt-br/centrais-de-conteudo/download/receitanetbx/receitanetbx
- O recibo se recupera reenviando o arquivo idêntico pelo PVA: http://sped.rfb.gov.br/estatico/52/99B4B74C086931501E2B8C8389F55449E333AD/Roteiro%20para%20recuperar%20recibo%20de%20transmiss%C3%A3o.pdf

## Reforma Tributária: CBS e IBS

### Plataformas e calendário

- Portal principal: https://consumo.tributos.gov.br, em beta. Portal piloto: https://piloto-cbs.tributos.gov.br
- A plataforma da CBS roda em beta de 13/01/2026 a dezembro de 2026. O sistema definitivo começa em 2027.
- Os documentos fiscais reais entram automaticamente na apuração assistida. Em 2026 o pagamento é só simulado.
- O Decreto 12.955, de 29/04/2026, regulamenta a CBS.
- Manual da plataforma, de 21/05/2026: https://www.gov.br/receitafederal/pt-br/centrais-de-conteudo/publicacoes/manuais/reforma-tributaria-do-consumo/manual-plataforma-cbs-21-maio-2026-07h40.pdf/@@download/file

### API de apuração da CBS, versão 2

Documentação versão 1.1, de 17/09/2026: https://docs.receitafederal.gov.br/apuracao-cbs/

- **Débitos:** POST https://api.receitafederal.gov.br/apuracao-cbs/v2/debitos/{cnpj8}. A produção restrita usa /apuracao-cbs-prr/v2/...
  - É incremental: cada chamada devolve só o que mudou desde a anterior, numa janela de 8 dias.
  - A primeira chamada começa no dia 1 do mês corrente.
  - O JSON lista os débitos por chave de acesso, com valores como apurado, extinto e saldo devedor.
- **Situação:** GET .../v2/situacao/{tiquete}. Quando o pedido termina, devolve um link de download assinado, válido por 48 horas.
- **Calendário anunciado em 14/09/2026, tudo gratuito:**
  - débitos e créditos no início de outubro de 2026;
  - pagamentos e recolhimentos no início de novembro;
  - geração de DARF no fim de novembro.
  - Fonte: https://www.gov.br/fazenda/pt-br/assuntos/noticias/2026/setembro/receita-federal-publica-nova-documentacao-tecnica-das-apis-de-apuracao-de-cbs

### Restrições de acesso que pesam para o nanci

- **Autenticação:** OAuth2 client credentials, não o certificado.
  - ClientId e ClientSecret saem do serviço "Gerar Credencial" do portal, com login gov.br representando o CNPJ.
  - As credenciais foram renovadas em 24/08/2026, e uma credencial passou a cobrir todas as APIs: https://www.gov.br/receitafederal/pt-br/assuntos/noticias/2026/agosto/credenciais-de-acesso-as-api-da-reforma-tributaria-serao-atualizadas
- **Limite:** a chamada de abertura aceita no máximo 4 por dia.
- **Webhook obrigatório:** todo pedido tem de informar uma URL HTTPS pública, que o servidor confere com um HEAD antes de aceitar.
  - Depois dá para consultar a situação por polling.
  - Mesmo assim, um app desktop precisa de alguma URL pública que passe na conferência.
  - Fonte: https://docs.receitafederal.gov.br/apuracao-cbs/guiaapiassincrona.html
- **Fluxo antigo:** o manual de maio ainda descreve o fluxo de tíquete anterior, em /rtc/apuracao-cbs/v1, com token em api.receitafederal.gov.br/token. Parece substituído pela versão 2.

### Calculadora de CBS e IBS

- Oficialmente, não há API pública hospedada.
- O que existe é um componente offline de código aberto, que roda localmente e serve em localhost:8080/api.
- Documentação e Swagger: https://consumo.tributos.gov.br/servico/calcular-tributos-consumo/calculadora/documentacao

### IBS

- Não há API do Comitê Gestor (CGIBS) para a empresa ver a própria apuração de IBS.
- Há um piloto no Rio Grande do Sul sobre NF-e desde 05/01/2026, com 123 empresas (relato de terceiros).

### Split payment

- A plataforma pública e a sua OpenAPI são para prestadores de serviço de pagamento.
- Não há data de início obrigatória. Não interessa ao nanci.
- Fonte (relato de terceiros): https://www.reformatributaria.com/comite-gestor-do-ibs-reforma-tributaria/receita-e-comite-gestor-do-ibs-aprovam-manuais-da-plataforma-publica-do-split-payment/

### Obrigações de 2026

Ato Conjunto RFB/CGIBS nº 1/2025, de 23/12/2025 (relato de terceiros, resumo da SEFAZ-ES): https://sefaz.es.gov.br/Noticia/ato-conjunto-da-receita-federal-e-comite-gestor-do-ibs-define-obrigacoes-acessorias-para-ibs-e-cbs-em-2026
- O documento fiscal é obrigatório nas operações com IBS e CBS.
- A falta dos campos de IBS e CBS não gera multa até o primeiro dia do quarto mês depois da publicação da parte comum do regulamento.
- A apuração de 2026 é só informativa.

Também existem, na produção restrita, APIs de eventos da DeRE e do ReOps.

## Open Finance

- Fora do escopo.
- Só instituições autorizadas pelo Banco Central podem receber os dados (Resolução Conjunta 1/2020, art. 1º). As demais só por parceria com um participante (art. 36).
- Fonte: https://normativos.bcb.gov.br/Lists/Normativos/Attachments/51028/Res_Conj_0001_v6_P.pdf
- Uma ferramenta desktop com certificado teria de usar um agregador, ou importar arquivos do banco em OFX ou CNAB.

## Não encontrado

- Limite de quanto tempo atrás o download de eventos do eSocial alcança. A regra de "10 anos" não foi confirmada.
- Versão REST dos serviços do eSocial.
- Consulta avulsa de totalizadores no modelo REST da EFD-Reinf, e qualquer serviço que baixe o conteúdo de eventos já enviados.
- Web service oficial gratuito de leitura de DCTFWeb ou MIT.
- API oficial gratuita do Simples Nacional, e layout verificado do arquivo de Simples e MEI dos dados abertos.
- Serviço do e-CAC para pedir cópia de arquivos SPED.
- API do Comitê Gestor do IBS para a empresa consultar a própria apuração.
- Forma de usar a API de apuração da CBS com o certificado A1, ou sem webhook público.
- Sistema "novo eSocial Simplificado" oficial e com data, além dos módulos de doméstico, MEI e segurado especial.
