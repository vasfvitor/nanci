---
title: "Perguntas Frequentes (FAQ)"
description: "Dúvidas comuns sobre certificados, XMLs, sincronização e ambiente."
weight: 40
icon: "help-circle"
toc: true
---

## Eu emito NFS-e todo mês, mas o Nanci não baixou nada. Por quê?

A consulta por NSU no Ambiente de Dados Nacional (ADN) Contribuintes **não deve ser tratada como uma lista absoluta e exaustiva** das notas emitidas pela própria empresa.

Dependendo da regra de distribuição e da maturidade da integração do município com o Padrão Nacional, notas emitidas pelo próprio CNPJ (na condição de prestador) podem não aparecer na fila de distribuição desse endpoint via NSU.

Para validar se sua integração está de fato funcionando (e se seu certificado/senha estão corretos), teste também com uma NFS-e em que seu CNPJ seja **tomador/interessado**, ou consulte uma chave de acesso específica diretamente no portal web oficial. 

Lembre-se: O Nanci reflete o que o Governo entrega na API. Se a API retornar "sem documentos", o problema não está no Nanci, mas sim na forma como as notas foram distribuídas no banco de dados nacional.

## Qual a diferença entre Produção e Produção Restrita?
- **Produção:** É o ambiente oficial da Receita Federal. Notas baixadas aqui têm validade jurídica.
- **Produção Restrita:** É o ambiente de homologação (testes) do Governo. As notas emitidas e consultadas aqui não têm valor fiscal. Se você está desenvolvendo ou testando uma integração e emitindo notas "falsas", deve usar o ambiente de Produção Restrita.

## Onde ficam meus XMLs e como faço backup?
Tudo fica no seu computador. Leia nossa página sobre [Privacidade e Backup de Dados](privacidade).

## O Nanci pode usar um certificado A3 (Token/Smartcard)?
Atualmente, suportamos apenas **Certificados A1** (`.pfx` ou `.p12`). O uso de certificados A3 requer integração com drivers criptográficos e leitura de porta USB/Smartcard, o que atualmente está fora do escopo da nossa sincronização automatizada sem interface em background.

## O Nanci manda meus dados para vocês?
**Não.** O código é aberto (Open Source), o banco de dados é gerado localmente em SQLite, e a conexão HTTPS (mTLS) ocorre diretamente entre sua máquina (seu IP) e os servidores do governo (SERPRO/Receita). Nós nunca recebemos cópias dos seus XMLs ou da sua chave privada.
