<p align="center">
  <img src="logo.svg" alt="Nanci" width="120">
</p>

<h1 align="center">Nanci</h1>

<p align="center">
  Baixe NFS-e, NF-e e CT-e direto do governo, com o certificado A1 da empresa.<br>
  Para Windows. Tudo fica no seu computador.
</p>

<p align="center">
  <a href="https://github.com/vasfvitor/nanci/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/vasfvitor/nanci"></a>
  <a href="LICENSE"><img alt="Licença GPL-3.0" src="https://img.shields.io/github/license/vasfvitor/nanci"></a>
  <a href="https://vasfvitor.github.io/nanci/"><img alt="Documentação" src="https://img.shields.io/badge/docs-vasfvitor.github.io%2Fnanci-blue"></a>
</p>

<p align="center">
  <img src="docs/screenshots/empresas-dark.png" alt="Tela de empresas do Nanci" width="90%">
</p>

## O que faz

| Documento | Origem |
|---|---|
| **NFS-e** | Ambiente de Dados Nacional (ADN) |
| **NF-e** | Distribuição DF-e da SEFAZ, com Manifestação do Destinatário |
| **CT-e** | Distribuição DF-e da SEFAZ |

Várias empresas, cada uma com o seu certificado A1. Exporta para Excel, CSV, XML e DANFSe. Não há servidor no meio: a conexão vai direto ao governo.

## Instalação

Baixe o instalador da [última versão](https://github.com/vasfvitor/nanci/releases/latest) e siga o [primeiro uso](https://vasfvitor.github.io/nanci/docs/instalacao/).

O CLI não vem no instalador. Compile com `go build -o nanci.exe ./cmd/nanci` e veja os [comandos](https://vasfvitor.github.io/nanci/docs/cli/).

## Documentação

- **Uso**: [vasfvitor.github.io/nanci](https://vasfvitor.github.io/nanci/)
- **Técnica**: [arquitetura](docs/ARCHITECTURE.md), [NFS-e/ADN](docs/NFSE_ADN.md), [NF-e/SEFAZ](docs/NFE_SEFAZ.md), [CT-e/SEFAZ](docs/CTE_SEFAZ.md), [certificados](docs/CERTIFICATES.md)
- **Desenvolvimento**: [DEVELOPMENT.md](docs/DEVELOPMENT.md), [CONTRIBUTING.md](CONTRIBUTING.md)

Nunca anexe certificado, senha, XMLs reais ou o banco em issues. Vulnerabilidades vão por e-mail, conforme o [SECURITY.md](SECURITY.md).

## Licença

[GPL-3.0](LICENSE).
