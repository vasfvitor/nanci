-- +goose Up
-- NF-e and CT-e exports get the same kinds as NFS-e: the spreadsheets and the
-- PDF of each document (DANFE, DACTE). Each kind keeps its own marks, so an
-- XML export does not make the CSV of the same document look exported.
-- SQLite cannot change a CHECK in place, so both tables are rebuilt.
CREATE TABLE company_nfe_export_marks_new (
    company_id TEXT NOT NULL,
    nfe_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml', 'csv', 'xlsx', 'danfe')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    exported_events INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, nfe_document_id, export_kind)
);
INSERT INTO company_nfe_export_marks_new (company_id, nfe_document_id, export_kind, exported_hash, exported_at, exported_events)
SELECT company_id, nfe_document_id, export_kind, exported_hash, exported_at, exported_events FROM company_nfe_export_marks;
DROP TABLE company_nfe_export_marks;
ALTER TABLE company_nfe_export_marks_new RENAME TO company_nfe_export_marks;

CREATE TABLE company_cte_export_marks_new (
    company_id TEXT NOT NULL,
    cte_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml', 'csv', 'xlsx', 'dacte')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    exported_events INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, cte_document_id, export_kind)
);
INSERT INTO company_cte_export_marks_new (company_id, cte_document_id, export_kind, exported_hash, exported_at, exported_events)
SELECT company_id, cte_document_id, export_kind, exported_hash, exported_at, exported_events FROM company_cte_export_marks;
DROP TABLE company_cte_export_marks;
ALTER TABLE company_cte_export_marks_new RENAME TO company_cte_export_marks;

-- +goose Down
-- The marks of the new kinds are dropped: the old CHECK allows only 'xml'.
-- Those documents show as pending again for those kinds after an Up.
CREATE TABLE company_cte_export_marks_old (
    company_id TEXT NOT NULL,
    cte_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    exported_events INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, cte_document_id, export_kind)
);
INSERT INTO company_cte_export_marks_old (company_id, cte_document_id, export_kind, exported_hash, exported_at, exported_events)
SELECT company_id, cte_document_id, export_kind, exported_hash, exported_at, exported_events FROM company_cte_export_marks
WHERE export_kind = 'xml';
DROP TABLE company_cte_export_marks;
ALTER TABLE company_cte_export_marks_old RENAME TO company_cte_export_marks;

CREATE TABLE company_nfe_export_marks_old (
    company_id TEXT NOT NULL,
    nfe_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    exported_events INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, nfe_document_id, export_kind)
);
INSERT INTO company_nfe_export_marks_old (company_id, nfe_document_id, export_kind, exported_hash, exported_at, exported_events)
SELECT company_id, nfe_document_id, export_kind, exported_hash, exported_at, exported_events FROM company_nfe_export_marks
WHERE export_kind = 'xml';
DROP TABLE company_nfe_export_marks;
ALTER TABLE company_nfe_export_marks_old RENAME TO company_nfe_export_marks;
