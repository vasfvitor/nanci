---
title: "Linha de comando"
description: "Os comandos do Nanci para terminal e rotinas agendadas."
weight: 40
---

O `nanci` faz pelo terminal o mesmo que o aplicativo e usa o mesmo banco. Ele não vem no instalador: é um download à parte.

## Instalar

1. Na [última versão](https://github.com/vasfvitor/nanci/releases/latest), baixe `nanci-cli-windows-amd64-<versão>.zip`, da mesma versão do aplicativo.
2. Extraia o `nanci.exe` para uma pasta, por exemplo `%LOCALAPPDATA%\Programs\nanci-cli`.
3. No menu Iniciar, abra **Editar as variáveis de ambiente para sua conta**, escolha **Path**, clique em **Editar**, depois em **Novo**, e cole o caminho da pasta. Não use o comando `setx`: ele corta um Path longo.
4. Abra um terminal novo e confira com `nanci --version`. A versão deve ser a mesma de **Configurações** › **Sobre o Nanci**.

{{< callout type="warning" >}}
A atualização automática do aplicativo não atualiza o `nanci`. Quando o aplicativo se atualizar, baixe o `nanci` da mesma versão. Um `nanci` mais antigo que o aplicativo recusa abrir os dados e avisa.
{{< /callout >}}

Para compilar a partir do código, com [Go](https://go.dev/dl/): `go build -o nanci.exe ./cmd/nanci` no repositório.

As opções de cada comando estão no `--help`, por exemplo `nanci nfe list --help`.

## Primeiros passos

```powershell
nanci company add --cnpj 12345678000199 --name "Minha Empresa" --cert C:\certs\empresa.pfx --uf SP

nanci pull --cnpj 12345678000199        # NFS-e
nanci nfe pull --cnpj 12345678000199    # NF-e
nanci cte pull --cnpj 12345678000199    # CT-e

nanci export xlsx --cnpj 12345678000199 --competencia 2026-09 --out setembro.xlsx
```

A senha do certificado é pedida no terminal na primeira vez e depois fica guardada no Windows. Para desistir no pedido da senha, use Ctrl+C. Em rotinas sem terminal, defina `NANCI_CERT_PASSWORD`.

## Comandos

| Comando | O que faz |
|---|---|
| `init` | Cria ou atualiza o banco e a pasta de dados. |
| `company add`, `update`, `list` | Cadastra, altera e lista empresas. `update --sync-start-date` troca a data inicial da NFS-e antes da primeira sincronização ou depois de um `reset`. |
| `company assign-credential --cnpj <cnpj> --credential-id <id>` | Liga a empresa a um certificado já cadastrado. O ID sai em `credential list`. |
| `credential add`, `list`, `update-path` | Cadastra certificados e atualiza o caminho do arquivo. |
| `pull`, `status`, `list` | Sincroniza, mostra a situação e lista as NFS-e. |
| `reset` | Recomeça a sincronização da NFS-e desde o NSU 0, sem apagar documentos, e libera a troca da data inicial. |
| `export xlsx`, `csv`, `zip`, `xml`, `danfse`, `danfse-zip` | Exporta as NFS-e. Nos formatos em lote, `--chave` (pode repetir) limita às notas informadas. |
| `nfe pull`, `status`, `list`, `pendentes` | Sincroniza e lista as NF-e e as manifestações pendentes. |
| `nfe ciencia`, `manifestar` | Registra a Manifestação do Destinatário. |
| `nfe export zip`, `xml`, `csv`, `xlsx`, `danfe-zip`, `danfe` | Exporta as NF-e em XML, planilha ou DANFE (PDF). O DANFE não sai para resumos. |
| `nfe import <arquivo>...` | Importa arquivos XML de NF-e e de eventos que a empresa já tem, sem consultar a SEFAZ. |
| `nfe buscar --chave <chave>` | Busca uma NF-e na SEFAZ pela chave de acesso. |
| `nfe recuperar-nsu`, `cte recuperar-nsu` | Pede à SEFAZ, um a um, os NSUs que a sincronização pulou. `--limite N` limita as consultas da execução; `--descartar` encerra a lista sem consultar. |
| `cte pull`, `status`, `list` | Sincroniza e lista os CT-e. `cte list --nfe <chave>` acha o frete de uma NF-e. |
| `cte export zip`, `xml`, `csv`, `xlsx`, `dacte`, `dacte-zip` | Exporta os CT-e. O DACTE só existe para o CT-e modelo 57. |
| `cte import <arquivo>...` | Importa arquivos XML de CT-e e de eventos que a empresa já tem, sem consultar a SEFAZ. |
| `nfe testar-conexao`, `cte testar-conexao` | Testa o certificado sem gastar consultas. |
| `nfe reset`, `cte reset` | Apaga as notas da empresa e recomeça a sincronização. |
| `cte desacordo` | Registra a prestação do serviço em desacordo de um CT-e em que a empresa é tomadora, com `--chave` e `--observacao` (15 a 255 caracteres). Mostra o prazo de 45 dias da autorização. |

{{< callout type="warning" >}}
`nfe ciencia`, `nfe manifestar`, `cte desacordo`, os `reset` e o `recuperar-nsu --descartar` só simulam. Para executar, repita com `--confirmar`. Uma manifestação ou um desacordo enviado não pode ser cancelado pelo nanci.
{{< /callout >}}

Cada `nfe buscar` gasta uma das 20 consultas por hora de NF-e da empresa, as mesmas do `nfe pull`. O `recuperar-nsu` gasta uma por NSU e, sem `--limite`, deixa 2 da hora para a sincronização.

## Rotina agendada

Para o Agendador de Tarefas do Windows:

```powershell
$cnpj = "12345678000199"
nanci pull --cnpj $cnpj
nanci nfe pull --cnpj $cnpj
nanci cte pull --cnpj $cnpj
nanci export zip --cnpj $cnpj --incremental --out "D:\fiscal\nfse-$(Get-Date -Format yyyyMMdd).zip"
```

Rodar de hora em hora é seguro: se a SEFAZ ainda não permitir nova consulta, o comando avisa e não envia nada. Os logs vão para o terminal; redirecione com `2>> nanci.log` se quiser guardá-los.

## Para scripts e agentes de IA

Com `--json`, o comando escreve o resultado como um único JSON no stdout, sem linhas de progresso. Um erro sai como a última linha do stderr, em JSON, com código de saída 1:

```json
{"error": {"code": "sync_running", "message": "erro: sincronização já em andamento para esta empresa (NF-e) em outro processo do nanci (aplicativo); aguarde a sincronização atual terminar"}}
```

Os avisos de log, quando há, também saem no stderr como uma linha JSON cada, antes do erro. O `code` vem vazio quando o erro não tem um código próprio. Os códigos são `canceled`, `sefaz_blocked`, `sync_running`, `no_requests_left`, `not_found`, `invalid_input`, `wrong_password` e `password_required` (a senha não está salva e não há terminal para pedi-la).

Um agente de IA, como Claude Code, Codex ou Gemini CLI, pode usar o `nanci` pelo terminal com `--json`. A senha do certificado sai do Windows (guardada na primeira vez que você a digita no terminal ou no aplicativo) ou de `NANCI_CERT_PASSWORD`; sem terminal, o `nanci` não pede a senha e avisa como resolver. O que envia evento ou apaga dados só simula sem `--confirmar`, então o agente pode mostrar o plano antes de executar. O aplicativo e o terminal usam o mesmo banco: se um estiver sincronizando uma empresa, o outro recusa a mesma origem com `sync_running` em vez de consultar a SEFAZ ao mesmo tempo.

O skill `nanci-cli-agente` ensina o agente a usar o `nanci`: o `--json`, o que fazer com cada código de erro, o limite de consultas e a regra de simular antes de `--confirmar`. Baixe `nanci-skill-<versão>.zip` da mesma versão e extraia a pasta `nanci-cli-agente` na pasta de skills do seu agente:

| Agente | Pasta |
|---|---|
| Codex e Gemini CLI | `%USERPROFILE%\.agents\skills` ([Codex](https://learn.chatgpt.com/docs/build-skills), [Gemini CLI](https://geminicli.com/docs/cli/skills/)) |
| Claude Code | `%USERPROFILE%\.claude\skills` ([Claude Code](https://code.claude.com/docs/en/skills)) |
| Antigravity | `%USERPROFILE%\.gemini\config\skills` ([Antigravity](https://antigravity.google/docs/skills.md)) |

As páginas oficiais escrevem `~`; no Windows, é a sua pasta de usuário (`%USERPROFILE%`). Quando baixar um `nanci` novo, troque o skill pelo da mesma versão.
