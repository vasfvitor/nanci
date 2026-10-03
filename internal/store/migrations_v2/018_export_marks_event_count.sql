-- +goose Up
-- How many events of the document the export carried. The pending check
-- compares it with the current count, so an event stored in the same second
-- as the mark, or while the ZIP was written, still makes the document pending.
ALTER TABLE company_nfe_export_marks ADD COLUMN exported_events INTEGER NOT NULL DEFAULT 0;
ALTER TABLE company_cte_export_marks ADD COLUMN exported_events INTEGER NOT NULL DEFAULT 0;

-- Existing marks count only the events stored strictly before them, so an
-- event of the same second is pending once. Exporting it again is harmless.
UPDATE company_nfe_export_marks
SET exported_events = (
    SELECT COUNT(*) FROM nfe_events e
    INNER JOIN nfe_documents d ON d.chave_acesso = e.chave_acesso
    WHERE d.id = company_nfe_export_marks.nfe_document_id
        AND e.created_at < company_nfe_export_marks.exported_at
);
UPDATE company_cte_export_marks
SET exported_events = (
    SELECT COUNT(*) FROM cte_events e
    INNER JOIN cte_documents d ON d.chave_acesso = e.chave_acesso
    WHERE d.id = company_cte_export_marks.cte_document_id
        AND e.created_at < company_cte_export_marks.exported_at
);

-- +goose Down
ALTER TABLE company_cte_export_marks DROP COLUMN exported_events;
ALTER TABLE company_nfe_export_marks DROP COLUMN exported_events;
