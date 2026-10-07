---
title: "Desenvolvimento"
description: "Como compilar o Nanci a partir do código-fonte e contribuir."
weight: 90
---

O Nanci é código aberto, sob a licença [GPL-3.0](https://github.com/vasfvitor/nanci/blob/main/LICENSE). O código é Go; o aplicativo desktop usa [Wails](https://wails.io/) com Vue 3 e Quasar.

A documentação para desenvolvedores fica no repositório, junto do código:

| Documento | Conteúdo |
|---|---|
| [DEVELOPMENT.md](https://github.com/vasfvitor/nanci/blob/main/docs/DEVELOPMENT.md) | Requisitos, ambiente local, rodar o desktop em modo de desenvolvimento, gerar o instalador e as capturas de tela. |
| [CONTRIBUTING.md](https://github.com/vasfvitor/nanci/blob/main/CONTRIBUTING.md) | Estrutura do projeto, certificado de teste, padrões de código, commits e testes. |
| [ARCHITECTURE.md](https://github.com/vasfvitor/nanci/blob/main/docs/ARCHITECTURE.md) | Pacotes, fluxo de dados e o loop de sincronização comum às três fontes. |
| [NFSE_ADN.md](https://github.com/vasfvitor/nanci/blob/main/docs/NFSE_ADN.md), [NFE_SEFAZ.md](https://github.com/vasfvitor/nanci/blob/main/docs/NFE_SEFAZ.md), [CTE_SEFAZ.md](https://github.com/vasfvitor/nanci/blob/main/docs/CTE_SEFAZ.md) | Detalhes de cada integração: endpoints, regras, limites e modelo de dados. |

## Compilar em poucos passos

Com Go, Node.js, pnpm e o Wails CLI instalados (versões em [DEVELOPMENT.md](https://github.com/vasfvitor/nanci/blob/main/docs/DEVELOPMENT.md#requisitos)):

```bash
git clone https://github.com/vasfvitor/nanci
cd nanci

# Linha de comando
go build -o nanci.exe ./cmd/nanci

# Desktop com recarga automática
cd internal/desktop
wails dev
```
