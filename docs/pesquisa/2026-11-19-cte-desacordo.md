# CT-e: prestação do serviço em desacordo (evento 610110)

Levantado em 19/11/2026.

Responde: o que o nanci precisa saber para enviar, como tomador, o evento 610110 à SEFAZ autorizadora do CT-e.

Todas as fontes foram consultadas em 19/11/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fornecedores, fóruns ou imprensa, sem confirmação em texto oficial. A tabela de endpoints e o prazo foram conferidos de novo na mesma data, direto nas páginas oficiais citadas.

Fontes oficiais usadas:

- [A] MOC CT-e 4.00, Visão Geral (ENCAT; o PDF diz 08/2022, a lista do portal diz fevereiro/2023): https://www.cte.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=0eu1YksSGXM= . Seções 3.2.2, 3.4.1, 3.7.1, 5 (pp. 52 a 57) e 6.5 a 6.6 (pp. 67 e 68).
- [B] Pacote de liberação dos schemas 4.00, `PL_CTe_400_NT2026.001 RTC_VincPgto_1.01c_corr` (08/06/2026): https://www.cte.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=2oAI3pZFSWk= .
- [C] Ajuste SINIEF 09/07, texto consolidado no CONFAZ: https://www.confaz.fazenda.gov.br/legislacao/ajustes/2007/AJ_009_07 .
- [D] Relação de web services do Portal Nacional do CT-e, produção https://www.cte.fazenda.gov.br/portal/webServices.aspx e homologação https://hom.cte.fazenda.gov.br/portal/webServices.aspx ; a mesma lista está em https://dfe-portal.svrs.rs.gov.br/Cte/Servicos .

## Base legal e prazo

Norma: Ajuste SINIEF 09/07 [C].

