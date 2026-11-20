---
name: nanci-cli-agente
description: Use when an AI agent runs the `nanci` CLI for someone, to sync, list, check or export NFS-e, NF-e or CT-e, register a manifestação, import XML or recover NSUs. Covers `--json` and reading its output, the error codes and what to do on each, the certificate password, the SEFAZ hourly budget, the simulate-then-`--confirmar` rule, and recipes per task. For changing the CLI itself, use cli-command instead.
---
<!-- Generated from .agents/skills/nanci-cli-agente/ by scripts/sync-skills.sh; edit there. -->

# Using the `nanci` CLI as an agent

`nanci` syncs Brazilian fiscal documents (NFS-e from the ADN, NF-e and CT-e from the SEFAZ distribution) for each registered company into a local SQLite database, lists them, exports them and registers the Manifestação do Destinatário. The desktop app uses the same database. Talk to the person in Portuguese and keep the fiscal terms as they are (chave de acesso, competência, manifestação, ciência, resumo, NSU, tomador).

## Always run with `--json`

Put `--json` on every command: `nanci nfe status -c 12345678000199 --json`.

- **Success:** stdout holds exactly one JSON value (an object or an array), with no progress lines; parse the whole of stdout. Exit code 0.
- **Failure:** exit code 1, and the last line of stderr is `{"error":{"code":"...","message":"..."}}`. stderr is JSON Lines: before the envelope there may be log records, one JSON object per line (warnings only, unless `-v`). Take the line with the `error` key. `message` is Portuguese and meant for a person; branch on `code`.
- **Batch with failed items** (`nfe ciencia --confirmar`, `nfe manifestar --confirmar`) and **`testar-conexao` that fails:** stdout still has the result JSON *and* stderr has the envelope, exit 1. Read stdout first to see which items failed. The same goes for `nfe ciencia --confirmar` with no eligible chave (`Skipped` says why each was left out) and for an import that was interrupted (the files done so far).
- **`nfe import` / `cte import`** fail (exit 1, envelope "nenhum arquivo importado") only when no file got in. With some files in, the exit is 0 even if others were rejected: always check `Rejected` and each file's `Outcome` and `Message`.
- Field names are the Go names (`ChaveAcesso`, `TotalValue`, `NextAllowedAt`). Money is `int64` cents (`123456` is R$ 1.234,56). Times are RFC 3339; a time or a day count that does not apply is `null`, never a zero date. Empty lists are `[]`.

Without `--json` the output is Portuguese text for a person. Never parse it.

## Error codes

| `code` | Meaning | What to do |
|---|---|---|
| `sync_running` | Another nanci process (the desktop app or another terminal) is using that company and source. The message names the holder ("aplicativo" or "CLI"). | Do not retry in a loop. Tell the person and try again in a few minutes; a crashed holder frees the source within a minute. |
| `sefaz_blocked` | The SEFAZ blocked queries for this company until a time, written in the message (for example after cStat 656, consumo indevido). | Stop. Report the time. Running `nfe status`/`cte status` shows `NextAllowedAt` and `BlockedReason`. Never work around it. |
| `no_requests_left` | The 20 queries per hour for this company and source are spent. | Wait. `status` shows `RequestsLastHour`, `RequestBudget` and `NextAllowedAt`. |
| `wrong_password` | The certificate password is wrong. | Ask the person to fix it (see Password). Never guess or try variations. |
| `password_required` | The password is not saved and there is no terminal to ask for it. | Ask the person to run the command once in their terminal (see Password). |
| `not_found` | The company, certificate or document does not exist. | Run `nanci company list --json` and check the CNPJ, or the chave de acesso. |
| `invalid_input` | A malformed chave de acesso, CNPJ or flag value, or a command line the CLI refused (unknown flag, missing required flag, wrong number of arguments). | Fix the argument, with `--help` if needed; nothing was sent. |
| `canceled` | The run was interrupted. | Run it again if the person still wants it. |
| `""` (empty) | Any other error. | Read `message` and report it; do not guess a fix. |

## Password and certificate

- The certificate password comes from the Windows credential store, saved the first time a person typed it, or from the `NANCI_CERT_PASSWORD` environment variable. There is no `--senha` flag.
- Never ask the person to paste the password into the chat, never write it in a command line or a file, and never guess it. When a command fails with `wrong_password` or `password_required`, ask the person to run that same command once in their own terminal so nanci prompts and saves it, or to open the desktop app, then continue.
- Registering companies and certificates (`init`, `company add/update/assign-credential`, `credential add/update-path`) is for the person, in the desktop app or their terminal. Run them only when the person asks, with the certificate path they gave you. With `--json` they answer with the record as stored; `credential add` returns the new `ID` that `company assign-credential --credential-id` takes.

## Stopping a command

