---
title: "Perguntas frequentes"
description: "Dúvidas comuns sobre documentos que não chegam, ambientes, certificados e privacidade."
weight: 70
---

## Emito NFS-e todo mês, mas o Nanci não baixou nada

A fila do ADN **não é uma lista completa** das notas emitidas pela empresa. Dependendo do município e de como ele se integrou ao padrão nacional, notas em que a empresa é prestadora podem não chegar por essa fila.

Para saber se o problema é a fila ou a configuração:

1. Confira se a empresa está em **Produção** (`producao`). Em Produção Restrita só aparecem notas de teste.
2. Teste com uma nota em que a empresa seja **tomadora**.
3. Consulte uma chave conhecida na página **Consulta Direta API**.

Se a fila responder sem documentos, o Nanci não tem o que baixar.

## Por que não aparecem as NF-e que a empresa emitiu?

A distribuição da SEFAZ não devolve ao emitente as notas que ele mesmo emitiu, nem na NF-e nem no CT-e. O Nanci só baixa o que a SEFAZ entrega. Para guardar as notas emitidas, use o XML gerado pelo seu sistema emissor.

## Por que a NF-e aparece só como resumo?

Para o destinatário, a SEFAZ entrega primeiro um resumo. O XML completo só é liberado depois da **Ciência da Operação** (ou de uma manifestação conclusiva). Registre a ciência e sincronize de novo. Veja [Manifestação do Destinatário](../nfe/#manifestação-do-destinatário).

## Consigo baixar NF-e de mais de 3 meses atrás?

Não por aqui. A SEFAZ guarda os documentos da distribuição por cerca de 90 dias (CT-e: cerca de 3 meses). O que já saiu da fila não volta. Por isso vale sincronizar com frequência.

## Por que o botão de sincronizar NF-e ou CT-e está desabilitado?

A SEFAZ permite poucas consultas e bloqueia por uma hora quem consulta demais. O Nanci respeita esse limite: depois de 20 consultas na última hora, ou quando a fila está em dia, ele espera e mostra o horário da próxima consulta permitida. Veja [Limite de consultas](../nfe/#limite-de-consultas).

## Qual a diferença entre Produção e Produção Restrita?

- **Produção** (`producao`): o ambiente oficial. Os documentos têm validade fiscal.
- **Produção Restrita** (`producao_restrita`): o ambiente de testes do governo (homologação). Os documentos não têm valor fiscal.

Empresas novas são cadastradas em Produção. O ambiente pode ser trocado em **Empresas → Editar**.

## Posso usar certificado A3 (token ou cartão)?

Não. Só certificados A1 (`.pfx` ou `.p12`).

## Funciona no macOS ou Linux?

O aplicativo desktop só é distribuído para Windows. A linha de comando pode ser compilada para outros sistemas a partir do código-fonte, mas só é testada no Windows.

## O Nanci envia meus dados para alguém?

Não. O Nanci não tem servidor. A conexão vai do seu computador direto para o ADN e para a SEFAZ, e os XMLs, o banco e as senhas ficam na sua máquina. O código é aberto e pode ser conferido. Veja [Privacidade e backup](../privacidade/).

## Onde ficam meus XMLs e como faço backup?

Em `%LOCALAPPDATA%\nanci`. Veja [Privacidade e backup](../privacidade/#backup).
