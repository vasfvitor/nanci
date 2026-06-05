# Storage Model And Reset Policy

This document records the current local storage contract after the document-identity refactor.

It is intentionally explicit because the schema changed in a breaking way and later scopes will build on top of it.

## Current model

The local database now distinguishes between:

- canonical document identity
- company-specific participation in that document

### `documents`

One row per `chave_acesso`.

This table stores canonical XML-derived fields only, such as:

- access key
- issue date
- competence
- prestador, tomador, and intermediary identifiers
- service and withholding values
- canonical status
- raw hash
- canonical XML file path

### `company_documents`

One row per `company_id + document_id`.

This table stores company-facing semantics, such as:

- `company_role`
- `visibility_reason`
- first and last NSU seen for that company
- first and last sync timestamps for that company

## Semantics

The important contract is:

- one NFS-e can belong to multiple managed companies locally
- each managed company keeps its own role and visibility basis
- company-facing reads must go through `company_documents`
- canonical document identity must not be overwritten by another company's sync lineage

### Role vs visibility

`company_role` is the internal source of truth for company-facing filtering and export grouping.

Allowed values today:

- `prestada`
- `tomada`
- `intermediario`
- `none`

`visibility_reason` explains why the note is visible from the queried company's perspective.

Allowed values today:

- `exact_prestador`
- `exact_tomador`
- `exact_intermediario`
- `same_root_only`
- `unknown`

Same-root visibility is stored as provenance only. It must not silently imply fiscal role.

## File storage

XML storage is currently canonical by document for this phase:

- one XML file per `chave_acesso`
- path stored on `documents.xml_path`

ZIP exports remain company-scoped at read time because they are driven by `company_documents`, not because files are duplicated on disk.

## Reset policy

This refactor is currently reset-oriented.

That means:

- fresh databases are supported
- new installs are supported
- existing SQLite files created before this schema split are not migrated in place yet

If you have a database created before the canonical `documents` plus `company_documents` split, discard it and let the app recreate a fresh one.

## What is intentionally deferred

The following are not closed by this phase:

- in-place migration for pre-refactor SQLite databases
- final event-ledger design
- final same-root credential and consultation-CNPJ model
- parser hardening against the full official fixture matrix

Those are handled by later audit groups.
