---
title: "NF-e"
description: "Distribuição de NF-e (modelo 55) pela SEFAZ, limite de consultas e Manifestação do Destinatário."
weight: 31
---

O Nanci baixa as NF-e (modelo 55) em que a empresa aparece pelo serviço **NFeDistribuicaoDFe** do Ambiente Nacional da SEFAZ, e registra a **Manifestação do Destinatário**.

{{< theme-image light="/img/screenshots/nfe-light.png" dark="/img/screenshots/nfe-dark.png" alt="Página de NF-e" >}}

## Antes de começar

- Cadastre a **UF** da empresa (no cadastro ou em **Editar** na tela Empresas). Sem ela, a sincronização falha antes de pedir a senha.
- O certificado é o mesmo da NFS-e. A raiz do CNPJ (8 primeiros dígitos) da empresa precisa ser igual à do certificado; o certificado da matriz serve para as filiais.
- Use **Testar Conexão** para conferir o certificado e a rede. O teste não gasta consultas.

## O que chega

A SEFAZ guarda os documentos por cerca de **90 dias**. O primeiro sync traz o que ainda estiver lá; notas mais antigas não podem ser baixadas por aqui.

| Papel da empresa | O que chega |
|---|---|
| Destinatário | Todas as NF-e autorizadas contra o CNPJ, de qualquer UF. Primeiro chega só o **resumo**; o XML completo só é liberado depois da Ciência da Operação ou de uma manifestação conclusiva. |
| Transportador | NF-e em que o CNPJ é o transportador. |
| Autorizado | NF-e em que o CNPJ foi informado no grupo `autXML`. |
| Emitente | A SEFAZ não devolve ao emitente as notas que ele mesmo emitiu. Uma nota só aparece com esse papel se chegar por outro motivo. |

## Limite de consultas

A SEFAZ bloqueia por uma hora o CNPJ que consulta demais (rejeição **656, consumo indevido**). Para evitar isso, o Nanci:

- faz no máximo **20 consultas por hora** por empresa e ambiente, contando também as que falham;
- espera **1 hora** quando a fila está em dia (não há documento novo);
- espera 1 hora se mesmo assim receber a rejeição 656.

Enquanto a consulta estiver bloqueada, a página mostra o horário da próxima consulta permitida e o botão de sincronizar fica desabilitado. Esse limite é separado do limite do CT-e.

## Manifestação do Destinatário

Só a empresa **destinatária** de uma NF-e **autorizada** pode manifestar.

| Evento | Natureza |
|---|---|
| Ciência da Operação | Não conclusiva. Libera o XML completo no próximo sync. |
| Confirmação da Operação | Conclusiva. |
| Desconhecimento da Operação | Conclusiva. |
| Operação não Realizada | Conclusiva. Exige justificativa de 15 a 255 caracteres. |

{{< callout type="warning" >}}
Uma manifestação enviada fica registrada na SEFAZ e o Nanci não consegue desfazê-la. Os dois fluxos abaixo mostram o que será enviado e pedem confirmação antes do envio.
{{< /callout >}}

- **Ciência em lote**: selecione as notas na aba **Notas** e clique em **Registrar ciência**. O diálogo mostra as notas elegíveis, as ignoradas com o motivo, e exige uma confirmação antes de enviar. A senha é pedida uma vez para todos os lotes. Para uma nota só, use **Registrar ciência** no menu da linha.
- **Manifestação conclusiva**: use **Manifestar…** na linha da nota. O diálogo pede o tipo, a justificativa quando exigida e uma revisão final.

{{< theme-image light="/img/screenshots/dialogo-ciencia-nfe-light.png" alt="Diálogo de Ciência da Operação" >}}

### Prazos

Os prazos contam da autorização da NF-e. São avisos: o Nanci não impede nenhum envio por prazo.

- **Ciência**: a nota sem manifestação é marcada como "ciência atrasada" 10 dias após a autorização.
- **Conclusiva**: até **90 dias** da autorização, com alerta a partir de 30 dias antes do fim. Fora do prazo a SEFAZ rejeita o evento (596).
- **Confirmação tácita**: passados os 90 dias sem evento conclusivo, a operação é considerada confirmada. A Ciência não interrompe esse prazo.

A aba **Pendências** lista as notas sem manifestação conclusiva, com o prazo de cada uma.

{{< theme-image light="/img/screenshots/nfe-pendencias-light.png" dark="/img/screenshots/nfe-pendencias-dark.png" alt="Aba de pendências de manifestação" >}}

Base legal: cláusula 15ª-C do Ajuste SINIEF 07/05, na redação dos Ajustes SINIEF 11/22 e 14/26.

## Exportar e redefinir

- **Exportar** gera um ZIP com os XMLs da lista filtrada. Marque **Incluir resumos** para levar também as notas que ainda não estão completas.
- **Exportar XML** na linha da nota salva o XML dela.
- **Redefinir NF-e** apaga as NF-e da empresa nos dois ambientes e reinicia a sincronização do zero. As manifestações já registradas na SEFAZ não são afetadas, e o histórico de envios é mantido. Um bloqueio da SEFAZ em vigor continua valendo.

Não há geração de DANFE em PDF.

## Detalhes técnicos

Endpoints, requisitos de TLS, regras de cada `cStat`, assinatura dos eventos e modelo de dados estão em [docs/NFE_SEFAZ.md](https://github.com/vasfvitor/nanci/blob/main/docs/NFE_SEFAZ.md). Os comandos de terminal estão na página [Linha de comando](../cli/#nf-e).
