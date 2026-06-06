-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;

CREATE TABLE credentials (
    id                  TEXT PRIMARY KEY,
    label               TEXT NOT NULL,
    cert_path           TEXT NOT NULL,
    environment         TEXT NOT NULL CHECK(environment IN ('producao_restrita', 'producao')),
    owner_cnpj          TEXT,
    owner_cnpj_root     TEXT,
    fingerprint_sha256  TEXT,
    subject_name        TEXT,
    not_before          TEXT,
    not_after           TEXT,
    inspected_at        TEXT,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX idx_credentials_env_path ON credentials(environment, cert_path);
CREATE INDEX idx_credentials_owner_root ON credentials(owner_cnpj_root);

CREATE TABLE companies (
    id              TEXT PRIMARY KEY,
    cnpj            TEXT NOT NULL UNIQUE,
    cnpj_root       TEXT NOT NULL,
    name            TEXT,
    credential_id   TEXT NOT NULL REFERENCES credentials(id),
    last_nsu        INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_companies_cnpj_root ON companies(cnpj_root);
CREATE INDEX idx_companies_credential_id ON companies(credential_id);

CREATE TABLE documents (
    id                  TEXT PRIMARY KEY,
    chave_acesso        TEXT NOT NULL UNIQUE,
    issue_date          TEXT,
    competence          TEXT,          -- formato YYYY-MM
    prestador_cnpj      TEXT,
    prestador_name      TEXT,
    tomador_cnpj        TEXT,
    tomador_name        TEXT,
    intermediario_cnpj  TEXT,
    intermediario_name  TEXT,
    service_value       REAL DEFAULT 0,
    iss_value           REAL DEFAULT 0,
    irrf_value          REAL DEFAULT 0,
    inss_value          REAL DEFAULT 0,
    pis_value           REAL DEFAULT 0,
    cofins_value        REAL DEFAULT 0,
    csll_value          REAL DEFAULT 0,
    total_retentions    REAL DEFAULT 0,
    status              TEXT DEFAULT 'normal',
    layout_version      TEXT,
    xml_path            TEXT NOT NULL,
    raw_hash            TEXT NOT NULL,  -- SHA-256 do XML bruto
    parse_error         TEXT,           -- se o parser falhou, guarda a mensagem
    parse_warnings      TEXT,           -- JSON string array de warnings
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_documents_chave ON documents(chave_acesso);
CREATE INDEX idx_documents_competence ON documents(competence);

CREATE TABLE company_documents (
    id                  TEXT PRIMARY KEY,
    company_id          TEXT NOT NULL REFERENCES companies(id),
    document_id         TEXT NOT NULL REFERENCES documents(id),
    company_role        TEXT NOT NULL CHECK(company_role IN ('tomada', 'prestada', 'intermediario', 'none')),
    visibility_reason   TEXT NOT NULL CHECK(visibility_reason IN ('exact_prestador', 'exact_tomador', 'exact_intermediario', 'same_root_only', 'unknown')),
    first_seen_nsu      INTEGER,
    last_seen_nsu       INTEGER,
    first_synced_at     TEXT NOT NULL,
    last_synced_at      TEXT NOT NULL,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(company_id, document_id)
);

CREATE INDEX idx_company_documents_company_role ON company_documents(company_id, company_role);
CREATE INDEX idx_company_documents_company_document ON company_documents(company_id, document_id);

CREATE TABLE events (
    id                         TEXT PRIMARY KEY,
    document_id                TEXT NOT NULL REFERENCES documents(id),
    chave_acesso               TEXT NOT NULL,
    event_type                 TEXT NOT NULL,
    event_at                   TEXT,
    replacement_chave_acesso   TEXT,
    description                TEXT,
    raw_xml_path               TEXT NOT NULL,
    raw_hash                   TEXT NOT NULL,
    parse_warnings             TEXT,
    created_at                 TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_events_document ON events(document_id);
CREATE INDEX idx_events_chave ON events(chave_acesso);
CREATE UNIQUE INDEX idx_events_unique_raw_hash ON events(raw_hash);

CREATE TABLE sync_runs (
    id                  TEXT PRIMARY KEY,
    company_id          TEXT NOT NULL REFERENCES companies(id),
    credential_id       TEXT REFERENCES credentials(id),
    credential_cnpj     TEXT,
    consultation_cnpj   TEXT,
    consultation_basis  TEXT CHECK(consultation_basis IN ('exact_certificate_cnpj', 'same_root_certificate')),
    started_at          TEXT NOT NULL,
    finished_at         TEXT,
    from_nsu            INTEGER,
    to_nsu              INTEGER,
    documents_found     INTEGER DEFAULT 0,
    errors_count        INTEGER DEFAULT 0,
    status              TEXT NOT NULL CHECK(status IN ('running', 'completed', 'failed', 'interrupted'))
);

CREATE INDEX idx_syncruns_company ON sync_runs(company_id);
CREATE INDEX idx_syncruns_credential ON sync_runs(credential_id);

PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;

DROP TABLE sync_runs;
DROP TABLE events;
DROP TABLE company_documents;
DROP TABLE documents;
DROP TABLE companies;
DROP TABLE credentials;

PRAGMA foreign_keys=ON;
-- +goose StatementEnd
