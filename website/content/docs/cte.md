---
title: "CT-e"
description: "Distribuição de CT-e, CT-e OS, GTV-e e CT-e Simplificado pela SEFAZ."
weight: 32
---

O Nanci baixa os conhecimentos de transporte em que a empresa é parte pelo serviço **CTeDistribuicaoDFe** do Ambiente Nacional da SEFAZ, com os eventos de cada um.

{{< theme-image light="/img/screenshots/cte-light.png" dark="/img/screenshots/cte-dark.png" alt="Página de CT-e" >}}

## Antes de começar

Os requisitos são os mesmos da [NF-e](../nfe/#antes-de-começar): UF da empresa cadastrada, certificado com a mesma raiz de CNPJ e, se quiser, **Testar Conexão** antes do primeiro sync.

## O que chega

| Modelo | Documento |
|---|---|
| 57 | CT-e e CT-e Simplificado |
| 67 | CT-e OS (outros serviços) |
| 64 | GTV-e (guia de transporte de valores) |

A SEFAZ guarda os documentos por cerca de **3 meses**. O XML completo chega direto, sem resumo e sem manifestação.

A empresa recebe os CT-e em que é **tomadora**, **destinatária**, **remetente**, **expedidora**, **recebedora** ou **autorizada** (grupo `autXML`). Uma empresa pode ocupar vários papéis no mesmo documento, por exemplo remetente e tomadora; o filtro de papel encontra o documento por qualquer um deles. Como na NF-e, a SEFAZ não devolve ao emitente os CT-e que ele emitiu.

O limite de consultas é o mesmo da NF-e (20 por hora, espera de 1 hora com a fila em dia ou com a rejeição 656), contado à parte. Veja [Limite de consultas](../nfe/#limite-de-consultas).

## Achar o frete de uma NF-e

O Nanci guarda as chaves das NF-e transportadas em cada CT-e. Para achar o CT-e do frete de uma nota, use o filtro de chave de NF-e na página CT-e, ou `nanci cte list --nfe <chave>` no terminal.

## Na página CT-e

- **Sincronizar CT-e** baixa documentos e eventos novos.
- Filtros por papel, modelo, situação, tomador e chave de NF-e.
- **Eventos** na linha abre os eventos do documento.
- **Exportar** gera um ZIP com os XMLs e eventos da lista filtrada; **Exportar XML** na linha salva um documento.
- **Redefinir CT-e** apaga os CT-e da empresa nos dois ambientes e reinicia a sincronização do zero.

Não há geração de DACTE em PDF.

## Prestação do serviço em desacordo

O evento 610110, com que o tomador declara que o transporte não foi prestado como consta no CT-e, aparece quando chega pela distribuição, mas o Nanci ainda não o envia.

## Detalhes técnicos

Schemas lidos, resolução do tomador, eventos distribuídos e modelo de dados estão em [docs/CTE_SEFAZ.md](https://github.com/vasfvitor/nanci/blob/main/docs/CTE_SEFAZ.md). Os comandos de terminal estão na página [Linha de comando](../cli/#ct-e).
