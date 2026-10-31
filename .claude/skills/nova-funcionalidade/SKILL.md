---
name: nova-funcionalidade
description: Start here for any new feature, change or fix in nanci, especially when the person asking is not a developer ("quero que o app faça...", "adiciona um botão", "muda a tela de...", "corrige..."). Turns the request into a plain-Portuguese plan for approval, picks the area skill, works on a local branch in small commits, runs scripts/verify.sh, stops before anything risky (SEFAZ events, production, real certificate, applied migrations, generated files, weakening checks) and explains the result without jargon.
---

# Nova funcionalidade: do pedido ao commit

Quem pede pode não ser desenvolvedor. Ele não vai revisar o código, então este skill e as verificações automáticas são a revisão. Fale com ele em português simples, sem jargão; o código, os commits e os comentários seguem o AGENTS.md (inglês, termos fiscais em português).

## 1. Entender antes de mexer

- Reescreva o pedido em uma ou duas frases: o que o usuário do app vai poder fazer ou ver depois. Pergunte o que faltar (em qual tela, para qual documento, o que acontece quando dá errado). Não adivinhe regra fiscal: se a dúvida for de legislação ou da SEFAZ, diga que precisa de fonte (skill `research-doc`).
- Procure se já existe algo parecido: `rg` pelo texto do botão ou da mensagem, `docs/ROADMAP.md` e `docs/PENDENCIAS.md` (pode já ter decisão tomada, inclusive "não fazer").

## 2. Escolher o caminho

| O pedido mexe em | Leia antes |
|---|---|
| Uma tela do app, um botão, uma ação nova no desktop | `desktop-method` |
| Um comando do `nanci` no terminal | `cli-command` |
| O que fica guardado no banco (tabela, coluna) | `migration` |
| Uma consulta ou envio à SEFAZ ou ao ADN | `sefaz-call` |
| Um tipo novo de documento (MDF-e, por exemplo) | `new-source` |
| O site de documentação | `website-docs` |
| Um fato sobre lei, SEFAZ, ADN ou biblioteca | `research-doc` |

`lane` e `integrate` servem para dividir um trabalho grande entre vários agentes; não são necessários para trabalhar sozinho.

## 3. Plano em português simples, e esperar o ok

Mostre, sem código:

- o que muda para quem usa o app (a tela, o botão, a mensagem);
- o que não muda;
- os passos, um por commit (veja abaixo);
- os riscos e o que vai precisar de decisão;
- como ele vai poder testar.

Só comece depois que ele aprovar. Se no meio do trabalho o plano mudar, pare e mostre de novo.

## 4. Trabalhar numa branch local

Não precisa de PR. O trabalho fica numa branch local e entra na `main` por fast-forward, com o histórico linear:

```bash
git switch main
git switch -c feat/<nome-curto>        # ou: git worktree add ../nanci-<nome> -b feat/<nome-curto>
```

- Um commit por passo, e cada commit compila e passa nos testes sozinho. A ordem costuma ser de baixo para cima: domínio e banco, depois `internal/app`, depois Wails e frontend, depois docs.
- Mensagem no formato `escopo: mudança` do AGENTS.md (`frontend: add the export button to the CT-e page`), sem rodapé de atribuição.
- Para corrigir um commit anterior da branch, faça `git commit --fixup <sha>` e, no fim, junte sem editor: `GIT_SEQUENCE_EDITOR=: git rebase -i --autosquash main`.
- Se a `main` andou: `git rebase main`. Conflito que você não entende por completo: pare e explique.
- No fim: `git switch main && git merge --ff-only feat/<nome-curto>`, e apague a branch. Não faça push: quem roda a sessão decide.

## 5. Verificar

```bash
bash scripts/verify.sh
```

Roda o mesmo que o CI: frontend, Go nos dois módulos, lint, `go.mod`, bindings do Wails e o teste de arquitetura. Rode antes de cada commit que mexe em código e sempre antes do fast-forward. Quando algo falhar, corrija o código. Não:

- apague, pule (`t.Skip`) ou afrouxe um teste para ele passar, a não ser que o comportamento tenha mudado de propósito e a pessoa tenha aprovado;
- mude `.golangci.yml`, `eslint.config.ts`, o `frozenMigrations` de `internal/store/migrations_frozen_test.go` ou o teste `TestBoundMethodsCrossOnlyDTOs` para a verificação passar;
- use `--no-verify`, `//nolint` ou `eslint-disable` sem um motivo escrito ao lado que a pessoa aprovou.

Quando a tela mudou, teste também no app de verdade (veja "Como testar" abaixo) e gere as capturas da página (`pnpm run screenshots <rota>` em `internal/desktop/frontend`).

## 6. Pare e pergunte antes de

- **Enviar qualquer coisa à SEFAZ que não se desfaz**: manifestação (ciência, confirmação, desconhecimento, operação não realizada), desacordo de CT-e, qualquer evento. Tem valor legal. Em teste, use só servidores falsos (veja `sefaz-call`).
- **Usar o ambiente de produção com um CNPJ real.** Na NF-e e no CT-e cada consulta gasta a cota de 20 por hora, e uma consulta fora de ordem bloqueia o CNPJ por uma hora (656), inclusive para o sistema do contador que usa o mesmo CNPJ. Desenvolva com `devdata` e o certificado de teste.
- **Usar um certificado real ou uma senha**, ou pôr um deles, um XML fiscal real ou o banco em arquivo, log ou commit.
- **Editar uma migration que já existe** em `internal/store/migrations_v2/`: mudança no banco é sempre uma migration nova (`migration`).
- **Editar à mão arquivo gerado**: `internal/desktop/frontend/bindings/` (gere com `wails3 generate bindings -clean=true -ts -i`) e `internal/store/sqlgen/` (gere com sqlc, veja `migration`).
- **Mudar as regras**: `AGENTS.md`, os skills, a configuração de lint, os testes de arquitetura.
- **Adicionar uma dependência nova** (Go ou npm) ou trocar a versão do Wails.
- **Apagar dados, telas, comandos ou testes** que existem.
- **Mexer em muitas áreas ao mesmo tempo** sem o plano ter previsto.
- **Reescrever a `main`**: `git push --force`, `git reset --hard` na `main`, rebase da `main`.

## 7. Como testar no app

```bash
go run ./cmd/seeddev                    # cria devdata/ com uma empresa e o certificado de teste (senha mockdata)
NANCI_DATA_DIR="$PWD/devdata" bash -c 'cd internal/desktop && wails3 dev'
```

O certificado de teste não passa numa SEFAZ ou ADN de verdade: sincronizar com ele falha, e isso é o esperado. Para ver uma tela com dados, use as capturas de tela (`pnpm run screenshots`), que usam dados de mentira.

## 8. Explicar o resultado

Termine com, em português simples:

- o que mudou para quem usa o app, e o que ficou igual;
- como experimentar (os comandos acima e o caminho na tela);
- os commits, um por linha;
- o resultado do `verify.sh`;
- o que ficou de fora ou precisa de decisão, e qualquer coisa estranha que você notou e não corrigiu.
