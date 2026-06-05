# Audit Next Steps And Proposed Grouping

This file now acts as a status-oriented execution map for the audit scopes.

Most of the original grouping has already been implemented in code. At this point, the document should answer two questions clearly:

1. what is already closed
2. what remains as the natural next implementation wave

## Current status

Implemented scopes:

- `01-document-identity-and-multi-company-storage`
- `02-events-schema-and-lifecycle`
- `03-nsu-checkpoint-advancement`
- `04-same-root-certificate-and-consulta-cnpj`
- `05-role-detection-by-establishment-vs-root`
- `07-cnpj-validation-and-rollout-timing`

Still open:

- `06-parser-schema-fixtures-and-domain-coverage`

## What is already closed

The codebase now includes:

- canonical `documents` plus `company_documents`
- company-scoped `company_role` and `visibility_reason`
- company-facing list/export/UI behavior aligned to the relation model
- authoritative NSU checkpointing based on ADN response watermarks
- canonical event ledger persistence plus status materialization on `documents.status`
- same-root credential reuse with explicit consultation CNPJ and sync audit trail
- numeric CNPJ DV validation with alphanumeric CNPJ explicitly blocked for now

That means the prior sequencing concern of "01/05 first, then 03, then 02, then 04, then 07" is no longer the active planning problem.

## Remaining active group

## Group 3: Parser Hardening And Evidence Coverage

Scope grouped here:

- `06-parser-schema-fixtures-and-domain-coverage`

Why this is now the natural next step:

- it is the only audit scope still open
- it validates the correctness claims made by the storage, sync, event, and same-root work already landed
- it is the main remaining gap between "implemented architecture" and "trusted payload coverage"

Natural next steps:

- check in a development reference set for official XSD/layout artifacts or document the exact acquisition flow
- build a fixture matrix for:
  - normal documents
  - cancellations
  - substitutions
  - namespace variants
  - missing optional fields
  - intermediary participation
  - same-root visibility without exact establishment participation
- tighten parser behavior so unsupported payloads fail explicitly and observably
- verify parser behavior against the event-ledger and same-root semantics already implemented
- document parser coverage and known unsupported cases
- review README claims so user-facing language matches actual parser coverage

Exit condition:

- parser claims match actual tested coverage
- production-like payload variance is no longer an unbounded risk

## Suggested issue breakdown

If this is turned into the next implementation ticket, the natural ticket is:

- Ticket E: parser fixture suite and schema-grounded coverage

Optional supporting sub-slices inside that ticket:

- fixture acquisition and storage policy
- document parser fixture matrix
- event parser fixture matrix
- unsupported-payload behavior and observability
- README / operator-doc claim alignment

## What I would do next in code

If continuing immediately from the current codebase, I would start with Scope `06` and keep it focused on:

1. official-schema and fixture acquisition policy
2. fixture-driven parser coverage for documents and events
3. documentation alignment so parser claims match tested reality
