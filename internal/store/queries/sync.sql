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

-- name: UpsertDocument :exec
INSERT INTO documents (
    id, chave_acesso, issue_date, competence,
    prestador_cnpj, prestador_name, tomador_cnpj, tomador_name,
    intermediario_cnpj, intermediario_name,
    service_value, iss_value, irrf_value, inss_value, pis_value, cofins_value, csll_value, total_retentions,
    status, layout_version, xml_path, raw_hash, parse_warnings,
    nfse_number, service_description, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chave_acesso) DO UPDATE SET
    status = excluded.status,
    updated_at = excluded.updated_at;

-- name: UpsertCompanyDocument :exec
INSERT INTO company_documents (
    relation_id, company_id, document_id, company_role, visibility_reason,
    first_seen_nsu, last_seen_nsu, first_seen_nsu_valid, last_seen_nsu_valid,
    first_synced_at, last_synced_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(company_id, document_id) DO UPDATE SET
    last_seen_nsu = excluded.last_seen_nsu,
    last_seen_nsu_valid = excluded.last_seen_nsu_valid,
    last_synced_at = excluded.last_synced_at;

-- name: InsertEvent :exec
INSERT INTO events (
    id, document_id, chave_acesso, type, event_at, event_at_valid,
    replacement_chave_acesso, description, raw_xml_path, raw_hash,
    parse_warnings, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(raw_hash) DO NOTHING;

-- name: ListCompanyDocuments :many
SELECT d.*, cd.company_role, cd.visibility_reason 
FROM documents d
JOIN company_documents cd ON d.id = cd.document_id
WHERE cd.company_id = ?
ORDER BY d.issue_date DESC;