- O evento é a "Prestação de serviço em desacordo com o informado no CT-e", cláusula 18ª-A, inciso XV (incluído pelo Ajuste SINIEF 10/16, efeitos a partir de 01/09/2016). O cancelamento desse evento é o inciso XXV (Ajuste SINIEF 25/23).
- Prazo: 45 dias. Cláusula 17ª, § 6º (redação do Ajuste SINIEF 31/22, efeitos a partir de 03/04/2023): "O prazo para registro de um dos eventos citados no inciso III alínea 'a' será de 45 (quarenta e cinco) dias". A cláusula 17ª-A, § 4º (Ajuste SINIEF 8/17) repete o prazo. O MOC conta da autorização de uso do CT-e e aplica a regra O05, com rejeição 787 [A].
- O CT-e de substituição exige um 610110 autorizado, salvo tomador no exterior; um 610111 anula o 610110 (NT 2025.001 RTC v1.14b, § 11, 30/04/2026: https://www.cte.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=k4vwqpRxU2M= ; Ajuste SINIEF 17/24 [C]).

## Quem envia e para quais modelos

Fonte: MOC 4.00, seções 3.7.1 e 6.5 [A].

- Autor: o tomador indicado no CT-e (tipo de autor 6). O certificado tem de ter a raiz do CNPJ do tomador. Autor diferente do tomador: rejeição 755 (regra N19).
- Tomador pessoa física envia pelo portal da SVRS com login gov.br (NT 2022.001 v1.00, 03/2022: https://www.cte.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=wwLVIQEvpZU= ). Fora do alcance do nanci, que só tem certificado de CNPJ.
- Modelos: 57 (CT-e, inclusive o CT-e Simplificado, que é modelo 57) e 67 (CT-e OS). A GTV-e (64) fica de fora: rejeição 595 (regra N15).

## Layout do evento

Fonte: schemas do pacote [B] (`eventoCTe_v4.00.xsd`, `eventoCTeTiposBasico_v4.00.xsd`, `evPrestDesacordo_v4.00.xsd`, `retEventoCTe_v4.00.xsd`, `procEventoCTe_v4.00.xsd`) e MOC 5.1.1 e 6.5 [A].

- Raiz `eventoCTe versao="4.00"`, namespace `http://www.portalfiscal.inf.br/cte`. Um evento por mensagem: não existe lote, ao contrário da NF-e.
- `infEvento`, na ordem: `cOrgao`, `tpAmb`, `CNPJ` (ou `CPF`), `chCTe`, `dhEvento` (`AAAA-MM-DDThh:mm:ssTZD`), `tpEvento` (`610110`), `nSeqEvento` (1 a 20; acima, rejeição 636), `detEvento versaoEvento="4.00"`.
- `Id` do `infEvento`: `ID` + tpEvento + chave + nSeqEvento com 3 dígitos, 55 caracteres (padrão `ID[0-9]{12}[A-Z0-9]{12}[0-9]{29}`, pronto para o CNPJ alfanumérico). Na NF-e o nSeqEvento do Id tem 2 dígitos. Id que não bate com os campos: rejeição 628.
- `cOrgao`: o código da UF da chave (o MOC diz que a SEFAZ confere a UF pela chave; divergência é a rejeição 677). Os valores aceitos são as UF mais 90, 91, 93, 94, 95 e 96; não existe 92.
- Grupo `evPrestDesacordo` dentro de `detEvento`:
  - `descEvento`: "Prestação do Serviço em Desacordo" (ou a forma sem acento, "Prestacao do Servico em Desacordo");
  - `indDesacordoOper`: sempre `1`;
  - `xObs`: 15 a 255 caracteres. O MOC lista o campo como opcional, mas o schema o exige; vale o schema.
- Assinatura: a mesma XMLDSig da NF-e (envelopada sobre `infEvento`, C14N, SHA-1, RSA-SHA1), em `ds:Signature` depois do `infEvento`.

## Serviço CTeRecepcaoEventoV4

Fonte: MOC 3.2.2, 3.4.1 e 5 [A].

- Síncrono, SOAP 1.2, document/literal, método `cteRecepcaoEvento`. O corpo leva `cteDadosMsg` com o `eventoCTe` sem compactação (só os serviços de recepção do CT-e usam gzip e Base64).
- Resposta `retEventoCTe`: `infEvento` com `tpAmb`, `verAplic`, `cOrgao`, `cStat`, `xMotivo` e, quando registrado, `chCTe`, `tpEvento`, `xEvento`, `nSeqEvento`, `dhRegEvento` e `nProt` (15 dígitos).
- `cStat` 135: evento registrado e vinculado ao CT-e; 134: registrado, mas o CT-e não está autorizado; 136: registrado, CT-e não encontrado.
- `cStat` pode ter 4 dígitos desde a NT 2025.001 RTC v1.14b, § 18 (tipo `TStat` `[0-9]{3,4}`).

Rejeições do 610110 [A], 6.5.1: 636 (nSeqEvento acima de 20), 205 (CT-e denegado), 218 (CT-e cancelado), 664 (CT-e já tem substituto), 787 (mais de 45 dias da autorização). Rejeições gerais de evento que importam aqui [A], 5.1.5: 755 (autor não é o tomador), 631 (evento duplicado), 217 (CT-e não consta na base), 627 e 905 (CNPJ ou CPF inválido), 628 (Id), 630 (schema), 634, 637 e 635 (dhEvento antes da emissão, antes da autorização ou depois do processamento, com 5 minutos de tolerância), 677 (cOrgao), 252 (tpAmb).

## Endpoints por autorizadora

Fonte: relação de web services do Portal Nacional do CT-e [D], conferida em 19/11/2026 nas páginas de produção e de homologação.

| Autorizadora | Produção | Homologação |
|---|---|---|
| SVRS e RS | `https://cte.svrs.rs.gov.br/ws/CTeRecepcaoEventoV4/CTeRecepcaoEventoV4.asmx` | `https://cte-homologacao.svrs.rs.gov.br/ws/CTeRecepcaoEventoV4/CTeRecepcaoEventoV4.asmx` |
| SP e SVSP | `https://nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoEventoV4.asmx` | `https://homologacao.nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoEventoV4.asmx` |
| MG | `https://cte.fazenda.mg.gov.br/cte/services/CTeRecepcaoEventoV4` | `https://hcte.fazenda.mg.gov.br/cte/services/CTeRecepcaoEventoV4` |
| MS | `https://producao.cte.ms.gov.br/ws/CTeRecepcaoEventoV4` | `https://homologacao.cte.ms.gov.br/ws/CTeRecepcaoEventoV4` |
| MT | `https://cte.sefaz.mt.gov.br/ctews2/services/CTeRecepcaoEventoV4` | `https://homologacao.sefaz.mt.gov.br/ctews2/services/CTeRecepcaoEventoV4` |
| PR | `https://cte.fazenda.pr.gov.br/cte4/CTeRecepcaoEventoV4` | `https://homologacao.cte.fazenda.pr.gov.br/cte4/CTeRecepcaoEventoV4` |

O portal lista as URLs de MT e PR com `?wsdl`; o endereço do serviço é o mesmo sem o sufixo.

- SVRS atende AC, AL, AM, BA, CE, DF, ES, GO, MA, PA, PB, PI, RJ, RN, RO, SC, SE e TO (texto da página).
- SVSP atende AP, PE e RR (texto da página).
- MG, MS, MT, PR, RS e SP têm autorizadora própria. A página não diz isso com essas palavras: são as UF que não aparecem nas duas listas e têm URL própria.

A autorizadora sai da UF da chave (cUF, posições 1 e 2), não da UF do tomador.

## Cancelamento do desacordo (610111)

Fonte: MOC 6.6 [A] e `evCancPrestDesacordo_v4.00.xsd` [B]. Não entra agora; fica registrado para o item em `docs/PENDENCIAS.md`.

- Grupo `evCancPrestDesacordo` (a tabela do MOC escreve `evPrestDesacordo` por engano; vale o schema), com `descEvento` "Cancelamento Prestação do Serviço em Desacordo" e `nProtEvPrestDes`, o protocolo de 15 dígitos do 610110.
- Autor: o tomador. Modelos 57 e 67. Rejeições próprias: 636, 664 e 866 (protocolo inexistente, de outro CT-e ou já cancelado). Nenhum prazo foi encontrado.

## Mudanças de 2025 e 2026

- NT 2025.001 RTC v1.14b: o § 11 (substituição exige o 610110) e o § 18 (`cStat` com 4 dígitos), citados acima. O layout do 610110 não mudou.
- NT 2026.002 v1.00 (10/06/2026, `conteudo=sMeQcKKmGXE=`) e v1.01 (16/07/2026, `conteudo=nYNUxSrXYaU=`), NT 2026.004 v1.01 (10/2026, `conteudo=urai6j7wfLM=`, cria o evento 211110 do tomador, só na SVRS) e NT 2026.005 v1.00 (10/2026, `conteudo=AtaVevRXCIQ=`): nenhuma fala do 610110 ou do 610111.

## Não encontrado

- A SOAP action e o namespace do WSDL do CTeRecepcaoEventoV4. O padrão dos outros serviços do CT-e dá `http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoEventoV4` e a action `.../CTeRecepcaoEventoV4/cteRecepcaoEvento`, mas o `?wsdl` da SVRS e de SP responde 403 sem certificado de cliente. Conferir com o certificado A1 antes do primeiro envio real.
- Texto oficial que diga que as UF atendidas pela SVRS ou pela SVSP põem o próprio código em `cOrgao`. O schema e a regra C-1 do MOC indicam que sim.
- Como a SEFAZ resolve o tomador (`toma3`, `toma4`, o `toma` do CT-e OS) para a regra 755. O MOC não detalha.
- Regra própria do CT-e Simplificado para o 610110. O Ajuste SINIEF 4/26 (cláusula 3ª-B, § 4º) dispensa o evento só na substituição do Simplificado para correção de valor [C].
- O conteúdo da NT 2026.001, que não foi aberta.
