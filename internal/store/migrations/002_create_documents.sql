-- +goose Up
-- +goose StatementBegin
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
    status              TEXT DEFAULT 'normal',
    xml_path            TEXT NOT NULL,
    raw_hash            TEXT NOT NULL,  -- SHA-256 do XML bruto
    parse_error         TEXT,           -- se o parser falhou, guarda a mensagem
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE company_documents;
DROP TABLE documents;
-- +goose StatementEnd
