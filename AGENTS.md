# Repository Guidelines

## Project Structure & Module Organization
`cmd/nanci` contains the CLI entrypoint. Core application logic lives in `internal/app`, with CLI adapters in `internal/cli`, sync orchestration in `internal/service/sync`, and persistence in `internal/store` plus `internal/store/migrations`. Domain-specific code is split across `internal/adn`, `internal/nfse`, `internal/report`, `internal/files`, and `internal/foundation`. The desktop app is under `internal/desktop`: Go/Wails backend files at the module root and the Vue 3 frontend in `internal/desktop/frontend/src`.

## Build, Test, and Development Commands
Use `go build -o nanci.exe ./cmd/nanci` to build the CLI from the repo root. `make fmt` runs `goimports` and `gofmt`; `make lint` runs `golangci-lint`; `make test` runs `go test ./...`; `make check` chains formatting, vulnerability, lint, test, and security checks. For the desktop frontend, run `pnpm install` once in `internal/desktop/frontend`, then `pnpm run dev`, `pnpm run build`, `pnpm run lint:check`, or `pnpm run test:unit`. Keep `pnpm run lint` as the fixing command. For Wails live development, use `wails dev` from `internal/desktop`.

## Coding Style & Naming Conventions
Follow standard Go formatting: tabs, `gofmt`, and `goimports` with the local prefix `github.com/vasfvitor/nanci`. Keep Go packages lowercase and focused; exported identifiers use `CamelCase`, internal helpers use `camelCase`. Vue/TypeScript files use the existing ESLint + Prettier setup. Match current naming patterns such as `DocumentsPage.vue`, `MainLayout.vue`, and `PasswordPromptDialog.vue`.

## Desktop Frontend Architecture
The Vue frontend uses Vue 3, Composition API, `<script setup lang="ts">`, Pinia, and the `@` source alias. Keep route pages as composition surfaces: pages wire stores, composables, dialogs, notifications, and router state, while feature logic lives in `src/composables` and cross-route state lives in `src/stores`.

Do not import generated Wails bindings or `wailsjs/go/models` from pages, stores, or components. The only frontend modules that should import generated Wails code are `src/platform/wails/client.ts`, `src/platform/wails/events.ts`, and `src/platform/wails/runtime.ts`. Frontend code should depend on `src/types/desktop.ts`, `desktopClient`, and the platform wrappers instead of generated bindings. Avoid `window.go`, `@ts-expect-error` Wails calls, raw `EventsOff`, and `v-html` for query or JSON output.

Use focused Pinia setup stores for state that must survive route navigation, component unmount/remount, or be visible in multiple places. Current examples are `console` for logs and console UI, `documents` for document filters/results, `query` for direct-query form/result, and `companySync` for in-flight sync state. Use this same pattern for any user-visible background operation or long-running state, including document export progress, credential/certificate inspection, password prompt requests, log subscriptions, import/sync queues, and selected filters the user expects to keep after visiting another screen. Route-local `ref` state is fine for disposable UI such as an open dialog flag, the currently selected row for an edit dialog, or form draft state that intentionally resets when the component closes.

Composables should orchestrate feature actions and return a small typed API. They may own short-lived loading flags for an action scoped to the current mounted view, but they should delegate durable state to Pinia. If a composable starts an async Wails operation whose result can outlive the current route, mark it in a store before awaiting and clear it in `finally`; add a regression test that creates a second composable instance while the promise is pending. Use `storeToRefs` when returning reactive Pinia state from composables.

Shared formatting and display logic belongs in utilities, not templates: use `src/utils/formatters.ts` for CPF/CNPJ, date/time, currency cents, and access-key formatting, and `src/utils/nfseDisplay.ts` for NFSe role/status/visibility labels and colors. Keep computed derivations out of templates when they are reused or non-trivial.

Dialogs should expose typed props and emits where practical. They should not reach into generated Wails APIs; if backend behavior is needed, call a feature composable or `desktopClient`. Event subscriptions must use the platform event wrapper and keep the unsubscribe function returned by `EventsOn`.

Desktop Go methods exposed to Wails should return DTOs from `internal/desktop/desktopapi`, not `internal/nfse` domain structs. Add or update DTO mappers when fields cross the desktop boundary, and keep file path construction/export dispatch in the backend for export flows.

## Testing Guidelines
Place Go tests next to the code they cover using `*_test.go`. Existing examples include `internal/store/companies_test.go` and `internal/foundation/cnpj/cnpj_test.go`. Run `make test` before opening a PR; use `go test ./...` for quick iteration. Add regression tests for parser, storage, and path-handling changes.

For the desktop frontend, use Vitest with happy-dom, Vue Test Utils, and Pinia test setup. Add focused tests for Wails client mapping/error normalization, event cleanup wrappers, Pinia stores, composables that build Wails requests, formatter utilities, and route-remount behavior for long-running operations. Run `pnpm run lint:check`, `pnpm run test:unit`, and `pnpm run build` in `internal/desktop/frontend` before handing off frontend changes. Also smoke-test key Wails flows manually when behavior crosses the desktop runtime: add company, add credential, sync, list documents, export, direct query, password prompt, and logs console.

## Commit & Pull Request Guidelines
Prefer scoped, imperative commit subjects in the form `scope: change`, for example `frontend: fix company dialog spacing`, `cli: validate --competencia input`, `frontend,cli: align export flow`, or `wails: wire new desktop command`. Use repo areas as scopes: `frontend`, `cli`, `app`, `store`, `sync`, `nfse`, `adn`, `wails`, or a small combination when one change truly spans layers. Keep commits narrow and descriptive; add an issue reference when relevant. PRs should explain the behavioral change, list verification commands, and include screenshots for desktop/frontend UI changes. Note any schema, migration, or certificate-handling impact explicitly.

## Security & Configuration Tips
Do not commit certificate files, passwords, SQLite data, or exported fiscal documents. Prefer `NANCI_CERT_PASSWORD` for local runs instead of hardcoding secrets. Review `make security` output before merging changes that touch networking, storage, or auth flows.
