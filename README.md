<p align="center">
  <img src="logo.svg" alt="Nanci" width="120">
</p>

# Nanci

Aplicativo desktop e de linha de comando (CLI) Open Source para baixar Notas Fiscais de Serviços Eletrônicas (NFS-e) diretamente do Ambiente de Dados Nacional (ADN) NF-e (modelo 55) e CT-e (modelos 57, 67 e 64) da distribuição DF-e da SEFAZ, utilizando seu Certificado Digital A1.

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
- Baixa os CT-e em que a empresa é tomadora, remetente, destinatária, expedidora, recebedora ou autorizada (CT-e, CT-e OS, GTV-e e CT-e Simplificado), com os eventos, pela distribuição DF-e do Ambiente Nacional da SEFAZ. Guarda as chaves das NF-e transportadas, para achar o CT-e do frete de uma nota. Detalhes em [docs/CTE_SEFAZ.md](docs/CTE_SEFAZ.md).

## O que o Nanci NÃO faz?

- **Não usa portal web municipal**: A consulta ocorre exclusivamente na infraestrutura nacional (ADN para NFS-e, Ambiente Nacional da SEFAZ para NF-e e CT-e).
- **Não baixa NFC-e, MDF-e, NFCom, NF3e nem CF-e SAT**, e não importa XML avulso.
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

A distribuição de NF-e exige a UF da empresa (`--uf` em `company add` ou `company update`). Os comandos ficam em `nanci nfe` e recebem a empresa por `--cnpj`:

```bash
# Cadastrar a UF da empresa
nanci.exe company update --cnpj 12345678000199 --uf SP

# Testar certificado e TLS sem gastar consultas
nanci.exe nfe testar-conexao --cnpj 12345678000199

# Baixar e listar
nanci.exe nfe pull --cnpj 12345678000199
nanci.exe nfe list --cnpj 12345678000199 --completude resumo

# Ciência da Operação, passo 1: simulação. Mostra o que seria enviado e não envia nada.
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos

# Passo 2: envio de verdade. Confira a simulação antes.
nanci.exe nfe ciencia --cnpj 12345678000199 --todos-resumos --confirmar
```

> **Atenção:** com `--confirmar`, `nfe ciencia` e `nfe manifestar` enviam eventos assinados à SEFAZ. Um evento registrado não pode ser desfeito pelo Nanci. Rode sempre a simulação primeiro. Se algum evento for rejeitado ou não for enviado, o comando mostra a tabela de resultados e termina com erro.

| Comando | O que faz |
|---|---|
| `nfe testar-conexao` | Carrega o certificado e testa o TLS com a SEFAZ, sem consumir consultas. |
| `nfe pull` | Baixa resumos, NF-e completas e eventos (até 20 consultas por hora). |
| `nfe status` | Mostra cursor, bloqueios, consultas da última hora e totais. |
| `nfe list` | Lista as NF-e. Filtros: `--competencia`/`-m`, `--situacao`, `--completude` (resumo, completa), `--papel`/`-p`, `--manifestacao`, `--emitente` e `--chave` (pode repetir). |
| `nfe ciencia` | Registra a Ciência da Operação em lote, por `--chave` (pode repetir) ou `--todos-resumos`. Simulação sem `--confirmar`. |
| `nfe manifestar` | Registra uma manifestação conclusiva de uma nota: `--chave`, `--tipo` (confirmacao, desconhecimento ou nao_realizada) e `--justificativa` (15 a 255 caracteres, obrigatória para nao_realizada). Simulação sem `--confirmar`. |
| `nfe pendentes` | Lista as notas sem manifestação conclusiva e seus prazos. `--vencendo-em N` mostra só as que vencem em até N dias. |
| `nfe export zip` | Exporta os XMLs em ZIP (`--out`/`-o`, padrão `nfe.zip`). Filtros: `--competencia`/`-m`, `--papel`/`-p` e `--chave`; `--incluir-resumos` inclui os resumos e `--incremental` exporta só o que ainda não foi exportado. |
| `nfe export xml` | Exporta o XML de uma nota (`--chave`) para `--out`/`-o`, por padrão `<chave>.xml`. |
| `nfe reset` | Remove as NF-e da empresa, nos dois ambientes, e reinicia a sincronização NF-e. Simulação sem `--confirmar`. |

Detalhes de limites, prazos e TLS em [docs/NFE_SEFAZ.md](docs/NFE_SEFAZ.md).



## Relato de Problemas e Contribuição

Encontrou algum problema ou erro? **Nunca anexe seu arquivo .pfx, senhas ou XMLs reais em issues públicas.**

- Verifique o [Guia de Troubleshooting](website/content/docs/troubleshooting.md) para erros comuns (como falso positivos de antivírus).
- Verifique as [Perguntas Frequentes (FAQ)](website/content/docs/faq.md).
- Leia nossas [Políticas de Segurança](SECURITY.md).
- Para reportar um bug seguro, use a [aba Issues](../../issues).

Para desenvolver, compilar localmente ou contribuir com o código fonte, veja as instruções em [DEVELOPMENT.md](docs/DEVELOPMENT.md).
