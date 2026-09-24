-- +goose Up
CREATE TABLE nfe_documents (
    id TEXT PRIMARY KEY,
    chave_acesso TEXT NOT NULL UNIQUE,
    modelo TEXT NOT NULL,
    serie TEXT NOT NULL,
    numero TEXT NOT NULL,
    issue_date TEXT NOT NULL,
    competence TEXT NOT NULL,
    authorized_at TEXT,
    protocolo TEXT NOT NULL,
    emitente_cnpj TEXT NOT NULL,
    emitente_name TEXT NOT NULL,
    emitente_ie TEXT NOT NULL,
    emitente_uf TEXT NOT NULL,
    destinatario_cnpj TEXT NOT NULL,
    destinatario_name TEXT NOT NULL,
    transportador_cnpj TEXT NOT NULL,
    autorizados_cnpj TEXT NOT NULL DEFAULT '',
    tp_nf TEXT NOT NULL CHECK (tp_nf IN ('0', '1')),
    fin_nfe TEXT NOT NULL,
    nat_op TEXT NOT NULL,
    total_value INTEGER NOT NULL DEFAULT 0,
    icms_value INTEGER NOT NULL DEFAULT 0,
    ipi_value INTEGER NOT NULL DEFAULT 0,
    situacao TEXT NOT NULL CHECK (situacao IN ('autorizada', 'denegada', 'cancelada')),
    completeness TEXT NOT NULL CHECK (completeness IN ('resumo', 'completa')),
    layout_version TEXT NOT NULL,
    raw_hash TEXT NOT NULL UNIQUE,
    resumo_raw_hash TEXT,
    parse_warnings TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_nfe_documents_emitente ON nfe_documents(emitente_cnpj);

CREATE TABLE company_nfe_documents (
    relation_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    nfe_document_id TEXT NOT NULL REFERENCES nfe_documents(id),
    company_role TEXT NOT NULL CHECK (company_role IN ('destinatario', 'emitente', 'transportador', 'autorizado', 'none')),
    visibility_reason TEXT NOT NULL CHECK (visibility_reason IN ('exact_destinatario', 'exact_emitente', 'exact_transportador', 'exact_autorizado', 'resumo_destinatario', 'same_root_only', 'unknown')),
    manifestacao TEXT NOT NULL DEFAULT 'nenhuma' CHECK (manifestacao IN ('nenhuma', 'ciencia', 'confirmada', 'desconhecida', 'nao_realizada')),
    manifestacao_at TEXT,
    first_seen_nsu INTEGER,
    last_seen_nsu INTEGER,
    first_synced_at TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    viewed_at TEXT,
    UNIQUE (company_id, nfe_document_id)
);
CREATE INDEX idx_company_nfe_documents_viewed ON company_nfe_documents(company_id, viewed_at);
CREATE INDEX idx_company_nfe_documents_manifestacao ON company_nfe_documents(company_id, manifestacao);

-- One row per (chave, tpEvento, nSeqEvento): a resEvento upgrades in place to
-- its procEventoNFe, and an event sent by nanci collapses with its later
-- distributed copy. A resumo never overwrites a completa.
CREATE TABLE nfe_events (
    id TEXT PRIMARY KEY,
    nfe_document_id TEXT REFERENCES nfe_documents(id),
    chave_acesso TEXT NOT NULL,
    tp_evento TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('cancelamento', 'carta_correcao', 'ciencia', 'confirmacao', 'desconhecimento', 'nao_realizada', 'unknown')),
    n_seq_evento INTEGER NOT NULL,
    event_at TEXT,
    registered_at TEXT,
    registered INTEGER NOT NULL DEFAULT 0,
    c_stat TEXT NOT NULL DEFAULT '',
    x_motivo TEXT NOT NULL DEFAULT '',
    protocolo TEXT NOT NULL,
    autor_cnpj TEXT NOT NULL,
    description TEXT NOT NULL,
    justificativa TEXT NOT NULL,
    correcao TEXT NOT NULL,
    completeness TEXT NOT NULL CHECK (completeness IN ('resumo', 'completa')),
    sent_by_nanci INTEGER NOT NULL DEFAULT 0,
    raw_hash TEXT NOT NULL,
    parse_warnings TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (chave_acesso, tp_evento, n_seq_evento)
);

-- Outbound manifestação attempts (audit and error display). Registered ones
-- also land in nfe_events.
CREATE TABLE nfe_manifestations (
    id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    chave_acesso TEXT NOT NULL,
    tp_evento TEXT NOT NULL,
    n_seq_evento INTEGER NOT NULL,
    justificativa TEXT NOT NULL,
    id_lote TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('registrada', 'ja_registrada', 'rejeitada', 'erro')),
    c_stat TEXT NOT NULL,
    x_motivo TEXT NOT NULL,
    protocolo TEXT NOT NULL,
    registered_at TEXT,
    request_raw_hash TEXT,
    response_raw_hash TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_nfe_manifestations_company_chave ON nfe_manifestations(company_id, chave_acesso);

CREATE TABLE company_nfe_export_marks (
    company_id TEXT NOT NULL,
    nfe_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    PRIMARY KEY (company_id, nfe_document_id, export_kind)
);

-- +goose Down
DROP TABLE company_nfe_export_marks;
DROP INDEX idx_nfe_manifestations_company_chave;
DROP TABLE nfe_manifestations;
DROP TABLE nfe_events;
DROP INDEX idx_company_nfe_documents_manifestacao;
DROP INDEX idx_company_nfe_documents_viewed;
DROP TABLE company_nfe_documents;
DROP INDEX idx_nfe_documents_emitente;
DROP TABLE nfe_documents;
