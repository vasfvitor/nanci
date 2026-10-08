---
name: cli-command
description: Use when adding or changing a `nanci` CLI command (internal/cli). Covers registration through CommandEnv and AppFactory, the flag conventions (-c/--cnpj, -m/--competencia, -d/--direcao vs -p/--papel, --chave, --incremental, -o/--out, --confirmar), Portuguese help and output, error wrapping, validation before AppFactory, password handling, the test harness, and the docs to update.
---

# CLI command: add or change a `nanci` command

The CLI is a thin Cobra adapter over `internal/app`. A command parses flags, validates what it can without the database, calls one `app` use case and prints the result. Business rules, filters and their validation messages live in `internal/app` (for example the filter builder in `internal/app/cte.go` that rejects `--modelo 58`).

## Registration

- `NewRootCommand(env CommandEnv)` in `internal/cli/root_factory.go` builds a fresh tree per call and adds each top-level command (`newNFeCommand(env)`, `newCTeCommand(env)`, ...). A new top-level command goes there; a new subcommand goes in its group constructor (`newNFeCommand` in `nfe.go`, `newCTeCommand` in `cte.go`, `newExportCommand` in `export.go`).
- `CommandEnv` (`env.go`) carries `Stdin`/`Stderr` (`*os.File`, for the password prompt), `Stdout io.Writer`, `AppFactory`, `Verbose`, `Trace`. Never read package globals; take everything from `env`.
- `AppFactory func(ctx) (*app.App, func(), error)`: production is `prodAppFactory` in `runtime.go` (opens SQLite, wires stores, keyring credential provider with `TerminalCredentialProvider` fallback). Every `RunE` does:
  ```go
  application, cleanup, err := env.AppFactory(cmd.Context())
  if err != nil {
      return fmt.Errorf("inicializar: %w", err)
  }
  defer cleanup()
  ```
- One constructor per leaf, `newNFeXxxCmd(env CommandEnv, cnpjFlag *string) *cobra.Command`, one file per leaf or small group (`nfe_pull.go`, `nfe_export.go`, `cte_reset.go`). Flag variables are locals of the constructor.
- The root sets `SilenceUsage`/`SilenceErrors`; `cmd/nanci/main.go` is the only place that prints the returned error and picks the exit code. Do not print errors yourself; return them.

## Flag conventions

| Flag | Rule | Precedent |
|---|---|---|
| `-c/--cnpj` | Required. Persistent on a group whose leaves all act on one company (`nfe`, `cte`, `export`: `MarkPersistentFlagRequired`); local and `MarkFlagRequired` on standalone leaves (`pull`, `list`, `status`, `company update`). Pass `*cnpjFlag` raw; `app` cleans and validates it. | `nfe.go`, `export.go`, `list.go` |
| `-m/--competencia` | `YYYY-MM`. NFS-e: competência; NF-e/CT-e: month of issue (help says "Mês de emissão"). | `export.go`, `nfe_export.go` |
| `-d/--direcao` | NFS-e only: `tomada`, `prestada`, `intermediario`. | `list.go`, `export.go` |
| `-p/--papel` | NF-e/CT-e only, the company's role (`destinatario`, `tomador`, ...). Never add `--direcao` to a SEFAZ command or `--papel` to an NFS-e one. `company add -p` is `--cert`, a legacy collision; do not repeat it. | `nfe_list.go`, `cte_list.go` |
| `--chave` | Filters and batch actions: `StringSliceVar`, help ends "(pode repetir)". Single-document actions: `StringVar` plus `MarkFlagRequired`. | `nfe_export.go` zip vs xml |
| `--incremental` | Export only what is new or changed since the last export marks. | `export.go`, `cte_export.go` |
| `-o/--out` | Default is a fixed name per format (`export.xlsx`, `nfe.zip`, `cte.zip`, `danfses.zip`); single-key exports default to `<chave>.xml` computed in `RunE`, with help "(padrão: <chave>.xml)". | `nfe_export.go` |
| `--nao-vistos` | Lists: only rows without `viewed_at`. | `list.go`, `nfe_list.go` |
| `--confirmar` | Mandatory for anything that sends an event or deletes data. See below. | `nfe_manifest.go`, `nfe_reset.go` |

Flag names and values are Portuguese and snake_case for enum values (`nao_realizada`); keep an accepted alias when renaming (`nao-realizada` still parses).

## Text and output

- `Short`, `Long` and flag help in Portuguese, sentence case, no trailing period on `Short`. A dry-run command says so in `Short`: "(simulação sem --confirmar)".
- Print to `cmd.OutOrStdout()` only, with `_, _ = fmt.Fprint...`. Tables use `text/tabwriter` with an UPPERCASE header row and a dashes row (`nfe_list.go`), dates through `formatDate`/`formatDateTime`, CNPJ through `cnpj.Format`, money through `.FormatBRL()`, long names through `truncateText(s, 30)`, empties through `dashIfEmpty` (helpers in `nfe.go`).
- Empty results print one line and return nil: "Nenhum documento encontrado.", "Nenhuma NF-e encontrada.", "Nenhum CT-e encontrado.", "Nenhuma manifestação pendente.". Lists end with "Total de N ... listado(s).".
- Exports print a progress line ("Gerando arquivo ZIP...") then reuse `printExportResult` (`export.go`), which already handles the empty and incremental-empty cases.
- SEFAZ pulls reuse `pullError` (prints "Próxima consulta permitida após: ..." for a `*sync.BlockedError`); connection tests reuse `printConnectionTest`; status output reuses `printIdleWarning`.

## Errors and validation

