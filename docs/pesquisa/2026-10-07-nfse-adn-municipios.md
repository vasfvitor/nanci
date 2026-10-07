# NFS-e: Ambiente de Dados Nacional, Sefin Nacional e municípios

Levantado em 07/10/2026.

Responde: o que o ADN e o Sistema Nacional NFS-e entregam para a empresa, o que mudou em 2025-2026 e como ficam as cidades que mantêm sistema próprio.

Todas as fontes foram consultadas em 07/10/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fornecedores, fóruns ou imprensa.

## API de contribuintes do ADN

Fonte principal: Manual dos Contribuintes, APIs do ADN, versão 1.0 de 12/02/2026: https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/documentacao-atual/manual-contribuintes-apis-adn-sistema-nacional-nfse.pdf

### Papéis

- O manual diz que o contribuinte consulta os documentos "em que figure como emitente, tomador ou intermediário".
  - Logo, as NFS-e emitidas pela empresa como prestadora também voltam.
- Um tópico antigo do TabNews (2023-2024) dizia que só as notas recebidas voltavam, o que contradiz o manual (relato de terceiros): https://www.tabnews.com.br/Crazynds/6930df61-f915-4476-980f-6ce69d7197a7

### Métodos documentados

- GET /DFe/{NSU} devolve o DF-e daquele NSU.
- GET /NFSe/{ChaveAcesso}/Eventos devolve os eventos de uma chave.

### Novidade de 2026: consulta por outro estabelecimento

- Um certificado com a mesma raiz de CNPJ pode consultar outro estabelecimento.
- Há um parâmetro novo no NSU para informar o "CNPJ de consulta", e o sistema valida a raiz.
- O código do ACBr chama esse parâmetro de ?cnpjConsulta= (relato de terceiros): https://www.projetoacbr.com.br/forum/topic/86917-captura-de-nfse-emitidos-contra-um-cnpj/

### Endereços

- Swagger da produção restrita: https://adn.producaorestrita.nfse.gov.br/contribuintes/docs/index.html
- Produção: https://adn.nfse.gov.br/contribuintes/DFe/{NSU} (relato de terceiros): https://www.projetoacbr.com.br/forum/topic/74320-acesso-%C3%A0-api-para-baixar-as-nfs-e-nacional-adn/
- O caminho que funciona usa o plural "contribuintes". Um Swagger antigo mostrava o singular.
- Os dois hosts do Swagger derrubaram a conexão sem certificado de cliente. Isso é compatível com mTLS, mas não foi verificado.

### Limites

- Lote de 50 documentos por chamada, só pelo TabNews (relato de terceiros). O manual oficial não traz o número.
- Um fornecedor relata HTTP 429 com tempo de espera (relato de terceiros): https://nfe.io/docs/distribuicao-processamento-e-periodicidade/
  - A espera de 1 hora quando não há documento novo é escolha do fornecedor, copiada da NF-e. Não é regra do ADN.

## Sefin Nacional, API de emissão

Fonte: Manual dos Contribuintes, APIs do Emissor Público Nacional. O arquivo diz versão 1.2 de out/2025, e o histórico interno começa na 1.0 de 17/03/2025: https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/documentacao-atual/manual-contribuintes-emissor-publico-api-sistema-nacional-nfs-e-v1-2-out2025.pdf/@@download/file

### Emissão e consulta

- POST /nfse é síncrono: recebe a DPS e devolve o XML da NFS-e ou o erro.
  - Uma DPS que informa a chave a substituir também gera o evento "Cancelamento por Substituição".
- GET /nfse/{chaveAcesso} busca a NFS-e pela chave.
- GET e HEAD /dps/{id}:
  - O id é formado por IBGE (7) + tipo de inscrição (1) + inscrição (14) + série (5) + número (15).
  - O GET devolve a chave só ao prestador, ao tomador ou ao intermediário.
  - O HEAD só informa se a NFS-e existe, para qualquer certificado válido.

### Eventos

- POST /nfse/{chave}/eventos recebe um envelope JSON com XML assinado.
- GET /nfse/{chave}/eventos[/{tipoEvento}[/{numSeq}]] lê os eventos.

### Parâmetros municipais

- GET /parametros_municipais/{cMun}/convenio
- GET /parametros_municipais/{cMun}/{codigoServico}, para alíquotas, regimes especiais e deduções.
- GET /parametros_municipais/{cMun}/{CPF/CNPJ}, listado duas vezes no manual: uma para retenções e outra para benefícios.
- Um usuário de fórum chamou a API em adn.nfse.gov.br e recebeu alíquotas nulas para Porto Alegre (relato de terceiros): https://portalspedbrasil.com.br/?p=20727

