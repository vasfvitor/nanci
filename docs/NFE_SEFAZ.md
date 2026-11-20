# Integração NF-e e Ambiente Nacional da SEFAZ

Detalhamento técnico sobre como o Nanci baixa NF-e (modelo 55) pela distribuição DF-e do Ambiente Nacional e registra a Manifestação do Destinatário.

---

## O que é a distribuição de NF-e

A SEFAZ mantém no Ambiente Nacional (AN) o webservice **NFeDistribuicaoDFe**, que entrega a um CNPJ os documentos em que ele aparece. Cada documento recebe um NSU (Número Sequencial Único) próprio daquele CNPJ, e a consulta funciona como uma fila: o Nanci envia o último NSU que já leu (`distNSU/ultNSU`) e recebe até 50 documentos seguintes, com os cursores atualizados (`ultNSU` e `maxNSU`). Fonte: NT 2014.002 v1.40 ([Portal da NF-e](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=)), resumida em `docs/pesquisa/2026-10-07-sefaz-dfe.md`.

- **Resumo (`resNFe`)**: dados básicos da nota (chave, emitente, valor, situação). Não é o documento fiscal.
- **Completa (`procNFe`)**: o XML autorizado da NF-e com o protocolo.
- **Eventos (`resEvento`, `procEventoNFe`)**: cancelamento, carta de correção e manifestações.

A fila guarda os documentos por até 90 dias depois da recepção no Ambiente Nacional (mesma NT, "Janela de 90 dias" na nota de pesquisa). Por isso, ao contrário da NFS-e, a política inicial da empresa (`--sync-start-policy`) **não** se aplica à NF-e: o primeiro pull traz o que a SEFAZ ainda tiver.

## O que a empresa recebe, por papel

| Papel | O que chega |
|---|---|
| `destinatario` | Todas as NF-e autorizadas contra o CNPJ, de qualquer UF. Primeiro chega o resumo; o XML completo só é distribuído depois da Ciência da Operação (ou de uma manifestação conclusiva). Cancelamentos chegam sem necessidade de manifestação. |
| `emitente` | A SEFAZ não distribui ao emitente as notas que ele mesmo emitiu. Uma nota só aparece com esse papel quando chega por outro motivo (por exemplo, o CNPJ também está no `autXML`). |
| `transportador` | NF-e em que o CNPJ é o transportador. |
| `autorizado` | NF-e em que o CNPJ foi informado no grupo `autXML` (pessoas autorizadas a obter o XML). |

Eventos de uma chave que a empresa ainda não tem localmente, no mesmo ambiente (`tpAmb`) do evento, são descartados, exceto os que a própria empresa assinou (suas manifestações), que ficam guardados e são ligados à nota quando ela chegar. A nota de outro ambiente não conta: um cancelamento de homologação não cancela a nota de produção com a mesma chave (decisão de 20/11/2026).

## Endpoints Consumidos

As chamadas são SOAP 1.2 sobre HTTPS com o certificado A1 da empresa. As URLs estão em `internal/sefaz/endpoints.go`; o ambiente `producao` da empresa usa produção (`tpAmb` 1) e `producao_restrita` usa homologação (`tpAmb` 2).

O ambiente pode ser trocado a qualquer momento, também depois da primeira sincronização: cada nota e cada evento guardam o seu `tpAmb`, e as telas, o `nfe status` e a exportação mostram só o ambiente atual da empresa. O cursor, o bloqueio da SEFAZ, a data da carga inicial e o orçamento por hora também são separados por ambiente, então um bloqueio recebido em produção não adia o pull em homologação, e continua valendo quando a empresa volta a produção.

A redefinição das NF-e (`nfe reset`) mantém o histórico das manifestações enviadas (`nfe_manifestacoes`), porque ele registra o que foi para a SEFAZ, e mantém um bloqueio em vigor, porque a SEFAZ continua bloqueando.

A distribuição exige o código IBGE da UF da empresa (`cUFAutor`); sem UF cadastrada, o pull falha antes de pedir a senha.

## Requisitos de TLS

Verificado em 23/09/2026:

