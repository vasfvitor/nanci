-- +goose Up
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;

CREATE TABLE companies_new (
    id          TEXT PRIMARY KEY,
    cnpj        TEXT NOT NULL UNIQUE,       -- suporta formato numérico e alfanumérico
    cnpj_root   TEXT NOT NULL,              -- primeiros 8 chars para agrupamento de filiais
    name        TEXT,
    cert_path   TEXT NOT NULL,
    environment TEXT NOT NULL DEFAULT 'producao_restrita'
                     CHECK(environment IN ('producao_restrita', 'producao', 'homologacao')),
    last_nsu    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO companies_new (id, cnpj, cnpj_root, name, cert_path, environment, last_nsu, created_at, updated_at)
SELECT id, cnpj, cnpj_root, name, cert_path, environment, last_nsu, created_at, updated_at
FROM companies;

DROP TABLE companies;
ALTER TABLE companies_new RENAME TO companies;

CREATE INDEX idx_companies_cnpj_root ON companies(cnpj_root);

PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;

UPDATE companies SET environment = 'producao_restrita' WHERE environment = 'homologacao';

CREATE TABLE companies_new (
    id          TEXT PRIMARY KEY,
    cnpj        TEXT NOT NULL UNIQUE,       
    cnpj_root   TEXT NOT NULL,              
    name        TEXT,
    cert_path   TEXT NOT NULL,
    environment TEXT NOT NULL DEFAULT 'producao_restrita'
                     CHECK(environment IN ('producao_restrita', 'producao')),
    last_nsu    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO companies_new (id, cnpj, cnpj_root, name, cert_path, environment, last_nsu, created_at, updated_at)
SELECT id, cnpj, cnpj_root, name, cert_path, environment, last_nsu, created_at, updated_at
FROM companies;

DROP TABLE companies;
ALTER TABLE companies_new RENAME TO companies;

CREATE INDEX idx_companies_cnpj_root ON companies(cnpj_root);

PRAGMA foreign_keys=ON;
-- +goose StatementEnd
