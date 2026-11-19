# SEFAZ: NSU faltante e consNSU

Levantado em 18/11/2026.

Responde: quando a distribuição de NF-e ou CT-e pula um NSU, o que a NT manda fazer e o que o `consNSU` responde.

Todas as fontes foram consultadas em 18/11/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fornecedores, fóruns ou imprensa, sem confirmação em texto oficial.

## NF-e: NFeDistribuicaoDFe

Norma: NT 2014.002, versão 1.40, publicada em 03/07/2026 no Portal da NF-e: https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg= (lido como texto com `pdftotext`; é o mesmo arquivo da cópia citada em `2026-10-07-sefaz-dfe.md`).

### Sequência de NSU

- O NSU é gerado "para cada CNPJ ou CPF" (§3.11.3, p. 12), de 14 dígitos: as aplicações que consultam "o mesmo CNPJ" devem seguir a mesma sequência (§3.11.4.1, p. 15). A raiz do CNPJ só entra na regra de continuidade dos 60 dias (§3.4, Obs. 2, p. 7) e na autenticação pelo certificado.
- A sequência não tem buracos: "Considerando que o Ambiente Nacional gera NSU sem lacunas [...] A identificação de alguma lacuna [...] indica que houve alguma falha" (§3.6, p. 9); o retorno deve conter "uma sequência de numeração sem intervalos" (§3.4, p. 7).
- O que fazer com a lacuna: "o interessado deve consultar pontualmente os NSU identificados como faltantes [...] informando o NSU desejado [...] na tag consNSU" (§3.6). O campo A09 (p. 6) diz o mesmo: "Geralmente esta consulta será utilizada quando identificado pelo interessado um NSU faltante [...] esta consulta fechará a lacuna".

### cStat 656 e o ultNSU

- Desde a versão 1.14, "o número da última consulta realizada para o CNPJ de 14 dígitos" volta no XML da rejeição 656, só no `distNSU` (§3.11.4.1, p. 15). A versão 1.14 entrou em produção em 24/03/2022 (relato de terceiros: https://www.projetoacbr.com.br/forum/topic/67494-nt2014002-v113-a-v115-web-service-de-distribui%C3%A7%C3%A3o-de-documentos-fiscais-eletr%C3%B4nicos-publicado-em-260520222/).
- "Se diversas aplicações do mesmo ator [...] efetuarem consultas por NSU para o mesmo CNPJ [...] devem seguir a mesma sequência [...] Caso contrário [...] uso indevido" (mesma página).
- Leitura do Nanci, não escrita na NT: quando outro programa (o ERP do contador) avançou a sequência, o 656 traz uma posição à frente do cursor do Nanci, e os NSUs entre os dois só chegaram ao outro programa. A NT não põe condição de "já consumido" no `consNSU`, só a janela de 90 dias e o limite por hora, então eles continuam pedíveis um a um.

### Respostas do consNSU (§3.6, p. 9; §3.10.2, p. 11; tabela da seção 4, p. 15-16)

- 138 "Documento localizado": o documento do NSU.
- 137 "Nenhum documento localizado": "o Ambiente Nacional não gerou o NSU e o interessado deve desconsiderá-lo".
- 589 "Número do NSU informado superior ao maior NSU do Ambiente Nacional" (regra H06, que a nota 1 aplica a `distNSU` e `consNSU`).
- 656 "Consumo Indevido".
- Fora da janela de 90 dias: o texto diz só "Após este período não será possível recuperar", sem cStat. O 632 ("Solicitação fora de prazo") é da regra H16, que a nota 2 limita ao `consChNFe`.

### Limite por hora (§3.11.4.2, p. 15)

- "número limitado de consultas por chave de acesso ou NSU em um período de 1 hora [...] 656 – Consumo indevido – ultrapassou o limite de 20 consultas por hora. Nesse caso, o CNPJ é bloqueado por 1 hora". Uma nova consulta antes de passar a hora "zera o tempo".
- A NT cita os dois tipos de consulta juntos, sem dizer com todas as letras que dividem o mesmo contador.
- 20 por hora por CNPJ em janela deslizante, contando só as consultas que devolvem documento (relato de terceiros: https://nfe.io/docs/distribuicao-processamento-e-periodicidade/index.md, atualizado em 24/09/2026).

### Na prática

- Usuários viram de 3% a 5% de NSUs pulados num dia, e o `consNSU` desses NSUs respondeu 137; uma resposta do Serpro repassada no fórum diz que o NSU pulado "não será mais reaproveitado" (relato de terceiros, agosto de 2017: https://www.projetoacbr.com.br/forum/topic/38289-didstribuicaodfe-pulou-nsu-dia-11/).

## CT-e: CTeDistribuicaoDFe

Norma: NT 2015.002, versão 1.05, publicada em 13/11/2024 no Portal do CT-e: https://www.cte.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=LLSQYxGtwv0= (lista em https://www.cte.fazenda.gov.br/portal/listaConteudo.aspx?tipoConteudo=Y0nErnoZpsg=).

- O `consNSU` funciona como o da NF-e (§2.4 e §2.6), com os mesmos 137, 138, 589 e 656. Não há consulta por chave.
- Janela de 3 meses.
- Nenhum número de consultas por hora: §2.11 diz só que tentativas repetidas recebem 656. Um fornecedor diz que valem 20 por hora por CNPJ também no CT-e (relato de terceiros: https://nfe.io/docs/distribuicao-processamento-e-periodicidade/index.md).

## Não encontrado

- cStat oficial do `consNSU` para um NSU com mais de 90 dias.
- Texto oficial dizendo se o ultNSU do 656 reflete a posição de outro sistema e se os NSUs pulados assim continuam pedíveis (a leitura acima é dedução).
- Texto oficial dizendo que `consNSU` e `consChNFe` dividem o contador de 20, e se consultas rejeitadas ou com 137 contam.
- Limite por hora na NT do CT-e.
- Se o 656 de uma consulta pontual também bloqueia o `distNSU` (os fornecedores divergem; o "o CNPJ é bloqueado" da NT sugere que sim).
