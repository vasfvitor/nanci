-- +goose Up
-- When the company marked the CT-e as viewed, as on the NFS-e and NF-e
-- relations. NULL means the CT-e is new.
ALTER TABLE company_cte_documents ADD COLUMN viewed_at TEXT;
CREATE INDEX idx_company_cte_documents_viewed_at ON company_cte_documents(company_id, viewed_at);

-- +goose Down
DROP INDEX idx_company_cte_documents_viewed_at;
ALTER TABLE company_cte_documents DROP COLUMN viewed_at;
