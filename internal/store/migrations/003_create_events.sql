-- +goose Up
-- +goose StatementBegin
CREATE TABLE events (
    id                         TEXT PRIMARY KEY,
    document_id                TEXT NOT NULL REFERENCES documents(id),
    chave_acesso               TEXT NOT NULL,
    event_type                 TEXT NOT NULL,
    event_at                   TEXT,
    replacement_chave_acesso   TEXT,
    description                TEXT,
    raw_xml_path               TEXT NOT NULL,
    raw_hash                   TEXT NOT NULL,
    created_at                 TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_events_document ON events(document_id);
CREATE INDEX idx_events_chave ON events(chave_acesso);
CREATE UNIQUE INDEX idx_events_unique_raw_hash ON events(raw_hash);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd
