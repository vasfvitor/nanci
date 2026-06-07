-- name: GetCredential :one
SELECT * FROM credentials WHERE id = ? LIMIT 1;

-- name: ListCredentials :many
SELECT * FROM credentials ORDER BY label ASC;

-- name: CreateCredential :exec
INSERT INTO credentials (
    id, label, cert_path, environment, owner_cnpj, owner_cnpj_root, 
    fingerprint_sha256, subject_name, not_before, not_after, inspected_at,
    created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteCredential :exec
DELETE FROM credentials WHERE id = ?;

-- name: UpdateCredential :exec
UPDATE credentials SET
    label = ?,
    cert_path = ?,
    environment = ?,
    owner_cnpj = ?,
    owner_cnpj_root = ?,
    fingerprint_sha256 = ?,
    subject_name = ?,
    not_before = ?,
    not_after = ?,
    inspected_at = ?,
    updated_at = ?
WHERE id = ?;
