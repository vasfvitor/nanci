-- +goose Up
-- +goose StatementBegin
CREATE TABLE sync_runs (
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

CREATE INDEX idx_syncruns_company ON sync_runs(company_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sync_runs;
-- +goose StatementEnd
