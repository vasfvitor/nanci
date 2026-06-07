-- name: CreateSyncRun :exec
INSERT INTO sync_runs (
    id, company_id, credential_id, credential_cnpj, consultation_cnpj,
    consultation_basis, started_at, from_nsu, to_nsu, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateSyncRunProgress :exec
UPDATE sync_runs
SET to_nsu = ?, documents_found = ?, errors_count = ?
WHERE id = ?;

-- name: FinishSyncRun :exec
UPDATE sync_runs
SET finished_at = ?, status = ?
WHERE id = ?;

-- name: UpsertDocument :one
INSERT INTO documents (
    id, chave_acesso, issue_date, competence,
    prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
    intermediario_cnpj, intermediario_name,
    service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value, total_retentions,
    status, layout_version, xml_path, raw_hash, parse_warnings,
    nfse_number, service_description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chave_acesso) DO UPDATE SET
    issue_date = excluded.issue_date,
    competence = excluded.competence,
    prestador_cnpj = excluded.prestador_cnpj,
    prestador_name = excluded.prestador_name,
    tomador_cnpj = excluded.tomador_cnpj,
    tomador_name = excluded.tomador_name,
    intermediario_cnpj = excluded.intermediario_cnpj,
    intermediario_name = excluded.intermediario_name,
    service_value = excluded.service_value,
    iss_value = excluded.iss_value,
    irrf_value = excluded.irrf_value,
    inss_value = excluded.inss_value,
    pis_value = excluded.pis_value,
    cofins_value = excluded.cofins_value,
    csll_value = excluded.csll_value,
    total_retentions = excluded.total_retentions,
    status = excluded.status,
    layout_version = excluded.layout_version,
    xml_path = excluded.xml_path,
    raw_hash = excluded.raw_hash,
    parse_warnings = excluded.parse_warnings,
    nfse_number = excluded.nfse_number,
    service_description = excluded.service_description,
    updated_at = excluded.updated_at
RETURNING id;

-- name: UpsertCompanyDocument :exec
INSERT INTO company_documents (
    relation_id, company_id, document_id, company_role, visibility_reason,
    first_seen_nsu, last_seen_nsu, first_seen_nsu_valid, last_seen_nsu_valid,
    first_synced_at, last_synced_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(company_id, document_id) DO UPDATE SET
    company_role = excluded.company_role,
    visibility_reason = excluded.visibility_reason,
    first_seen_nsu = CASE
        WHEN company_documents.first_seen_nsu_valid = 0 THEN excluded.first_seen_nsu
        WHEN excluded.first_seen_nsu_valid = 0 THEN company_documents.first_seen_nsu
        ELSE MIN(company_documents.first_seen_nsu, excluded.first_seen_nsu)
    END,
    first_seen_nsu_valid = MAX(company_documents.first_seen_nsu_valid, excluded.first_seen_nsu_valid),
    last_seen_nsu = CASE
        WHEN company_documents.last_seen_nsu_valid = 0 THEN excluded.last_seen_nsu
        WHEN excluded.last_seen_nsu_valid = 0 THEN company_documents.last_seen_nsu
        ELSE MAX(company_documents.last_seen_nsu, excluded.last_seen_nsu)
    END,
    last_seen_nsu_valid = MAX(company_documents.last_seen_nsu_valid, excluded.last_seen_nsu_valid),
    last_synced_at = excluded.last_synced_at;

-- name: InsertEvent :exec
INSERT INTO events (
    id, document_id, chave_acesso, type, event_at, event_at_valid,
    replacement_chave_acesso, description, raw_xml_path, raw_hash,
    parse_warnings, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(raw_hash) DO NOTHING;

-- name: GetDocumentIDByAccessKey :one
SELECT id FROM documents WHERE chave_acesso = ? LIMIT 1;

-- name: LinkEventsToDocument :exec
UPDATE events SET document_id = ? WHERE chave_acesso = ? AND document_id IS NULL;

-- name: ListEventTypesByAccessKey :many
SELECT type FROM events WHERE chave_acesso = ?;

-- name: UpdateDocumentStatusByAccessKey :exec
UPDATE documents SET status = ?, updated_at = ? WHERE chave_acesso = ?;
