-- +goose Up
-- The tpAmb (1 produção, 2 homologação) each manifestação was sent to.
-- Rows sent before this column existed keep it empty.
ALTER TABLE nfe_manifestations ADD COLUMN tp_amb TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE nfe_manifestations DROP COLUMN tp_amb;
