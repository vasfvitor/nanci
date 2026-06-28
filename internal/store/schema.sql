CREATE TABLE companies (
    id TEXT PRIMARY KEY,
    cnpj TEXT NOT NULL UNIQUE,
    cnpj_root TEXT NOT NULL,
    name TEXT NOT NULL,
    credential_id TEXT,
    credential_label TEXT,
    credential_cert_path TEXT,
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    sync_start_policy TEXT NOT NULL DEFAULT 'from_now' CHECK (sync_start_policy IN ('all', 'since_date', 'from_now')),
    sync_start_date TEXT,
    initial_sync_completed_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    cert_path TEXT NOT NULL,
    owner_cnpj TEXT NOT NULL,
    owner_cnpj_root TEXT NOT NULL,
    fingerprint_sha256 TEXT NOT NULL,
    subject_name TEXT NOT NULL,
    not_before TEXT,
    not_after TEXT,
    inspected_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    chave_acesso TEXT NOT NULL UNIQUE,
    issue_date TEXT NOT NULL,
    competence TEXT NOT NULL,
    prestador_cnpj TEXT NOT NULL,
    prestador_name TEXT NOT NULL,
    tomador_cnpj TEXT NOT NULL,
    tomador_name TEXT NOT NULL,
    intermediario_cnpj TEXT NOT NULL,
    intermediario_name TEXT NOT NULL,
    service_value INTEGER NOT NULL DEFAULT 0,
    iss_value INTEGER NOT NULL DEFAULT 0,
    irrf_value INTEGER NOT NULL DEFAULT 0,
    inss_value INTEGER NOT NULL DEFAULT 0,
    pis_value INTEGER NOT NULL DEFAULT 0,
    cofins_value INTEGER NOT NULL DEFAULT 0,
    csll_value INTEGER NOT NULL DEFAULT 0,
    total_retentions INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('normal', 'cancelada', 'substituida')),
    layout_version TEXT NOT NULL,
    xml_path TEXT NOT NULL,
    raw_hash TEXT NOT NULL UNIQUE,
    parse_warnings TEXT, 
    nfse_number TEXT NOT NULL,
    service_description TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE company_documents (
    relation_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    document_id TEXT NOT NULL REFERENCES documents(id),
    company_role TEXT NOT NULL CHECK (company_role IN ('tomada', 'prestada', 'intermediario', 'none')),
    visibility_reason TEXT NOT NULL CHECK (visibility_reason IN ('exact_prestador', 'exact_tomador', 'exact_intermediario', 'same_root_only', 'unknown')),
    first_seen_nsu INTEGER,
    last_seen_nsu INTEGER,
    first_synced_at TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    viewed_at TEXT,
    UNIQUE(company_id, document_id)
);

CREATE TABLE company_document_export_marks (
    company_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml', 'csv', 'xlsx', 'danfse')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    PRIMARY KEY (company_id, document_id, export_kind)
);

CREATE INDEX idx_company_documents_viewed_at ON company_documents(company_id, viewed_at);
CREATE INDEX idx_export_marks_kind ON company_document_export_marks(company_id, export_kind, exported_at);

CREATE TABLE events (
    id TEXT PRIMARY KEY,
    document_id TEXT REFERENCES documents(id),
    chave_acesso TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('cancelamento', 'substituicao', 'unknown')),
    event_at TEXT,
    replacement_chave_acesso TEXT NOT NULL,
    description TEXT NOT NULL,
    raw_xml_path TEXT NOT NULL,
    raw_hash TEXT NOT NULL UNIQUE,
    parse_warnings TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE sync_runs (
    id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    credential_id TEXT NOT NULL REFERENCES credentials(id),
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    credential_cnpj TEXT NOT NULL,
    consultation_cnpj TEXT NOT NULL,
    consultation_basis TEXT NOT NULL CHECK (consultation_basis IN ('exact_certificate_cnpj', 'same_root_certificate')),
    mode TEXT NOT NULL CHECK (mode IN ('normal', 'first_setup')),
    started_at TEXT NOT NULL,
    finished_at TEXT,
    from_nsu INTEGER NOT NULL,
    to_nsu INTEGER NOT NULL,
    checked_count INTEGER NOT NULL DEFAULT 0,
    documents_found INTEGER NOT NULL DEFAULT 0,
    empty_count INTEGER NOT NULL DEFAULT 0,
    consecutive_empty_count INTEGER NOT NULL DEFAULT 0,
    errors_count INTEGER NOT NULL DEFAULT 0,
    last_found_nsu INTEGER,
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'interrupted')),
    stop_reason TEXT CHECK (stop_reason IN ('empty_limit', 'context_canceled', 'fetch_error', 'process_error'))
);

CREATE TABLE sync_state (
    company_id TEXT NOT NULL REFERENCES companies(id),
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    consultation_cnpj TEXT NOT NULL,
    last_checked_nsu INTEGER NOT NULL DEFAULT 0,
    last_found_nsu INTEGER,
    last_empty_streak INTEGER NOT NULL DEFAULT 0,
    last_success_at TEXT,
    last_error_at TEXT,
    last_error_code TEXT,
    last_error_message TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (company_id, environment, consultation_cnpj)
);

CREATE UNIQUE INDEX idx_sync_runs_running ON sync_runs(company_id) WHERE status = 'running';
