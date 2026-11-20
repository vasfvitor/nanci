-- +goose Up
-- How many events of the NFS-e the export carried, as 018 did for the NF-e
-- and the CT-e. A cancelamento changes the note's status but not its raw
-- hash, so the pending check also compares the event count; before this, an
-- incremental export never brought back a note cancelled after it.
ALTER TABLE company_document_export_marks ADD COLUMN exported_events INTEGER NOT NULL DEFAULT 0;

-- Existing marks count the events stored before them, so a note that got an
-- event after its last export is pending once. Exporting it again is
-- harmless.
UPDATE company_document_export_marks
SET exported_events = (
    SELECT COUNT(*) FROM events e
    INNER JOIN documents d ON d.chave_acesso = e.chave_acesso
    WHERE d.id = company_document_export_marks.document_id
        AND e.created_at < company_document_export_marks.exported_at
);

-- +goose Down
ALTER TABLE company_document_export_marks DROP COLUMN exported_events;
