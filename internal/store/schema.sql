CREATE TABLE companies (
    id TEXT PRIMARY KEY,
    cnpj TEXT NOT NULL UNIQUE,
    cnpj_root TEXT NOT NULL,
    name TEXT NOT NULL,
    credential_id TEXT,
    credential_label TEXT,
    credential_cert_path TEXT,
    environment TEXT NOT NULL CHECK (environment IN ('producao', 'producao_restrita')),
    sync_start_policy TEXT NOT NULL DEFAULT 'from_now' CHECK (sync_start_policy IN ('all', 'since_date', 'from_now')),
    sync_start_date TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    uf TEXT NOT NULL DEFAULT ''
);

CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    cert_path TEXT NOT NULL,
    owner_cnpj TEXT NOT NULL,
    owner_cnpj_root TEXT NOT NULL,
    fingerprint_sha256 TEXT NOT NULL,
    subject_name TEXT NOT NULL,
    not_before TEXT,
    not_after TEXT,
    inspected_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    chave_acesso TEXT NOT NULL UNIQUE,
    issue_date TEXT NOT NULL,
    competence TEXT NOT NULL,
    prestador_cnpj TEXT NOT NULL,
    prestador_name TEXT NOT NULL,
    tomador_cnpj TEXT NOT NULL,
    tomador_name TEXT NOT NULL,
    intermediario_cnpj TEXT NOT NULL,
    intermediario_name TEXT NOT NULL,
    service_value INTEGER NOT NULL DEFAULT 0,
    iss_value INTEGER NOT NULL DEFAULT 0,
    irrf_value INTEGER NOT NULL DEFAULT 0,
    inss_value INTEGER NOT NULL DEFAULT 0,
    pis_value INTEGER NOT NULL DEFAULT 0,
    cofins_value INTEGER NOT NULL DEFAULT 0,
    csll_value INTEGER NOT NULL DEFAULT 0,
    total_retentions INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('normal', 'cancelada', 'substituida')),
    layout_version TEXT NOT NULL,
    xml_path TEXT NOT NULL,
    raw_hash TEXT NOT NULL UNIQUE,
    parse_warnings TEXT, 
    nfse_number TEXT NOT NULL,
    service_description TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE company_documents (
    relation_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    document_id TEXT NOT NULL REFERENCES documents(id),
    company_role TEXT NOT NULL CHECK (company_role IN ('tomada', 'prestada', 'intermediario', 'none')),
    visibility_reason TEXT NOT NULL CHECK (visibility_reason IN ('exact_prestador', 'exact_tomador', 'exact_intermediario', 'same_root_only', 'unknown')),
    first_seen_nsu INTEGER,
    last_seen_nsu INTEGER,
    first_synced_at TEXT NOT NULL,
    last_synced_at TEXT NOT NULL,
    viewed_at TEXT,
    UNIQUE(company_id, document_id)
);

CREATE TABLE company_document_export_marks (
    company_id TEXT NOT NULL,
    document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml', 'csv', 'xlsx', 'danfse')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    PRIMARY KEY (company_id, document_id, export_kind)
);

CREATE INDEX idx_company_documents_viewed_at ON company_documents(company_id, viewed_at);
CREATE INDEX idx_export_marks_kind ON company_document_export_marks(company_id, export_kind, exported_at);

CREATE TABLE events (
    id TEXT PRIMARY KEY,
    document_id TEXT REFERENCES documents(id),
    chave_acesso TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('cancelamento', 'substituicao', 'unknown')),
    event_at TEXT,
    replacement_chave_acesso TEXT NOT NULL,
    description TEXT NOT NULL,
    raw_xml_path TEXT NOT NULL,
    raw_hash TEXT NOT NULL UNIQUE,
    parse_warnings TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE sync_runs (
    id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL REFERENCES companies(id),
    credential_id TEXT NOT NULL REFERENCES credentials(id),
    environment TEXT NOT NULL DEFAULT 'producao_restrita',
    credential_cnpj TEXT NOT NULL,
    consultation_cnpj TEXT NOT NULL,
    consultation_basis TEXT NOT NULL CHECK (consultation_basis IN ('exact_certificate_cnpj', 'same_root_certificate')),
    mode TEXT NOT NULL DEFAULT 'normal',
    started_at TEXT NOT NULL,
    finished_at TEXT,
    from_nsu INTEGER NOT NULL,
    to_nsu INTEGER NOT NULL,
    checked_count INTEGER NOT NULL DEFAULT 0,
    documents_found INTEGER NOT NULL DEFAULT 0,
    empty_count INTEGER NOT NULL DEFAULT 0,
    consecutive_empty_count INTEGER NOT NULL DEFAULT 0,
    errors_count INTEGER NOT NULL DEFAULT 0,
    last_found_nsu INTEGER,
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'interrupted')),
    stop_reason TEXT,
    source TEXT NOT NULL DEFAULT 'nfse' CHECK (source IN ('nfse', 'nfe', 'cte'))
);

CREATE UNIQUE INDEX idx_sync_runs_running ON sync_runs(company_id, source) WHERE status = 'running';

CREATE TABLE sync_state (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    environment TEXT NOT NULL,
    consultation_cnpj TEXT NOT NULL,
    last_checked_nsu INTEGER NOT NULL DEFAULT 0,
    last_found_nsu INTEGER,
    max_nsu INTEGER,
    last_empty_streak INTEGER NOT NULL DEFAULT 0,
    last_success_at TEXT,
    last_error_at TEXT,
    last_error_code TEXT,
    last_error_message TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    failed_nsu INTEGER,
    failed_nsu_attempts INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, source, environment, consultation_cnpj)
);

CREATE TABLE company_sync_sources (
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    initial_sync_completed_at TEXT,
    blocked_until TEXT,
    blocked_reason TEXT,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (company_id, source)
);

CREATE TABLE sync_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    company_id TEXT NOT NULL REFERENCES companies(id),
    source TEXT NOT NULL CHECK (source IN ('nfse', 'nfe', 'cte')),
    requested_at TEXT NOT NULL
);

CREATE INDEX idx_sync_requests_window ON sync_requests(company_id, source, requested_at);

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
    updated_at TEXT NOT NULL,
    tp_amb TEXT NOT NULL DEFAULT '' CHECK (tp_amb IN ('', '1', '2'))
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
CREATE INDEX idx_company_nfe_documents_viewed_at ON company_nfe_documents(company_id, viewed_at);
CREATE INDEX idx_company_nfe_documents_manifestacao ON company_nfe_documents(company_id, manifestacao);
CREATE INDEX idx_company_nfe_documents_document ON company_nfe_documents(nfe_document_id);

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
    tp_amb TEXT NOT NULL DEFAULT '' CHECK (tp_amb IN ('', '1', '2')),
    UNIQUE (chave_acesso, tp_evento, n_seq_evento)
);

-- Outbound manifestação attempts (audit and error display). Registered ones
-- also land in nfe_events.
CREATE TABLE nfe_manifestacoes (
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
    created_at TEXT NOT NULL,
    tp_amb TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_nfe_manifestacoes_company_chave ON nfe_manifestacoes(company_id, chave_acesso);

CREATE TABLE company_nfe_export_marks (
    company_id TEXT NOT NULL,
    nfe_document_id TEXT NOT NULL,
    export_kind TEXT NOT NULL CHECK (export_kind IN ('xml')),
    exported_hash TEXT NOT NULL,
    exported_at TEXT NOT NULL,
    PRIMARY KEY (company_id, nfe_document_id, export_kind)
);

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
    masked_keys INTEGER NOT NULL DEFAULT 0,
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
