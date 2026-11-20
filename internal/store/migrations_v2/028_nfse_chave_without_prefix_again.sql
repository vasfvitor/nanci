-- +goose Up
-- Migration 020 rewrote the "NFS" + 50 digits chaves to the 50 digits, but a
-- build older than that parser fix kept writing them on databases that had
-- already run 020: a later sync of the same NSU then failed on
-- UNIQUE(documents.raw_hash). The same rewrite runs again, with the same
-- rules: a collision with another document's digits, and an Id that is not
-- "NFS" + 50 digits, stay as they are.
CREATE TABLE nfse_prefixed_028 AS
SELECT id, substr(chave_acesso, 4) AS chave
FROM documents
WHERE length(chave_acesso) = 53
  AND substr(chave_acesso, 1, 3) = 'NFS'
  AND substr(chave_acesso, 4) NOT GLOB '*[^0-9]*'
  AND NOT EXISTS (SELECT 1 FROM documents o WHERE o.chave_acesso = substr(documents.chave_acesso, 4));

UPDATE documents
SET chave_acesso = (SELECT p.chave FROM nfse_prefixed_028 p WHERE p.id = documents.id),
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id IN (SELECT id FROM nfse_prefixed_028);

-- Link the events of the chave, as storing the document does.
UPDATE events
SET document_id = (SELECT p.id FROM nfse_prefixed_028 p WHERE p.chave = events.chave_acesso)
WHERE document_id IS NULL
  AND chave_acesso IN (SELECT chave FROM nfse_prefixed_028);

-- Recompute the status with the rule of recomputeDocumentStatus: a
-- substituição wins over a cancelamento.
UPDATE documents
SET status = CASE
    WHEN EXISTS (SELECT 1 FROM events e WHERE e.chave_acesso = documents.chave_acesso AND e.type = 'substituicao') THEN 'substituida'
    WHEN EXISTS (SELECT 1 FROM events e WHERE e.chave_acesso = documents.chave_acesso AND e.type = 'cancelamento') THEN 'cancelada'
    ELSE 'normal'
END
WHERE id IN (SELECT id FROM nfse_prefixed_028);

DROP TABLE nfse_prefixed_028;

-- +goose Down
-- Nothing to undo: the 50 digits are the canonical chave, and which rows had
-- the prefix is not kept.
