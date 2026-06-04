-- +goose Up
-- +goose StatementBegin
CREATE TABLE documents (
    id              TEXT PRIMARY KEY,
    company_id      TEXT NOT NULL REFERENCES companies(id),
    chave_acesso    TEXT NOT NULL UNIQUE,
    nsu             INTEGER,
    direction       TEXT CHECK(direction IN ('tomada', 'prestada', 'intermediario')),
    issue_date      TEXT,
    competence      TEXT,          -- formato YYYY-MM
    prestador_cnpj  TEXT,
    prestador_name  TEXT,
    tomador_cnpj    TEXT,
    tomador_name    TEXT,
    service_value   REAL DEFAULT 0,
    iss_value       REAL DEFAULT 0,
    irrf_value      REAL DEFAULT 0,
    inss_value      REAL DEFAULT 0,
    pis_value       REAL DEFAULT 0,
    cofins_value    REAL DEFAULT 0,
    csll_value      REAL DEFAULT 0,
    status          TEXT DEFAULT 'normal',
    xml_path        TEXT NOT NULL,
    raw_hash        TEXT NOT NULL,  -- SHA-256 do XML bruto
    parse_error     TEXT,           -- se o parser falhou, guarda a mensagem
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_documents_company_competence ON documents(company_id, competence);
CREATE INDEX idx_documents_chave ON documents(chave_acesso);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE documents;
-- +goose StatementEnd
