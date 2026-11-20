# Integração CT-e e Ambiente Nacional da SEFAZ

Detalhamento técnico sobre como o Nanci baixa CT-e (modelo 57, inclusive o CT-e Simplificado), CT-e OS (modelo 67) e GTV-e (modelo 64) pela distribuição DF-e do Ambiente Nacional.

---

## O que é a distribuição de CT-e

A SEFAZ mantém no Ambiente Nacional (AN) o webservice **CTeDistribuicaoDFe**, com o mesmo desenho do `NFeDistribuicaoDFe` descrito em [NFE_SEFAZ.md](NFE_SEFAZ.md): fila por NSU, até 50 documentos por consulta. A fila guarda os documentos por cerca de 3 meses, e a política inicial da empresa (`--sync-start-policy`) **não** se aplica ao CT-e: o primeiro pull traz o que a SEFAZ ainda tiver.

Diferenças em relação à NF-e:

- **Não há resumo.** O XML completo, com o protocolo de autorização, chega direto.
- **Não há manifestação** para liberar documentos. O único evento que o tomador pode enviar é a prestação do serviço em desacordo (veja abaixo).
- **Não há consulta por chave.** O serviço só aceita `distNSU` e `consNSU`.

Schemas distribuídos e lidos pelo Nanci:

| Schema | Documento |
|---|---|
| `procCTe` | CT-e, modelo 57. Em algumas versões também traz o CT-e Simplificado. |
| `procCTeOS` | CT-e OS (outros serviços), modelo 67. |
| `procGTVe` | GTV-e (guia de transporte de valores), modelo 64. |
| `procCTeSimp` | CT-e Simplificado, modelo 57. |
| `procEventoCTe` | Eventos. |

Os layouts 2.00, 3.00 e 4.00 podem aparecer na janela de 3 meses. Qualquer outro schema é guardado sem leitura e marcado como não suportado.

## O que a empresa recebe, por papel

O AN distribui o CT-e ao remetente, ao destinatário, ao expedidor, ao recebedor, ao tomador (inclusive o tomador terceiro do `toma4`) e aos CNPJ/CPF do grupo `autXML`. Ao emitente, não distribui o CT-e que ele mesmo emitiu, só os eventos de MDF-e sobre ele; esses eventos são descartados, porque a empresa não tem o documento.

Uma empresa pode ocupar vários papéis no mesmo CT-e, por exemplo remetente e tomador. O papel principal é o primeiro na ordem tomador, destinatário, remetente, expedidor, recebedor, emitente, autorizado; todos ficam guardados, e o filtro de papel casa com qualquer um deles.

Para terceiros do `autXML`, o AN troca as chaves de documentos relacionados (`infDoc`, `docAnt`, `refCTe`) por `9999…`. O Nanci descarta essas chaves (veja "Ligação com a NF-e").

## Endpoints Consumidos

A chamada é SOAP 1.2 síncrona sobre HTTPS com o certificado A1 da empresa, método `cteDistDFeInteresse`, pedido `distDFeInt` versão `1.00` no namespace `http://www.portalfiscal.inf.br/cte`. As URLs estão em `internal/sefaz/endpoints.go`, e o ambiente segue o da empresa, como na NF-e. A action SOAP (`http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe/cteDistDFeInteresse`) segue a do serviço de NF-e e foi aceita pelo serviço de produção em 24/09/2026 (resposta `cStat` 137 para uma empresa sem CT-e na fila).

Cada CT-e e cada evento guardam o seu `tpAmb`. Sem `ide/tpAmb`, o documento usa o `tpAmb` do protocolo (`infProt/tpAmb`), e o evento, o do `retEventoCTe`. Um `tpAmb` ausente ou inválido não impede a gravação: o XML é guardado com o `tpAmb` do ambiente consultado e um aviso, sem as novas tentativas de uma falha de leitura. A mesma regra vale para a NF-e. Cursor, bloqueio e orçamento são separados por ambiente, como na NF-e.

A redefinição dos CT-e mantém os documentos e os eventos assinados por outra empresa cadastrada.

## Requisitos de TLS