- Wrap with Portuguese context and `%w`: `"inicializar: %w"` for the factory, `"erro ao gerar arquivo ZIP: %w"` / `"erro ao exportar XML: %w"` / `"erro ao adicionar empresa: %w"` for a named action, plain `"erro: %w"` for list/pull/status. Never `errors.New(err.Error())`.
- Validate before `env.AppFactory` whatever needs no database, so a typo never opens the DB or asks for a password: `dfe.ParseAccessKey(chaveFlag)` in `newNFeExportXMLCmd` and `newNFeManifestarCmd` ("chave de acesso inválida: %w"), mutually exclusive flags in `newNFeCienciaCmd` ("informe --chave ou --todos-resumos, apenas um dos dois"), ranges in `newNFePendentesCmd`.
- A command whose result is a set of per-item outcomes prints the table first, then returns a non-nil error when any item failed (`nfeOutcomesError` in `nfe_manifest.go`), so scripts see a non-zero exit.

## Side effects need `--confirmar`

Any command that sends something to SEFAZ/ADN that cannot be undone, or deletes local data, runs as a simulation by default:

- Without `--confirmar`: call the `Plan*`/`Preview*` use case (`PlanCiencia`, `PlanManifestacao`, `PreviewReset`), print what would happen, and end with "Nada foi enviado. Use --confirmar para ..." or "Nada foi alterado. Use --confirmar para redefinir." The plan path must not ask for the certificate password (the test asserts zero requests).
- With `--confirmar`: call the `Register*`/`Reset` use case. Flag help: "Envia ... à SEFAZ; sem esta flag nada é enviado" or "Remove de fato; sem esta flag nada é alterado".
- Precedents: `nfe ciencia` and `nfe manifestar` in `internal/cli/nfe_manifest.go`, `nfe reset` in `nfe_reset.go`, `cte reset` in `cte_reset.go`. Map `app.ErrSyncRunning` to "...; aguarde a sincronização atual terminar".

## Passwords

Commands never prompt themselves. `app` asks the `CredentialProvider` when it loads the certificate: the keyring first (`app.KeyringCredentialProvider`), then `TerminalCredentialProvider` (`credential.go`), which reads `NANCI_CERT_PASSWORD` before prompting with `term.ReadPassword`. Pass a purpose string through the use case so the prompt says why ("Sincronização NF-e"). Do not add a `--senha` flag.

## Tests

Tests are white-box (`package cli`) next to the command (`nfe_test.go`, `cte_test.go`, `list_test.go`, `company_test.go`).

- `newNFeTestRoot(t)` builds a root over a real SQLite (`storetest.OpenTestDB`) with one company (`nfeTestCNPJ`, UF SP, production) and a `refusingPasswords` provider that counts requests. `newCTeTestRoot(t)` wraps it with the CT-e repository. `newInMemTestRoot(t)` and `newTestRootForApp(app)` (`company_test.go`) are the lower-level builders; `newTestRoot` + `tempCommand` (`root_test.go`) test the root itself.
- `env.run(args...)` resets `env.out`, sets args and executes; assert on `env.out.String()` and the returned error (`errors.Is(err, dfe.ErrInvalidAccessKey)` or `strings.Contains`).
- Seed with `env.seed("procnfe.xml", nsu)` (fixtures in `internal/nfe/testdata`), `env.seedCTe("procte.xml", nsu)` / `env.seedListFixtures()` (`internal/cte/testdata`), `env.seedNFSe(id, chave)` (`list_test.go`), `env.setLastQuery(source, at)` for idle warnings; read back with helpers like `env.manifestacaoCount()`.
- Flag values stick to a command tree: Cobra does not reset a flag that a later invocation leaves out, and `RunE` may write to its flag variable (the `<chave>.xml` default). Build a fresh root per invocation inside table-driven loops, with the comment `// flag values stick to a command tree` (`TestCTeList_RejectsInvalidFlags`).
- For a `--confirmar` command test three things: the dry run prints the plan and "Nada foi ...", `env.passwords.requests == 0` and nothing was recorded; the confirmed path changes state (`TestNFeReset_DryRunThenConfirm`).
- Run `go test ./internal/cli/` and `golangci-lint run ./internal/cli/...`.

## Docs to update in the same change

- `website/content/docs/cli.md`: the "Comandos" table (one row per command group) and the `--confirmar` callout when the command has side effects; "Primeiros passos" or "Rotina agendada" when the command belongs in a routine.
- `docs/NFE_SEFAZ.md` or `docs/CTE_SEFAZ.md`, section "Uso pela linha de comando": the numbered example block and the bullet that explains the flags and defaults. NFS-e commands have no such section in `docs/NFSE_ADN.md`; describe behavior there only if it changes the ADN contract.
- `README.md` has no command examples, only the build line and a link to the CLI page; touch it only if that changes.
- If the command mirrors a desktop action, keep the wording of both aligned (button label, confirmation text).

## Checklist

- [ ] Constructor `newXxxCmd(env, ...)` registered in its group or in `NewRootCommand`.
- [ ] `--cnpj` required (persistent on the group or local); flag names and shorthands follow the table.
- [ ] Portuguese `Short`, flag help and output; "Nenhum ..." line for empty results.
- [ ] Database-free validation happens before `env.AppFactory`.
- [ ] `defer cleanup()` right after the factory; errors wrapped with `%w`, no printing of errors.
- [ ] Side effects behind `--confirmar`, with a dry-run plan that asks no password.
- [ ] Non-zero exit when any item of a batch failed.
- [ ] Tests with a fresh root per invocation; dry-run test asserts zero password requests.
- [ ] `cli.md` table, `docs/<SOURCE>_SEFAZ.md` command section updated.
