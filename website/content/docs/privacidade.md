---
title: "Privacidade e backup"
description: "Onde ficam os dados e como fazer backup."
weight: 60
---

O Nanci não tem servidor nem telemetria. Ele só se comunica com o ADN (`adn.nfse.gov.br`) e com a SEFAZ (`*.nfe.fazenda.gov.br`, `*.cte.fazenda.gov.br`), usando o certificado da empresa.

## Onde ficam os dados

Em `%LOCALAPPDATA%\nanci`. **Configurações → Abrir Pasta de Dados** abre a pasta.

| Item | Conteúdo |
|---|---|
| `nanci-v1.db` | Banco SQLite com empresas, notas e histórico. |
| `blobs/` | Os XMLs originais. Não apague: as exportações saem daqui. |
| `logs/` | Logs do aplicativo. Guardam o CNPJ das empresas em claro, para diagnóstico local; só o pacote gerado por **Exportar Pacote de Diagnóstico** mascara CNPJ e chaves de acesso. |

As senhas ficam no Gerenciador de Credenciais do Windows. O banco e os XMLs não são criptografados; use o BitLocker se o computador for compartilhado.

## Backup

Feche o Nanci e copie a pasta inteira. Para restaurar, copie de volta para o mesmo caminho antes de abrir o aplicativo.

## Apagar tudo

Desinstalar remove só o programa. Para apagar os dados, apague a pasta e as entradas `nanci_certs` do Gerenciador de Credenciais.
