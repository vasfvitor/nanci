---
title: "Certificado A1"
description: "Certificados aceitos e como o Nanci guarda a senha."
weight: 50
---

O Nanci aceita certificado e-CNPJ **A1** (`.pfx` ou `.p12`). A3, em token ou cartão, não funciona. O mesmo certificado serve para NFS-e, NF-e e CT-e, e o da matriz serve para as filiais.

O arquivo não é copiado; o Nanci guarda o caminho. Se você mover ou renovar o certificado, atualize o caminho em **Credenciais**.

{{< theme-image light="/img/screenshots/credenciais-light.png" dark="/img/screenshots/credenciais-dark.png" alt="Tela de credenciais" >}}

## Senha

A senha não vai para o banco do Nanci. Depois do primeiro uso ela fica no Gerenciador de Credenciais do Windows, numa entrada que começa com `nanci_certs`. Para esquecê-la, apague essa entrada.

Na linha de comando, a senha também pode vir da variável `NANCI_CERT_PASSWORD`, definida no ambiente ou num arquivo `.env.local` na pasta onde o comando roda, na pasta do executável ou em `%LOCALAPPDATA%\nanci`.

Em **Configurações**, **Testar Conexão** confere o certificado e a conexão com o ADN da NFS-e. A SEFAZ (NF-e e CT-e) fica de fora.