- **TLS 1.2.** Os servidores do AN não negociam TLS 1.3.
- **Certificado do cliente por renegociação.** O servidor (IIS) não pede o certificado no primeiro handshake; ele renegocia a conexão ao receber a requisição para um `.asmx` real e responde 403 se o cliente não apresentar certificado. A renegociação do Go só funciona em HTTP/1.1, então o transporte comum desliga o HTTP/2.
- **Cadeias públicas.** Os certificados dos servidores são emitidos por cadeias públicas (GlobalSign e Let's Encrypt/ISRG), já confiáveis no sistema operacional. O Nanci não embute raízes ICP-Brasil e não desliga a verificação.

`nanci nfe testar-conexao` faz só o handshake TLS e não gasta consultas. Como a SEFAZ só pede o certificado do cliente na primeira consulta real, um teste bem-sucedido confirma a rede e a cadeia do servidor, mas não que a SEFAZ aceitará o certificado.

## Limites de consulta

A SEFAZ bloqueia o CNPJ que consulta demais com a rejeição **656 (consumo indevido)**, por 1 hora. As regras vigentes desde 10/03/2022 (NT 2014.002, conhecidas por relatos de terceiros; ver `docs/pesquisa/2026-10-07-sefaz-dfe.md`) são duas: `consNSU` e `consChNFe` somados têm limite de 20 por hora, e o `distNSU` tem de esperar 1 hora depois de um `cStat` 137 ou de `ultNSU` igual a `maxNSU`. O `distNSU` não entra no limite de 20 por hora; o Nanci o conta mesmo assim, por escolha conservadora. O Nanci aplica as regras antes de enviar:

| Regra | Comportamento |
|---|---|
| 20 consultas por hora | Orçamento por empresa, origem e ambiente, em janela móvel de 1 hora (`sync_requests`), que inclui o `distNSU` (escolha do Nanci). Cada requisição é registrada antes do envio, então tentativas que falham também contam. Esgotado o orçamento, o pull para com `rate_budget` e a origem fica bloqueada até a consulta mais antiga da janela completar 1 hora. |
| Fila em dia | Com `cStat` 137 (nenhum documento) ou `ultNSU` igual a `maxNSU`, o pull para com `caught_up` e a próxima consulta só é permitida 1 hora depois. |
| `cStat` 656 | O pull para com `consumo_indevido` e espera 1 hora. O `ultNSU` devolvido só é adotado se avançar o cursor. |
| Intervalo entre páginas | 2 segundos entre requisições do mesmo pull. |

O bloqueio fica em `company_sync_sources.blocked_until`, e um pull dentro desse intervalo é recusado antes de pedir a senha. Zerar o estado local de sincronização não remove o bloqueio, porque a SEFAZ continua bloqueando.

**Regra dos 60 dias.** O Ambiente Nacional só gera NSU para a raiz de CNPJ que consultou o `distNSU` nos últimos 60 dias. Depois de uma pausa maior, a primeira consulta devolve 137 e os documentos do período parado não chegam mais pela distribuição. A regra entrou na NT 2014.002 v1.10, em produção desde 10/11/2021; ela é conhecida só por relatos de terceiros, que concordam entre si, e não foi conferida no texto da NT. Quando a última consulta respondida pela SEFAZ (`sync_state.last_success_at` da origem `nfe` no ambiente atual) passa de **45 dias**, o `nfe status` e a página de NF-e mostram um aviso. Uma empresa que nunca sincronizou NF-e não recebe o aviso. O limite está em `sync.DistIdleWarningDays`. Consultas feitas por outro programa com o mesmo CNPJ, como o ERP do contador, também contam para a SEFAZ, mas o Nanci não as vê, então o aviso pode aparecer sem que haja risco real.

A distribuição nunca é reenviada automaticamente pelo transporte, já que cada envio conta no limite. Um documento que falha ao ser lido três vezes seguidas no mesmo NSU é guardado como não suportado e o cursor avança, para que um único documento não trave a fila.

## Recuperar NSUs faltantes

O Ambiente Nacional numera os documentos de cada CNPJ "sem lacunas", e a NT manda o cliente pedir cada NSU faltante com `consNSU` (NT 2014.002 v1.40, §3.6 e campo A09; `docs/pesquisa/2026-11-18-sefaz-consnsu.md`). O Nanci registra, a cada resposta da distribuição, os NSUs que o cursor passou sem receber documento (`sync_nsu_gaps`):

- **Salto do 656.** Desde a v1.14, o 656 devolve a posição da sequência do CNPJ no servidor, e o Nanci adota esse `ultNSU` quando ele está à frente do cursor. Os NSUs entre os dois foram entregues a outro programa que consulta o mesmo CNPJ, como o ERP do contador (leitura do Nanci, não escrita na NT).
- **Buraco num lote 138.** Um NSU do intervalo (cursor, `ultNSU`] que veio sem `docZip`.
- **Não são lacunas:** a caminhada a partir do NSU 0, que a distribuição começa no NSU mais antigo da janela; um 137, que não tem nada a entregar; uma pausa de mais de 60 dias, em que o Ambiente Nacional não gera NSU (ver "Regra dos 60 dias").

A recuperação (`nfe recuperar-nsu` e o botão **Recuperar NSUs**) pede cada NSU faltante, do mais antigo ao mais novo, sem mover o cursor:

- **Orçamento.** Cada NSU gasta 1 das 20 consultas da hora. Sem limite informado, a execução deixa 2 consultas para o pull (`sync.RecoverNSUReserve`). Como na busca pela chave, `consumo_indevido` e `rate_budget` recusam a execução, e a espera depois de `caught_up` não.
- **Respostas.** 138 grava o documento; 137 conta como "sem documento" (a NT: "o Ambiente Nacional não gerou o NSU e o interessado deve desconsiderá-lo"); 656 bloqueia a origem por 1 hora e para; 589 (NSU acima do `maxNSU`) e 632 (fora de prazo) recusam só aquele NSU, que conta como "sem documento"; outra rejeição (certificado, CNPJ, serviço parado) para com erro e deixa a lacuna onde está.
- **Janela.** Uma lacuna cujo NSU mais antigo pode ter passado de 90 dias (`sync.NSUGapWindow`) é encerrada sem consulta: a NT não dá cStat para um NSU fora da janela, só diz que "não será possível recuperar". Os NSUs pulados numa retomada depois de mais de 90 dias parados já nascem fora da janela.
- **Descartar.** Uma lacuna grande (milhares de NSUs de um 656) levaria dias a 20 consultas por hora. Descartar encerra todas as lacunas abertas sem consultar a SEFAZ; os documentos delas só entram importando o XML.

## Manifestação do Destinatário

Só a empresa **destinatária** de uma NF-e **autorizada** pode manifestar. Os quatro eventos:

| Evento | `tpEvento` | Natureza |
|---|---|---|
| Ciência da Operação | 210210 | Não conclusiva. Libera o XML completo na distribuição. |
| Confirmação da Operação | 210200 | Conclusiva. |
| Desconhecimento da Operação | 210220 | Conclusiva. |
| Operação não Realizada | 210240 | Conclusiva; exige justificativa de 15 a 255 caracteres. |

A SEFAZ aceita registrar cada evento conclusivo até duas vezes, e vale só o mais recente; essa retificação pode ser feita em até 30 dias da primeira manifestação (cláusula 15ª-C, §§ 2º e 3º, do [Ajuste SINIEF 07/05](https://www.confaz.fazenda.gov.br/legislacao/ajustes/2005/AJ007_05), com o § 2º na redação do Ajuste SINIEF 43/23, em vigor desde 01/08/2024; conferido em 20/11/2026). O Nanci envia uma só manifestação conclusiva por NF-e e não retifica (`nfe.ConclusiveBlockReason`).

Nada é enviado sem confirmação explícita. A ciência vai em lotes de até 20 eventos; as conclusivas, nota a nota. Resultados por nota: `registrada` (`cStat` 135/136), `já registrada` (573 duplicidade), `rejeitada` (outro `cStat`, com o `xMotivo` da SEFAZ) e `não enviada` (o lote não teve resposta). Um lote sem resposta HTTP é reenviado uma vez; se ele já tinha chegado, a SEFAZ responde 573 e a nota conta como já registrada. Se continuar sem resposta, o envio dos lotes seguintes é interrompido e as notas não enviadas podem ser enviadas de novo.

Uma ciência respondida com 655 conta como rejeitada, com a mensagem "NF-e já possui manifestação conclusiva". A SEFAZ não registrou a ciência, então o Nanci não grava evento nem muda a manifestação da nota.

### Prazos

Os prazos contam da autorização da NF-e (ou da emissão, quando a data de autorização não está disponível). São só avisos; o Nanci **não bloqueia** nenhuma manifestação por prazo.

- **Ciência:** a nota sem nenhuma manifestação passa a ser marcada como "ciência atrasada" 10 dias após a autorização.
- **Conclusiva:** Confirmação, Desconhecimento e Operação não Realizada podem ser registradas em até 90 dias da autorização, com alerta a partir de 30 dias antes. A SEFAZ rejeita um evento fora do prazo com `cStat` 596, e o `xMotivo` é mostrado.
- **Confirmação tácita:** passados os 90 dias sem nenhum evento conclusivo, a operação é considerada ocorrida, com os mesmos efeitos da Confirmação da Operação. A Ciência da Operação não interrompe esse prazo nem impede a presunção. O Nanci mostra essas notas como "confirmada tacitamente".

Fontes: cláusula 15ª-C do Ajuste SINIEF 07/05, na redação dos Ajustes SINIEF 11/22 e 14/26 (este em vigor desde 1º de junho de 2026), e a NT 2020.001 v1.60 para a rejeição 596.

### Assinatura

Cada evento é assinado no perfil XMLDSig da NF-e: assinatura envelopada sobre `infEvento`, canonicalização C14N 1.0 inclusiva, digest SHA-1, assinatura RSA-SHA1 e o certificado da empresa em `KeyInfo`. O Nanci escreve o `infEvento` já na forma canônica, então usa só a biblioteca padrão do Go (`crypto/sha1` e `crypto/rsa`), sem biblioteca de C14N. O formato foi conferido contra um evento de ciência real aceito pela SEFAZ: o `infEvento` gerado é idêntico byte a byte e produz o mesmo `DigestValue` (`TestInfEventoDigest_RealEvent` em `internal/sefaz/xmldsig_test.go`). O teste opcional `go test -tags xmlsec ./internal/sefaz/` verifica a assinatura com o `xmlsec1`.

## Importação de XML

A empresa pode guardar no Nanci XMLs de NF-e que já tem em mãos, sem consultar a SEFAZ e sem gastar o limite por hora: as notas que ela mesma emitiu (a SEFAZ não as distribui ao emitente), as de mais de 90 dias, que já saíram da fila, ou as que o fornecedor enviou por e-mail. São recusados: arquivo ilegível ou maior que 20 MiB, uma `NFe` sem o protocolo de autorização, XML de outro ambiente, nota em que a empresa não aparece em papel nenhum, e evento de uma nota que a empresa não tem, a menos que a própria empresa seja a autora do evento, como na distribuição.

## Busca pela chave

A busca pela chave (`consChNFe`, no mesmo NFeDistribuicaoDFe) serve para uma nota que a empresa sabe que existe e que ainda não chegou, ou para trazer o XML completo depois da Ciência da Operação sem esperar o próximo pull. A SEFAZ devolve a nota sem os eventos.

- **Orçamento.** Cada busca gasta 1 das 20 consultas por hora da origem `nfe`. Um bloqueio por `consumo_indevido` ou `rate_budget` recusa a busca antes de pedir a senha; a hora de espera depois de `caught_up` não, porque ela vale só para o `distNSU`. Um 137 ("NF-e não localizada ou sem acesso para a empresa") também gasta a consulta.
- **Rejeições:** 640 (a empresa não é destinatária, transportadora nem autorizada nesta NF-e), 641 (a SEFAZ não entrega ao emitente a própria NF-e; o caminho é importar o XML), 632 (chave fora da janela de 90 dias), 653 (NF-e cancelada), 654 (NF-e denegada) e 618 (modelo não aceito). Os códigos estão em `docs/pesquisa/2026-10-07-sefaz-dfe.md` (NT 2014.002).

## Exportação

- A NF-e denegada não tem DANFE (decisão de 20/11/2026): o `fiscal-renderer` v0.8.1 não tem marca para ela, e o PDF passaria por uma nota válida. O ZIP de DANFE a deixa de fora sem marcar como exportada, e o DANFE avulso a recusa.
- A exportação incremental volta a trazer a nota quando o hash do XML ou a quantidade de eventos muda. Fica uma lacuna: um `resEvento` que depois chega como `procEvento` atualiza o mesmo registro (mesma chave, `tpEvento` e `nSeqEvento`), sem mudar a contagem, então o evento completo só sai numa exportação sem `--incremental`.

## CNPJ alfanumérico

Conferido em 07/10/2026 contra os textos oficiais. Vale para NF-e, CT-e e NFS-e; fica aqui porque a regra da chave nasce na NT da NF-e.

O que os textos dizem:

- **NT Conjunta 2025.001 v1.00** (ENCAT e RFB, 25/04/2025; homologação 06/04/2026, produção 06/07/2026), para NF-e, NFC-e, CT-e, CT-e OS, GTV-e, MDF-e, BP-e, NF3e e NFCom ([Portal da NF-e](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=5ZkvIZt10mQ%3D)):
  - O CNPJ continua com 14 posições: as 12 primeiras aceitam letras maiúsculas e as 2 últimas, os dígitos verificadores, são numéricas (`[A-Z0-9]{12}[0-9]{2}`). O DV é o módulo 11 de sempre, com cada caractere valendo o código ASCII menos 48 (`A` = 17). Exemplo da NT: `12.ABC.345/01DE-35`.
  - A chave de acesso continua com 44 posições e **não** é convertida em número: as letras do CNPJ vão na própria chave, que segue `[0-9]{6}[A-Z0-9]{12}[0-9]{26}` (letras só nas posições 7 a 18).
  - O DV da chave troca cada um dos caracteres pelo código ASCII menos 48 e aplica o módulo 11 de sempre: pesos de 2 a 9 da direita para a esquerda, e o resultado 10 ou 11 vira 0 (rotina do Anexo II). Uma chave só com números mantém o DV de antes.
  - A NT cita a possível exclusão das letras I, O, U, Q e F, mas diz que ela "precisa ser confirmada".
  - O código de barras do DANFE e do DACTE passa a ser CODE-128 híbrido (C e A), porque o CODE-128C só codifica números.
- **NT 2014.002 v1.40** (distribuição da NF-e, produção desde 08/07/2026) muda os campos de CNPJ de numérico para caractere ([cópia de terceiros](https://www.reformatributaria.com/wp-content/uploads/2026/07/NT2014.002_v1.40-WsNFeDistribuicaoDFe-CNPJ-alfa.pdf), ver `docs/pesquisa/2026-10-07-sefaz-dfe.md`).
- **NT SE/CGNFS-e 009** (v1.0 de 04/06/2026 e v1.01 de 01/10/2026, [portal da NFS-e](https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/rtc/nota-tecnica-009-se-cgnfs-e-v-1-01.pdf)) só muda o tipo de todos os campos CNPJ de N para C. Não fala da chave nem do DV dela. Os esquemas de produção restrita de 27/07/2026 (`esquemas-nfse-rtc-v1-01-20260727.zip`) trazem `TSCNPJ` `[0-9A-Z]{14}`, `TSIdNFSe` `NFS[0-9]{9}[0-9A-Z]{14}[0-9]{27}` (letras só na inscrição federal, posições 10 a 23 da chave), `TSIdDPS` com letras só quando o tipo de inscrição é 2 (CNPJ) e `TSChaveNFSe` `[0-9]{6}([0-9A-Z]{14})[0-9]{30}`. Os esquemas listados em "Documentação atual" (v1.01 de 09/02/2026) ainda são só numéricos.

No código, `dfe.ParseAccessKey` (NF-e e CT-e) exige DV válido de um CNPJ com letras, o que também barra letras nas posições 19 e 20, como no padrão da NT.

Em aberto:

- **DV da chave de NFS-e com letras.** Nenhum texto da NFS-e o define. O Nanci nunca conferiu o DV da chave de NFS-e e continua sem conferir.
- **`TSChaveNFSe` contradiz `TSIdNFSe`.** O primeiro põe as letras nas posições 7 a 20; o segundo e a regra de formação, nas posições 10 a 23. O Nanci segue o `TSIdNFSe`. Rever quando os esquemas com CNPJ alfanumérico chegarem à "Documentação atual".
- **Letras vedadas.** A NT 2025.001 v1.00 não confirma a exclusão de I, O, U, Q e F; o Nanci aceita qualquer letra de A a Z.
- **CNPJ alfanumérico solto no log exportado.** Fora de uma chave e sem pontuação, continua sem máscara, para não mascarar identificadores hexadecimais (decisão do teste `raw alphanumeric untouched`).
- **Código de barras.** O DANFE e o DACTE saem pelo `fiscal-renderer` v0.8.1 desde 16/10/2026; não foi verificado se ele gera o CODE-128 híbrido para chave com CNPJ alfanumérico. Conferir com um XML real quando houver.
- **Sem chave real nos testes.** Os testes usam o CNPJ do exemplo da NT; não há XML real de emitente com CNPJ alfanumérico.

## Fora do escopo

- NFC-e (modelo 65), NFCom, NF3e e CF-e SAT.
- Recuperar pela SEFAZ as notas emitidas pela própria empresa; o XML delas pode ser importado (ver "Importação de XML").

## Atribuição

Partes de `internal/sefaz` (montagem e leitura do `distDFeInt`, envelope SOAP, estruturas de resposta e o formato do assinador) foram adaptadas de [gonfe](https://github.com/mschunke/gonfe), sob licença MIT. Os arquivos adaptados trazem um cabeçalho de atribuição, e a licença está em [`third_party/gonfe/LICENSE`](../third_party/gonfe/LICENSE).

## Referências

- [Portal Nacional da NF-e](https://www.nfe.fazenda.gov.br/portal/principal.aspx) (Notas Técnicas, em especial a NT 2014.002 da distribuição DF-e).
- [Regras de consumo indevido para DF-e (NS Tecnologia)](https://blog.nstecnologia.com.br/regras-de-consumo-indevido-para-dfe/).
- [Documentação de métodos do sped-nfe](https://github.com/nfephp-org/sped-nfe/tree/master/docs/metodos), usada só como referência de comportamento.
