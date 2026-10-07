---
name: lane
description: Use when delegating an implementation step of nanci to a subagent in an isolated worktree (a "lane"). Builds the agent prompt with the repo's non-negotiable mechanics (base commit, go.work and dist setup, LF, signed commits without trailers, bindings plus mocks, verification per area, report format) so nothing is re-derived from memory.
---

# Lane: delegate one implementation step

A lane is one subagent, one worktree, a few commits, disjoint files from every other running lane. The orchestrator integrates with the `integrate` skill afterwards.

## Before spawning

1. Decide the scope in files: two lanes must not touch the same file. `client.ts`, `types/desktop.ts`, `wailsjs/` and `scripts/generate-screenshots.ts` are one unit; `go.mod`/`go.sum` of a module are one unit; `docs/ROADMAP.md` and `docs/PENDENCIAS.md` are edited by the orchestrator only.
2. Pick the base: `main` for maintenance, the integration branch for a feature. Agent worktrees may start at a stale commit, so the prompt always begins with a reset to the base tip.
3. Spawn with `model: opus`, `isolation: worktree`, a short `name`, and the prompt below filled in.

## Prompt skeleton

```
Repo nanci. Read AGENTS.md (and <area docs>). You are in an isolated worktree:
first `git reset --hard <base>` (worktree is clean). Work only there; do not push.
Setup (gitignored, never commit): `cp /v/_code_v/nanci/go.work ./go.work`;
for the desktop module `mkdir -p internal/desktop/frontend/dist && cp -r
/v/_code_v/nanci/internal/desktop/frontend/dist/* internal/desktop/frontend/dist/`;
for frontend work `pnpm install --frozen-lockfile` in internal/desktop/frontend.
Files stay LF (check with `file`). Commits are signed by the repo config; do NOT
add any attribution trailer. Subjects `scope: change` per AGENTS.md.

Task: <what, with file paths, signatures, and the decisions already made>.
Commits: <one line each, in order>.
Verify: <the block for the area, below>.
Report: SHAs, worktree path, verification output, judgment calls, anything that
looks like a pre-existing bug (do not fix it unless asked).
```

## Verification blocks

- Go root: `go build ./... && go vet ./... && go test ./...`; `golangci-lint run --allow-parallel-runners ./...`; `govulncheck ./...` when `go.mod` changed; `GOWORK=off go mod tidy` with no diff.
- Desktop Go: `cd internal/desktop && go build ./... && go test ./...` after copying `dist`. Any change to an exported `app` API needs this even if the lane did not touch `internal/desktop`.
- Bindings: `cd internal/desktop && wails generate module`, then `git diff frontend/wailsjs` shows only the expected methods, and the `vi.mock` list in `src/platform/wails/client.test.ts` plus the mocks in `scripts/generate-screenshots.ts` are updated in the same commit (the build does not type-check that script).
- Frontend: `pnpm run lint:check && pnpm run test:unit && pnpm run build`; `pnpm run screenshots` when a page or Quasar changed.
- Store: `go test ./internal/store -run Migration -v`; after `schema.sql` edits regenerate sqlc with `CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate` (the cgo link fails with TDM-GCC on this machine) and commit `sqlgen`. Never edit an applied migration; add a new one.
- Docs: every factual claim about an external system carries its source URL and the date checked; third-party-only claims say so.

## Rules the agent must not relearn

- One lane per worktree; no edits to `docs/ROADMAP.md`, `docs/PENDENCIAS.md`, `MEMORY`.
- Portuguese fiscal vocabulary stays Portuguese in identifiers (AGENTS.md).
- A pre-existing bug found on the way is reported, not fixed, unless the prompt says otherwise.
- Nothing is pushed; `gh` mutations are blocked.
