-- +goose Up
ALTER TABLE companies DROP COLUMN last_nsu;

CREATE TABLE company_documents_new (
    relation_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    document_id TEXT NOT NULL REFERENCES documents(id),
    company_role TEXT NOT NULL CHECK (company_role IN ('tomada', 'prestada', 'intermediario', 'none')),
    visibility_reason TEXT NOT NULL CHECK (visibility_reason IN ('exact_prestador', 'exact_tomador', 'exact_intermediario', 'same_root_only', 'unknown')),
    first_seen_nsu INTEGER,
    last_seen_nsu INTEGER,
    first_synced_at TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    viewed_at TEXT,
    UNIQUE(company_id, document_id)
);

INSERT INTO company_documents_new (relation_id, company_id, document_id, company_role, visibility_reason, first_seen_nsu, last_seen_nsu, first_synced_at, last_synced_at, viewed_at)
SELECT relation_id, company_id, document_id, company_role, visibility_reason,
       CASE WHEN first_seen_nsu_valid = 1 THEN first_seen_nsu ELSE NULL END,
       CASE WHEN last_seen_nsu_valid = 1 THEN last_seen_nsu ELSE NULL END,
       first_synced_at, last_synced_at, viewed_at
FROM company_documents;

DROP TABLE company_documents;
ALTER TABLE company_documents_new RENAME TO company_documents;
CREATE INDEX idx_company_documents_viewed_at ON company_documents(company_id, viewed_at);

-- +goose Down
-- Add rollback logic for recreating company_documents with valid flags and adding last_nsu to companies.
ALTER TABLE companies ADD COLUMN last_nsu INTEGER NOT NULL DEFAULT 0;

CREATE TABLE company_documents_old (
    relation_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    document_id TEXT NOT NULL REFERENCES documents(id),
    company_role TEXT NOT NULL CHECK (company_role IN ('tomada', 'prestada', 'intermediario', 'none')),
    visibility_reason TEXT NOT NULL CHECK (visibility_reason IN ('exact_prestador', 'exact_tomador', 'exact_intermediario', 'same_root_only', 'unknown')),
    first_seen_nsu INTEGER NOT NULL,
    last_seen_nsu INTEGER NOT NULL,
    first_seen_nsu_valid INTEGER NOT NULL DEFAULT 0,
    last_seen_nsu_valid INTEGER NOT NULL DEFAULT 0,
    first_synced_at TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    viewed_at TEXT,
    UNIQUE(company_id, document_id)
);

INSERT INTO company_documents_old (relation_id, company_id, document_id, company_role, visibility_reason, first_seen_nsu, last_seen_nsu, first_seen_nsu_valid, last_seen_nsu_valid, first_synced_at, last_synced_at, viewed_at)
SELECT relation_id, company_id, document_id, company_role, visibility_reason,
       COALESCE(first_seen_nsu, 0), COALESCE(last_seen_nsu, 0),
       CASE WHEN first_seen_nsu IS NOT NULL THEN 1 ELSE 0 END,
       CASE WHEN last_seen_nsu IS NOT NULL THEN 1 ELSE 0 END,
       first_synced_at, last_synced_at, viewed_at
FROM company_documents;

DROP TABLE company_documents;
ALTER TABLE company_documents_old RENAME TO company_documents;
CREATE INDEX idx_company_documents_viewed_at ON company_documents(company_id, viewed_at);
