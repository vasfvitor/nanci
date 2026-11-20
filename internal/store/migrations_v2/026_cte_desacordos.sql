-- +goose Up
-- Every prestação do serviço em desacordo (CT-e event 610110) nanci sent,
-- also the rejected and the unanswered ones, as nfe_manifestacoes does for
-- the NF-e. A registered event is also stored in cte_events, like one that
-- arrives by the distribution. A reset of the company's CT-e keeps these rows
-- as the audit trail of what was sent.
CREATE TABLE cte_desacordos (
    id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    chave_acesso TEXT NOT NULL,
    tp_amb TEXT NOT NULL CHECK (tp_amb IN ('1', '2')),
    n_seq_evento INTEGER NOT NULL,
    observacao TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('registrada', 'ja_registrada', 'rejeitada', 'erro')),
    c_stat TEXT NOT NULL,
    x_motivo TEXT NOT NULL,
    protocolo TEXT NOT NULL,
    event_at TEXT NOT NULL,
    registered_at TEXT,
    request_raw_hash TEXT,
    response_raw_hash TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_cte_desacordos_company_chave ON cte_desacordos(company_id, chave_acesso);

-- +goose Down
DROP INDEX idx_cte_desacordos_company_chave;
DROP TABLE cte_desacordos;
