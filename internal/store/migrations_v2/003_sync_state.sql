-- +goose Up

CREATE TABLE sync_state (
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

ALTER TABLE sync_runs ADD COLUMN environment TEXT NOT NULL DEFAULT 'producao_restrita';
ALTER TABLE sync_runs ADD COLUMN mode TEXT NOT NULL DEFAULT 'normal';
ALTER TABLE sync_runs ADD COLUMN stop_reason TEXT;
ALTER TABLE sync_runs ADD COLUMN checked_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_runs ADD COLUMN empty_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_runs ADD COLUMN consecutive_empty_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sync_runs ADD COLUMN last_found_nsu INTEGER;
