 # Detailed Go Architecture and Reliability Refactor

  ## Objectives

  Perform one atomic refactor PR that:

  - Removes confirmed correctness and security defects.
  - Replaces handwritten retry and SQL boilerplate with maintained tooling.
  - Applies Ardan Labs principles: data-oriented design, consistent semantics, consumer-owned interfaces, explicit error handling, and strong
    package boundaries.
  - Preserves CLI flags and desktop workflows.
  - Starts with a new nanci-v2.db; the existing database remains untouched.

  ## Target Architecture

  Dependencies flow inward:

  cmd/nanci ──> internal/cli ──> internal/app
  internal/desktop ────────────> internal/app

  internal/app ──> internal/nfse
  internal/app ──> consumer-owned repository interfaces
  internal/app ──> ADN and storage interfaces

  internal/store ──> internal/store/sqlgen
  internal/adn ───> net/http + go-retry

  Rules:

  - internal/nfse owns fiscal domain data and validation.
  - internal/app owns use cases and interfaces required by those use cases.
  - internal/adn, internal/store, and disk storage are concrete adapters.
  - CLI and Wails are composition roots and presentation adapters.
  - Generated SQL and Wails code never define domain contracts.

  ## 1. Domain Types

  Introduce explicit domain values:

  type Money int64 // cents

  type Environment string
  const (
      EnvironmentProduction Environment = "producao"
      EnvironmentRestricted Environment = "producao_restrita"
  )

  type DocumentStatus string
  type CompanyRole string
  type VisibilityReason string
  type EventType string
  type SyncStatus string
  type ConsultationBasis string

  Each enum provides:

  func ParseEnvironment(string) (Environment, error)
  func (e Environment) Valid() bool
  func (e Environment) String() string

  Apply equivalent parsing and validation to every enum. Do not permit arbitrary strings inside domain entities.

  Define money operations:

  func ParseMoney(string) (Money, error)
  func NewMoneyFromCents(int64) Money
  func (m Money) Cents() int64
  func (m Money) Add(Money) (Money, error)
  func (m Money) Sub(Money) (Money, error)
  func (m Money) FormatBRL() string

  ParseMoney accepts the XSD representation: non-negative decimal text with zero or two decimal places and at most 15 integer digits. It
  rejects commas, exponents, excess scale, overflow, signs, and malformed input.

  Change all fiscal values in Document from float64 to Money. Reports use integer cents and convert only at output boundaries.

  Introduce validated identifiers:

  type AccessKey string
  func ParseAccessKey(string) (AccessKey, error) // exactly 50 ASCII digits

  type DocumentID string
  type CompanyID string
  type CredentialID string
  type SyncRunID string

  Use value semantics for IDs, money, enums, filters, and results. Use pointers for mutable aggregate entities managed through repositories.

  Remove or consolidate:

  - Duplicate CompanyStats definitions.
  - Unused nfse.CompanyStats.
  - ParseError if failed parses are never persisted as documents.
  - Boolean validity pairs where pointers or explicit option types already express absence.
  - PullResult.EventsFound until synchronization can populate it accurately.

  ## 2. XML and Payload Parsing

  Split the current parser into:

  internal/nfse/payload.go
  internal/nfse/document_parser.go
  internal/nfse/event_parser.go
  internal/nfse/participation.go

  Payload API:

  type PayloadLimits struct {
      CompressedBytes   int64
      UncompressedBytes int64
  }

  func DecodePayload(encoded string, limits PayloadLimits) (DecodedPayload, error)

  type DecodedPayload struct {
      XML    []byte
      SHA256 string
  }

  Defaults:

  - Maximum base64-decoded compressed payload: 10 MiB.
  - Maximum decompressed XML: 50 MiB.
  - Reject trailing gzip corruption and invalid base64.
  - Compute SHA-256 while reading the bounded decompressed stream.

  Document parser:

  func ParseDocumentXML(data []byte) (Document, []Warning, error)

  Use encoding/xml.Decoder with namespace-aware paths. Do not use a shared currentText variable across arbitrary nesting.

  Track exact paths for:

  - infNFSe/@versao
  - chNFSe
  - dhEmi
  - compNFSe
  - provider, customer, and intermediary CNPJ/name
  - service and retention amounts
  - NFS-e number
  - service description

  Unknown fields are ignored. Missing optional fields produce typed warnings. Missing or invalid identity fields, malformed money, malformed
  timestamps, or invalid XML return errors.

  Event parser:

  func ParseEventXML(data []byte) (Event, []Warning, error)

  Classify using structured event elements and codes. Support XSD 1.00 and 1.01, including chSubstituta. Never classify by strings.Contains
  over the complete XML.

  Add fixtures for cancellation, cancellation by substitution, fiscal-analysis events, unknown event codes, namespaces, nested signatures,
  duplicate tag names, and malformed documents.

  ## 3. ADN HTTP Client

  Replace the general Request(method, path, body, dest) API with endpoint-specific methods internally backed by:

  type Client struct {
      baseURL    *url.URL
      httpClient *http.Client
      backoff    retry.Backoff
  }

  type APIError struct {
      Method     string
      URL        string
      StatusCode int
      Body       string
      Retryable  bool
  }

  Construction:

  type ClientConfig struct {
      Environment Environment
      HTTPClient  *http.Client
      Retry       RetryConfig
  }

  func NewClient(ClientConfig) (*Client, error)

  Transport policy:

  - Clone http.DefaultTransport.
  - Set a cloned tls.Config with the certificate and TLS 1.2 minimum.
  - Preserve default proxy and dial behavior.
  - Remove TLS renegotiation unless a documented integration test requires it.
  - Set request timeout through context; retain a defensive client timeout.

  Retry policy using github.com/sethvargo/go-retry v0.3.0:

  - Exponential backoff from 1 second.
  - Maximum 5 retries.
  - Maximum individual delay 30 seconds.
  - Jitter of 20%.
  - Retry transport errors only when the context is still active.
  - Retry HTTP 408, 425, 429, and 500–599.
  - Do not retry certificate, validation, JSON, or ordinary 4xx failures.
  - Honor valid Retry-After values without exceeding the configured cap.

  Every attempt constructs a fresh request and fresh body reader. Close each response body before the next attempt.

  Limit:

  - Successful JSON response to 20 MiB.
  - Error body to 64 KiB.
  - Reject unknown JSON fields only after confirming ADN response stability; initially accept them for compatibility.

  Add httptest.Server coverage for all status and cancellation paths.

  ## 4. Certificate Handling

  Replace duplicated loaders with:

  type LoadedCertificate struct {
      TLS        tls.Certificate
      Inspection Inspection
  }

  func LoadPKCS12(path, password string) (LoadedCertificate, error)

  File handling:

  - Reject directories and unsupported extensions.
  - Wrap file errors with path context while preserving fs.ErrNotExist.
  - Preserve ErrInvalidPassword as a sentinel.

  Inspection:

  - Read the ICP-Brasil CNPJ from its documented subject OID.
  - Validate extracted CNPJ through the domain CNPJ package.
  - Do not inspect issuer text, email addresses, DNS names, or arbitrary numeric substrings.
  - Validate current certificate validity separately from decoding.
  - Return fingerprint, subject, validity interval, owner CNPJ, and owner root.

  Add tests using generated test PKCS#12 fixtures for correct owner, incorrect password, missing CNPJ OID, expired/not-yet-valid certificates,
  and malformed chains.

  ## 5. Persistence and SQL Generation

  Pin sqlc v1.31.1 with the Go 1.25 tool directive. Add:

  sqlc.yaml
  internal/store/schema.sql
  internal/store/queries/*.sql
  internal/store/sqlgen/*

  sqlgen is generated and must contain the standard generated-code header. Generated structs remain private to the persistence adapter.

  Configure sqlc for:

  - SQLite.
  - database/sql.
  - JSON tags disabled unless required internally.
  - SQL included as comments.
  - Exact generated names for query parameters.
  - No generated public repository interface.

  Handwritten internal/store code maps generated rows to domain values and validates enum/time/JSON fields. Corrupt stored values return
  errors; they must never silently become zero values.

  Remove the package-level Store interface. Define interfaces where consumed:

  // internal/app
  type CompanyRepository interface {
      CreateCompany(context.Context, *nfse.Company) error
      CompanyByCNPJ(context.Context, cnpj.Value) (*nfse.Company, error)
      ListCompanies(context.Context) ([]nfse.Company, error)
      AssignCredential(context.Context, nfse.CompanyID, nfse.CredentialID) error
  }

  type CredentialRepository interface { ... }

  type DocumentReader interface {
      ListCompanyDocuments(context.Context, nfse.CompanyID, DocumentFilter) ([]nfse.CompanyDocument, error)
  }

  Synchronization receives one workflow repository:

  type SyncRepository interface {
      StartRun(context.Context, StartRunParams) (nfse.SyncRun, error)
      ApplyDocument(context.Context, ApplyDocumentParams) error
      ApplyEvent(context.Context, ApplyEventParams) error
      AdvanceCheckpoint(context.Context, AdvanceCheckpointParams) error
      FinishRun(context.Context, FinishRunParams) error
  }

  ApplyDocument executes one transaction containing:

  1. Canonical document upsert.
  2. Company-document relation upsert.
  3. Pending event attachment for the access key.
  4. Status recomputation.
  5. Monotonic NSU checkpoint update.

  ApplyEvent executes one transaction containing:

  1. Event upsert by raw hash.
  2. Document attachment if already present.
  3. Pending storage if the document has not arrived.
  4. Status recomputation when attached.
  5. Monotonic NSU checkpoint update.

  AdvanceCheckpoint handles valid empty batches. It rejects checkpoint regression.

  All write methods return ErrNotFound, ErrConflict, or wrapped root errors consistently. No (nil, nil) not-found result is allowed.

  ## 6. New Database Schema

  Use nanci-v2.db.

  Schema changes:

  - Monetary columns become INTEGER NOT NULL DEFAULT 0.
  - Enum columns retain SQLite CHECK constraints matching domain constants.
  - Timestamps remain RFC3339 UTC text and are validated on read.
  - Add an event attachment state so events may exist before documents.
  - Add a partial unique index preventing more than one running sync per company.
  - Preserve unique raw hashes and company-document uniqueness.
  - Add indexes matching all filter/order clauses used by queries.

  Startup behavior:

  - If only nanci.db exists, create nanci-v2.db.
  - Log and display a one-time notice that legacy data was preserved but not imported.
  - Never rename, modify, or delete nanci.db.
  - Do not build an automatic conversion migration.

  Use Goose v3.27.1 for future v2 migrations. The consolidated v2 schema is migration 001.

  ## 7. Synchronization Workflow

  Replace mutable caller-owned company state with IDs and immutable input:

  type SyncRequest struct {
      CompanyID        nfse.CompanyID
      CredentialID     nfse.CredentialID
      ConsultationBasis nfse.ConsultationBasis
  }

  type SyncResult struct {
      FromNSU          int64
      ToNSU            int64
      DocumentsApplied int
      EventsApplied    int
      Warnings         int
      Duration         time.Duration
  }

  The service loads required state itself through narrow repository methods.

  Processing rules:

  - Start one persisted run before network access.
  - Process envelopes in ascending NSU order; reject duplicate/conflicting ordering within one response.
  - Determine payload type from schema metadata plus structured root element verification.
  - Apply each envelope transactionally with its checkpoint.
  - Empty batches advance to ultNSU.
  - Reject ultNSU < requestedNSU, maxNSU < ultNSU, or non-progressing repeated pages.
  - Cancellation marks the run interrupted.
  - Protocol, parsing, persistence, and transport errors mark it failed.
  - Finalization uses context.WithoutCancel(ctx) with a 5-second timeout.
  - A finalization failure is joined with the primary error using errors.Join.
  - Progress notifications are observational only and never used to compute the result.

  ## 8. Disk Storage

  Define in the synchronization consumer:

  type XMLStore interface {
      Put(context.Context, XMLObject) (StoredXML, error)
  }

  type XMLObject struct {
      Kind      XMLKind
      AccessKey nfse.AccessKey
      SHA256    string
      Data      []byte
  }

  Concrete disk layout:

  xml/documents/<first-2-hash>/<sha256>.xml
  xml/events/<first-2-hash>/<sha256>.xml

  Write algorithm:

  1. Validate hash against data.
  2. Resolve destination beneath the configured root.
  3. Create parent directory with 0750.
  4. Write a temporary file with 0600.
  5. Flush and close.
  6. Atomically rename.
  7. Treat an existing identical destination as success.

  No XML-derived text becomes a directory or filename.

  ZIP exports use human-readable validated entry names, but reject absolute paths, separators, .., control characters, and duplicates.

  ## 9. Reports

  Create one shared row projection:

  type ReportRow struct {
      IssueDate          time.Time
      AccessKey          nfse.AccessKey
      Role               nfse.CompanyRole
      CounterpartyCNPJ   cnpj.Value
      ServiceValue       nfse.Money
      ...
  }

  func BuildRows([]nfse.CompanyDocument) []ReportRow

  Export APIs:

  func WriteCSV(io.Writer, []ReportRow) error
  func WriteXLSX(io.Writer, []ReportRow) error
  func WriteZIP(context.Context, io.Writer, XMLReader, []nfse.CompanyDocument) ([]Warning, error)

  Rules:

  - Presentation adapters create output files and handle close errors.
  - CSV uses deterministic headers and checks writer.Error().
  - XLSX checks every Excelize call and returns close errors with errors.Join.
  - ZIP missing-file conditions become returned warnings, not fmt.Printf.
  - Reports format cents only at rendering time.
  - Preserve current sheet names and CLI output filenames.

  ## 10. Application Layer

  Replace public mutable fields on app.App with private dependencies:

  type Dependencies struct {
      Companies       CompanyRepository
      Credentials     CredentialRepository
      Documents       DocumentReader
      Sync            SyncService
      XML             XMLReader
      Passwords       CredentialProvider
      Logger          *slog.Logger
      Clock           Clock
  }

  func New(Dependencies) (*App, error)

  Validate every required dependency during construction.

  Extract shared operations:

  - resolveCompanyByCNPJ
  - resolveCredentialForCompany
  - validateDocumentFilter
  - listDocumentsForCompany

  The app layer wraps errors once with use-case context. Adapters decide how to display them. Library packages never print or log expected
  operational failures.

  Inject a clock into code that persists timestamps or measures runs:

  type Clock interface {
      Now() time.Time
  }

  Use a real clock in production and deterministic test clocks.

  ## 11. CLI

  Replace global command variables and init registration:

  type Config struct {
      In     io.Reader
      Out    io.Writer
      ErrOut io.Writer
      OpenApp func(context.Context, bool) (Application, io.Closer, error)
  }

  func NewCommand(Config) *cobra.Command
  func Execute(context.Context, Config) error

  Each command:

  - Uses cmd.Context().
  - Returns errors from RunE.
  - Never calls os.Exit.
  - Never writes directly to process-global stdout/stderr.
  - Defers resource closure and joins close failures.
  - Uses Cobra argument and flag validation before opening the database.

  Root configuration:

  - SilenceErrors: true
  - SilenceUsage: true
  - Main prints one error and returns exit code 1.
  - Usage errors return code 2 through a typed CLI error.
  - SIGINT/SIGTERM cancel the root context.

  Factor repeated export flags and application lifecycle into helpers without hiding command behavior.

  ## 12. Wails Adapter

  Initialize the core application before calling wails.Run, so initialization failures terminate cleanly.

  Expose a thin Wails facade with DTOs rather than raw domain entities where Money or enum serialization would be ambiguous.

  Password protocol:

  type PasswordRequest struct {
      ID string
      app.CertPasswordRequest
  }

  Maintain map[string]chan passwordResult under a mutex.

  - Emit request ID to the frontend.
  - SubmitCertPassword(id, password) completes only that request.
  - CancelCertPassword(id) cancels only that request.
  - Context cancellation removes the pending request.
  - Duplicate/unknown submissions return an error.
  - Shutdown cancels all pending requests.

  All facade methods check initialization state and return ErrApplicationUnavailable, never panic.

  ## 13. Generated Code

  SQL:

  - go generate ./internal/store/... invokes pinned sqlc.
  - CI regenerates sqlc output and requires a clean diff.
  - Generated SQL code is never manually edited.

  Wails:

  - Keep generated frontend bindings tracked because frontend compilation imports them.
  - Remove tracked build/windows/installer/wails_tools.nsh; Wails owns it.
  - Add a binding verification command using the pinned Wails CLI/version.
  - CI runs generation/build and fails if tracked bindings change.
  - Ensure generated models expose DTOs, not persistence entities.

  Add generated headers where missing so linters and reviewers distinguish generated files.

  ## 14. Tooling and Repository Hygiene

  Run go mod tidy and commit normalized go.mod/go.sum.

  Pin:

  - Go 1.25.1
  - Wails v2.12.0
  - sqlc v1.31.1
  - Goose v3.27.1
  - sethvargo/go-retry v0.3.0

  Make targets:

  make generate
  make generate-check
  make fmt
  make lint
  make test
  make test-race
  make build
  make check

  make check must be non-mutating and run:

  1. gofmt/goimports verification.
  2. go mod tidy -diff.
  3. sqlc/Wails generation verification.
  4. go vet.
  5. golangci-lint.
  6. go test -race ./....
  7. CLI build.
  8. Desktop frontend build.
  9. Wails production build where platform dependencies are available.

  Fix all current lint findings rather than suppressing them. Narrow existing exclusions and require reasons for every nolint.

  ## 15. Implementation Order Within the Single PR

  1. Add characterization tests for existing CLI, sync, parser, and persistence behavior.
  2. Add domain enums, IDs, and Money; migrate callers at compile time.
  3. Replace XML/payload parsing and add XSD-derived fixtures.
  4. Replace retry/client and certificate inspection.
  5. Add v2 schema, sqlc configuration, queries, and generated adapter.
  6. Introduce consumer-owned interfaces and transactional sync repository.
  7. Refactor synchronization around atomic envelope application.
  8. Replace disk paths and export writers.
  9. Refactor application construction.
  10. Rebuild CLI and Wails adapters.
  11. Normalize generation, dependencies, Make targets, and CI.
  12. Remove obsolete code only after all replacement tests pass.

  Use separate commits for these steps even though they land in one PR.

  ## Acceptance Tests

  The refactor is complete only when:

  - Existing CLI command names and flags still work.
  - Existing nanci.db remains byte-for-byte untouched.
  - nanci-v2.db initializes and syncs from an empty state.
  - A cancellation or crash cannot advance past the last atomically persisted envelope.
  - Invalid XML money never silently becomes zero.
  - XSD 1.01 chSubstituta is captured.
  - XML and ZIP inputs cannot escape configured roots.
  - Concurrent sync for one company is rejected; different companies may sync concurrently.
  - CLI command errors run all deferred closers and print exactly once.
  - Concurrent Wails password prompts cannot cross-deliver results.
  - Generated-code verification produces no diff.
  - go test -race ./..., lint, tidy verification, CLI build, frontend build, and Wails build pass.
  - Modified core packages have at least 80% statement coverage.

  ## Explicit Assumptions

  - This is one refactor PR with multiple ordered commits.
  - Backward database compatibility is intentionally not implemented.
  - Legacy data remains preserved in nanci.db.
  - No external decimal library is added.
  - Full XSD Go code generation is rejected because it would create a large generated model surface; focused stdlib parsing is the intended
    design.
  - sqlc is used only for SQL type safety and scanner generation, not as the domain layer.
  - Public CLI behavior is preserved; internal Go APIs may change freely because this is an application repository, not a published library.