---
title: "Instalação e Uso"
description: "Como instalar o Nanci no Windows e configurar o primeiro uso com certificado A1."
weight: 20
icon: "download"
toc: true
---

## Download

O Nanci está disponível para Windows. Você não precisa instalar linguagens de programação ou dependências complexas.

1. Acesse a página de [Releases no GitHub](https://github.com/vasfvitor/nanci/releases/latest).
2. Baixe o arquivo `nanci-desktop-windows-amd64-installer.exe` mais recente.
3. Execute o instalador no seu computador.

Por padrão, o executável desktop é instalado em `%LOCALAPPDATA%\Programs\Nanci Desktop`.

O banco de dados, os logs e as configurações ficam na sua pasta de usuário em `%LOCALAPPDATA%\nanci`. Nada é enviado para a nuvem.

## Primeiro Uso (Desktop)

Ao abrir o Nanci pela primeira vez:

1. Vá na aba **Credenciais** na barra lateral.
2. Adicione uma credencial informando o arquivo do seu certificado `.pfx` (A1).
3. Vá na aba **Empresas**.
4. Clique em **Adicionar Empresa**, insira o CNPJ e vincule a credencial que você acabou de adicionar.
5. Pronto! Agora você pode clicar no botão de **Sincronizar** (ícone de nuvem) ao lado do nome da empresa para começar a baixar as notas.

Dependendo do volume de notas da sua empresa, a primeira sincronização pode demorar um pouco. As próximas serão incrementais (apenas o que for novo).
