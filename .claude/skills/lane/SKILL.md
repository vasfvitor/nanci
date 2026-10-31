---
name: lane
description: Use when delegating an implementation step of nanci to a subagent in an isolated worktree (a "lane"). Builds the agent prompt with the repo's non-negotiable mechanics (base commit, go.work and dist setup, LF, signed commits without trailers, bindings plus mocks, verification per area, report format) so nothing is re-derived from memory.
---

# Lane: delegate one implementation step

A lane is one subagent, one worktree, a few commits, disjoint files from every other running lane. The orchestrator integrates with the `integrate` skill afterwards.

## Before spawning

1. Decide the scope in files: two lanes must not touch the same file. `src/platform/wails/client/`, `src/types/desktop/`, `frontend/bindings/` and `scripts/generate-screenshots.ts` are one unit; `go.mod`/`go.sum` of a module are one unit; `docs/ROADMAP.md` and `docs/PENDENCIAS.md` are edited by the orchestrator only.
2. Pick the base: `main` for maintenance, the integration branch for a feature. Agent worktrees may start at a stale commit, so the prompt always begins with a reset to the base tip.
3. Spawn with `isolation: worktree`, a short `name`, an explicit `model` when the session has a policy for subagents, and the prompt below filled in.

## Prompt skeleton

```
Repo nanci. Read AGENTS.md (and <area docs>). You are in an isolated worktree:
first `git reset --hard <base>` (worktree is clean). Work only there; do not push.
Setup (gitignored, never commit): `go work init . ./internal/desktop
./third_party/go-pkcs12`; for the desktop module without frontend work, stub the
embedded bundle as CI does: `mkdir -p internal/desktop/frontend/dist && echo
'<div></div>' > internal/desktop/frontend/dist/index.html`; for frontend work
`pnpm install --frozen-lockfile` in internal/desktop/frontend (`pnpm run build`
then writes the real dist). Files stay LF (check with `file`). Keep the repo's
commit signing if it is configured; do NOT add any attribution trailer. Subjects
`scope: change` per AGENTS.md.

Task: <what, with file paths, signatures, and the decisions already made>.
Commits: <one line each, in order>.
Verify: <the block for the area, below>.
Report: SHAs, worktree path, verification output, judgment calls, anything that
looks like a pre-existing bug (do not fix it unless asked).
```

## Verification blocks

- Go root: `go build ./... && go vet ./... && go test ./...`; `golangci-lint run --allow-parallel-runners ./...` with a private `GOLANGCI_LINT_CACHE` (the shared cache reports findings from other worktrees with foreign paths); `govulncheck ./...` when `go.mod` changed; `GOWORK=off go mod tidy` with no diff.
- Desktop Go: `cd internal/desktop && go build ./... && go test ./...` with `frontend/dist` present (built or stubbed, see the setup above). Any change to an exported `app` API needs this even if the lane did not touch `internal/desktop`.
- Bindings: `cd internal/desktop && wails3 generate bindings -clean=true -ts -i` (CLI pinned: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`), then `git status frontend/bindings` shows only the expected methods and models, and the `vi.mock` list in `src/platform/wails/client.test.ts` plus the mocks in `scripts/generate-screenshots.ts` are updated in the same commit (`vue-tsc -b` type-checks that script through `tsconfig.node.json`).
- Frontend: `pnpm run lint:check && pnpm run test:unit && pnpm run build`; `pnpm run screenshots` when a page or Quasar changed.
- Store: `go test ./internal/store -run Migration -v`; after a migration regenerate sqlc (it reads the goose Up sections of `migrations_v2`) with `CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate` (`CGO_ENABLED=0` avoids needing a C compiler, and the cgo link fails with some Windows toolchains) and commit `sqlgen`. Never edit an applied migration; add a new one.
- Docs: every factual claim about an external system carries its source URL and the date checked; third-party-only claims say so.

## Rules the agent must not relearn

- One lane per worktree; no edits to `docs/ROADMAP.md` or `docs/PENDENCIAS.md`.
- Portuguese fiscal vocabulary stays Portuguese in identifiers (AGENTS.md).
- A pre-existing bug found on the way is reported, not fixed, unless the prompt says otherwise.
- Nothing is pushed and nothing is written on GitHub (PR, issue, comment): the person running the session does that.
