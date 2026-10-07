---
name: research-doc
description: Use when recording a fact about an external system (SEFAZ, ADN, Receita, a library) or resolving a pending item, so the docs do not drift. Says where each fact lives, how to cite it (NT number, version, date, URL, third-party marking), how research notes, ROADMAP and PENDENCIAS are kept, and to re-verify aged claims before repeating them.
---

# Research doc: record a fact once, with its source

A fact about an external system goes into the doc that owns it, with the source URL and the date checked. Research notes are dated snapshots; the area docs are what the code relies on. Read the target doc before writing, and keep the house language (Portuguese prose, short sentences).

## Where facts live

| Fact | Owner |
|---|---|
| NF-e distribution, endpoints, TLS, query limits (20/h, 656, 60-day rule), manifestação, CNPJ alfanumérico | `docs/NFE_SEFAZ.md` |
| CT-e distribution, actors, events, desacordo, link with NF-e | `docs/CTE_SEFAZ.md` |
| ADN endpoints, DANFSe, environments, chave de acesso | `docs/NFSE_ADN.md` |
| Certificate loading, password storage, use with SEFAZ | `docs/CERTIFICATES.md` |
| Packages, data flow, sync origins, dependency rules | `docs/ARCHITECTURE.md` |
| Toolchain, build, checks, installer, screenshots, site | `docs/DEVELOPMENT.md` |
| What the user does and sees | `website/content/docs/*.md` (see `website-docs`) |
| Raw survey of an external landscape | `docs/pesquisa/AAAA-MM-DD-<tema>.md` |
| Work order and decisions | `docs/ROADMAP.md`; items without a plan: `docs/PENDENCIAS.md` |

The site states the user-facing consequence and cites the official source inline; the area doc carries the detail and the code names (`sync.DistIdleWarningDays`). Never copy a section from one to the other.

## Research notes: `docs/pesquisa/`

Copy the header of `docs/pesquisa/2026-10-07-sefaz-dfe.md`:

```
# <Tema>

Levantado em 07/10/2026.

Responde: <a pergunta, numa linha>.

Todas as fontes foram consultadas em 07/10/2026. Afirmações marcadas "(relato de terceiros)" vêm só de fornecedores, fóruns ou imprensa.
```

- One `## ` section per system; each opens with `Fonte:` (or `Norma:`) naming the document, its version and the URL.
- A claim backed only by vendors, forums, press or code of other projects (ACBr, sped-nfe) ends with "(relato de terceiros)" and its URL.
- End with `## Não encontrado`: what was looked for and not confirmed (`2026-10-07-esocial-reinf-reforma.md`).
- A survey of the repo itself names the commit (`2026-10-07-inventario-manutencao.md`: "Levantado em 07/10/2026 no commit `a0ca132`").
- Do not update an old note in place; write a new dated one. A fact the code depends on moves into the area doc.

## Citing

- A claim about a Nota Técnica cites number, version, date (published or in production) and URL: "NT 2014.002, versão vigente 1.40, em produção desde 08/07/2026". If only a third-party copy was read, say so and name the official host (`2026-10-07-sefaz-dfe.md`: "Cópia da NT: <url>. O original fica no Portal da NF-e"). If the rule is known only from third parties, say that too (`NFE_SEFAZ.md`, "Regra dos 60 dias": "conhecida só por relatos de terceiros ... não foi conferida no texto da NT").
- Prefer official hosts: `www.nfe.fazenda.gov.br/portal` and `www.cte.fazenda.gov.br/portal` (NT links are `exibirArquivo.aspx?conteudo=...`), `www.gov.br/nfse` (manuals under `biblioteca/documentacao-tecnica`), `adn.nfse.gov.br` for endpoints, `www.confaz.fazenda.gov.br/legislacao` for Ajustes SINIEF, `gov.br/esocial`, `gov.br/sped`, `*.receitafederal.gov.br`, `*.serpro.gov.br`. When a host blocks the fetch, say how the data was read (`2026-10-07-federal-serpro.md`, Loja Serpro via the `ccstore` API).
- A rejection code or limit used in code cites the NT where it is defined, and the code constant that applies it.

## ROADMAP and PENDENCIAS

- `docs/ROADMAP.md` orders work in phases. When a phase is done, delete it and record its decisions in the area doc; decisions taken in planning go in "Decisões tomadas em <data>" with the date.
- `docs/PENDENCIAS.md` is the short list of items without a plan. On resolving one, remove it and record the decision (what, why, date) in the area doc, not in the commit message only.
- Both files are edited by the orchestrator only (see `lane`); a lane reports the change it would make.

## Memory versus repo

A Claude memory (`~/.claude/projects/V---code-v-nanci/memory/`) holds session pointers, never the only copy of a fact. Anything a contributor needs (a source, a rule, a decision, a known bug) goes in the repo doc that owns it, and the memory points there.

## Verify before asserting

A dated claim (memory, research note, PENDENCIAS, an old commit message) is a lead, not a fact. Before repeating it, check it at `HEAD`: run the command (`golangci-lint run ./...`, `go test`), grep the code, or reopen the source. On 07/10/2026 a memory written around `a0ca132` reported a `depguard` violation that no longer existed. Write down the date and commit you checked against.

## Checklist

- [ ] Fact placed in the owning doc from the table; site only for what the user acts on
- [ ] Source URL and date checked on every external claim; official host preferred
- [ ] NT cited with number, version, date and URL; third-party copy or third-party-only rule said so
- [ ] "(relato de terceiros)" on every non-official claim; "Não encontrado" section in research notes
- [ ] Resolved pending item removed from `PENDENCIAS.md`, decision with date in the area doc
- [ ] Aged claims re-verified at `HEAD` before being repeated
- [ ] Nothing left only in a Claude memory