- `nanci` never waits for input when there is no terminal, as when you run it: a missing password fails at once with `password_required`. A command that runs long is working (a pull walks NSU after NSU), not waiting for you.
- To stop one, end the process. Ctrl+C or SIGTERM stops it cleanly: what was stored stays, an interrupted import or ciência prints what it got through, and the source is free at once. A process killed without a signal (TerminateProcess, closing its window) leaves its source busy: the next command gets `sync_running` for up to a minute, then runs.

## Query budget and timing (NF-e and CT-e)

- Each company has 20 SEFAZ queries per hour per source (NF-e and CT-e count separately), shared with the desktop app. `nfe pull`/`cte pull` stop by themselves at the limit (`StopReason: "rate_budget"`) or when they reach the end (`"caught_up"`); running a pull when the SEFAZ does not allow it yet returns `sefaz_blocked` without sending anything. Running pulls hourly is safe.
- `nfe buscar --chave` spends one query. `recuperar-nsu` spends one per NSU and, without `--limite`, leaves 2 of the hour for the pull. Check `RequestsLastHour`/`RequestBudget` in `status` before spending queries the person did not ask for.
- `testar-conexao` spends nothing; use it to check a certificate.
- `IdleDays` that is not `null` in a status means the company has not queried the distribution for that many days. The Ambiente Nacional only generates NSUs for whoever queried in the last 60 days, so tell the person and suggest a pull.

## Simulate first, then `--confirmar`

`nfe ciencia`, `nfe manifestar`, `cte desacordo`, `reset`, `nfe reset`, `cte reset` and `recuperar-nsu --descartar` only simulate without `--confirmar`: they print the plan as JSON, send nothing, change nothing and ask no password.

1. Run without `--confirmar` and show the person the plan in plain words: which notes, number, emitente, value, deadline.
2. Run with `--confirmar` only after the person agrees to that plan. A manifestação sent to the SEFAZ cannot be cancelled, and `nfe reset` or `cte reset` deletes the company's documents of that source.
3. Never add `--confirmar` on your own because the plan "looks fine".

`recuperar-nsu` *without* `--descartar` is not a simulation: it queries the SEFAZ right away. Treat it as spending queries.

## Flags you will use

| Flag | Meaning |
|---|---|
| `-c, --cnpj` | The company, required on almost every command. Digits only or formatted; the list is in `nanci company list --json`. |
| `-m, --competencia YYYY-MM` | NFS-e: competência. NF-e/CT-e: month of issue. |
| `-d, --direcao` | NFS-e only: `tomada`, `prestada`, `intermediario`. |
| `-p, --papel` | NF-e/CT-e only, the company's role: NF-e `destinatario`, `emitente`, `transportador`, `autorizado`; CT-e also `tomador`, `remetente`, `expedidor`, `recebedor`. |
| `--chave` | Chave de acesso (44 digits). Repeatable on lists and batch exports; single on `buscar`, `manifestar` and the single-file exports (`xml`, `danfe`, `danfse`, `dacte`). |
| `--incremental` | Export only what was not exported yet or changed since. |
| `-o, --out` | Output file. Use an absolute path the person chose; the JSON answers with `OutPath`. |
| `--nao-vistos` | Lists: only documents nobody has opened yet. |

`nanci <command> --help` lists the rest (Portuguese).

## Recipes

**Which companies are there.** `nanci company list --json` returns an array of `{CNPJ, Name, CredentialLabel, Environment, UF, LastSyncAt, ...}`. Use these CNPJs; do not invent one.

**State of a company.** `nanci nfe status -c <cnpj> --json`, then `cte status` and `status` (NFS-e). From NF-e status report `LastSyncAt`, `NextAllowedAt`, `RequestsLastHour`/`RequestBudget`, `TotalResumos`, `PendingCiencia`, `PendingConclusiva`, `CienciaOverdue`, `NSUFaltantes` and `IdleDays`. The NFS-e `status` has `NextAllowedAt` and `BlockedReason` too.

**Bring new documents.** `nanci pull -c <cnpj> --json` (NFS-e), `nanci nfe pull -c <cnpj> --json`, `nanci cte pull -c <cnpj> --json`. Report the counts (`CompletasSaved`, `ResumosSaved`, `DocumentsSaved`, `EventsSaved`, `Errors`) and `StopReason`. Run them one after another, not in parallel: each source of a company takes the lock.

**List documents.** `nanci nfe list -c <cnpj> -m 2026-10 --json` (array of NF-e). Useful filters: `--papel destinatario`, `--completude resumo`, `--manifestacao nenhuma`, `--emitente <cnpj>`, `--situacao cancelada`. `nanci cte list --nfe <chave> --json` finds the freight (CT-e) of an NF-e. `nanci list -c <cnpj> -m 2026-10 -d tomada --json` for NFS-e.

**What still needs a manifestação.** `nanci nfe pendentes -c <cnpj> --json`, optionally `--vencendo-em 10`. Each item is an NF-e plus `Kind` (`sem_ciencia` or `sem_conclusiva`) and `CienciaOverdue`. Deadlines are already computed: `CienciaDue`, `ConclusiveDue`, `DaysLeft`, `CienciaDaysLeft`; `TacitlyConfirmed` true means the deadline passed and the operation counts as confirmed. Do not compute legal deadlines yourself.

