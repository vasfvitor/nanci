-- name: GetCompanyByCNPJ :one
-- The NFS-e initial sync comes from company_sync_sources; it is NULL for a
-- company that never finished one.
SELECT sqlc.embed(companies), company_sync_sources.initial_sync_completed_at AS nfse_initial_sync_completed_at
FROM companies
LEFT JOIN company_sync_sources
    ON company_sync_sources.company_id = companies.id AND company_sync_sources.source = 'nfse'
WHERE companies.cnpj = ? LIMIT 1;

-- name: ListCompanies :many
-- Same NFS-e initial sync join as GetCompanyByCNPJ.
SELECT sqlc.embed(companies), company_sync_sources.initial_sync_completed_at AS nfse_initial_sync_completed_at
FROM companies
LEFT JOIN company_sync_sources
    ON company_sync_sources.company_id = companies.id AND company_sync_sources.source = 'nfse'
ORDER BY companies.name ASC;

-- name: CreateCompany :exec
INSERT INTO companies (
    id, cnpj, cnpj_root, name, credential_id, credential_label,
    credential_cert_path, environment, sync_start_policy,
    sync_start_date, created_at, updated_at, uf
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: AssignCredentialToCompany :execrows
UPDATE companies
SET credential_id = sqlc.arg(credential_id),
    credential_label = (
        SELECT label FROM credentials
        WHERE credentials.id = sqlc.arg(credential_id)
    ),
    credential_cert_path = (
        SELECT cert_path FROM credentials
        WHERE credentials.id = sqlc.arg(credential_id)
    ),
    updated_at = sqlc.arg(updated_at)
WHERE companies.id = sqlc.arg(company_id)
  AND EXISTS (
      SELECT 1 FROM credentials
      WHERE credentials.id = sqlc.arg(credential_id)
  );

-- name: UpdateCompany :exec
UPDATE companies
SET name = ?,
    environment = ?,
    sync_start_policy = ?,
    sync_start_date = ?,
    uf = ?,
    updated_at = ?
WHERE id = ?;
