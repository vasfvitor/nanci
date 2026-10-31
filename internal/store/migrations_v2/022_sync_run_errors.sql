-- +goose Up
-- A run that ends in error keeps its error code (persist_error, fetch_error,
-- process_error, ...) and message, so the run history says why it failed.
-- Existing runs keep NULL: their errors were never recorded.
ALTER TABLE sync_runs ADD COLUMN error_code TEXT;
ALTER TABLE sync_runs ADD COLUMN error_message TEXT;

-- +goose Down
ALTER TABLE sync_runs DROP COLUMN error_message;
ALTER TABLE sync_runs DROP COLUMN error_code;
