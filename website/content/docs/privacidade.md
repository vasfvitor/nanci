---
title: "Privacidade e Segurança"
description: "Onde seus dados ficam armazenados e como o Nanci lida com as suas informações."
weight: 30
icon: "shield"
toc: true
---

O **Nanci** foi concebido desde o princípio como uma aplicação *Local-First*. Isso significa que nós não hospedamos seus dados, não criamos uma cópia na nuvem, e não processamos seus documentos fiscais em nossos servidores.

Toda a extração, conexão com o certificado digital e armazenamento ocorre localmente na sua máquina.

## Onde o app é instalado?

No Windows, o instalador desktop padrão coloca o executável em:

- `%LOCALAPPDATA%\Programs\Nanci Desktop`

## Onde os dados ficam salvos?

O Nanci armazena seus dados e configurações em uma pasta específica do seu usuário:

- **Windows:** `C:\Users\SeuUsuario\AppData\Local\nanci`

### O que é salvo nessa pasta?

1. `nanci-v1.db`: O banco de dados SQLite principal, onde ficam salvas as configurações das empresas (CNPJs cadastrados) e os metadados das notas fiscais extraídos.
2. `blobs/`: Pasta contendo os arquivos originais temporários `.xml` das NFS-es baixadas da base de dados do governo. Não é um backup e o app pode limpar essa pasta periodicamente para economizar espaço. 
3. `.env.local`: Arquivo opcional para variáveis de ambiente locais, como `NANCI_CERT_PASSWORD`.
4. `logs/`: Pasta com os arquivos `nanci-desktop.log` e `wails.log` de diagnóstico da aplicação.

## O que NÃO é enviado para lugar nenhum

- Os arquivos `.xml` da pasta `blobs/`.
- O seu certificado digital e a senha dele (que reside apenas na sua máquina e/ou no gerenciador de credenciais seguro do seu Sistema Operacional).
- O banco de dados SQLite.
- A lista dos CNPJs que você está pesquisando.

A única comunicação de rede externa feita pela aplicação é HTTPS (com mTLS usando o certificado digital do respectivo CNPJ) diretamente para os endereços (URLs) oficiais do Ambiente de Dados Nacional (ADN) e RFB.

## Como fazer backup?

Como a arquitetura é local, o backup é simples: copie o arquivo `nanci-v1.db` e as demais pastas mencionadas acima para o seu sistema de armazenamento seguro preferido.

Se precisar reinstalar o Nanci em outra máquina, basta copiar esses dados para o caminho correspondente no novo sistema antes de abrir a aplicação, e ela lerá todos os dados como você os deixou.

## Como apagar meus dados?

Para remover todos os seus dados capturados e rastros locais da aplicação, basta:

1. Fechar o Nanci.
2. Deletar completamente a pasta do caminho informado na seção "Onde os dados ficam salvos?".
3. Remover a senha do certificado do chaveiro/gerenciador de credenciais do sistema operacional (se salva).
