---
title: "Privacidade e backup"
description: "Onde o Nanci guarda os dados, com quem ele se comunica e como fazer backup ou apagar tudo."
weight: 60
---

O Nanci não tem servidor. Não há conta, telemetria nem cópia na nuvem: os documentos, o banco de dados e as senhas ficam no seu computador.

## Com quem o Nanci se comunica

Só com os serviços oficiais, por HTTPS com o certificado da empresa (mTLS):

| Serviço | Endereços |
|---|---|
| ADN Contribuintes (NFS-e) | `adn.nfse.gov.br`, `adn.producaorestrita.nfse.gov.br` |
| Ambiente Nacional da NF-e | `www1.nfe.fazenda.gov.br`, `www.nfe.fazenda.gov.br`, `hom1.nfe.fazenda.gov.br` |
| Ambiente Nacional do CT-e | `www1.cte.fazenda.gov.br`, `hom1.cte.fazenda.gov.br` |

## Onde ficam os arquivos

O instalador coloca o programa em `%LOCALAPPDATA%\Programs\Nanci Desktop`. Os dados ficam em outra pasta:

- **Windows**: `%LOCALAPPDATA%\nanci` (por exemplo, `C:\Users\Maria\AppData\Local\nanci`).
- Linux e macOS (só linha de comando): `~/.config/nanci` e `~/Library/Application Support/nanci`.

A variável de ambiente `NANCI_DATA_DIR` troca essa pasta por outra.

No desktop, **Configurações → Abrir Pasta de Dados** abre a pasta no Explorer.

### O que tem na pasta

| Item | Conteúdo |
|---|---|
| `nanci-v1.db` | Banco SQLite: empresas, credenciais (o caminho do certificado, não a senha), dados das notas, estado da sincronização e histórico de manifestações. |
| `blobs/` | Os XMLs originais de todas as notas e eventos, como o governo entregou, um arquivo por documento. As exportações de XML e DANFSe saem daqui. **Não apague**. |
| `logs/` | `nanci-desktop.log` (com até 3 arquivos antigos) e `wails.log`, do aplicativo desktop. A linha de comando não grava log em arquivo. |
| `.env.local` | Opcional. Variáveis como `NANCI_CERT_PASSWORD`. |

Fora dessa pasta ficam o próprio certificado (onde você o deixou) e as senhas, no cofre de senhas do sistema operacional.

{{< callout type="warning" >}}
O banco SQLite e os XMLs não são criptografados pelo Nanci. Use a criptografia de disco do sistema (BitLocker) e uma conta de usuário própria. Em computadores compartilhados, qualquer pessoa com acesso ao seu perfil pode ler esses arquivos.
{{< /callout >}}

## Backup

1. Feche o Nanci.
2. Copie a pasta `%LOCALAPPDATA%\nanci` inteira, incluindo `blobs/`.
3. Guarde também o arquivo do certificado e a senha num lugar seguro.

Para restaurar em outro computador, instale o Nanci, copie a pasta para o mesmo caminho antes de abrir o aplicativo e, se o certificado estiver em outro lugar, atualize o caminho na tela **Credenciais**. A senha será pedida de novo na primeira sincronização.

## Apagar tudo

Desinstalar o Nanci remove só o programa; os dados continuam na pasta. Para apagar tudo:

1. Feche o Nanci e desinstale-o pelo Windows, se quiser.
2. Apague a pasta `%LOCALAPPDATA%\nanci`.
3. No Gerenciador de Credenciais do Windows, remova as entradas que começam com `nanci_certs`.
