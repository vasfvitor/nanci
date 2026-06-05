# Repository Guidelines

## Project Structure & Module Organization
`cmd/nanci` contains the CLI entrypoint. Core application logic lives in `internal/app`, with CLI adapters in `internal/cli`, sync orchestration in `internal/service/sync`, and persistence in `internal/store` plus `internal/store/migrations`. Domain-specific code is split across `internal/adn`, `internal/nfse`, `internal/report`, `internal/files`, and `internal/foundation`. The desktop app is under `internal/desktop`: Go/Wails backend files at the module root and the Vue 3 frontend in `internal/desktop/frontend/src`.

## Build, Test, and Development Commands
Use `go build -o nanci.exe ./cmd/nanci` to build the CLI from the repo root. `make fmt` runs `goimports` and `gofmt`; `make lint` runs `golangci-lint`; `make test` runs `go test ./...`; `make check` chains formatting, vulnerability, lint, test, and security checks. For the desktop frontend, run `pnpm install` once in `internal/desktop/frontend`, then `pnpm run dev`, `pnpm run build`, or `pnpm run lint`. For Wails live development, use `wails dev` from `internal/desktop`.

## Coding Style & Naming Conventions
Follow standard Go formatting: tabs, `gofmt`, and `goimports` with the local prefix `github.com/vasfvitor/nanci`. Keep Go packages lowercase and focused; exported identifiers use `CamelCase`, internal helpers use `camelCase`. Vue/TypeScript files use the existing ESLint + Prettier setup. Match current naming patterns such as `DocumentsPage.vue`, `MainLayout.vue`, and `PasswordPromptDialog.vue`.

## Testing Guidelines
Place Go tests next to the code they cover using `*_test.go`. Existing examples include `internal/store/companies_test.go` and `internal/foundation/cnpj/cnpj_test.go`. Run `make test` before opening a PR; use `go test ./...` for quick iteration. Add regression tests for parser, storage, and path-handling changes. No frontend test suite is configured yet, so at minimum run `pnpm run lint` and exercise key flows in `wails dev`.

## Commit & Pull Request Guidelines
Prefer scoped, imperative commit subjects in the form `scope: change`, for example `frontend: fix company dialog spacing`, `cli: validate --competencia input`, `frontend,cli: align export flow`, or `wails: wire new desktop command`. Use repo areas as scopes: `frontend`, `cli`, `app`, `store`, `sync`, `nfse`, `adn`, `wails`, or a small combination when one change truly spans layers. Keep commits narrow and descriptive; add an issue reference when relevant. PRs should explain the behavioral change, list verification commands, and include screenshots for desktop/frontend UI changes. Note any schema, migration, or certificate-handling impact explicitly.

## Security & Configuration Tips
Do not commit certificate files, passwords, SQLite data, or exported fiscal documents. Prefer `NANCI_CERT_PASSWORD` for local runs instead of hardcoding secrets. Review `make security` output before merging changes that touch networking, storage, or auth flows.
