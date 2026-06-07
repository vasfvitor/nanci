 # Go Architecture and Quality Refactor

  ## Summary

  Refactor the codebase in one PR while preserving CLI/Wails behavior. Replace fragile handwritten infrastructure, align packages with Ardan
  Labs principles, isolate generated code, and correct identified reliability/security defects.

  Confirmed issues include:

  - Unsafe HTTP retries, XML string parsing, float64 money, unchecked errors, path traversal risk, duplicated certificate parsing, stringly
    typed domain state, an oversized producer-owned Store interface, CLI os.Exit calls bypassing defers, unsafe desktop initialization/
    password coordination, stale generated bindings, and direct dependencies incorrectly marked indirect.
  - Coverage is 0% for adn, app, cli, files, and retry; lint currently reports 16 violations.

  ## Implementation Changes

  ### 1. Domain Model

  - Introduce value types for Money, Environment, DocumentStatus, CompanyRole, VisibilityReason, EventType, SyncStatus, and ConsultationBasis.
  - Represent Money as int64 cents. The official XSD allows 15 integer digits plus exactly two decimals, which fits safely in int64.
  - Parse monetary XML values strictly; invalid values return contextual parser errors instead of silently becoming zero.
  - Use value semantics for immutable identifiers/enums and pointer semantics consistently for mutable entities.
  - Remove duplicate/dead models such as the two incompatible CompanyStats types, unused ParseError, and the unpopulated EventsFound path.
  - Keep the local CNPJ implementation, but expose a validated cnpj.Value; an external package is not justified because rollout rules include
    project-specific alphanumeric support.

  ### 2. ADN and Certificate Integration

  - Delete internal/foundation/retry; use pinned github.com/sethvargo/go-retry v0.3.0.
  - Clone http.DefaultTransport so proxy, dialing, pooling, and default transport behavior are retained; override only mTLS-specific settings.
  - Remove TLS renegotiation unless an integration test proves the ADN endpoint requires it.
  - Recreate requests and bodies for every attempt.
  - Retry only temporary transport errors and HTTP 408, 425, 429, and 5xx; honor Retry-After, cap delays, and never retry normal 4xx.
  - Add a typed adn.APIError containing status, endpoint, bounded response body, and retryability.
  - Limit JSON and error response bodies to prevent unbounded reads.
  - Consolidate PKCS#12 loading into one implementation; LoadPKCS12 delegates to inspection-capable decoding.
  - Extract certificate CNPJ from the ICP-Brasil subject OID, never from issuer text or arbitrary digit regex matches.

  ### 3. Structured XML Processing

  - Split payload decoding, document parsing, and event parsing into focused components using encoding/xml.
  - Limit compressed and decompressed payload sizes to prevent gzip bombs.
  - Parse namespace-aware XML tokens and exact element paths instead of maintaining a shared text buffer or searching raw XML strings.
  - Classify events by structured event elements/codes from XSD 1.00 and 1.01.
  - Support the actual 1.01 replacement field chSubstituta; preserve unsupported events as typed unknown records with warnings.
  - Validate access keys as exactly 50 digits before using them as identifiers.
  - Keep focused handwritten wire structs/token extraction; do not generate the complete XSD object graph.

  ### 4. Persistence

  - Adopt sqlc v1.31.1, pinned through the Go tool directive, for routine SQLite queries and row scanning.
  - Place generated code in an isolated internal/store/sqlgen package with standard generated headers; domain types must not depend on
    generated types.
  - Keep handwritten store code only for domain mapping and transactional workflows.
  - Replace the producer-owned store.Store interface with small consumer-owned interfaces.
  - Give synchronization a deep repository contract: StartRun, ApplyDocument, ApplyEvent, AdvanceCheckpoint, and FinishRun.
  - Make each envelope application transactional: canonical upsert, company relation/event ledger, status recomputation, and NSU checkpoint
    commit together.
  - Permit pending events with no current document; attach them by access key when the document arrives.
  - Prevent concurrent synchronization of the same company with a database-enforced active-run constraint.
  - Return ErrNotFound consistently instead of nil, nil; propagate timestamp and JSON decoding failures.
  - Configure SQLite foreign keys, WAL, busy timeout, and bounded connection counts explicitly.
  - Upgrade Goose to v3.27.1 and avoid package-global migration configuration where its provider API permits.

  ### 5. Database Reset

  - Create a consolidated v2 schema using INTEGER cent columns and checked domain values.
  - Change the active database filename to nanci-v2.db.
  - Leave an existing nanci.db untouched and emit a clear startup notice that v1 data is not imported.
  - Do not delete, overwrite, or silently migrate the old database.

  ### 6. Files and Reports

  - Replace path-oriented files.Writer with a BlobStore consumer interface and a concrete disk implementation.
  - Store XML by trusted SHA-256-derived paths, using temporary files plus atomic rename.
  - Treat an orphaned content-addressed file after a database failure as harmless and reusable.
  - Validate every resolved path remains below the configured data directory.
  - Change CSV/ZIP/XLSX generation to accept io.Writer where supported; adapters own destination file creation.
  - Check CSV flush errors, Excelize calls, archive closes, and file closes.
  - Return structured export warnings for missing XML instead of printing from library code.
  - Sanitize ZIP entry names and use validated access keys only.

  ### 7. Application and Adapters

  - Make application dependencies private and inject repositories, blob storage, client factory, logger, and credential provider through a
    constructor.
  - Move path resolution, SQLite opening, migrations, and resource ownership to CLI/Desktop composition roots.
  - Make Close ownership explicit and return close errors; remove the concrete store type assertion.
  - Consolidate duplicate company lookup/filtering logic used by list, export, and status.
  - Return a synchronization result directly instead of reconstructing counts from progress callbacks.
  - Finalize sync runs using context.WithoutCancel plus a short timeout, not unrestricted context.Background.

  ### 8. CLI and Wails

  - Build the Cobra tree through cli.NewCommand instead of global commands, flags, and init.
  - Use cmd.Context, cmd.OutOrStdout, and cmd.ErrOrStderr everywhere.
  - Remove every os.Exit from command handlers; handlers return wrapped errors so defers always execute.
  - Configure Cobra with SilenceErrors and SilenceUsage; only main determines the process exit code.
  - Initialize the desktop core before exposing Wails methods; startup failure must prevent method dispatch rather than leave core == nil.
  - Replace the global password channel with request IDs and per-request channels protected by a mutex, preventing concurrent prompts from
    receiving each other’s passwords.
  - Return a stable “application unavailable” error for desktop initialization failures.

  ### 9. Generated Code and Tooling

  - Run go mod tidy; classify UUID, Goose, Cobra, Excelize, x/term, SQLite, and PKCS#12 as direct dependencies.
  - Remove the tracked generated wails_tools.nsh; let Wails recreate it during packaging.
  - Keep Wails bindings tracked, but add a CI check that runs the pinned Wails build/codegen and fails on a dirty diff.
  - Add go generate/Make targets for sqlc and generated-artifact verification.
  - Remove narration comments, speculative comments such as “assuming endpoint,” numbered implementation comments, and ignored errors.
  - Make go test, golangci-lint, go mod tidy -diff, sqlc generation verification, CLI build, and Wails frontend build mandatory CI checks.

  ## Public Interface Changes

  - Monetary fields change from float64 to Money; Wails-facing DTOs expose cents plus formatted display strings.
  - Repository interfaces move to their consuming packages and become workflow-oriented.
  - Export functions accept writers instead of output paths.
  - Synchronization returns a typed result with separate document, event, warning, and error counts.
  - CLI commands and flags remain compatible; only error formatting becomes centralized.

  ## Test Plan

  - httptest coverage for retryable/non-retryable statuses, request body recreation, Retry-After, cancellation, bounded bodies, and typed API
    errors.
  - Fixture tests for namespaced XSD 1.00/1.01 documents and events, including chSubstituta, invalid money, malformed XML, unknown events, and
    decompression limits.
  - Store tests for atomic envelope application, pending-event attachment, checkpoint monotonicity, duplicate delivery, concurrent-run
    rejection, invalid stored JSON/time, and transaction rollback.
  - File tests for traversal attempts, atomic replacement, idempotent hashes, and ZIP entry sanitization.
  - CLI tests asserting errors are returned without process termination and resources are closed.
  - Desktop concurrency tests for two simultaneous password requests, cancellation, duplicate submission, and startup failure.
  - Fuzz tests for payload decoding and XML parsers.
  - Require modified core packages to reach at least 80% statement coverage and leave golangci-lint with zero findings.

  ## Assumptions and Design Basis

  - Delivered as one refactor PR, organized into reviewable commits but merged atomically.
  - Existing nanci.db data is not imported; it remains preserved on disk.
  - Selective dependencies only: sqlc and the already-present retry library are adopted; stdlib XML and integer cents remain preferred.
  - Design follows Ardan Labs package-oriented design (https://www.ardanlabs.com/blog/2017/02/package-oriented-design.html), interface
    semantics (https://www.ardanlabs.com/blog/2017/07/interface-semantics.html), Go’s context guidance
    (https://go.dev/blog/context-and-structs), error guidance (https://go.dev/blog/go1.13-errors), and transaction guidance
    (https://go.dev/doc/database/execute-transactions).