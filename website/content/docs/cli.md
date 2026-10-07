---
title: "Linha de comando"
description: "Referência dos comandos do Nanci para terminal, rotinas agendadas e automações."
weight: 40
---

O `nanci` faz pelo terminal o mesmo que o aplicativo desktop: cadastra empresas, sincroniza, lista e exporta. Os dois usam o mesmo banco de dados, então uma empresa cadastrada num aparece no outro.

## Instalar

O instalador traz só o aplicativo desktop. Para ter o `nanci.exe`, compile a partir do código-fonte (requer [Go](https://go.dev/dl/) na versão do `go.mod`):

```bash
git clone https://github.com/vasfvitor/nanci
cd nanci
go build -o nanci.exe ./cmd/nanci
```

Todos os comandos têm ajuda: `nanci --help`, `nanci nfe --help`, `nanci nfe list --help`. As opções globais são `-v`/`--verbose` (logs de depuração) e `--trace` (registra também as requisições e o corpo das respostas, com CNPJ e chaves mascarados).

## Primeiros passos

```bash
# Cadastrar a empresa em produção, com o certificado
nanci company add --cnpj 12345678000199 --name "Minha Empresa" --cert C:\certs\empresa.pfx --env producao --uf SP

# Baixar NFS-e, NF-e e CT-e
nanci pull --cnpj 12345678000199
nanci nfe pull --cnpj 12345678000199
nanci cte pull --cnpj 12345678000199

# Exportar as NFS-e de setembro para Excel
nanci export xlsx --cnpj 12345678000199 --competencia 2026-09 --out setembro.xlsx
```

{{< callout type="warning" >}}
Sem `--env producao`, a empresa é cadastrada em `producao_restrita`, o ambiente de testes do governo, e nenhuma nota real aparece. Para trocar depois: `nanci company update --cnpj 12345678000199 --env producao`.
{{< /callout >}}

## Senha do certificado

O Nanci procura a senha nesta ordem:

1. O cofre de senhas do sistema operacional (Gerenciador de Credenciais do Windows), se ela já foi informada antes.
2. A variável de ambiente `NANCI_CERT_PASSWORD`.
3. Um pedido no terminal.

Uma senha correta é guardada no cofre, então nas próximas execuções ela não é pedida. Para uma rotina agendada, rode o comando uma vez de forma interativa, ou defina `NANCI_CERT_PASSWORD`:

```powershell
$env:NANCI_CERT_PASSWORD = "minha-senha"
nanci pull --cnpj 12345678000199
```

A variável também pode ficar num arquivo `.env.local`; veja [Certificado A1](../certificados/#arquivo-envlocal).

## Empresas e credenciais

Uma credencial é um certificado `.pfx`/`.p12` cadastrado; várias empresas podem usar a mesma.

| Comando | O que faz |
|---|---|
| `company add` | Cadastra uma empresa. Obrigatórios: `--cnpj`/`-c` e `--name`/`-n`. Certificado: `--cert`/`-p` (cria uma credencial, com rótulo opcional em `--credential-label`) ou `--credential-id` (usa uma existente). `--env`/`-e`: `producao` ou `producao_restrita` (padrão). `--uf`: UF, exigida para NF-e e CT-e. |
| `company update` | Muda `--name`, `--env` ou `--uf` (vazio remove) da empresa `--cnpj`. |
| `company list` | Lista as empresas. |
| `company assign-credential` | Liga a credencial `--credential-id` à empresa `--cnpj`. |
| `credential add` | Cadastra um certificado: `--cert` e `--label`. |
| `credential list` | Lista as credenciais. |
| `credential update-path` | Atualiza o caminho do certificado (`--cert`) da credencial `--credential-id`, por exemplo depois de mover o arquivo. |
| `init` | Cria o banco e as pastas locais. Opcional: qualquer comando faz isso. |

A importação inicial da NFS-e é escolhida no `company add`: `--sync-start-policy from_now` (padrão, só documentos a partir de hoje), `all` (todo o histórico) ou `since_date` com `--sync-start-date AAAA-MM-DD`. Os atalhos `--last-12-months` e `--last-5-years` fazem o mesmo que `since_date`.

## NFS-e

| Comando | O que faz |
|---|---|
| `pull` | Baixa as NFS-e e eventos novos do ADN. |
| `status` | Mostra o cursor, a última sincronização e os totais. |
| `list` | Lista as NFS-e. Filtros: `--competencia`/`-m` (`AAAA-MM`), `--direcao`/`-d` (`tomada`, `prestada`, `intermediario`) e `--nao-vistos`. |
| `export xlsx` | Planilha Excel (padrão `export.xlsx`). |
| `export csv` | Planilha CSV (padrão `export.csv`). |
| `export zip` | ZIP com os XMLs originais (padrão `export.zip`). |
| `export danfse` | DANFSe em PDF de uma nota, por `--chave` (padrão `danfse.pdf`). |
| `export danfse-zip` | ZIP com os DANFSe (padrão `danfses.zip`). |

Todos recebem `--cnpj`/`-c`. Os `export` aceitam `--out`/`-o`, `--competencia`/`-m`, `--direcao`/`-d` e `--incremental` (só o que ainda não foi exportado).

## NF-e

Os comandos ficam em `nanci nfe` e recebem a empresa por `--cnpj`/`-c`. A empresa precisa ter UF cadastrada.

| Comando | O que faz |
|---|---|
| `nfe testar-conexao` | Testa o certificado e o TLS com a SEFAZ sem gastar consultas. |
| `nfe pull` | Baixa resumos, NF-e completas e eventos (até 20 consultas por hora). |
| `nfe status` | Mostra cursor, bloqueio, consultas da última hora e totais. |
| `nfe list` | Lista as NF-e. Filtros: `--competencia`/`-m`, `--situacao`, `--completude` (`resumo`, `completa`), `--papel`/`-p`, `--manifestacao`, `--emitente`, `--chave` (pode repetir) e `--nao-vistos`. |
| `nfe ciencia` | Registra a Ciência da Operação, por `--chave` (pode repetir) ou `--todos-resumos`. |
| `nfe manifestar` | Registra uma manifestação conclusiva de uma nota: `--chave`, `--tipo` (`confirmacao`, `desconhecimento`, `nao_realizada`) e `--justificativa` (15 a 255 caracteres, obrigatória para `nao_realizada`). |
| `nfe pendentes` | Lista as notas sem manifestação conclusiva e seus prazos. `--vencendo-em N` mostra só as que vencem em até N dias. |
| `nfe export zip` | ZIP com os XMLs e eventos (padrão `nfe.zip`). Filtros: `--competencia`/`-m`, `--papel`/`-p`, `--chave`; `--incluir-resumos` e `--incremental`. |
| `nfe export xml` | XML de uma nota (`--chave`), por padrão `<chave>.xml`. |
| `nfe reset` | Apaga as NF-e da empresa nos dois ambientes e reinicia a sincronização. |

{{< callout type="warning" >}}
`nfe ciencia`, `nfe manifestar` e `nfe reset` só simulam: mostram o que fariam e não mudam nada. Para executar, repita o comando com `--confirmar`. Uma manifestação enviada à SEFAZ não pode ser desfeita. Se algum evento for rejeitado ou não for enviado, o comando mostra a tabela de resultados e termina com código de saída diferente de zero.
{{< /callout >}}

```bash
# 1. Simulação: lista o que seria enviado
nanci nfe ciencia --cnpj 12345678000199 --todos-resumos

# 2. Envio, depois de conferir a simulação
nanci nfe ciencia --cnpj 12345678000199 --todos-resumos --confirmar

# 3. No próximo pull chegam os XMLs completos
nanci nfe pull --cnpj 12345678000199
```

## CT-e

Os comandos ficam em `nanci cte`, recebem a empresa por `--cnpj`/`-c` e usam o mesmo certificado e a mesma UF da NF-e.

| Comando | O que faz |
|---|---|
| `cte testar-conexao` | Testa o certificado e o TLS com a SEFAZ sem gastar consultas. |
| `cte pull` | Baixa CT-e, CT-e OS, GTV-e e eventos (até 20 consultas por hora). |
| `cte status` | Mostra cursor, bloqueio, consultas da última hora e totais por papel. |
| `cte list` | Lista os CT-e. Filtros: `--competencia`/`-m`, `--situacao`, `--papel`/`-p` (casa com qualquer papel da empresa no documento), `--modelo` (`57`, `64`, `67`), `--emitente`, `--tomador`, `--nfe` (chave de uma NF-e transportada), `--chave` (pode repetir) e `--nao-vistos`. |
| `cte export zip` | ZIP com os XMLs e eventos (padrão `cte.zip`). Filtros: `--competencia`/`-m`, `--papel`/`-p`, `--chave`; `--incremental`. |
| `cte export xml` | XML de um CT-e (`--chave`), por padrão `<chave>.xml`. |
| `cte reset` | Apaga os CT-e da empresa nos dois ambientes e reinicia a sincronização. Simulação sem `--confirmar`. |

## Rotina agendada

Um exemplo de script para o Agendador de Tarefas do Windows, que baixa tudo e exporta as NFS-e novas:

```powershell
$cnpj = "12345678000199"
nanci pull --cnpj $cnpj
nanci nfe pull --cnpj $cnpj
nanci cte pull --cnpj $cnpj
nanci export zip --cnpj $cnpj --incremental --out "D:\fiscal\nfse-$(Get-Date -Format yyyyMMdd).zip"
```

Os `pull` de NF-e e CT-e respeitam o limite da SEFAZ: se a consulta estiver bloqueada, o comando avisa o horário da próxima consulta permitida e não envia nada. Agendar a cada hora ou menos é seguro.

A CLI escreve os logs no terminal (saída de erro), não em arquivo. Para guardá-los, redirecione: `nanci pull --cnpj $cnpj 2>> nanci.log`.
