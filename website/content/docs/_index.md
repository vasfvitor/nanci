---
title: "Documentação"
weight: 10
---

O Nanci baixa documentos fiscais eletrônicos direto da infraestrutura nacional, usando o certificado digital A1 da empresa:

| Documento | Origem | O que chega |
|---|---|---|
| [NFS-e](nfse/) | Ambiente de Dados Nacional (ADN) | Notas de serviço em que a empresa é tomadora, prestadora ou intermediária, com eventos. |
| [NF-e](nfe/) (modelo 55) | Distribuição DF-e da SEFAZ | Notas recebidas, primeiro como resumo e completas depois da Ciência da Operação. |
| [CT-e](cte/) (modelos 57, 64 e 67) | Distribuição DF-e da SEFAZ | Conhecimentos de transporte em que a empresa é parte, com eventos. |

Tudo roda no seu computador. O Nanci não tem servidor próprio: a conexão vai da sua máquina direto para o governo, e os XMLs ficam no seu disco.

## Por onde começar

{{< cards >}}
  {{< card link="instalacao" title="Instalação e primeiro uso" icon="download" >}}
  {{< card link="certificados" title="Certificado A1" icon="key" >}}
  {{< card link="cli" title="Linha de comando" icon="terminal" >}}
  {{< card link="faq" title="Perguntas frequentes" icon="question-mark-circle" >}}
{{< /cards >}}

## O que o Nanci não faz

- Não emite notas. Só baixa o que o governo distribui.
- Não acessa portais municipais, não resolve CAPTCHA e não automatiza navegador.
- Não baixa NFC-e, MDF-e, NFCom, NF3e nem CF-e SAT, e não importa XML avulso.
- Não recupera as NF-e e CT-e que a própria empresa emitiu: a SEFAZ não os distribui ao emitente. Na NFS-e, as notas emitidas pela empresa também podem não aparecer (veja a [FAQ](faq/)).
- Não substitui a conferência contábil e fiscal.
