<p align="center">
  <img src="logo.svg" alt="Nanci" width="120">
</p>

# Nanci

Aplicativo desktop e de linha de comando (CLI) Open Source para baixar Notas Fiscais de Serviços Eletrônicas (NFS-e) diretamente do Ambiente de Dados Nacional (ADN) e NF-e (modelo 55) da distribuição DF-e da SEFAZ, utilizando seu Certificado Digital A1.

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
- Baixa as NF-e (modelo 55) recebidas pela empresa na distribuição DF-e do Ambiente Nacional da SEFAZ, respeitando o limite de consultas por hora.
- Registra a Manifestação do Destinatário da NF-e: Ciência da Operação em lote e manifestações conclusivas nota a nota, sempre com confirmação explícita.

## O que o Nanci NÃO faz?

- **Não usa portal web municipal**: A consulta ocorre exclusivamente na infraestrutura nacional (ADN para NFS-e, Ambiente Nacional da SEFAZ para NF-e).
- **Não baixa NFC-e, CT-e, NFCom, NF3e nem CF-e SAT**, e não importa XML avulso.
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

#### NF-e (modelo 55)

A distribuição de NF-e exige a UF da empresa. Os comandos ficam em `nanci nfe` e recebem a empresa por `--cnpj`:

```bash
# Cadastrar a UF da empresa
nanci.exe company update --cnpj 12345678000199 --uf SP

# Testar certificado e TLS sem gastar consultas
nanci.exe nfe testar-conexao --cnpj 12345678000199

# Baixar e listar
nanci.exe nfe pull --cnpj 12345678000199
nanci.exe nfe list --cnpj 12345678000199

# Ciência da Operação: sem --confirmar, só mostra o que seria enviado
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos --confirmar
```

| Comando | O que faz |
|---|---|
| `nfe testar-conexao` | Carrega o certificado e testa o TLS com a SEFAZ, sem consumir consultas. |
| `nfe pull` | Baixa resumos, NF-e completas e eventos (até 20 consultas por hora). |
| `nfe status` | Mostra cursor, bloqueios, consultas da última hora e totais. |
| `nfe list` | Lista as NF-e, com filtros por competência, situação, tipo, papel e manifestação. |
| `nfe ciencia` | Registra a Ciência da Operação em lote. |
| `nfe manifestar` | Registra confirmação, desconhecimento ou operação não realizada de uma nota. |
| `nfe pendentes` | Lista as notas sem manifestação conclusiva e seus prazos. |
| `nfe export zip` / `nfe export xml` | Exporta os XMLs em ZIP ou uma nota avulsa. |
| `nfe reset` | Remove as NF-e da empresa e reinicia a sincronização NF-e; necessário antes de trocar o ambiente. Simulação sem `--confirmar`. |

Detalhes de limites, prazos e TLS em [docs/NFE_SEFAZ.md](docs/NFE_SEFAZ.md).



## Relato de Problemas e Contribuição

Encontrou algum problema ou erro? **Nunca anexe seu arquivo .pfx, senhas ou XMLs reais em issues públicas.**

- Verifique o [Guia de Troubleshooting](website/content/docs/troubleshooting.md) para erros comuns (como falso positivos de antivírus).
- Verifique as [Perguntas Frequentes (FAQ)](website/content/docs/faq.md).
- Leia nossas [Políticas de Segurança](SECURITY.md).
- Para reportar um bug seguro, use a [aba Issues](../../issues).

Para desenvolver, compilar localmente ou contribuir com o código fonte, veja as instruções em [DEVELOPMENT.md](docs/DEVELOPMENT.md).
