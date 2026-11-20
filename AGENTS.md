# Repository Guidelines

*Clear is better than clever.* Go code should be boring in the best possible way—predictable, consistent, and immediately understandable to a new developer opening the file for the first time. When unsure, delete the abstraction.

## Project Structure & Module Organization
Package map, data flow and dependency rules: `docs/ARCHITECTURE.md`. The desktop app is under `internal/desktop`, a separate Go module: the Go/Wails backend at its root and the Vue 3 frontend in `internal/desktop/frontend/src`.

## Build, Test, and Development Commands
Build the CLI with `go build -o nanci.exe ./cmd/nanci` from the repo root. Run `bash scripts/verify.sh` (or `make verify`) before every commit: it runs everything CI checks, Go and frontend. In `internal/desktop/frontend`, `pnpm run lint` is the fixing command and `pnpm run lint:check` the checking one. For Wails live development, use `wails3 dev` from `internal/desktop` (Wails v3.1.0; see `docs/DEVELOPMENT.md`).

## Coding Style & Naming Conventions
Run `goimports` with the local prefix `github.com/vasfvitor/nanci` (`make fmt`). Brazilian fiscal domain terms from the tax authority vocabulary stay in Portuguese across all layers (Go, DTO, frontend types, DB columns): `ChaveAcesso`/`ChavesAcesso` (chave de acesso da NFS-e, NF-e ou CT-e), `CNPJ`, `Competence` (competência), `tomada`/`prestada`/`intermediario` (document direction), `tomador`/`remetente`/`expedidor`/`recebedor` (CT-e actors), `chave_acesso` (DB column). Do not rename these to English. The rule covers identifiers too: manifestação is `Manifestacao*` in Go, DTO and TypeScript names (`ManifestacaoRecord`, `TipoManifestacao`) and `manifestacao`/`manifestacoes` in SQL (`nfe_manifestacoes`). English verbs around a Portuguese noun are fine, as in `RegisterManifestacao` or `ListNFePendingManifestacoes`.

## Desktop Frontend Architecture
The Vue frontend uses Vue 3, Composition API, `<script setup lang="ts">`, Pinia, and the `@` source alias. Keep route pages as composition surfaces: pages wire stores, composables, dialogs, notifications, and router state, while feature logic lives in `src/composables` and cross-route state lives in `src/stores`.

Do not import the generated Wails bindings (`frontend/bindings`) or `@wailsio/runtime` from pages, stores, or components. The only frontend modules that should import them are `src/platform/wails/client/` (its files), `src/platform/wails/events.ts`, and `src/platform/wails/runtime.ts`. Frontend code should depend on `src/types/desktop/` (imported as `@/types/desktop`), `desktopClient`, and the platform wrappers instead of generated bindings. Avoid `window.go`, `@ts-expect-error` Wails calls, raw `EventsOff`, and `v-html` for query or JSON output.

Use focused Pinia setup stores for state that must survive route navigation, component unmount/remount, or be visible in multiple places: any user-visible background operation or long-running state, such as sync, export progress, password prompt requests, log subscriptions, import queues, and selected filters the user expects to keep after visiting another screen. Route-local `ref` state is fine for disposable UI such as an open dialog flag or the currently selected row for an edit dialog. Unsaved company and credential form drafts go through `useFormDraft`.

Composables should orchestrate feature actions and return a small typed API. They may own short-lived loading flags for an action scoped to the current mounted view, but they should delegate durable state to Pinia. If a composable starts an async Wails operation whose result can outlive the current route, mark it in a store before awaiting and clear it in `finally`; add a regression test that creates a second composable instance while the promise is pending. Use `storeToRefs` when returning reactive Pinia state from composables.

Shared formatting and display logic belongs in `src/utils`, not templates. Keep computed derivations out of templates when they are reused or non-trivial.

Dialogs should expose typed props and emits where practical. They should not reach into generated Wails APIs; if backend behavior is needed, call a feature composable or take a loader prop from the page. Components and pages never import `desktopClient`: the calls live in composables and stores. Event subscriptions must use the platform event wrapper and keep the unsubscribe function returned by `Events.On`.

