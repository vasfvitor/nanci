-- +goose Up
-- Lets the NF-e reset find documents no other company references without a
-- full scan of company_nfe_documents per document.
CREATE INDEX idx_company_nfe_documents_document ON company_nfe_documents(nfe_document_id);

-- +goose Down
DROP INDEX idx_company_nfe_documents_document;
