-- +goose Up
-- The NSU whose document last failed to decode or parse, and how many times
-- in a row it failed. After three failures the loop skips it as unsupported.
ALTER TABLE sync_state ADD COLUMN failed_nsu INTEGER;
ALTER TABLE sync_state ADD COLUMN failed_nsu_attempts INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sync_state DROP COLUMN failed_nsu_attempts;
ALTER TABLE sync_state DROP COLUMN failed_nsu;