Desktop Go methods exposed to Wails should return DTOs from `internal/desktop/desktopapi`, not domain structs. Add or update DTO mappers when fields cross the desktop boundary, and keep file path construction/export dispatch in the backend for export flows.

Do not hardcode style colors or use Tailwind-style theme selectors (such as `dark:bg-grey-9`). For dark theme styling adaptations, use Quasar-native dynamic class bindings based on `$q.dark.isActive` (e.g. `:class="$q.dark.isActive ? 'bg-grey-9' : 'bg-grey-1'"`) or leverage Quasar's built-in CSS variables (e.g. `var(--q-separator-color)`).

## Testing Guidelines
Add regression tests for parser, storage, and path-handling changes.

For the desktop frontend, use Vitest with happy-dom, Vue Test Utils, and Pinia test setup. Add focused tests for Wails client mapping/error normalization, event cleanup wrappers, Pinia stores, composables that build Wails requests, formatter utilities, and route-remount behavior for long-running operations. Also smoke-test key Wails flows manually when behavior crosses the desktop runtime: add company, add credential, sync, list documents, export, direct query, password prompt, and logs console.

## Commit & Pull Request Guidelines
Commit subjects are scoped and imperative, `scope: change` (`frontend: fix company dialog spacing`); the scopes are listed in `CONTRIBUTING.md`. Keep commits narrow; add an issue reference when relevant. PRs should explain the behavioral change, list verification commands, and include screenshots for desktop/frontend UI changes. Note any schema, migration, or certificate-handling impact explicitly.

## Working With Agents
This file is the one instruction file for every coding agent (Codex and Claude Code read it as is; `.gemini/settings.json` points Gemini CLI at it). Do not add a `CLAUDE.md` or `GEMINI.md`: Claude Code stops reading `AGENTS.md` when a `CLAUDE.md` exists.

The project skills live in `.agents/skills/<name>/SKILL.md`, in the Agent Skills format, which Codex, Gemini CLI and Antigravity load from there. `.claude/skills/` is a copy for Claude Code generated by `scripts/sync-skills.sh`: edit `.agents/skills` and run the script; `scripts/verify.sh` and CI fail when the copy is stale. A tool that does not load skills on its own reads the `SKILL.md` named below when the task matches:

- `nova-funcionalidade`: start of any new feature, change or fix, whoever drives the agent, especially when the person asking is not a developer.
- `desktop-method`: a backend capability exposed in the desktop app, through every layer.
- `cli-command`: a new or changed `nanci` command.
- `migration`: any SQLite schema or data change.
- `sefaz-call`: a new or changed call to SEFAZ or the ADN.
- `new-source`: a new document source (such as MDF-e) or a feature that spans all sources.
- `website-docs`: a page of the website or its screenshots.
- `research-doc`: a fact about an external system, or a resolved pending item.
- `nanci-cli-agente`: running the `nanci` CLI for someone, with `--json`.
- `lane` and `integrate`: delegating a step to another agent in its own worktree and landing its commits.

Rules for every change, delegated or not:

- Every file is LF. `.gitattributes` normalizes on commit, but scripts that write files on Windows must still write LF (Python: `newline='\n'`).
- Commits carry no attribution trailer. When the clone signs commits (the upstream repo does), keep them signed: a history rewrite drops signatures, so re-sign before handing off.
- A change to Wails bindings regenerates `frontend/bindings` and updates the mocks in `src/platform/wails/client.test.ts` and `scripts/generate-screenshots.ts` in the same commit (steps in the `desktop-method` skill). CI fails on stale bindings or a stale `go.mod`.
- Exported `app` API changes must keep `internal/desktop` building; run its build and tests even when the change did not touch it.
- `docs/ROADMAP.md` and `docs/PENDENCIAS.md` are edited by the orchestrator only; a resolved item is deleted, not left behind. Facts about external systems go to `docs/pesquisa/` with source URL and date.
- Maintenance lands on `main` as individual commits, no PR, no squash. Features that need review go through a PR. Agents do not push; the person running the session does.

## Security & Configuration Tips
Do not commit certificate files, passwords, SQLite data, or exported fiscal documents. Prefer `NANCI_CERT_PASSWORD` for local runs instead of hardcoding secrets. Review `make security` output before merging changes that touch networking, storage, or auth flows.
