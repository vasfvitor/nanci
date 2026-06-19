<p align="center">
  <img src="logo.svg" alt="Nanci" width="120">
</p>

# Nanci

Aplicativo desktop e de linha de comando (CLI) Open Source para baixar Notas Fiscais de Serviços Eletrônicas (NFS-e) diretamente do Ambiente de Dados Nacional (ADN) utilizando seu Certificado Digital A1.

<p align="center">
  <img src="docs/screenshots/empresas-dark.png" alt="Nanci Tela Empresas" width="90%">
</p>

## O que é e para quem é?

O **Nanci** é uma ferramenta focada em resolver o problema da captura de NFS-e em âmbito nacional. É ideal para contadores, desenvolvedores e empreendedores que precisam baixar XMLs de múltiplos CNPJs sem precisar depender de serviços externos além do necessário.

Toda a operação ocorre localmente na sua máquina (Local-First).

## O que o Nanci faz?

- Consulta documentos fiscais disponíveis no ADN Contribuintes por NSU de forma incremental.
- Salva o XML bruto da nota localmente na sua máquina.
- Extrai metadados principais.
- Consulta eventos associados aos documentos.
- Exporta dados em Excel (`.xlsx`), CSV, ZIP de XMLs e PDF (DANFSE).
- Permite automação através da sua interface de linha de comando (CLI).
- Suporta cadastro de múltiplas empresas e credenciais A1 (PFX/P12).

## O que o Nanci NÃO faz?

- **Não usa portal web municipal**: A consulta ocorre exclusivamente na infraestrutura nacional (ADN).
- **Não faz scraping ou usa automação de navegador**: Não resolve CAPTCHAs nem simula navegação.
- **Não envia seus XMLs ou Certificados para servidores de terceiros**: A comunicação ocorre apenas entre sua máquina e o Governo.
- **Não garante que notas emitidas pela sua própria empresa apareçam**: O ADN possui regras de distribuição estritas. Não utilize o app como garantidor absoluto de notas emitidas. Veja a [FAQ de documentos vazios](website/content/docs/faq.md).
- **Não substitui validação contábil e fiscal**: O Nanci é uma ferramenta de apoio à extração de dados.

## Onde ficam os dados?

Toda a arquitetura do Nanci é **local**. Certificados, banco de dados SQLite e XMLs originais são armazenados diretamente no seu disco, garantindo privacidade absoluta das suas notas fiscais.
Para entender os caminhos das pastas e como fazer backup, leia nossa [Política de Privacidade de Dados](website/content/docs/privacidade.md).

## Instalação e Uso

### Desktop (Recomendado)

A maneira mais fácil de começar:

1. Vá até a página de [Releases](../../releases) do GitHub.
2. Baixe o instalador mais recente para Windows.
3. Instale e abra o Nanci. A interface gráfica guiará você na adição do certificado e da primeira empresa.

### Linha de Comando (CLI) para Automações

Se você precisa automatizar a captura localmente:

```bash
# Adicionar empresa
nanci.exe company add --cnpj 12345678000199 --name "Minha Empresa" --cert cert.pfx

# Sincronizar
nanci.exe pull --cnpj 12345678000199

# Exportar relatórios
nanci.exe export xlsx --cnpj 12345678000199 --out relatorio.xlsx
```

*A senha do certificado pode ser informada por prompt de comando seguro ou via variável de ambiente `NANCI_CERT_PASSWORD`.*



## Relato de Problemas e Contribuição

Encontrou algum problema ou erro? **Nunca anexe seu arquivo .pfx, senhas ou XMLs reais em issues públicas.**

- Verifique o [Guia de Troubleshooting](website/content/docs/troubleshooting.md) para erros comuns (como falso positivos de antivírus).
- Verifique as [Perguntas Frequentes (FAQ)](website/content/docs/faq.md).
- Leia nossas [Políticas de Segurança](SECURITY.md).
- Para reportar um bug seguro, use a [aba Issues](../../issues).

Para desenvolver, compilar localmente ou contribuir com o código fonte, veja as instruções em [DEVELOPMENT.md](docs/DEVELOPMENT.md).
