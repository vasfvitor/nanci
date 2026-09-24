-- +goose Up
-- company_sync_sources is the only record of the initial sync since 007.
-- Copy any NFS-e value still missing there before dropping the old column.
INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, updated_at)
SELECT id, 'nfse', initial_sync_completed_at, updated_at FROM companies WHERE initial_sync_completed_at IS NOT NULL
ON CONFLICT (company_id, source) DO UPDATE SET
    initial_sync_completed_at = COALESCE(company_sync_sources.initial_sync_completed_at, excluded.initial_sync_completed_at);
ALTER TABLE companies DROP COLUMN initial_sync_completed_at;

-- +goose Down
ALTER TABLE companies ADD COLUMN initial_sync_completed_at TEXT;
UPDATE companies
SET initial_sync_completed_at = (
    SELECT initial_sync_completed_at FROM company_sync_sources
    WHERE company_sync_sources.company_id = companies.id AND company_sync_sources.source = 'nfse'
);
