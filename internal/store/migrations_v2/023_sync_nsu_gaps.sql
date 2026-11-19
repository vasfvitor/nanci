-- +goose Up
-- NSU ranges the NF-e or CT-e distribution moved the cursor past without
-- delivering a document to nanci: the jump a cStat 656 makes when another
-- program consumed the CNPJ's sequence, or a hole inside a page. Recovery
-- asks for each NSU with consNSU, walking next_nsu up to last_nsu.
-- Gaps are only known from this version on; no existing data describes them.
CREATE TABLE sync_nsu_gaps (
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
    detected_at TEXT NOT NULL,
    finished_at TEXT,
    CHECK (first_nsu <= last_nsu),
    UNIQUE (company_id, source, environment, first_nsu)
);

-- +goose Down
DROP TABLE sync_nsu_gaps;
