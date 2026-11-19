-- +goose Up
-- A lease per company and source, so two nanci processes (the desktop app
-- and the CLI, possibly run by an AI agent) never query or change the same
-- source at once. The holder renews expires_at while it works; a crashed
-- holder's lease runs out and the next process takes it over.
CREATE TABLE source_locks (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    owner TEXT NOT NULL,
    holder TEXT NOT NULL,
    pid INTEGER NOT NULL,
    acquired_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    PRIMARY KEY (company_id, source)
);

-- +goose Down
DROP TABLE source_locks;
