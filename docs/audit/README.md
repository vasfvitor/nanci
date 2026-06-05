# Audit Scopes

This directory breaks the audit into independent scopes so each issue can be planned, assigned, and verified separately.

## Scopes

- `01-document-identity-and-multi-company-storage.md`
- `02-events-schema-and-lifecycle.md`
- `03-nsu-checkpoint-advancement.md`
- `04-same-root-certificate-and-consulta-cnpj.md`
- `05-role-detection-by-establishment-vs-root.md`
- `06-parser-schema-fixtures-and-domain-coverage.md`
- `07-cnpj-validation-and-rollout-timing.md`

## Shared context

The project is trying to solve the bulk NFS-e XML download problem without portal scraping, CAPTCHA bypass, or browser automation. The intended path is the official ADN Contribuintes API, local XML storage, SQLite indexing, and safe NSU checkpointing.

That means the main audit standard is not just "code works", but:

- ADN API usage must match the published contributor flow.
- The data model must fit accounting-office reality, where one note can matter to more than one managed company.
- Checkpointing must be safe under retries, empty batches, and interruptions.
- Retention reports must reflect establishment-level and event-level truth.

## External references

- Manual dos Contribuintes - Guia para utilizacao das APIs do ADN, versao 1.0, 12/02/2026:
  https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/documentacao-atual/manual-contribuintes-apis-adn-sistema-nacional-nfse.pdf
- Documentacao Atual da NFS-e, atualizada em 17/04/2026:
  https://www.gov.br/nfse/pt-br/biblioteca/documentacao-tecnica/documentacao-atual
- CNPJ Alfanumerico - Receita Federal:
  https://www.gov.br/receitafederal/pt-br/acesso-a-informacao/acoes-e-programas/programas-e-atividades/cnpj-alfanumerico
