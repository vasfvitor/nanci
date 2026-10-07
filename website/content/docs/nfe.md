---
title: "NF-e"
description: "NF-e recebidas pela SEFAZ e Manifestação do Destinatário."
weight: 31
fonte:
  nome: "Portal Nacional da NF-e"
  url: "https://www.nfe.fazenda.gov.br/portal/principal.aspx"
referencia: "docs/NFE_SEFAZ.md"
---

O Nanci baixa as NF-e em que a empresa aparece, pela distribuição DF-e da SEFAZ, e registra a Manifestação do Destinatário. A empresa precisa ter UF cadastrada.

{{< theme-image light="/img/screenshots/nfe-light.png" dark="/img/screenshots/nfe-dark.png" alt="Página de NF-e" >}}

A SEFAZ guarda as notas por cerca de 90 dias; o que saiu da fila não volta. Como destinatária, a empresa recebe primeiro um **resumo**. O XML completo só chega depois da **Ciência da Operação**.

## Limite de consultas

A SEFAZ bloqueia por uma hora quem consulta demais. O Nanci faz no máximo 20 consultas por hora e, quando a fila está em dia, espera uma hora antes de consultar de novo. Enquanto isso, o botão de sincronizar mostra quando volta a funcionar.

## Manifestação do Destinatário

{{< callout type="warning" >}}
Uma manifestação enviada não pode ser desfeita.
{{< /callout >}}

- **Ciência da Operação**: selecione as notas e clique em **Registrar ciência**.
- **Confirmação, Desconhecimento ou Operação não Realizada**: use **Manifestar…** na linha da nota.

{{< theme-image light="/img/screenshots/dialogo-ciencia-nfe-light.png" alt="Diálogo de Ciência da Operação" >}}

A aba **Pendências** mostra as notas sem manifestação conclusiva e o prazo de cada uma: 90 dias da autorização. Depois disso a operação é considerada confirmada.

{{< theme-image light="/img/screenshots/nfe-pendencias-light.png" dark="/img/screenshots/nfe-pendencias-dark.png" alt="Aba de pendências de manifestação" >}}

## Exportar

**Exportar** gera um ZIP com os XMLs da lista. **Exportar XML** salva uma nota.
