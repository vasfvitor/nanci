-- name: GetCompanyByCNPJ :one
SELECT * FROM companies WHERE cnpj = ? LIMIT 1;

-- name: ListCompanies :many
SELECT * FROM companies ORDER BY name ASC;

-- name: CreateCompany :exec
INSERT INTO companies (
    id, cnpj, cnpj_root, name, credential_id, credential_label,
    credential_cert_path, environment, last_nsu, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?);

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
    environment = (
        SELECT environment FROM credentials
        WHERE credentials.id = sqlc.arg(credential_id)
    ),
    updated_at = sqlc.arg(updated_at)
WHERE companies.id = sqlc.arg(company_id)
  AND EXISTS (
      SELECT 1 FROM credentials
      WHERE credentials.id = sqlc.arg(credential_id)
  );

-- name: UpdateCompanyNSU :exec
UPDATE companies
SET last_nsu = ?, updated_at = ?
WHERE id = ?;
