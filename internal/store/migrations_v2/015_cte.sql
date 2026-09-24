-- +goose Up
CREATE TABLE cte_documents (
    id TEXT PRIMARY KEY,
    chave_acesso TEXT NOT NULL UNIQUE,
    tp_amb TEXT NOT NULL CHECK (tp_amb IN ('1', '2')),
    modelo TEXT NOT NULL CHECK (modelo IN ('57', '64', '67')),
    tipo_documento TEXT NOT NULL CHECK (tipo_documento IN ('cte', 'cte_os', 'gtve', 'cte_simplificado')),
    serie TEXT NOT NULL,
    numero TEXT NOT NULL,
    cfop TEXT NOT NULL,
    nat_op TEXT NOT NULL,
    issue_date TEXT NOT NULL,
    competence TEXT NOT NULL,
    authorized_at TEXT,
    protocolo TEXT NOT NULL,
    tp_cte TEXT NOT NULL,
    tp_serv TEXT NOT NULL,
    modal TEXT NOT NULL,
    mun_ini_codigo TEXT NOT NULL,
    mun_ini_nome TEXT NOT NULL,
    uf_ini TEXT NOT NULL,
    mun_fim_codigo TEXT NOT NULL,
    mun_fim_nome TEXT NOT NULL,
    uf_fim TEXT NOT NULL,
    emitente_cnpj TEXT NOT NULL,
    emitente_name TEXT NOT NULL,
    emitente_ie TEXT NOT NULL,
    emitente_uf TEXT NOT NULL,
    remetente_cnpj TEXT NOT NULL,
    remetente_name TEXT NOT NULL,
    destinatario_cnpj TEXT NOT NULL,
    destinatario_name TEXT NOT NULL,
    expedidor_cnpj TEXT NOT NULL,
    expedidor_name TEXT NOT NULL,
    recebedor_cnpj TEXT NOT NULL,
    recebedor_name TEXT NOT NULL,
    tomador_indicador TEXT NOT NULL,
    tomador_cnpj TEXT NOT NULL,
    tomador_name TEXT NOT NULL,
    tomador_ie TEXT NOT NULL,
    tomador_uf TEXT NOT NULL,
    autorizados_cnpj TEXT NOT NULL DEFAULT '',
    nfe_chaves TEXT NOT NULL DEFAULT '',
    total_value INTEGER NOT NULL DEFAULT 0,
    receivable_value INTEGER NOT NULL DEFAULT 0,
    icms_value INTEGER NOT NULL DEFAULT 0,
    tot_trib_value INTEGER NOT NULL DEFAULT 0,
    carga_value INTEGER NOT NULL DEFAULT 0,
    produto_predominante TEXT NOT NULL,
    situacao TEXT NOT NULL CHECK (situacao IN ('autorizada', 'denegada', 'cancelada')),
    layout_version TEXT NOT NULL,
    raw_hash TEXT NOT NULL UNIQUE,
    parse_warnings TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_cte_documents_emitente ON cte_documents(emitente_cnpj);
CREATE INDEX idx_cte_documents_tomador ON cte_documents(tomador_cnpj);
CREATE INDEX idx_cte_documents_tp_amb_competence ON cte_documents(tp_amb, competence);

-- papeis lists every role the company plays, comma-separated in priority
-- order; company_role is the first one.
CREATE TABLE company_cte_documents (
    relation_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    cte_document_id TEXT NOT NULL REFERENCES cte_documents(id),
    company_role TEXT NOT NULL CHECK (company_role IN ('tomador', 'destinatario', 'remetente', 'expedidor', 'recebedor', 'emitente', 'autorizado', 'none')),
    papeis TEXT NOT NULL,
    visibility_reason TEXT NOT NULL CHECK (visibility_reason IN ('exact_tomador', 'exact_destinatario', 'exact_remetente', 'exact_expedidor', 'exact_recebedor', 'exact_emitente', 'exact_autorizado', 'same_root_only', 'unknown')),
    first_seen_nsu INTEGER,
    last_seen_nsu INTEGER,
    first_synced_at TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    UNIQUE (company_id, cte_document_id)
);
CREATE INDEX idx_company_cte_documents_document ON company_cte_documents(cte_document_id);

-- One row per (chave, tpEvento, nSeqEvento). An event may arrive before its
-- document; cte_document_id is filled when the document arrives.
CREATE TABLE cte_events (
    id TEXT PRIMARY KEY,
    cte_document_id TEXT REFERENCES cte_documents(id),
    chave_acesso TEXT NOT NULL,
    tp_amb TEXT NOT NULL CHECK (tp_amb IN ('1', '2')),
    c_orgao TEXT NOT NULL,
    tp_evento TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('cancelamento', 'carta_correcao', 'epec', 'registro_multimodal', 'gtv', 'comprovante_entrega', 'cancelamento_comprovante_entrega', 'insucesso_entrega', 'cancelamento_insucesso_entrega', 'prestacao_desacordo', 'cancelamento_desacordo', 'mdfe_autorizado', 'mdfe_cancelado', 'unknown')),
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
    observacao TEXT NOT NULL,
    correcao TEXT NOT NULL,
    condicao_uso TEXT NOT NULL,
    raw_hash TEXT NOT NULL,
    parse_warnings TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (chave_acesso, tp_evento, n_seq_evento)
);

CREATE TABLE company_cte_export_marks (
    company_id TEXT NOT NULL,
    cte_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    PRIMARY KEY (company_id, cte_document_id, export_kind)
);

-- +goose Down
DROP TABLE company_cte_export_marks;
DROP TABLE cte_events;
DROP INDEX idx_company_cte_documents_document;
DROP TABLE company_cte_documents;
DROP INDEX idx_cte_documents_tp_amb_competence;
DROP INDEX idx_cte_documents_tomador;
DROP INDEX idx_cte_documents_emitente;
DROP TABLE cte_documents;
