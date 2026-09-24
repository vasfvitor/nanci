-- +goose Up
-- The tpAmb (1 produção, 2 homologação) of each NF-e and event, so a
-- company can switch environments without mixing their notes. Until now a
-- company could not change its environment after an NF-e sync, so the rows a
-- company sees take its environment. Rows no company sees stay empty.
ALTER TABLE nfe_documents ADD COLUMN tp_amb TEXT NOT NULL DEFAULT '' CHECK (tp_amb IN ('', '1', '2'));
ALTER TABLE nfe_events ADD COLUMN tp_amb TEXT NOT NULL DEFAULT '' CHECK (tp_amb IN ('', '1', '2'));

UPDATE nfe_documents
SET tp_amb = COALESCE((
    SELECT CASE c.environment WHEN 'producao' THEN '1' WHEN 'producao_restrita' THEN '2' END
    FROM company_nfe_documents cd
    INNER JOIN companies c ON c.id = cd.company_id
    WHERE cd.nfe_document_id = nfe_documents.id AND c.environment IN ('producao', 'producao_restrita')
    ORDER BY c.environment -- producao first, should two companies disagree
    LIMIT 1
), '');

UPDATE nfe_events
SET tp_amb = COALESCE((
    SELECT d.tp_amb FROM nfe_documents d WHERE d.chave_acesso = nfe_events.chave_acesso
), '');

-- +goose Down
ALTER TABLE nfe_events DROP COLUMN tp_amb;
ALTER TABLE nfe_documents DROP COLUMN tp_amb;
