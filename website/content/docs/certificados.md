---
title: "Certificados A1"
description: "Como gerenciar senhas e uso de certificados digitais A1 no Nanci."
weight: 40
icon: "key"
toc: true
---

O Nanci suporta certificados digitais e-CNPJ tipo A1 nos formatos `.pfx` e `.p12`.

## Senha do Certificado

Por padrão, a senha do certificado é solicitada via interface gráfica no momento do uso (ao cadastrar ou ao sincronizar caso não tenha sido salva).

Para uso automatizado via CLI, a senha pode ser passada via variável de ambiente:

```bash
export NANCI_CERT_PASSWORD="senha"
```

## Arquivo `.env.local`

Para desenvolvedores ou usuários avançados, o Nanci carrega variáveis de ambiente de um arquivo `.env.local`. Ele procura nesse arquivo na seguinte ordem:

1. No diretório atual onde o comando foi rodado
2. No diretório onde o executável do Nanci está
3. Na pasta de dados do app: `%APPDATA%\nanci\.env.local` (Windows)

**Exemplo de `.env.local`:**
```dotenv
NANCI_CERT_PASSWORD=senha-super-secreta
```

> **Atenção:** Nunca compartilhe seu certificado A1 ou sua senha com ninguém. O Nanci foi desenhado especificamente para rodar localmente no seu computador para garantir que suas credenciais nunca sejam enviadas para servidores de terceiros.
