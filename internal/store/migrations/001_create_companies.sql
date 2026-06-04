-- +goose Up
-- +goose StatementBegin
CREATE TABLE companies (
    id          TEXT PRIMARY KEY,
    cnpj        TEXT NOT NULL UNIQUE,       -- suporta formato numérico e alfanumérico
    cnpj_root   TEXT NOT NULL,              -- primeiros 8 chars para agrupamento de filiais
    name        TEXT,
    cert_path   TEXT NOT NULL,
    environment TEXT NOT NULL DEFAULT 'producao_restrita'
                     CHECK(environment IN ('producao_restrita', 'producao')),
    last_nsu    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_companies_cnpj_root ON companies(cnpj_root);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE companies;
-- +goose StatementEnd
