---
title: "Instalação e primeiro uso"
description: "Instalar o Nanci no Windows e fazer a primeira sincronização."
weight: 20
---

Você precisa de Windows 10 ou 11 e de um certificado e-CNPJ A1 (`.pfx` ou `.p12`).

## Instalar

Baixe `nanci-desktop-windows-amd64-<versão>-installer.exe` da [última versão](https://github.com/vasfvitor/nanci/releases/latest) e execute. Não precisa de administrador.

O instalador não tem assinatura de código, então o SmartScreen pode avisar. Para conferir o arquivo, compare o hash com o `nanci-checksums.txt` da mesma versão:

```powershell
Get-FileHash .\nanci-desktop-windows-amd64-v0.6.0-installer.exe -Algorithm SHA256
```

## Atualizar

O Nanci procura uma versão nova quando abre. Se houver, aparece o aviso **Nova versão disponível**: clique em **Atualizar e reiniciar**, e o Nanci baixa a versão nova, fecha e abre de novo já atualizado. Para procurar a qualquer momento, use **Procurar atualizações** em **Configurações** › **Sobre o Nanci**.

A atualização só é instalada se tiver a assinatura do projeto. Ela não começa enquanto uma sincronização estiver em andamento. Seus dados e as empresas cadastradas continuam onde estavam.

Se o aviso não aparecer, baixe o instalador da última versão e execute por cima da instalação atual; os dados são mantidos.

## Cadastrar a empresa

Sem empresa cadastrada, a primeira tela mostra os três passos até as primeiras notas, e o botão **Cadastrar primeira empresa** abre o cadastro.

{{< theme-image light="/img/screenshots/boas-vindas-light.png" dark="/img/screenshots/boas-vindas-dark.png" alt="Tela de boas-vindas" >}}

Em **Empresas**, clique em **Adicionar** e informe CNPJ, nome e certificado. Preencha também:

- **UF**, se for baixar NF-e ou CT-e.
- **O que deseja importar no primeiro sync?**: vale só para a NFS-e. NF-e e CT-e trazem o que a SEFAZ ainda tiver, cerca de 3 meses.

{{< theme-image light="/img/screenshots/dialogo-adicionar-empresa-light.png" dark="/img/screenshots/dialogo-adicionar-empresa-dark.png" alt="Diálogo de adicionar empresa" >}}

## Sincronizar

Escolha a empresa no menu lateral, abra **NFS-e**, **NF-e** ou **CT-e** e clique em **Sincronizar**. A senha do certificado é pedida uma vez e fica guardada no Windows.

Na **NFS-e**, o botão **Sincronizar NFS-e** traz a marca **Primeira sincronização** até a empresa sincronizar pela primeira vez.

{{< theme-image light="/img/screenshots/primeira-sincronizacao-light.png" alt="Botão de sincronizar com a marca de primeira sincronização" >}}
