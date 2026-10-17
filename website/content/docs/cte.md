---
title: "CT-e"
description: "CT-e, CT-e OS, GTV-e e CT-e Simplificado pela SEFAZ."
weight: 32
fonte:
  nome: "Portal Nacional do CT-e"
  url: "https://www.cte.fazenda.gov.br/portal/"
referencia: "docs/CTE_SEFAZ.md"
---

O Nanci baixa os CT-e (modelos 57, 64 e 67) em que a empresa é parte, com os eventos. Precisa de UF cadastrada, como a [NF-e](../nfe/), e tem o mesmo [limite de consultas](../nfe/#limite-de-consultas), contado à parte. O aviso **Sem consulta há N dias**, depois de 45 dias sem sincronizar, também vale para o CT-e.

{{< theme-image light="/img/screenshots/cte-light.png" dark="/img/screenshots/cte-dark.png" alt="Página de CT-e" >}}

A SEFAZ guarda os CT-e por 3 meses ([NT 2015.002](https://www.cte.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=LLSQYxGtwv0=)). O XML completo chega direto, sem resumo nem manifestação.

## Exportar

**Exportar** gera, da lista filtrada ou da seleção, uma planilha Excel ou CSV, um ZIP com os XMLs e eventos, ou um ZIP com os DACTE em PDF. Cada documento também pode ser salvo sozinho em XML ou DACTE. O DACTE existe só para o CT-e modelo 57; CT-e OS, GTV-e e CT-e Simplificado ficam fora do ZIP de DACTE, e a tela diz quantos ficaram de fora. **Somente não exportados** repete só o que mudou desde a última exportação naquele formato.

## Importar XML

**Importar XML** guarda os CT-e que a empresa já tem em arquivo, sem consultar a SEFAZ e sem gastar consultas: os que ela mesma emitiu, os de mais de 3 meses e os recebidos por e-mail. Aceita CT-e, CT-e OS, GTV-e e CT-e Simplificado autorizados e os eventos, vários arquivos de uma vez. As recusas são as da [NF-e](../nfe/#importar-e-buscar), e o Nanci mostra o motivo de cada arquivo. Na lista, o CT-e importado leva a marca **Importado**.

Não há busca pela chave para o CT-e: a distribuição do CT-e só entrega pela fila de NSU.

## Achar o frete de uma NF-e

Filtre por **Chave de NF-e transportada** para ver os CT-e que levaram aquela nota.
