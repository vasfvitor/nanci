-- +goose Up
-- A gap can be discarded by the user: a jump of thousands of NSUs would take
-- days at 20 queries an hour, and the documents may come in by XML import
-- instead. A discarded gap is never asked for again. SQLite cannot change a
-- CHECK in place, so the table is rebuilt.
CREATE TABLE sync_nsu_gaps_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfe', 'cte')),
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    first_nsu INTEGER NOT NULL,
    last_nsu INTEGER NOT NULL,
    next_nsu INTEGER NOT NULL,
    cause TEXT NOT NULL CHECK (cause IN ('consumo_indevido', 'batch_hole')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'done', 'expired', 'discarded')),
    recovered INTEGER NOT NULL DEFAULT 0,
    empty INTEGER NOT NULL DEFAULT 0,
    nsus_since TEXT NOT NULL,
    detected_at TEXT NOT NULL,
    finished_at TEXT,
    CHECK (first_nsu <= last_nsu),
    UNIQUE (company_id, source, environment, first_nsu)
);
INSERT INTO sync_nsu_gaps_new (id, company_id, source, environment, first_nsu, last_nsu, next_nsu, cause,
    status, recovered, empty, nsus_since, detected_at, finished_at)
SELECT id, company_id, source, environment, first_nsu, last_nsu, next_nsu, cause,
    status, recovered, empty, nsus_since, detected_at, finished_at
FROM sync_nsu_gaps;
DROP TABLE sync_nsu_gaps;
ALTER TABLE sync_nsu_gaps_new RENAME TO sync_nsu_gaps;

-- +goose Down
-- Discarded gaps become expired: closed and never asked for, as before.
CREATE TABLE sync_nsu_gaps_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfe', 'cte')),
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    first_nsu INTEGER NOT NULL,
    last_nsu INTEGER NOT NULL,
    next_nsu INTEGER NOT NULL,
    cause TEXT NOT NULL CHECK (cause IN ('consumo_indevido', 'batch_hole')),
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'done', 'expired')),
    recovered INTEGER NOT NULL DEFAULT 0,
    empty INTEGER NOT NULL DEFAULT 0,
    nsus_since TEXT NOT NULL,
    detected_at TEXT NOT NULL,
    finished_at TEXT,
    CHECK (first_nsu <= last_nsu),
    UNIQUE (company_id, source, environment, first_nsu)
);
INSERT INTO sync_nsu_gaps_old (id, company_id, source, environment, first_nsu, last_nsu, next_nsu, cause,
    status, recovered, empty, nsus_since, detected_at, finished_at)
SELECT id, company_id, source, environment, first_nsu, last_nsu, next_nsu, cause,
    CASE WHEN status = 'discarded' THEN 'expired' ELSE status END, recovered, empty, nsus_since, detected_at, finished_at
FROM sync_nsu_gaps;
DROP TABLE sync_nsu_gaps;
ALTER TABLE sync_nsu_gaps_old RENAME TO sync_nsu_gaps;
