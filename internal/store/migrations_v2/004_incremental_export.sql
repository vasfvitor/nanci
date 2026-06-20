-- +goose Up
ALTER TABLE company_documents ADD COLUMN viewed_at TEXT;

CREATE TABLE company_document_export_marks (
    company_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml', 'csv', 'xlsx', 'danfse')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    PRIMARY KEY (company_id, document_id, export_kind)
);

CREATE INDEX idx_company_documents_viewed_at ON company_documents(company_id, viewed_at);
CREATE INDEX idx_export_marks_kind ON company_document_export_marks(company_id, export_kind, exported_at);

-- +goose Down
DROP INDEX idx_export_marks_kind;
DROP INDEX idx_company_documents_viewed_at;
DROP TABLE company_document_export_marks;
-- Note: SQLite does not fully support DROP COLUMN safely without table recreation, so we leave viewed_at.
