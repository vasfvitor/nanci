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

A SEFAZ guarda as notas por 90 dias e não reenvia o que já saiu da fila. Dentro desse prazo, uma nota que não chegou pode ser buscada pela chave; depois dele, só entra pelo XML (ver [Importar e buscar](#importar-e-buscar)). Como destinatária, a empresa recebe primeiro um **resumo**. O XML completo chega depois da **Ciência da Operação**, da Confirmação ou da Operação não Realizada ([NT 2014.002](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=)).

## Limite de consultas

A SEFAZ bloqueia por uma hora quem consulta demais ([NT 2014.002](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=)). O Nanci faz no máximo 20 consultas por hora e, quando a fila está em dia, espera uma hora antes de consultar de novo. Enquanto isso, o botão de sincronizar mostra quando volta a funcionar.

Não deixe a empresa mais de 60 dias sem sincronizar. A SEFAZ só gera NSU para quem consultou nesse prazo, e as notas do período parado não chegam mais (regra da NT 2014.002 v1.10, conhecida por relatos de terceiros). Depois de 45 dias sem consulta, a página mostra o aviso **Sem consulta há N dias**; sincronizar reinicia a contagem.

## Manifestação do Destinatário

{{< callout type="warning" >}}
Uma manifestação enviada não pode ser cancelada. A SEFAZ aceita retificar uma manifestação conclusiva em até 30 dias ([Ajuste SINIEF 07/05](https://www.confaz.fazenda.gov.br/legislacao/ajustes/2005/AJ007_05)), mas o Nanci ainda não faz isso.
{{< /callout >}}

- **Ciência da Operação**: selecione as notas e clique em **Registrar ciência**.
- **Confirmação, Desconhecimento ou Operação não Realizada**: use **Manifestar…** na linha da nota.

{{< theme-image light="/img/screenshots/dialogo-ciencia-nfe-light.png" alt="Diálogo de Ciência da Operação" >}}

A aba **Pendências** mostra as notas sem manifestação conclusiva e o prazo de cada uma: 90 dias da autorização. Depois disso a operação é considerada confirmada ([Ajuste SINIEF 07/05](https://www.confaz.fazenda.gov.br/legislacao/ajustes/2005/AJ007_05), cláusula 15ª-C).

{{< theme-image light="/img/screenshots/nfe-pendencias-light.png" dark="/img/screenshots/nfe-pendencias-dark.png" alt="Aba de pendências de manifestação" >}}

## Importar e buscar

**Importar XML** guarda as NF-e que a empresa já tem em arquivo, sem consultar a SEFAZ e sem gastar consultas: as notas que ela mesma emitiu, que a SEFAZ não entrega ao emitente, as de mais de 90 dias e as que o fornecedor mandou por e-mail. Aceita a nota autorizada (com o protocolo), o resumo e os eventos, vários arquivos de uma vez. Um arquivo de outro ambiente, ilegível ou de uma nota em que a empresa não aparece é recusado sem impedir os outros, e o Nanci mostra o motivo de cada um. Na lista, a nota importada leva a marca **Importado**.

**Buscar pela chave** pede uma NF-e à SEFAZ pela chave de acesso. Funciona para a destinatária, a transportadora e quem foi autorizado no XML, dentro dos 90 dias; a SEFAZ não entrega ao emitente a própria nota, que só entra pela importação. Cada busca gasta 1 das 20 consultas da hora ([NT 2014.002](https://www.nfe.fazenda.gov.br/portal/exibirArquivo.aspx?conteudo=uWO2d/gTuWg=)). Enquanto a empresa não manifesta a nota, a SEFAZ entrega só o resumo: registre a Ciência da Operação e busque de novo, ou espere a próxima sincronização.

## Exportar

**Exportar** gera, da lista filtrada ou da seleção, uma planilha Excel ou CSV, um ZIP com os XMLs e eventos, ou um ZIP com os DANFE em PDF. Cada nota também pode ser salva sozinha em XML ou DANFE. Resumos (notas sem o XML completo) entram na planilha e no ZIP de XML só com **Incluir resumos**, e nunca têm DANFE: o ZIP de DANFE avisa quantos ficaram de fora. **Somente não exportados** repete só o que mudou desde a última exportação naquele formato.