### Hosts e evento de cancelamento

- Produção: sefin.nfse.gov.br/SefinNacional. Produção restrita: sefin.producaorestrita.nfse.gov.br/SefinNacional. Os dois vêm só de posts da comunidade (relato de terceiros).
- O código do evento de cancelamento, e110001, vem da nfe.io (relato de terceiros).

## Mudanças técnicas de 2026

### DANFSe

- A NT SE/CGNFS-e 008/2026, de 05/05/2026, fixou um leiaute nacional obrigatório de DANFSe para os sistemas emissores.
- A mesma NT anunciou o fim da API atual de geração de DANFSe a partir de 01/07/2026: https://www.gov.br/nfse/pt-br/noticias/se-cgnfs-e-publica-nota-tecnica-no-008-2026-com-regras-para-emissao-do-danfse
- A versão 1.02, de 14/07/2026, adiou o prazo para 03/08/2026: https://www.gov.br/nfse/pt-br/noticias/danfse-novos-ajustes-de-leiaute-e-prorrogacao-do-prazo-para-adequacao
- Os comunicados não dizem em que host ficava a API descontinuada. Se o nanci busca PDF de DANFSe no ADN, conferir.

### CNPJ alfanumérico

- A NT 009, de junho de 2026, muda todos os campos de CNPJ da DPS e da NFS-e de numérico para caractere.
  - Cópia de terceiros: https://www.reformatributaria.com/wp-content/uploads/2026/06/nt-009-se-cgnfse-v1-0-1.pdf
- Produção restrita desde 27/07/2026 e produção desde 10/08/2026, segundo o CRCMS citando o Portal Brasil: https://crcms.org.br/?p=37244

## Obrigações

### LC 214/2025, art. 62

- O §1º diz que, a partir de 01/01/2026, municípios e Distrito Federal devem escolher uma de duas opções:
  - permitir a emissão no ambiente nacional;
  - manter emissor próprio e compartilhar os documentos com o ADN no leiaute padrão.
- Usar o emissor nacional não é obrigatório.
- O §7º permite à União suspender transferências voluntárias ao município que não cumprir.
- Fontes:
  - Apresentação da RFB: https://www.gov.br/receitafederal/pt-br/centrais-de-conteudo/publicacoes/apresentacoes/reforma-tributaria-do-consumo/nfs-e_live_rj_es.pdf/@@download/file
  - Texto do artigo: https://www.legjur.com/legislacao/art/lec_00002142025-62
- O Planalto estava fora do ar, então o texto do artigo não foi conferido lá.

### Adesão

- O gov.br diz que todos os 5.571 entes aderiram, incluindo todas as capitais: https://www.gov.br/nfse/pt-br/municipios/monitoramento-adesoes
- A página não tem data. O nome da planilha vinculada contém 20260928.
- Uma matéria sem data, do fim de 2025, falava em só 1.843 municípios plenamente operacionais (relato de terceiros): https://amdjus.com.br/?p=14220

### Notas técnicas de IBS/CBS

- NT 003, de julho de 2025: https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/rtc/nt-003-1-2-se-cgnfse-novo-layout-rtc.pdf
- NT 004, de 19/08/2025, é o leiaute base de 2026. A versão 2.0, de 10/12/2025, suspendeu a obrigatoriedade do grupo IBSCBS (relato de terceiros): https://documentacao.senior.com.br/exigenciaslegais/noticias/federal/2025/2025-12-11-nota-tecnica-reforma-tributaria-adequacao-na-nfse-nacional
- NT 005, de novembro de 2025.
- NT 007: https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/rtc/nt-007-se-cgnfse-v1-0.pdf
- O Ato Conjunto RFB/CGIBS 4, de 30/07/2026, fixou novo calendário nacional dos campos de IBS/CBS. Fonte oficial, da SEEC-DF: https://www.economia.df.gov.br/documents/d/seec/_notas-fiscais_-conheca-o-cronograma-da-obrigatoriedade-de-emissao_-1-

### Simples Nacional: Resolução CGSN 191/2026