Os mesmos da NF-e (veja [NFE_SEFAZ.md](NFE_SEFAZ.md#requisitos-de-tls)). Na verificação de 23/09/2026, `www1.cte.fazenda.gov.br` e `hom1.cte.fazenda.gov.br` apresentavam certificados da AC SERPRO sob a raiz GlobalSign Root R46, já confiável no sistema operacional.

## Limites de consulta

O Nanci aplica ao CT-e as mesmas regras da NF-e (veja [NFE_SEFAZ.md](NFE_SEFAZ.md#limites-de-consulta)), com orçamento e bloqueio próprios da origem `cte`.

A NT 2015.002 não publica um limite de consultas por hora para o CT-e, e não foram encontradas regras de 656 próprias do CT-e; o Nanci supõe que valem as da NF-e (ver `docs/pesquisa/2026-10-07-sefaz-dfe.md`, "Não encontrado"). Se a SEFAZ contar as consultas de NF-e e de CT-e juntas por CNPJ, um pull de CT-e logo depois de um de NF-e pode receber 656. O bloqueio é gravado e respeitado, então o efeito é uma hora de espera.

**Regra dos 60 dias.** Não foi encontrada fonte que confirme na distribuição de CT-e (NT 2015.002) a regra dos 60 dias da NF-e; o Nanci a aplica ao CT-e por prudência, com o mesmo aviso depois de 45 dias sem consulta.

## Recuperar NSUs faltantes

O CT-e registra e recupera os NSUs pulados como a NF-e (ver [NFE_SEFAZ.md](NFE_SEFAZ.md#recuperar-nsus-faltantes)): o `consNSU` do CTeDistribuicaoDFe responde com os mesmos 137, 138, 589 e 656 (NT 2015.002 v1.05, §2.4 e §2.6; `docs/pesquisa/2026-11-18-sefaz-consnsu.md`), e é o único jeito de buscar um CT-e fora da fila, já que não há consulta por chave. O orçamento é o da origem `cte`, com a mesma reserva de 2 consultas para o pull e a mesma janela de 90 dias para encerrar lacunas antigas (a NT do CT-e fala em 3 meses).

## Eventos distribuídos

Um evento só conta como registrado quando o `retEventoCTe` traz `cStat` 134, 135 ou 136; os outros são guardados com um aviso de leitura e não mudam a situação do documento. Um `tpEvento` desconhecido é guardado como `unknown`.

## Prestação do serviço em desacordo

O evento 610110 é a forma de o **tomador** declarar que o serviço de transporte não foi prestado como consta no CT-e. Só o tomador pode enviá-lo, para o CT-e (modelo 57) e o CT-e OS (67), em até 45 dias da autorização (Ajuste SINIEF 09/07, cláusula 17ª, § 6º), com uma observação de 15 a 255 caracteres, e o envio vai à **SEFAZ autorizadora do CT-e** (`CTeRecepcaoEventoV4` da UF da chave, SVRS ou SVSP para a maioria), não ao Ambiente Nacional. Layout, endpoints e rejeições, com as fontes: `docs/pesquisa/2026-11-19-cte-desacordo.md`.

O Nanci envia o evento pelo `nanci cte desacordo` e pelo item **Prestação em desacordo** do menu da linha do CT-e (decisões de 19/11/2026):

- Antes de pedir a senha, o Nanci confere a observação pela regra do schema, que o CNPJ da empresa é exatamente o do tomador resolvido do CT-e (a regra da rejeição 755), o modelo, a situação autorizada e que não há desacordo registrado em vigor (um 610110 registrado sem um 610111 registrado depois). O `nSeqEvento` é o número de desacordos registrados mais 1.
- O prazo de 45 dias é mostrado, contado da autorização, e nunca bloqueia: a SEFAZ decide, com a rejeição 787. É a mesma regra da manifestação conclusiva da NF-e.
- O envio vai sozinho, sem lote, com `cOrgao` igual ao código da UF da chave. Os `cStat` 134, 135 e 136 registram o evento; 631 é duplicidade (registrado antes); o resto é rejeição.
- Todo envio fica em `cte_desacordos`, também o rejeitado e o que ficou sem resposta.
- A SOAP action (`http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoEventoV4/cteRecepcaoEvento`) segue o padrão dos outros serviços do CT-e; o WSDL oficial só responde com certificado de cliente e não foi lido. Os testes usam só servidores falsos, com respostas montadas a partir do schema.

## Ligação com a NF-e

As chaves das NF-e transportadas ficam em `nfe_chaves`: `infCTeNorm/infDoc/infNFe/chave` nos layouts 3.00 e 4.00, `rem/infNFe/chave` no 2.00 e `det/infNFe/chNFe` no CT-e Simplificado. Só entram chaves que passam na validação da chave de acesso (44 dígitos e dígito verificador), sem repetição. As chaves `9999…` que o AN envia aos terceiros do `autXML` são descartadas, e o documento fica marcado como cópia mascarada (`masked_keys`); para esses documentos a ligação com a NF-e não existe.

O CT-e é guardado uma vez por chave, para todas as empresas que o recebem. Quando uma empresa recebe o documento completo e outra, terceira do `autXML`, recebe a cópia mascarada, o Nanci fica com o completo: a cópia mascarada nunca substitui o completo, e o completo substitui a cópia mascarada quando chega depois. A ligação com a NF-e e a exportação usam sempre o XML completo quando alguma empresa o recebeu.

## Exportação

O DACTE só existe para o CT-e modelo 57 propriamente dito: o leiaute do [fiscal-renderer](https://github.com/awafinance/fiscal-renderer) não cobre CT-e OS (modelo 67), GTV-e (modelo 64) nem CT-e Simplificado. No ZIP de DACTE esses documentos ficam de fora, são contados e não recebem marca de exportação, então uma exportação incremental volta a contá-los, como os resumos da NF-e.

## Importação de XML

Como na NF-e (ver [NFE_SEFAZ.md](NFE_SEFAZ.md#importação-de-xml)), com as mesmas recusas. A raiz decide o tipo: `cteProc`, `cteOSProc`, `GTVeProc` e `cteSimpProc` são documentos, `procEventoCTe` é evento; não há resumo.

## Modelo de dados

`tp_cte`, `tp_serv` e `modal` em `cte_documents` são texto bruto, sem `CHECK`, porque os domínios mudaram entre os layouts 3.00 e 4.00.

A situação vem do `cStat` do protocolo (100 e 150 → `autorizada`; 110, 205, 301, 302 e 303 → `denegada`) e passa a `cancelada` quando há um cancelamento registrado. Um protocolo com outro `cStat` é tratado como falha de leitura. No CT-e, 110 ("Uso Denegado"), 205 ("CT-e está denegado na base de dados da SEFAZ"), 301 e 302 (irregularidade fiscal do emitente e do destinatário) estão confirmados nas tabelas de cStat do CT-e; 303 fica na lista por ser da mesma família "Uso Denegado" na NF-e, e é inofensivo se nunca ocorrer.

## Fora do escopo

- Cancelamento da prestação do serviço em desacordo (610111); ver `docs/PENDENCIAS.md`.
- MDF-e: os documentos em si e os eventos de MDF-e sobre CT-e que a empresa não tem.
- Recuperar pela SEFAZ os CT-e emitidos pela própria empresa; o XML deles pode ser importado (ver "Importação de XML").
- Ligação visual entre CT-e e NFS-e nas telas.

## Atribuição

O CT-e usa a mesma montagem do `distDFeInt`, o mesmo envelope SOAP e a mesma leitura do `retDistDFeInt` da NF-e, adaptados de [gonfe](https://github.com/mschunke/gonfe). Veja a seção "Atribuição" de [NFE_SEFAZ.md](NFE_SEFAZ.md#atribuição).

## Referências

- [Portal Nacional do CT-e](https://www.cte.fazenda.gov.br/portal/) (NT 2015.002 v1.05, distribuição DF-e de CT-e, e MOC CT-e 4.00).
- [sped-cte](https://github.com/nfephp-org/sped-cte), usado só como referência de comportamento.
- [NFE_SEFAZ.md](NFE_SEFAZ.md), para as regras compartilhadas com a NF-e.
