---
title: "Certificado A1"
description: "Certificados aceitos, como o Nanci guarda a senha e o que a SEFAZ exige do certificado."
weight: 50
---

O Nanci usa certificados digitais e-CNPJ tipo **A1**, nos formatos `.pfx` e `.p12`. Certificados A3 (token USB ou cartão) não são suportados.

O mesmo certificado serve para NFS-e, NF-e e CT-e. Não é preciso outra configuração.

## Onde o certificado fica

O Nanci não copia o arquivo: guarda só o caminho. Se você mover ou renovar o certificado, atualize o caminho na tela **Credenciais** (ou com `nanci credential update-path`).

Uma credencial pode ser usada por várias empresas. O certificado da matriz serve para as filiais, desde que a raiz do CNPJ (8 primeiros dígitos) seja a mesma.

{{< theme-image light="/img/screenshots/credenciais-light.png" dark="/img/screenshots/credenciais-dark.png" alt="Tela de credenciais" >}}

## Senha

A senha **não** é gravada no banco de dados do Nanci. Na primeira vez que o certificado é usado, o Nanci pede a senha; se ela abrir o certificado, fica guardada no cofre de senhas do sistema operacional (no Windows, o Gerenciador de Credenciais, numa entrada que começa com `nanci_certs`). Nas próximas vezes ela não é pedida.

- **Desktop**: o pedido aparece num diálogo que informa a finalidade, por exemplo "Sincronização NF-e" ou "Assinatura: Ciência da Operação (12 notas)".
- **Linha de comando**: se o cofre não tiver uma senha válida, o Nanci usa a variável `NANCI_CERT_PASSWORD` e, sem ela, pergunta no terminal.

{{< theme-image light="/img/screenshots/dialogo-senha-certificado-light.png" dark="/img/screenshots/dialogo-senha-certificado-dark.png" alt="Diálogo de senha do certificado" >}}

Para apagar uma senha guardada, abra o Gerenciador de Credenciais do Windows, em **Credenciais do Windows**, e remova as entradas que começam com `nanci_certs`.

## Arquivo `.env.local`

A linha de comando lê variáveis de um arquivo `.env.local`, procurado nesta ordem:

1. Na pasta em que o comando foi executado.
2. Na pasta do executável.
3. Na pasta de dados do Nanci (`%LOCALAPPDATA%\nanci\.env.local` no Windows).

Variáveis já definidas no ambiente têm prioridade sobre o arquivo.

```dotenv
NANCI_CERT_PASSWORD=senha-do-certificado
```

Esse arquivo guarda a senha em texto simples. Prefira deixar o Nanci guardá-la no cofre do sistema.

## Exigências da SEFAZ (NF-e e CT-e)

- **Raiz do CNPJ**: a empresa consultada precisa ter a mesma raiz de CNPJ do certificado. O Nanci confere antes de enviar; a SEFAZ também recusa a consulta (`cStat` 593) ou o evento (`cStat` 631) quando a raiz difere.
- **Chave RSA**: as manifestações da NF-e são assinadas com a chave privada do certificado, que precisa ser RSA, como nos A1 da ICP-Brasil.
- **TLS**: a SEFAZ pede o certificado do cliente renegociando a conexão TLS 1.2. O Nanci faz isso sozinho, sem instalar cadeias ICP-Brasil e sem desligar a verificação do servidor.

**Testar Conexão** (ou `nanci nfe testar-conexao` / `nanci cte testar-conexao`) abre o certificado e faz o handshake TLS sem gastar consultas. Um teste bem-sucedido confirma a senha, a rede e a cadeia do servidor, mas não que a SEFAZ aceitará o certificado: isso só aparece na primeira consulta real.

{{< callout type="error" >}}
Nunca anexe o arquivo `.pfx`/`.p12`, a senha ou um `.env.local` em issues, e-mails ou commits.
{{< /callout >}}
