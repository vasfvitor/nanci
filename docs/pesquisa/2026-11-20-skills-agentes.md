# Skills e instruções para agentes de código

Levantado em 20/11/2026.

Responde: em que pasta cada agente de código procura skills e que arquivo de instruções ele lê, para que os skills do nanci funcionem em Claude Code, Codex, Gemini CLI e Antigravity.

Todas as fontes foram consultadas em 20/11/2026. Afirmações marcadas "(relato de terceiros)" vêm só de blogs ou fóruns.

## Formato Agent Skills

Fonte: especificação Agent Skills, https://agentskills.io/specification

- Um skill é uma pasta com um `SKILL.md`: front matter YAML e o texto em Markdown.
- Campos obrigatórios: `name` (até 64 caracteres, minúsculas, números e hífens, igual ao nome da pasta) e `description` (até 1024 caracteres, o que o skill faz e quando usar). Opcionais: `license`, `compatibility`, `metadata`, `allowed-tools` (experimental).
- O agente carrega só `name` e `description` no início e lê o `SKILL.md` inteiro quando o skill se aplica.

Os 11 skills do nanci seguem esse formato sem campos de uma ferramenta só.

## Codex

Fonte: OpenAI, "Build skills", https://learn.chatgpt.com/docs/build-skills (o endereço antigo `developers.openai.com/codex/skills` redireciona para lá).

- Lê `.agents/skills` em cada pasta, da pasta atual até a raiz do repositório; do usuário, `$HOME/.agents/skills`.
- `name` e `description` são obrigatórios.
- Segue pastas de skill que são symlinks.

## Gemini CLI

Fontes: https://geminicli.com/docs/cli/skills/ e https://geminicli.com/docs/cli/gemini-md/

- Skills do projeto em `.gemini/skills/` ou no apelido `.agents/skills/`; quando os dois existem, o apelido vale.
- O arquivo de instruções padrão é `GEMINI.md`. Outros nomes entram por `context.fileName` no `settings.json`, por exemplo `["AGENTS.md", "GEMINI.md"]`. O nanci põe `{"context": {"fileName": ["AGENTS.md"]}}` em `.gemini/settings.json`.
- As duas páginas avisam que, para quem usa o plano gratuito ou o Google One, o Gemini CLI foi substituído pelo Antigravity CLI em 18/06/2026.

## Antigravity

Fonte: https://antigravity.google/docs/skills.md

- Skills do projeto em `<raiz do workspace>/.agents/skills/` e em `.agents/skills/` de cada pasta acima, até a raiz do repositório; globais em `~/.gemini/config/skills/`. A pasta antiga `.agent/skills` ainda funciona.
- A página não fala de `AGENTS.md`. Leitura nativa do `AGENTS.md` desde a versão 1.20.3 (05/03/2026), com o `GEMINI.md` valendo em caso de conflito, só aparece em blogs (relato de terceiros): https://agentpedia.codes/blog/antigravity-agents-md-guide

## Claude Code

Fontes: https://code.claude.com/docs/en/skills e https://code.claude.com/docs/en/memory

- Skills do projeto só em `.claude/skills/<nome>/SKILL.md`, na pasta atual e nas de cima até a raiz do repositório. A página não menciona `.agents/skills`.
- Uma pasta de skill do projeto pode ser symlink. No Windows o git faz checkout de um symlink como arquivo de texto, a não ser com `core.symlinks` ligado; por isso o nanci gera uma cópia com `scripts/sync-skills.sh`.
- O front matter só é lido quando o `---` de abertura está na primeira linha; o aviso de cópia gerada fica logo depois dele.
- Lê o `AGENTS.md` como instruções do projeto desde a v2.1.277, desde que não haja `CLAUDE.md`, `.claude/CLAUDE.md` nem `CLAUDE.local.md` na pasta atual ou acima. Com um deles presente, lê só o `CLAUDE.md`, a não ser que ele importe `@AGENTS.md`.

## Não encontrado

- Se o Antigravity segue pastas de skill que são symlinks.
- Texto oficial do Antigravity sobre `AGENTS.md` e sua precedência em relação ao `GEMINI.md`.
- Se o Gemini CLI precisa de alguma configuração para ativar skills; a página só fala de `/skills enable` e `/skills disable`.