**Ciência for the resumos** (makes the SEFAZ release the full NF-e on the next pull):

1. `nanci nfe ciencia -c <cnpj> --todos-resumos --json` → `{Eligible: [NF-e...], Skipped: [{ChaveAcesso, Reason}]}`.
2. Show the eligible notes, get a yes, then repeat with `--confirmar`.
3. The answer is `{Outcomes: [{ChaveAcesso, Status, CStat, XMotivo, Protocolo, RegisteredAt}], Skipped, Interrupted}`. `Status` is `registrada`, `ja_registrada` (already on the SEFAZ, fine), `rejeitada` or `erro`. A non-empty `Interrupted` means the batch stopped early; report it.
4. Then `nfe pull` to bring the full documents.

**Conclusive manifestação** (one note at a time): `nanci nfe manifestar -c <cnpj> --chave <chave> --tipo confirmacao --json` (or `desconhecimento`, or `nao_realizada` with `--justificativa` of 15 to 255 characters). The plan shows `Numero`, `EmitenteName`, `TotalValue`, `Manifestacao`, the deadlines and `TacitlyConfirmed`. The choice of `--tipo` is always the person's: never pick it from the data. Confirm, then repeat with `--confirmar`.

**Prestação em desacordo** (CT-e, the company as tomador, one at a time): `nanci cte desacordo -c <cnpj> --chave <chave> --observacao "<what was not provided as written>" --json`. The plan shows `Modelo`, `Numero`, `EmitenteName`, `TotalValue`, `Due` and `DaysLeft` (45 days from the authorization; negative means passed, and the SEFAZ will likely answer cStat 787). The observation is the person's words, 15 to 255 characters, no curly quotes or emoji: never write it for them. Confirm, then repeat with `--confirmar` → `{ChaveAcesso, TpEvento, Status, CStat, XMotivo, Protocolo}`; `Status` other than `registrada` or `ja_registrada` exits 1. It goes to the SEFAZ of the chave's UF, not the Ambiente Nacional, and spends no hourly query.

**One NF-e by its chave.** `nanci nfe list -c <cnpj> --chave <chave> --json` first (free). Only if it is not there, `nanci nfe buscar -c <cnpj> --chave <chave> --json` (one query) → `{Outcome: "completa" | "resumo" | "not_found", Message}`; read `Message` for what to do next.

**Export.** NF-e: `nanci nfe export zip|xml|csv|xlsx|danfe|danfe-zip -c <cnpj> -m 2026-10 -o <path> --json`; batch formats answer `{OutPath, Format, Incremental, ExportedCount, SkippedResumos}`, single files (`xml`, `danfe` with `--chave`) answer `{OutPath}`. Resumos are left out unless `--incluir-resumos`, and have no DANFE. CT-e: `nanci cte export ... --json` (`SkippedSemDACTE`; DACTE only for modelo 57). NFS-e: `nanci export xlsx|csv|zip|xml|danfse|danfse-zip -c <cnpj> -m 2026-10 -o <path> --json`. `ExportedCount: 0` is not an error: there was nothing (new) to export.

**Import XML the company already has.** `nanci nfe import <file>... -c <cnpj> --json` (or `cte import`) → `{Files: [{Path, Kind, ChaveAcesso, Outcome, Message}], Imported, Updated, Rejected}`; `Outcome` is `imported`, `updated` or `rejected`. No SEFAZ query.

**NSUs the distribution skipped.** When status shows `NSUFaltantes > 0`: `RecoveryAllowance` is how many a recovery would ask now and `RecoveryReadyAt` when one can start. Ask the person before spending queries, then `nanci nfe recuperar-nsu -c <cnpj> --limite 5 --json` → `{Asked, Recovered, Empty, Unreadable, Expired, Missing, StopReason, NextAllowedAt, ...}`. To give up on them: `recuperar-nsu --descartar --json` (plan `{Discarded, Confirmed: false}`), then `--confirmar` after a yes.

**Reset** (rarely right; only when the person asks): `nanci nfe reset -c <cnpj> --json` shows the counts that would be removed (`Documents`, `Events`, `ExportMarks`, `ManifestacoesKept`), `Confirmed: false`. Repeat with `--confirmar` only on an explicit request. The NFS-e `nanci reset -c <cnpj> --json` removes nothing: it puts the NFS-e cursor back to NSU 0 (`LastProcessedNSU` is where it stood), so the next `pull` walks the whole distribution again. It is the step before `company update --sync-start-date <YYYY-MM-DD>` when the person wants an earlier start date; without it that update fails and names the reset.

## Do not

- Run without `--json`, or scrape the text output.
- Loop on `sync_running`, `sefaz_blocked` or `no_requests_left`.
- Put `--confirmar` on a command whose plan the person has not seen and approved.
- Handle the certificate password in any form.
- Touch the SQLite file or the data folder directly; everything goes through `nanci`.
