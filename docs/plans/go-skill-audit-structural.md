# Go Skill Audit: Long-Term Structural Changes

## Summary

This plan is the larger alignment pass for the `go` and `cobra-viper` guidance. It intentionally goes beyond local cleanup and changes the shape of the CLI and application boundaries so the codebase becomes easier to test, easier to navigate, and less dependent on package-global state.

This plan should be executed after the easy-wins pass, not in parallel with it. The expected outcome is a command-first CLI with explicit dependency assembly, consumer-owned interfaces, and reduced layering noise in the runtime path.

## Implementation Changes

### 1. Replace package-global Cobra state with command factories

- Refactor `internal/cli` so commands are built by functions instead of package-global `var` declarations.
- Introduce a small command environment struct that holds shared dependencies needed by command constructors:
  - application factory
  - stdin/stdout/stderr
  - optional runtime config overrides for tests
- Build the root command from a `NewRootCommand(env CommandEnv) *cobra.Command` function.
- Convert each command file from `init()` registration to explicit construction and registration inside the root builder.
- Keep the user-visible command tree unchanged:
  - `company`
  - `credential`
  - `export`
  - `list`
  - `pull`
  - `status`
  - `init`

### 2. Make the CLI layer a thin adapter

- Move runtime assembly out of command handlers and into a dedicated constructor/factory module under `internal/cli`.
- Replace repeated `newApp()` calls in handlers with an injected application factory shaped like:
  - `type AppFactory func(ctx context.Context) (*app.App, func(), error)`
- Keep Cobra-specific logic in the CLI package only:
  - flag parsing
  - argument validation
  - writer selection
  - calling the app layer
- Ensure `internal/app`, `internal/service/*`, and domain packages continue to have zero Cobra imports.

### 3. Consolidate interface ownership

- Remove duplicate provider/consumer abstractions where the same dependency is modeled in more than one package.
- Choose the consumer package as the single owner for each operational interface.
- Apply this first to XML storage:
  - keep one `XMLStore` interface definition
  - update `internal/app`, `internal/service/sync`, and `internal/files` so the concrete blob store implements the consumer-owned interface
  - delete the duplicate interface declaration from the implementor package
- Review other interfaces in `internal/app/repositories.go` and `internal/service/sync/sync.go` and keep only the ones that reflect a real consumer need.

### 4. Simplify application/runtime composition

- Split dependency validation from the `app.App` data structure so the runtime object is less "bag of dependencies" and more explicit orchestration API.
- Introduce narrower constructors for major subsystems where it reduces field sprawl:
  - sync use cases
  - export use cases
  - credential/password flows
- Keep `app.App` only if it remains the thinnest practical facade for CLI and desktop callers; otherwise replace it with smaller feature-focused services.
- Do not introduce Java-style service/repository layering or generic base abstractions.

### 5. Revisit package boundaries with a domain-first bias

- Keep `internal/` if you want the compiler boundary, but reduce layer-named packages where they add navigation cost without strong value.
- Preferred end state for runtime-oriented code:
  - CLI adapter package
  - domain packages such as `nfse`, `adn`, `report`, `files`
  - a smaller orchestration surface instead of broad `app` plus `service` layering
- For this pass, do not move desktop frontend code or Wails DTO packages.
- Any package move must be paired with import cleanup and a focused regression test pass.

### 6. Build a durable CLI test harness

- Once command factories exist, add a shared test helper that constructs a fresh root command per test with injected fakes.
- Add table-driven tests for:
  - argument validation
  - flag interactions
  - success output
  - runtime failures
  - interactive password-prompt paths
- Tests should execute commands in memory and assert captured output streams directly.

## Execution Order

1. Convert root command creation to a factory.
2. Convert one command group end-to-end, preferably `company`, and establish the pattern.
3. Migrate the remaining command groups.
4. Consolidate `XMLStore` and any other duplicate interfaces exposed by the refactor.
5. Introduce the reusable CLI test harness and expand coverage.
6. Reassess whether `app.App` should remain a single facade or be split.

## Test Plan

- Run `go test ./internal/cli ./internal/app ./internal/service/sync ./internal/files`.
- Run `go test ./...` after each major migration step, not only at the end.
- Smoke test CLI parity before and after the refactor by exercising the main commands with the same sample data.
- Smoke test desktop startup to confirm the app-layer constructor changes did not break Wails runtime assembly.

## Acceptance Criteria

- No Cobra command registration relies on package `init()` side effects or package-global command instances.
- A fresh root command can be constructed per test with injected dependencies.
- The CLI layer owns Cobra concerns only and delegates business operations through injected factories.
- `XMLStore` exists in one place only, owned by the consumer side.
- The resulting structure reduces repeated runtime wiring and improves direct CLI test coverage.

## Assumptions

- This plan intentionally pushes the codebase closer to the named skills, even where that challenges the current repository shape.
- User-visible command names and core behaviors stay stable unless a compatibility issue is explicitly accepted.
- Desktop/Wails concerns remain out of scope except where app-constructor changes require small compatibility updates.
