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

INSERT INTO credentials (
    id, label, cert_path, environment, created_at, updated_at
)
SELECT
    lower(hex(randomblob(16))),
    COALESCE(NULLIF(name, ''), cnpj),
    cert_path,
    environment,
    MIN(created_at),
    MAX(updated_at)
FROM companies
GROUP BY cert_path, environment;

CREATE UNIQUE INDEX idx_credentials_env_path ON credentials(environment, cert_path);
CREATE INDEX idx_credentials_owner_root ON credentials(owner_cnpj_root);

CREATE TABLE companies_new (
    id              TEXT PRIMARY KEY,
    cnpj            TEXT NOT NULL UNIQUE,
    cnpj_root       TEXT NOT NULL,
    name            TEXT,
    credential_id   TEXT NOT NULL REFERENCES credentials(id),
    last_nsu        INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO companies_new (id, cnpj, cnpj_root, name, credential_id, last_nsu, created_at, updated_at)
SELECT
    comp.id,
    comp.cnpj,
    comp.cnpj_root,
    comp.name,
    cred.id,
    comp.last_nsu,
    comp.created_at,
    comp.updated_at
FROM companies comp
JOIN credentials cred
  ON cred.cert_path = comp.cert_path
 AND cred.environment = comp.environment;

DROP TABLE companies;
ALTER TABLE companies_new RENAME TO companies;
CREATE INDEX idx_companies_cnpj_root ON companies(cnpj_root);
CREATE INDEX idx_companies_credential_id ON companies(credential_id);

CREATE TABLE sync_runs_new (
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

INSERT INTO sync_runs_new (
    id, company_id, started_at, finished_at, from_nsu, to_nsu, documents_found, errors_count, status
)
SELECT
    id, company_id, started_at, finished_at, from_nsu, to_nsu, documents_found, errors_count, status
FROM sync_runs;

DROP TABLE sync_runs;
ALTER TABLE sync_runs_new RENAME TO sync_runs;
CREATE INDEX idx_syncruns_company ON sync_runs(company_id);
CREATE INDEX idx_syncruns_credential ON sync_runs(credential_id);

PRAGMA foreign_keys=ON;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
PRAGMA foreign_keys=OFF;

CREATE TABLE companies_old (
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

INSERT INTO companies_old (id, cnpj, cnpj_root, name, cert_path, environment, last_nsu, created_at, updated_at)
SELECT
    comp.id,
    comp.cnpj,
    comp.cnpj_root,
    comp.name,
    cred.cert_path,
    cred.environment,
    comp.last_nsu,
    comp.created_at,
    comp.updated_at
FROM companies comp
JOIN credentials cred ON cred.id = comp.credential_id;

DROP TABLE companies;
ALTER TABLE companies_old RENAME TO companies;
CREATE INDEX idx_companies_cnpj_root ON companies(cnpj_root);

CREATE TABLE sync_runs_old (
    id              TEXT PRIMARY KEY,
    company_id      TEXT NOT NULL REFERENCES companies(id),
    started_at      TEXT NOT NULL,
    finished_at     TEXT,
    from_nsu        INTEGER,
    to_nsu          INTEGER,
    documents_found INTEGER DEFAULT 0,
    errors_count    INTEGER DEFAULT 0,
    status          TEXT NOT NULL CHECK(status IN ('running', 'completed', 'failed', 'interrupted'))
);

INSERT INTO sync_runs_old (
    id, company_id, started_at, finished_at, from_nsu, to_nsu, documents_found, errors_count, status
)
SELECT
    id, company_id, started_at, finished_at, from_nsu, to_nsu, documents_found, errors_count, status
FROM sync_runs;

DROP TABLE sync_runs;
ALTER TABLE sync_runs_old RENAME TO sync_runs;
CREATE INDEX idx_syncruns_company ON sync_runs(company_id);

DROP TABLE credentials;

PRAGMA foreign_keys=ON;
-- +goose StatementEnd
