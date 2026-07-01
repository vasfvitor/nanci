# Go Skill Audit: Easy Wins

## Summary

This plan targets low-risk changes that improve alignment with the `go` and `cobra-viper` guidance without changing the repository's current high-level architecture. The work stays inside the existing `cmd/nanci` + `internal/cli` + `internal/app` layout and avoids cross-package moves.

The goal is to improve Cobra error behavior, stdout/stderr discipline, testability at the CLI boundary, and a few clear Go-idiom mismatches that can be corrected with limited churn.

## Implementation Changes

### 1. Tighten Cobra root command behavior

- Update `internal/cli/root.go` so the root command sets both `SilenceUsage: true` and `SilenceErrors: true`.
- Keep `cmd/nanci/main.go` as the single process-exit boundary.
- Change `internal/cli.Execute` to return an `error` instead of an exit code, and let `cmd/nanci/main.go` print the final error to `stderr` and choose the exit code.
- Do not print command errors inside subcommands once the root boundary handles them.

### 2. Fix CLI output routing

- Replace direct `fmt.Print*` success/info output in `internal/cli/*.go` with Cobra command writers where practical:
  - use `cmd.OutOrStdout()` for normal command output
  - use `cmd.ErrOrStderr()` only for user-facing failures or prompts
- Preserve the current special case in `TerminalCredentialProvider` where password prompts stay on `stderr` to keep `stdout` pipe-friendly.
- Keep the actual textual output stable unless a test requires a minor normalization.

### 3. Add focused in-memory CLI tests

- Add a new `internal/cli/root_test.go` covering:
  - runtime errors do not print usage
  - final errors are surfaced once
  - root persistent flags still parse correctly
- Add one command-level test file for a representative command group, preferably `export` or `company`, that executes commands in memory with `SetArgs`, `SetOut`, and `SetErr`.
- Reset any package-global command state between tests so tests can run reliably even if they are not parallelized.
- Do not add compiled-binary `os/exec` tests.

### 4. Remove obvious "log and return" duplication at non-boundaries

- In `internal/app/pull.go`, stop logging the sync failure immediately before returning it.
- In `internal/service/sync/sync.go`, keep structured context on parse/decode failures, but avoid emitting the same failure at a level that will duplicate the eventual boundary log.
- Default rule for this pass:
  - application and service layers return wrapped errors
  - CLI main and desktop entrypoints remain the logging/reporting boundaries
- Do not redesign logging infrastructure in this pass.

### 5. Prefer newer stdlib helpers where the replacement is mechanical

- Replace `sort.Slice` in `internal/service/sync/sync.go` with `slices.SortFunc`.
- Only make replacements that are behavior-preserving and local.
- Do not perform repo-wide "modernization" beyond cases touched by this plan.

## Test Plan

- Run `go test ./internal/cli ./internal/app ./internal/service/sync`.
- Run `go test ./...` before handoff.
- Manually smoke test:
  - `nanci --help`
  - one successful read-only command such as `nanci company list`
  - one failing command path to confirm usage is not printed on runtime failure
  - one interactive flow that prompts for certificate password

## Acceptance Criteria

- Root-command runtime errors are printed once and do not include usage text.
- CLI success output goes through Cobra writers rather than raw `fmt.Print*` in command handlers.
- There is direct test coverage for at least the root command and one representative subcommand path.
- `internal/app/pull.go` no longer logs and returns the same failure.
- The `sort.Slice` call in sync code is removed in favor of `slices.SortFunc`.

## Assumptions

- The existing layered repository structure stays in place for this plan.
- Backward-compatible CLI text is preferred over rewording output.
- It is acceptable for the CLI package to keep package-global commands for now, provided tests explicitly control and reset state.
