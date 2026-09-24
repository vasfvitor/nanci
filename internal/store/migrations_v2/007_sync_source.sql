-- +goose Up
CREATE TABLE sync_state_new (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    environment TEXT NOT NULL,
    consultation_cnpj TEXT NOT NULL,
    last_checked_nsu INTEGER NOT NULL DEFAULT 0,
    last_found_nsu INTEGER,
    max_nsu INTEGER,
    last_empty_streak INTEGER NOT NULL DEFAULT 0,
    last_success_at TEXT,
    last_error_at TEXT,
    last_error_code TEXT,
    last_error_message TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (company_id, source, environment, consultation_cnpj)
);

INSERT INTO sync_state_new (company_id, source, environment, consultation_cnpj, last_checked_nsu, last_found_nsu,
    max_nsu, last_empty_streak, last_success_at, last_error_at, last_error_code, last_error_message, created_at, updated_at)
SELECT company_id, 'nfse', environment, consultation_cnpj, last_checked_nsu, last_found_nsu,
    NULL, last_empty_streak, last_success_at, last_error_at, last_error_code, last_error_message, created_at, updated_at
FROM sync_state;

DROP TABLE sync_state;
ALTER TABLE sync_state_new RENAME TO sync_state;

ALTER TABLE sync_runs ADD COLUMN source TEXT NOT NULL DEFAULT 'nfse' CHECK (source IN ('nfse', 'nfe', 'cte'));
DROP INDEX idx_sync_runs_running;
CREATE UNIQUE INDEX idx_sync_runs_running ON sync_runs(company_id, source) WHERE status = 'running';

-- Per (company, source) facts. ResetSyncState clears initial_sync_completed_at
-- but keeps blocked_until: the tax authority still blocks after a local reset.
CREATE TABLE company_sync_sources (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    initial_sync_completed_at TEXT,
    blocked_until TEXT,
    blocked_reason TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (company_id, source)
);

INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, updated_at)
SELECT id, 'nfse', initial_sync_completed_at, updated_at FROM companies WHERE initial_sync_completed_at IS NOT NULL;

-- One row per outbound distribution request, for the rolling hourly budget.
CREATE TABLE sync_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL,
    requested_at TEXT NOT NULL
);
CREATE INDEX idx_sync_requests_window ON sync_requests(company_id, source, requested_at);

-- +goose Down
DROP INDEX idx_sync_requests_window;
DROP TABLE sync_requests;
DROP TABLE company_sync_sources;

UPDATE sync_runs SET status = 'interrupted', stop_reason = 'context_canceled'
WHERE status = 'running' AND source <> 'nfse';
DROP INDEX idx_sync_runs_running;
CREATE UNIQUE INDEX idx_sync_runs_running ON sync_runs(company_id) WHERE status = 'running';
-- Note: sync_runs.source stays; SQLite cannot drop a column with a CHECK safely without table recreation.

CREATE TABLE sync_state_old (
    company_id TEXT NOT NULL REFERENCES companies(id),
    environment TEXT NOT NULL,
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

INSERT INTO sync_state_old (company_id, environment, consultation_cnpj, last_checked_nsu, last_found_nsu,
    last_empty_streak, last_success_at, last_error_at, last_error_code, last_error_message, created_at, updated_at)
SELECT company_id, environment, consultation_cnpj, last_checked_nsu, last_found_nsu,
    last_empty_streak, last_success_at, last_error_at, last_error_code, last_error_message, created_at, updated_at
FROM sync_state
WHERE source = 'nfse';

DROP TABLE sync_state;
ALTER TABLE sync_state_old RENAME TO sync_state;
