-- +goose Up
-- The SEFAZ block, the initial sync and the hourly request budget belong to
-- one environment, as the sync cursor in sync_state already does, so a
-- company that switches environments does not carry the other one's block.
-- SQLite cannot change a primary key in place, so both tables are rebuilt.
-- Existing rows take the company's current environment.
CREATE TABLE company_sync_sources_new (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    initial_sync_completed_at TEXT,
    blocked_until TEXT,
    blocked_reason TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (company_id, source, environment)
);
INSERT INTO company_sync_sources_new (company_id, source, environment, initial_sync_completed_at,
    blocked_until, blocked_reason, updated_at)
SELECT s.company_id, s.source, c.environment, s.initial_sync_completed_at,
    s.blocked_until, s.blocked_reason, s.updated_at
FROM company_sync_sources s
INNER JOIN companies c ON c.id = s.company_id;
DROP TABLE company_sync_sources;
ALTER TABLE company_sync_sources_new RENAME TO company_sync_sources;

CREATE TABLE sync_requests_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    requested_at TEXT NOT NULL
);
INSERT INTO sync_requests_new (id, company_id, source, environment, requested_at)
SELECT r.id, r.company_id, r.source, c.environment, r.requested_at
FROM sync_requests r
INNER JOIN companies c ON c.id = r.company_id;
DROP INDEX idx_sync_requests_window;
DROP TABLE sync_requests;
ALTER TABLE sync_requests_new RENAME TO sync_requests;
CREATE INDEX idx_sync_requests_window ON sync_requests(company_id, source, environment, requested_at);

-- +goose Down
-- Only the rows of the company's current environment are kept.
CREATE TABLE sync_requests_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    requested_at TEXT NOT NULL
);
INSERT INTO sync_requests_old (id, company_id, source, requested_at)
SELECT r.id, r.company_id, r.source, r.requested_at
FROM sync_requests r
INNER JOIN companies c ON c.id = r.company_id AND c.environment = r.environment;
DROP INDEX idx_sync_requests_window;
DROP TABLE sync_requests;
ALTER TABLE sync_requests_old RENAME TO sync_requests;
CREATE INDEX idx_sync_requests_window ON sync_requests(company_id, source, requested_at);

CREATE TABLE company_sync_sources_old (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    initial_sync_completed_at TEXT,
    blocked_until TEXT,
    blocked_reason TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (company_id, source)
);
INSERT INTO company_sync_sources_old (company_id, source, initial_sync_completed_at,
    blocked_until, blocked_reason, updated_at)
SELECT s.company_id, s.source, s.initial_sync_completed_at,
    s.blocked_until, s.blocked_reason, s.updated_at
FROM company_sync_sources s
INNER JOIN companies c ON c.id = s.company_id AND c.environment = s.environment;
DROP TABLE company_sync_sources;
ALTER TABLE company_sync_sources_old RENAME TO company_sync_sources;
