> **Status (2026-08-24):** ported from branch `logs`. Implemented on `main`:
> ADN client masks `cnpjConsulta` in logged URLs and in `APIError.URL`, logs
> response bodies only at trace level, and caps error bodies in log records
> and error messages (`internal/adn/client.go`). Still open: `cnpj.Mask`,
> sanitizing log files during export, `diagnostics.json`, the startup notice
> and the `SettingsPage.vue` wording. The `version`/`environment`/`nsu`/
> `document_count` attributes were not added.

# Design Specification: Logs Bons, Mas Seguros & Diagnósticos

## Context and Purpose
This document outlines the design and implementation details for secure structured logging (sanitizing keys, secrets, and raw XMLs during execution) and the diagnostic package export feature for the Nanci application.

To preserve maximum troubleshooting ability for developers locally, raw CNPJs (PII) will remain unmasked in the logs on the local disk. However, when logs are exported through the application, they will be sanitized on-the-fly.

## Proposed Architecture

### 1. CNPJ Masking Utility (`internal/foundation/cnpj`)
A utility function `Mask(cnpj string) string` will be introduced in the `cnpj` package.
*   **Behavior**:
    1.  Normalize the input using the existing `Clean(cnpj)` function.
    2.  If the normalized CNPJ is exactly 14 characters (numeric or alphanumeric), mask it to `XX.***.***/****-XX` format using the first 2 and last 2 characters (e.g., `12.***.***/****-95`).
    3.  If the normalized CNPJ is not 14 characters, return the cleaned string as-is.

### 2. Client Response & Request Logging (`internal/adn`)
The ADN API client (`internal/adn/client.go`) logs HTTP requests and responses. The following changes will structure and sanitize logs at write-time:
*   **URL Sanitization**:
    *   Add a private helper `sanitizeURL(rawURL string) string` to mask the `cnpjConsulta` query parameter in URLs using `cnpj.Mask`.
*   **Leak Prevention at Write-Time**:
    *   Remove all raw/truncated response bodies (`body` key) and raw XML base64 strings from both success and error logging in `client.go`.
    *   Log only structured metadata:
        *   `version`: Version of the app from `buildinfo.Version`
        *   `environment`: Environment configuration (`producao` or `producao_restrita`)
        *   `method`: HTTP Verb
        *   `url`: Sanitized URL (query parameters masked)
        *   `status`: HTTP response status code
        *   `latency`: Request duration
        *   `nsu`: Extracted NSU (from successful `DocumentResponse`)
        *   `document_count`: Number of documents found (from successful `DocumentResponse`)
*   **Information Extraction**:
    *   In the HTTP request log wrapper, check if `dest` is a `*DocumentResponse` using type assertion. If so, read `UltNSU` and `len(Docs)` for logging.

### 3. Core App Logging Integration (`internal/app` & `internal/service/sync`)
*   **Client Setup**:
    *   In `Pull` (`internal/app/pull.go`) and `buildClientForQuery` (`internal/app/query.go`), pass `Log: a.Log` and `Environment: company.Environment` to the ADN client.
*   **Startup / Console Warning**:
    *   Log a clear message at start of synchronization or application startup:
        `"Para exportar logs limpos e sanitizados (sem dados pessoais), utilize a opção 'Exportar Pacote de Diagnóstico (Logs)' na tela de Configurações."`

### 4. Desktop Diagnostics & Logs Export Evolvement (`internal/desktop`)
Instead of creating a new API, the existing `ExportLogs()` method in `internal/desktop/app.go` will be evolved into a complete diagnostics package export.
*   **Database Inspection**:
    Query the SQLite database via `a.core.DB` to retrieve the following data safely (CNPJs masked using `cnpj.Mask` in memory):
    *   Configured companies (name, cnpj, environment, last_nsu).
    *   NSU cursors from `sync_state` table.
    *   The last 20 failed or error runs from `sync_runs` table (where `status = 'failed' OR errors_count > 0`).
*   **ZIP Construction**:
    *   Write `diagnostics.json` containing the database inspection metadata and system information (OS, architecture, app version, data directory).
    *   Gather `nanci-desktop.log` and its backups, plus `wails.log`.
    *   **Sanitization during compression**:
        Read log files line by line (or chunk by chunk) and use regexes to replace CNPJs:
        *   Formatted CNPJs: `\b[A-Za-z0-9]{2}\.[A-Za-z0-9]{3}\.[A-Za-z0-9]{3}/[A-Za-z0-9]{4}-[A-Za-z0-9]{2}\b`
        *   Raw numeric CNPJs: `\b\d{14}\b` (restricted to numeric to prevent corrupting build hashes or hex IDs).
        For every match, clean the CNPJ, parse it, and replace it dynamically with `cnpj.Mask(match)` (instead of a static placeholder) to maintain consistency and partial traceability.
    *   Write the sanitized files to the ZIP.

### 5. Frontend & UI Reuse (`internal/desktop/frontend`)
*   **No duplicate APIs**: Re-use the existing Wails mapping for `desktopClient.exportLogs()`.
*   **UI Wording**: Update the success message and button label in `SettingsPage.vue` to indicate that a complete sanitized diagnostics package has been saved.

## Acceptance Criteria & Tests
1.  **CNPJ Masking Unit Tests**:
    *   Verify `cnpj.Mask` works for formatted, raw numeric, raw alphanumeric, and short inputs.
2.  **ADN Client Log Verification**:
    *   Verify ADN request/response logs contain structured keys (version, environment, status, latency) and do not leak XML base64 or response bodies.
    *   Verify URL logged has query parameters sanitized.
3.  **Sanitization in Disk vs ZIP Verification**:
    *   Verify that `nanci-desktop.log` on local disk contains raw CNPJs.
    *   Verify that logs exported inside the ZIP package have all CNPJs masked correctly as `XX.***.***/****-XX`.
    *   Verify that non-CNPJ alphanumeric strings of length 14 (like hex IDs) are NOT affected.
4.  **Diagnostics JSON Verification**:
    *   Verify `diagnostics.json` exists in the ZIP and lists companies, state cursors, system details, and runs errors with masked CNPJs.
