-- +goose Up
ALTER TABLE companies ADD COLUMN sync_start_policy TEXT NOT NULL DEFAULT 'all';
ALTER TABLE companies ADD COLUMN sync_start_date TEXT;
ALTER TABLE companies ADD COLUMN initial_sync_completed_at TEXT;

UPDATE companies
SET initial_sync_completed_at = updated_at
WHERE initial_sync_completed_at IS NULL;

-- +goose Down
-- SQLite does not support dropping these columns safely without table recreation.
