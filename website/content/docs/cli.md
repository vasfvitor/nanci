---
title: "Linha de comando"
description: "Os comandos do Nanci para terminal e rotinas agendadas."
weight: 40
---

O `nanci` faz pelo terminal o mesmo que o aplicativo e usa o mesmo banco. O instalador não traz o CLI; compile com [Go](https://go.dev/dl/):

```bash
git clone https://github.com/vasfvitor/nanci
cd nanci
go build -o nanci.exe ./cmd/nanci
```

As opções de cada comando estão no `--help`, por exemplo `nanci nfe list --help`.

## Primeiros passos

```powershell
nanci company add --cnpj 12345678000199 --name "Minha Empresa" --cert C:\certs\empresa.pfx --uf SP

nanci pull --cnpj 12345678000199        # NFS-e
nanci nfe pull --cnpj 12345678000199    # NF-e
nanci cte pull --cnpj 12345678000199    # CT-e

nanci export xlsx --cnpj 12345678000199 --competencia 2026-09 --out setembro.xlsx
```

A senha do certificado é pedida no terminal na primeira vez e depois fica guardada no Windows. Em rotinas sem terminal, defina `NANCI_CERT_PASSWORD`.

## Comandos

| Comando | O que faz |
|---|---|
| `company add`, `update`, `list` | Cadastra, altera e lista empresas. |
| `credential add`, `list`, `update-path` | Cadastra certificados e atualiza o caminho do arquivo. |
| `pull`, `status`, `list` | Sincroniza, mostra a situação e lista as NFS-e. |
| `export xlsx`, `csv`, `zip`, `danfse`, `danfse-zip` | Exporta as NFS-e. |
| `nfe pull`, `status`, `list`, `pendentes` | Sincroniza e lista as NF-e e as manifestações pendentes. |
| `nfe ciencia`, `manifestar` | Registra a Manifestação do Destinatário. |
| `nfe export zip`, `xml` | Exporta as NF-e. |
| `cte pull`, `status`, `list` | Sincroniza e lista os CT-e. `cte list --nfe <chave>` acha o frete de uma NF-e. |
| `cte export zip`, `xml` | Exporta os CT-e. |
| `nfe testar-conexao`, `cte testar-conexao` | Testa o certificado sem gastar consultas. |
| `nfe reset`, `cte reset` | Apaga as notas da empresa e recomeça a sincronização. |

{{< callout type="warning" >}}
`nfe ciencia`, `nfe manifestar` e os `reset` só simulam. Para executar, repita com `--confirmar`. Uma manifestação enviada não pode ser desfeita.
{{< /callout >}}

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
