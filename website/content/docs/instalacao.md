---
title: "Instalação e primeiro uso"
description: "Como instalar o Nanci no Windows, cadastrar a primeira empresa e fazer a primeira sincronização."
weight: 20
---

## Requisitos

- Windows 10 ou 11, 64 bits.
- Certificado digital e-CNPJ tipo A1 (arquivo `.pfx` ou `.p12`) e a senha dele. Certificados A3 (token ou cartão) não são suportados.

## Instalar

1. Abra a [última versão no GitHub](https://github.com/vasfvitor/nanci/releases/latest).
2. Baixe o arquivo `nanci-desktop-windows-amd64-<versão>-installer.exe`.
3. Execute o instalador. Ele instala só para o seu usuário, em `%LOCALAPPDATA%\Programs\Nanci Desktop`, e não pede permissão de administrador.

O executável não tem assinatura de código, então o Windows SmartScreen pode avisar na primeira execução. Para conferir o download, compare o SHA-256 do instalador com o arquivo `nanci-checksums.txt` publicado na mesma versão:

```powershell
Get-FileHash .\nanci-desktop-windows-amd64-v0.5.1-installer.exe -Algorithm SHA256
```

Veja também [Solução de problemas](../troubleshooting/#aviso-do-smartscreen-ou-do-antivírus).

## Cadastrar a primeira empresa

1. Abra o Nanci. A tela inicial é **Empresas**.
2. Clique em **Adicionar** e preencha o CNPJ e o nome.
3. Na parte da credencial, escolha **Criar nova credencial**, dê um rótulo e informe o caminho do arquivo `.pfx` ou `.p12`. Se você já cadastrou o certificado na tela **Credenciais**, escolha **Usar credencial existente**.
4. Em **Ambiente da Empresa**, deixe **Produção** para baixar documentos reais. **Produção restrita** é o ambiente de testes do governo, onde notas reais não aparecem.
5. Informe a **UF** da empresa se for baixar NF-e ou CT-e. A SEFAZ exige a UF na consulta.
6. Escolha a política de importação inicial da NFS-e: a partir de hoje, últimos 12 meses, últimos 5 anos, a partir de uma data ou todo o histórico. NF-e e CT-e ignoram essa escolha, porque a SEFAZ só guarda os documentos por cerca de 3 meses.

{{< theme-image light="/img/screenshots/dialogo-adicionar-empresa-light.png" dark="/img/screenshots/dialogo-adicionar-empresa-dark.png" alt="Diálogo de adicionar empresa" >}}

O certificado não é copiado: o Nanci guarda o caminho do arquivo. Se você mover o `.pfx`, atualize o caminho na tela **Credenciais**.

## Sincronizar

Escolha a empresa no seletor do menu lateral e abra a página do documento que quer baixar:

- **NFS-e**: clique em **Sincronizar NFS-e**.
- **NF-e**: clique em **Sincronizar NF-e**. Use **Testar Conexão** antes para conferir o certificado sem gastar consultas.
- **CT-e**: clique em **Sincronizar CT-e**.

Na primeira vez, o Nanci pede a senha do certificado. Se ela estiver correta, fica guardada no Gerenciador de Credenciais do Windows e não é pedida de novo.

A primeira sincronização pode demorar, dependendo do volume. As seguintes são incrementais: o Nanci guarda até onde leu (o NSU) e continua dali. A NF-e e o CT-e têm limite de 20 consultas por hora; veja [NF-e](../nfe/#limite-de-consultas).

## Próximos passos

- [NFS-e](../nfse/), [NF-e](../nfe/) e [CT-e](../cte/): o que cada fonte entrega e como usar cada página.
- [Privacidade e backup](../privacidade/): onde os dados ficam e como copiá-los.
- [Linha de comando](../cli/): automatizar a sincronização e a exportação.
