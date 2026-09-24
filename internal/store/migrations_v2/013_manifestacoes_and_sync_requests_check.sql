-- +goose Up
-- Keeps manifestação in Portuguese in the schema, as in the rest of the code.
ALTER TABLE nfe_manifestations RENAME TO nfe_manifestacoes;
DROP INDEX idx_nfe_manifestations_company_chave;
CREATE INDEX idx_nfe_manifestacoes_company_chave ON nfe_manifestacoes(company_id, chave_acesso);

-- Matches the name of idx_company_documents_viewed_at on the NFS-e side.
DROP INDEX idx_company_nfe_documents_viewed;
CREATE INDEX idx_company_nfe_documents_viewed_at ON company_nfe_documents(company_id, viewed_at);

-- sync_requests gets the same CHECK on source as the other sync tables.
-- SQLite cannot add a CHECK in place, so the table is rebuilt.
CREATE TABLE sync_requests_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    requested_at TEXT NOT NULL
);
INSERT INTO sync_requests_new (id, company_id, source, requested_at)
SELECT id, company_id, source, requested_at FROM sync_requests;
DROP INDEX idx_sync_requests_window;
DROP TABLE sync_requests;
ALTER TABLE sync_requests_new RENAME TO sync_requests;
CREATE INDEX idx_sync_requests_window ON sync_requests(company_id, source, requested_at);

-- +goose Down
CREATE TABLE sync_requests_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL,
    requested_at TEXT NOT NULL
);
INSERT INTO sync_requests_old (id, company_id, source, requested_at)
SELECT id, company_id, source, requested_at FROM sync_requests;
DROP INDEX idx_sync_requests_window;
DROP TABLE sync_requests;
ALTER TABLE sync_requests_old RENAME TO sync_requests;
CREATE INDEX idx_sync_requests_window ON sync_requests(company_id, source, requested_at);

DROP INDEX idx_company_nfe_documents_viewed_at;
CREATE INDEX idx_company_nfe_documents_viewed ON company_nfe_documents(company_id, viewed_at);

DROP INDEX idx_nfe_manifestacoes_company_chave;
ALTER TABLE nfe_manifestacoes RENAME TO nfe_manifestations;
CREATE INDEX idx_nfe_manifestations_company_chave ON nfe_manifestations(company_id, chave_acesso);
