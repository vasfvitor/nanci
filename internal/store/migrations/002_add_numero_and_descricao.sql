-- +goose Up
-- +goose StatementBegin
ALTER TABLE documents ADD COLUMN nfse_number TEXT;
ALTER TABLE documents ADD COLUMN service_description TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE documents DROP COLUMN service_description;
ALTER TABLE documents DROP COLUMN nfse_number;
-- +goose StatementEnd