Resolução de 04/08/2026: https://fenacon.org.br/reforma-tributaria/simples-nacional-nfs-e-nacional-sera-obrigatoria-para-me-e-epp-a-partir-de-1o-de-novembro-de-2026/
- ME e EPP optantes passam a emitir pelo Emissor Nacional, web ou API, a partir de 01/11/2026.
- As regras de IBS/CBS para essas empresas começam em 01/01/2027.
- A resolução revogou a CGSN 189, que fixava 01/09/2026.

### MEI

- O padrão nacional é obrigatório desde 01/09/2023 para serviços prestados a empresas, pela Resolução CGSN 169/2022 com alterações (relato de terceiros):
  - https://fenacon.org.br/?p=24431
  - https://agenciasebrae.com.br/cultura-empreendedora/sebrae-esclarece-duvidas-sobre-nfs-e-no-padrao-nacional/

## As 10 maiores cidades por PIB

Base: PIB dos Municípios 2023 do IBGE, divulgado em 19/12/2025 (relato de terceiros): https://economicnewsbrasil.com.br/2025/12/19/pib-dos-municipios-2023-ibge/
- Fortaleza e Campinas ficam fora das 10 primeiras.
- Maricá é a 4ª, puxada pelo petróleo.

| # | Cidade | Situação em 2026 | Observações e fontes |
|---|---|---|---|
| 1 | São Paulo | Emissor próprio (Nota Paulistana). Converte e compartilha com o ADN desde 2026 (relato de terceiros) | Web service SOAP, ver abaixo. A CGSN 191 pode levar ME/EPP do Simples ao Emissor Nacional; não confirmado para SP |
| 2 | Rio de Janeiro | Emissor Nacional obrigatório desde 01/01/2026 (Decreto 56.921/2025) | A Nota Carioca segue para notas anteriores a 2026, a declaração mensal e a guia (relato de terceiros): https://convergenciadigital.com.br/mercado/rio-de-janeiro-adota-nota-fiscal-nacional-para-iss-a-partir-de-2026/ |
| 3 | Brasília (DF) | Emissor próprio (ISSnet), compartilha com o ADN | ABRASF 2.04 aceito até 30/09/2026. A partir de 01/10/2026, só o leiaute nacional, no web service próprio (nfse.fazenda.df.gov.br): https://static.fazenda.df.gov.br/arquivos/servico-1202/comunicado_NFSe_padrao_nacional_04-11-2025.pdf |
| 4 | Maricá | Emissor Nacional desde janeiro de 2026 | A Nota Maricá segue para notas anteriores a 2026: https://www.marica.rj.gov.br/wp-content/uploads/2025/12/JOM_1817_10-12-2025_.pdf |
| 5 | Belo Horizonte | Só Emissor Nacional (Portaria SMFA 75/2025, em fases até 01/01/2026, ajustada pela Portaria 88/2025) | Situação de 2026 não confirmada: https://prefeitura.pbh.gov.br/noticias/belo-horizonte-adere-ao-emissor-nacional-de-nota-fiscal-de-servico-eletronica |
| 6 | Manaus | Emissor Nacional desde 01/01/2026 (Decreto 6.743/2025; Portaria 3 SUBREC/SEMEF, abril de 2026) (relato de terceiros) | O DAM segue na Nota Manaus: https://www.coad.com.br/home/noticias-detalhe/136986/ |
| 7 | Curitiba | Só Emissor Nacional desde 01/01/2026 | O DAM é emitido pelo ISS-Curitiba: https://nota.curitiba.pr.gov.br/docs/Manual/Prestador/NFSE-NACIONAL_ManualDeIntegracao_Curitiba-v1.pdf |
| 8 | Osasco | Sistema próprio (EISS), manual de web service v2.1 de abril de 2026 | Compartilhamento com o ADN afirmado só pela nfe.io (relato de terceiros): https://nfe.io/docs/prefeituras-integradas/sao-paulo/osasco-sp-3534401/ |
| 9 | Porto Alegre | Só Emissor Nacional desde 01/11/2025 | A Nota Legal foi desativada: https://prefeitura.poa.br/smf/noticias/nfs-e-nacional-passa-ser-obrigatoria-em-porto-alegre-em-1o-de-novembro |
| 10 | Guarulhos | Sistema próprio (GissOnline). O Simples passa ao Emissor Nacional em 01/11/2026 (relato de terceiros) | Compartilhamento com o ADN não confirmado: https://notagateway.com.br/blog/guarulhos-sp-adia-obrigatoriedade-da-nfs-e-no-emissor-nacional-para-1o-de-novembro/ |

