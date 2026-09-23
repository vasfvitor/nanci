-- name: GetNFeDocumentByChave :one
SELECT * FROM nfe_documents WHERE chave_acesso = ? LIMIT 1;

-- name: UpsertNFeDocument :one
INSERT INTO nfe_documents (
    id, chave_acesso, modelo, serie, numero, issue_date, competence, authorized_at, protocolo,
    emitente_cnpj, emitente_name, emitente_ie, emitente_uf,
    destinatario_cnpj, destinatario_name, transportador_cnpj, autorizados_cnpj,
    tp_nf, fin_nfe, nat_op, total_value, icms_value, ipi_value,
    situacao, completeness, layout_version, raw_hash, resumo_raw_hash, parse_warnings,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chave_acesso) DO UPDATE SET
    modelo = excluded.modelo,
    serie = excluded.serie,
    numero = excluded.numero,
    issue_date = excluded.issue_date,
    competence = excluded.competence,
    authorized_at = excluded.authorized_at,
    protocolo = excluded.protocolo,
    emitente_cnpj = excluded.emitente_cnpj,
    emitente_name = excluded.emitente_name,
    emitente_ie = excluded.emitente_ie,
    emitente_uf = excluded.emitente_uf,
    destinatario_cnpj = excluded.destinatario_cnpj,
    destinatario_name = excluded.destinatario_name,
    transportador_cnpj = excluded.transportador_cnpj,
    autorizados_cnpj = excluded.autorizados_cnpj,
    tp_nf = excluded.tp_nf,
    fin_nfe = excluded.fin_nfe,
    nat_op = excluded.nat_op,
    total_value = excluded.total_value,
    icms_value = excluded.icms_value,
    ipi_value = excluded.ipi_value,
    situacao = excluded.situacao,
    completeness = excluded.completeness,
    layout_version = excluded.layout_version,
    raw_hash = excluded.raw_hash,
    resumo_raw_hash = excluded.resumo_raw_hash,
    parse_warnings = excluded.parse_warnings,
    updated_at = excluded.updated_at
RETURNING id;

-- name: UpdateNFeSituacao :exec
UPDATE nfe_documents SET situacao = ?, updated_at = ? WHERE chave_acesso = ?;

-- name: CompanyNFeDocumentExists :one
SELECT COUNT(*) FROM company_nfe_documents cd
INNER JOIN nfe_documents d ON d.id = cd.nfe_document_id
WHERE cd.company_id = ? AND d.chave_acesso = ?;

-- name: UpsertCompanyNFeDocument :exec
INSERT INTO company_nfe_documents (
    relation_id, company_id, nfe_document_id, company_role, visibility_reason,
    first_seen_nsu, last_seen_nsu, first_synced_at, last_synced_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(company_id, nfe_document_id) DO UPDATE SET
    company_role = excluded.company_role,
    visibility_reason = excluded.visibility_reason,
    first_seen_nsu = CASE
        WHEN company_nfe_documents.first_seen_nsu IS NULL THEN excluded.first_seen_nsu
        WHEN excluded.first_seen_nsu IS NULL THEN company_nfe_documents.first_seen_nsu
        ELSE MIN(company_nfe_documents.first_seen_nsu, excluded.first_seen_nsu)
    END,
    last_seen_nsu = CASE
        WHEN company_nfe_documents.last_seen_nsu IS NULL THEN excluded.last_seen_nsu
        WHEN excluded.last_seen_nsu IS NULL THEN company_nfe_documents.last_seen_nsu
        ELSE MAX(company_nfe_documents.last_seen_nsu, excluded.last_seen_nsu)
    END,
    last_synced_at = excluded.last_synced_at;

-- name: ListCompanyNFeRelationsByChave :many
SELECT cd.relation_id, c.cnpj FROM company_nfe_documents cd
INNER JOIN nfe_documents d ON d.id = cd.nfe_document_id
INNER JOIN companies c ON c.id = cd.company_id
WHERE d.chave_acesso = ?;

-- name: UpdateCompanyNFeManifestacao :exec
UPDATE company_nfe_documents SET manifestacao = ?, manifestacao_at = ? WHERE relation_id = ?;

-- name: NFeEventExists :one
SELECT COUNT(*) FROM nfe_events WHERE chave_acesso = ? AND tp_evento = ? AND n_seq_evento = ?;

-- name: UpsertNFeEvent :exec
-- Keeps the row id, created_at, the document link and sent_by_nanci, and
-- never lets a resumo overwrite a completa.
INSERT INTO nfe_events (
    id, nfe_document_id, chave_acesso, tp_evento, type, n_seq_evento, event_at, registered_at,
    registered, c_stat, x_motivo, protocolo, autor_cnpj, description, justificativa, correcao,
    completeness, sent_by_nanci, raw_hash, parse_warnings, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chave_acesso, tp_evento, n_seq_evento) DO UPDATE SET
    nfe_document_id = COALESCE(nfe_events.nfe_document_id, excluded.nfe_document_id),
    type = excluded.type,
    event_at = excluded.event_at,
    registered_at = excluded.registered_at,
    registered = excluded.registered,
    c_stat = excluded.c_stat,
    x_motivo = excluded.x_motivo,
    protocolo = excluded.protocolo,
    autor_cnpj = excluded.autor_cnpj,
    description = excluded.description,
    justificativa = excluded.justificativa,
    correcao = excluded.correcao,
    completeness = excluded.completeness,
    sent_by_nanci = MAX(nfe_events.sent_by_nanci, excluded.sent_by_nanci),
    raw_hash = excluded.raw_hash,
    parse_warnings = excluded.parse_warnings,
    updated_at = excluded.updated_at
WHERE NOT (nfe_events.completeness = 'completa' AND excluded.completeness = 'resumo');

-- name: LinkNFeEventsToDocument :exec
UPDATE nfe_events SET nfe_document_id = ? WHERE chave_acesso = ? AND nfe_document_id IS NULL;

-- name: ListNFeEventsByChave :many
SELECT * FROM nfe_events
WHERE chave_acesso = ?
ORDER BY COALESCE(registered_at, event_at, created_at), tp_evento, n_seq_evento;

-- name: InsertNFeManifestation :exec
INSERT INTO nfe_manifestations (
    id, company_id, chave_acesso, tp_evento, n_seq_evento, justificativa, id_lote,
    status, c_stat, x_motivo, protocolo, registered_at, request_raw_hash, response_raw_hash, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: MarkNFeExported :exec
INSERT INTO company_nfe_export_marks (company_id, nfe_document_id, export_kind, exported_hash, exported_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(company_id, nfe_document_id, export_kind) DO UPDATE SET
    exported_hash = excluded.exported_hash,
    exported_at = excluded.exported_at;
