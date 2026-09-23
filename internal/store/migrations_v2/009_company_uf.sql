-- +goose Up
-- Optional state sigla of the company (e.g. "SP"); empty when unknown.
-- NF-e distribution sends it as cUFAutor.
ALTER TABLE companies ADD COLUMN uf TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE companies DROP COLUMN uf;
