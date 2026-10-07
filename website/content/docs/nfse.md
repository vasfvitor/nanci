---
title: "NFS-e"
description: "NFS-e do Ambiente de Dados Nacional."
weight: 30
fonte:
  nome: "Portal Nacional da NFS-e"
  url: "https://www.gov.br/nfse/pt-br"
referencia: "docs/NFSE_ADN.md"
aliases:
  - /docs/nfse-adn/
---

As NFS-e vêm do ADN Contribuintes, a API nacional da Receita Federal ([manual da API](https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/documentacao-atual/manual-contribuintes-apis-adn-sistema-nacional-nfse.pdf)): notas tomadas, prestadas e intermediadas, com cancelamentos e substituições.

{{< theme-image light="/img/screenshots/documentos-light.png" dark="/img/screenshots/documentos-dark.png" alt="Página de NFS-e" >}}

Exporte a lista filtrada em Excel, CSV, ZIP com os XMLs ou ZIP com os DANFSe. Cada nota também pode ser salva sozinha em XML ou PDF.

**Consulta Direta API** busca os eventos de uma nota pela chave de acesso, sem sincronizar.

## Notas que não aparecem

A fila do ADN não traz necessariamente todas as notas que a empresa emitiu; depende do município. Veja a [FAQ](../faq/#emito-nfs-e-mas-o-nanci-não-baixou-nada).