- Em seis das dez cidades (Rio, Maricá, Belo Horizonte, Manaus, Curitiba e Porto Alegre), a distribuição do ADN deve cobrir as notas desde 2026.
- Em São Paulo, Brasília, Osasco e Guarulhos, a cobertura depende de o município compartilhar corretamente.

## Web service de NFS-e de São Paulo

Fonte: manual do web service, versão 3.3.9 de outubro de 2026: https://notadomilhao.sf.prefeitura.sp.gov.br/wp-content/uploads/2026/10/NFe_Web_Service-v3.3.9.pdf

### Endpoints

- Síncrono: https://nfews.prefeitura.sp.gov.br/lotenfe.asmx
- Assíncrono: lotenfeasync.asmx.
- O endereço antigo, nfe.prefeitura.sp.gov.br/ws/lotenfe.asmx, não suporta o leiaute v2 e será desligado.

### ConsultaNFeRecebidas

- É feita para tomadores e intermediários.
- Entrada: CPF ou CNPJ, CCM opcional, data inicial, data final e número da página.
- Devolve até 50 notas por página. Continua-se paginando até vir uma página com menos de 50.

### Certificado e assinatura

- Certificado ICP-Brasil A1, A3 ou A4, usado na conexão TLS e na assinatura XML envelopada (RSA-SHA1, C14N).
- Contador ou usuário autorizado pode assinar as consultas.
- As mensagens têm limite de 500 KB.

### Outros serviços

- O mesmo web service tem métodos de guia de ISS por incidência: EmissaoGuiaAsync, ConsultaSituacaoGuia e ConsultaGuia.
- A NFTS declara serviços tomados de prestadores de fora de SP. Substituiu a antiga DES e tem manual de web service próprio, versão 3.1.
- O envio de lote em TXT foi desativado para fatos a partir de 01/01/2026: https://notadomilhao.sf.prefeitura.sp.gov.br/manuais/

## Provedores ABRASF

- O ABRASF v1 não tem consulta de serviços tomados.
- Os manuais v2.x definem ConsultarNfseServicoTomado, mas a implementação varia por cidade (relato de terceiros, ACBr):
  - https://www.projetoacbr.com.br/forum/topic/86917-captura-de-nfse-emitidos-contra-um-cnpj/
  - https://www.projetoacbr.com.br/forum/topic/50028-consulta-de-nfs-e-%E2%80%93-servi%C3%A7os-tomados/
- Betha: o manual v2.02 lista "Serviços Tomados ou Intermediados" (relato de terceiros).
- ISSNet/DF: um consultor do ACBr diz que o ABRASF 2.04 não tem esse serviço, o que conflita com o anterior (relato de terceiros).

## ISS: guias, livro fiscal e declarações

- **São Paulo:** guias pelo web service e serviços tomados pela NFTS, ambos descritos acima.
- **Rio de Janeiro:** declaração mensal de serviços prestados, com guia gerada na Nota Carioca a partir dos dados do ADN. Sem API encontrada.
- **Manaus e Curitiba:** o DAM continua nos sistemas locais. Sem API encontrada.
- **Módulo de Apuração Nacional (MAN):**
  - Está na produção restrita desde 14/04/2026 e é opcional para os municípios.
  - Gera uma guia nacional única (DNA), com valor mínimo de R$ 10 (relato de terceiros): https://fenacon.org.br/noticias/modulo-de-apuracao-nacional-para-o-issqn-e-disponibilizado-no-ambiente-de-producao-restrita-da-nfs-e/
- Livro fiscal: nenhuma API encontrada.

## Não encontrado

- Limite de chamadas, janela de retenção e tamanho de lote do ADN em fonte oficial.
- Se a API de parâmetros municipais exige certificado.
- Conteúdo do Swagger de produção. Os hosts parecem exigir mTLS.
- Host da API de DANFSe descontinuada pela NT 008/2026.
- Confirmação oficial dos hosts da Sefin Nacional e do código de evento de cancelamento e110001.
- Confirmação de que notas de São Paulo, Osasco e Guarulhos aparecem para o tomador na distribuição do ADN. Só um teste com nota real resolve.
- Número oficial de municípios que usam o emissor nacional, separado dos que só compartilham dados.
- Confirmação, por provedor, de ConsultarNfseServicoTomado em GINFES, IPM, e-Governe, WebISS e SimplISS.
- Data de produção e especificação de API do MAN.
- APIs de livro fiscal ou de guia de ISS fora de São Paulo.
