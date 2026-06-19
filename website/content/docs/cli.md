---
title: "Interface de Linha de Comando (CLI)"
description: "Guia completo de comandos para usar o Nanci pelo terminal ou em automações."
weight: 30
icon: "terminal"
toc: true
---

O Nanci também possui uma CLI completa, ideal para automações, rotinas de servidor (cron jobs) ou para quem prefere usar o terminal.

> **Importante:** A CLI e o Desktop compartilham o mesmo banco de dados local. Empresas adicionadas em um aparecerão no outro.

## Compilando o CLI

O CLI não é distribuído compilado no instalador padrão no momento. Para compilá-lo a partir do código-fonte:

```bash
git clone https://github.com/vasfvitor/nanci
cd nanci
go build -o nanci.exe ./cmd/nanci
```

## Comandos Principais

### Adicionar uma Empresa e Credencial

```bash
./nanci.exe company add --cnpj 12345678000199 --name "Minha Empresa" --cert "caminho/para/cert.pfx"
```

### Sincronizar Notas (Puxar)

Baixa notas novas da API ADN.

```bash
./nanci.exe pull --cnpj 12345678000199
```

### Exportar Dados

O Nanci suporta exportação em lote para Excel, CSV, ZIP (com os XMLs) ou PDFs (DANFSE).

```bash
# Exportar relatórios tabulares
./nanci.exe export xlsx --cnpj 12345678000199 --out relatorio.xlsx
./nanci.exe export csv  --cnpj 12345678000199 --out relatorio.csv

# Exportar documentos
./nanci.exe export zip        --cnpj 12345678000199 --out xmls.zip
./nanci.exe export danfse-zip --cnpj 12345678000199 --out pdfs.zip

# Exportar um PDF único por chave
./nanci.exe export danfse --cnpj 12345678000199 --chave 35... --out nota.pdf
```

## Senha do Certificado

Para automações silenciosas, você não vai querer que a CLI peça a senha no terminal. Você pode providenciar a senha do certificado através da variável de ambiente `NANCI_CERT_PASSWORD`:

No PowerShell:
```powershell
$env:NANCI_CERT_PASSWORD="minha-senha"
./nanci.exe pull --cnpj 12345678000199
```

No CMD:
```cmd
set NANCI_CERT_PASSWORD=minha-senha
nanci.exe pull --cnpj 12345678000199
```

Você também pode criar um arquivo `.env.local` na raiz onde rodar o comando, contendo a variável `NANCI_CERT_PASSWORD=minha-senha`.
