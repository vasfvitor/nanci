-- name: GetCompanyByCNPJ :one
SELECT * FROM companies WHERE cnpj = ? LIMIT 1;

-- name: ListCompanies :many
SELECT * FROM companies ORDER BY name ASC;

-- name: CreateCompany :exec
INSERT INTO companies (
    id, cnpj, cnpj_root, name, environment, last_nsu, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 0, ?, ?);

-- name: AssignCredentialToCompany :exec
UPDATE companies
SET credential_id = ?,
    credential_label = ?,
    credential_cert_path = ?,
    updated_at = ?
WHERE id = ?;

-- name: UpdateCompanyNSU :exec
UPDATE companies
SET last_nsu = ?, updated_at = ?
WHERE id = ?;
