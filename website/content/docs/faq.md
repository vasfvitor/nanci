---
title: "Perguntas frequentes"
description: "Por que um documento não aparece, ambientes e certificados."
weight: 70
---

## Emito NFS-e, mas o Nanci não baixou nada

A fila do ADN não traz necessariamente as notas que a empresa emitiu; depende do município. Teste com uma nota em que a empresa seja tomadora, ou consulte uma chave conhecida em **Consulta Direta API**.

## Por que não aparecem as NF-e que a empresa emitiu?

A SEFAZ não distribui ao emitente as notas que ele mesmo emitiu. O mesmo vale para o CT-e.

## Por que a NF-e aparece só como resumo?

Falta a [Ciência da Operação](../nfe/#manifestação-do-destinatário). Depois dela, o XML completo chega no próximo sync.

## Consigo baixar notas antigas?

NF-e e CT-e, só dos últimos 3 meses mais ou menos: é o que a SEFAZ guarda. Para NFS-e, escolha o período ao cadastrar a empresa.

## Por que o botão de sincronizar está desabilitado?

A SEFAZ limita as consultas. Veja [Limite de consultas](../nfe/#limite-de-consultas).

## Produção ou Produção restrita?

Produção é o ambiente real. Produção restrita é o ambiente de testes do governo, sem valor fiscal. Para trocar, use **Empresas → Editar**.

## Funciona com certificado A3, macOS ou Linux?

Não. Só certificado A1, e o aplicativo só existe para Windows.
