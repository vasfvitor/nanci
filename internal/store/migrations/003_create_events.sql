-- +goose Up
-- +goose StatementBegin
CREATE TABLE events (
    id            TEXT PRIMARY KEY,
    document_id   TEXT NOT NULL REFERENCES documents(id),
    chave_acesso  TEXT NOT NULL,
    event_type    TEXT,
    event_date    TEXT,
    description   TEXT,
    raw_xml_path  TEXT,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_events_document ON events(document_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE events;
-- +goose StatementEnd
