---
title: "NFS-e"
description: "Como o Nanci baixa NFS-e do Ambiente de Dados Nacional (ADN), o que chega e como exportar."
weight: 30
aliases:
  - /docs/nfse-adn/
---

O Nanci baixa as NFS-e do padrão nacional pelo **ADN Contribuintes**, a API do Ambiente de Dados Nacional mantida pela Receita Federal. Não usa portal municipal.

{{< theme-image light="/img/screenshots/documentos-light.png" dark="/img/screenshots/documentos-dark.png" alt="Página de NFS-e" >}}

## Como funciona

O ADN entrega a cada CNPJ uma fila de documentos numerados por NSU (Número Sequencial Único). O Nanci pede os documentos depois do último NSU que já leu, guarda o XML de cada um e avança o cursor. Por isso as sincronizações seguintes só trazem o que é novo.

Cada nota recebe o papel da empresa:

- **Tomada**: a empresa contratou o serviço.
- **Prestada**: a empresa prestou o serviço.
- **Intermediário**: a empresa aparece como intermediária.

Os eventos (cancelamento, substituição) chegam pela mesma fila e mudam a situação da nota. Uma substituição prevalece sobre um cancelamento.

A NFS-e não tem limite de consultas por hora.

## Importação inicial

Ao cadastrar a empresa você escolhe até onde o primeiro sync vai buscar: só a partir de hoje, últimos 12 meses, últimos 5 anos, a partir de uma data ou todo o histórico disponível. Depois da carga inicial, as sincronizações seguem incrementais.

## Na página NFS-e

- **Sincronizar NFS-e** baixa os documentos novos da empresa escolhida no menu lateral.
- A lista segue a empresa e a competência do menu lateral. Filtre por papel e marque notas como vistas.
- Em cada linha: **Eventos**, **Exportar XML** e **Exportar DANFSe** (PDF).
- **Exportar** gera, para a lista filtrada, planilha CSV, planilha Excel (XLSX), ZIP com os XMLs originais ou ZIP com os DANFSe. A opção **Somente não exportados** leva só o que ainda não saiu numa exportação anterior.

A página **Consulta Direta API** busca os eventos de uma nota pela chave de acesso de 50 dígitos (`GET NFSe/{chave}/Eventos`), sem passar pela fila.

## Ambientes

| Ambiente | Valor no Nanci | URL base |
|---|---|---|
| Produção | `producao` | `https://adn.nfse.gov.br/contribuintes` |
| Produção Restrita (testes) | `producao_restrita` | `https://adn.producaorestrita.nfse.gov.br/contribuintes` |

Notas da Produção Restrita não têm valor fiscal. O ambiente é escolhido por empresa e pode ser trocado depois; o cursor é guardado separado para cada ambiente.

## Notas emitidas pela própria empresa

A fila do ADN não é uma lista completa das notas que a empresa emitiu. Dependendo do município, notas prestadas podem não chegar. Veja a [FAQ](../faq/#emito-nfs-e-todo-mês-mas-o-nanci-não-baixou-nada).

## Detalhes técnicos

Endpoints, tratamento da chave de acesso e migrações estão em [docs/NFSE_ADN.md](https://github.com/vasfvitor/nanci/blob/main/docs/NFSE_ADN.md) no repositório.
