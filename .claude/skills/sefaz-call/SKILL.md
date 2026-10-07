---
name: sefaz-call
description: Use when adding a new call to SEFAZ (NF-e/CT-e Ambiente Nacional or a UF service) or to the ADN (NFS-e), or changing an existing one. Covers the shared mTLS httpclient, SOAP 1.2 envelopes and endpoints in internal/sefaz, the ADN REST client, cStat handling and the hourly budget, XMLDSig signing, log redaction, tests with fake servers, the confirmation rule for events, and the SEFAZ rules an implementer must respect.
---

# SEFAZ / ADN call: add a request to a government web service

`internal/sefaz` (SOAP) and `internal/adn` (REST/JSON) only speak the protocol. Budgets, blocking, storage and the meaning of a result for the NSU cursor belong to the caller (`internal/sync`, `internal/app`). Read `docs/NFE_SEFAZ.md`, `docs/CTE_SEFAZ.md` or `docs/NFSE_ADN.md` and `docs/pesquisa/2026-10-07-sefaz-dfe.md` first.

## Shared transport: `internal/foundation/httpclient`

- `httpclient.New(Config)` gives a `*Client`; `Do(ctx, Request)` sends one request. `NewTransport` (`transport.go`) clones `http.DefaultTransport` with TLS 1.2 minimum, `Renegotiation: tls.RenegotiateFreelyAsClient`, `ForceAttemptHTTP2 = false` and `GetClientCertificate` returning the A1. The Ambiente Nacional (IIS) asks for the client certificate by renegotiation after the request reaches a real `.asmx`; HTTP/2 or no renegotiation means a 403. Do not build another transport.
- Retry (`retry.go`): transport failures and 408/425/429/5xx, exponential backoff with jitter, `Retry-After` honored, capped by `RetryConfig.MaxDelay`. `MaxRetries` 0 means no retry.
- Bodies: accepted bodies are read up to `Request.MaxBytes` (default `DefaultMaxBytes`, 20 MiB) and a larger one is `ErrResponseTooLarge`; rejected bodies up to `MaxErrorBodyBytes` (64 KiB), with `MaxErrorLogBodyBytes` (2 KiB) in the log and in `StatusError.Error()`.
- Logs: request at trace, response status/latency at debug, response body only at trace, error responses at error level. `Config.RedactURL` and `Config.RedactBody` mask identifiers before anything is logged or put in an error message; `StatusError.Body` stays raw for classification.
- The caller owns: headers (`Request.Header`), payload encoding, which non-2xx statuses are answers (`Request.Expect`, e.g. 500 for SOAP faults, 404 for an empty ADN queue) and `NewStatusError` when an expected status turns out to be a failure.

## SEFAZ: `internal/sefaz`

- Endpoints: `endpoints.go` holds one constant per service and environment (`DistribuicaoProducao`, `DistribuicaoCTeHomologacao`, `RecepcaoEventoProducao`, ...), the `Endpoints` struct and `EndpointsFor(env)`. A new Ambiente Nacional service adds a constant pair, a field in `Endpoints`, both cases in `EndpointsFor`, and a field in `newFakeClient` (`helpers_test.go`). URLs are only overridable through `ClientConfig.Endpoints` (tests); no flag or env var.
- `tpAmb`: `TpAmb(env)` maps `producao` to `"1"` and `producao_restrita` to `"2"` (homologação). The client stores it at `NewClient`; never take it from the user. On the app side `environmentTpAmb(comp)` (`internal/app/nfe.go`) scopes queries to the company's current environment.
- `cUFAutor`: the IBGE code of the company's UF, from `companyUFCode` in `internal/sync/manager.go` (`uf.Code`), checked 11..53 by `buildDistDFeInt`. A company without UF fails before the password is asked.
- Envelope (`soap.go`): `post(ctx, url, action, body, transportRetries)` wraps the body with `envelope()` (SOAP 1.2, no Header) and sets `Content-Type: application/soap+xml; charset=utf-8; action="..."` (`contentType`); SOAP 1.2 has no `SOAPAction` header. HTTP 500 is expected and parsed as `*FaultError`. Build request XML as compact strings with no whitespace between tags (SEFAZ answers 588), and decode answers with `decodeElement`/`findElements`, which ignore prefixes.
- `NewClient` sets `Retry{MaxRetries: 0}`: every request counts against the hourly limit, so the transport never retries. `post` decides per call: distribution passes 0; `EnviarEventos` passes `eventTransportRetries` (1), safe because a resent lote that already arrived comes back as 573.
- Distribution (`distribuicao.go`, `cte_distribuicao.go`): a `distService` value (`nfeDist`, `cteDist`) names namespace, wsdl, action, versão, wrapper and endpoint; `distribuicao()` builds `distDFeInt` and parses `retDistDFeInt`. cStat 137 (`CStatNenhumDocumento`), 138 (`CStatDocumentoLocalizado`) and 656 (`CStatConsumoIndevido`) are results; any other cStat is a `*RejectionError` (e.g. 593, CNPJ-Base differs from the certificate). `distBatch` in `internal/sync/dist.go` turns them into stop rules: 137 or `ultNSU >= maxNSU` is caught up and 656 is blocked, both for 1 hour; the next cursor is `max(cursor, ultNSU)`.
- Events (`evento.go`): `EnviarEventos(ctx, signer, idLote, eventos)` signs each `Evento`, sends one `envEvento` (up to `MaxEventosPorLote` = 20) to `NFeRecepcaoEvento4` with `cOrgao` 91 (`COrgaoAN`), and matches each `retEvento` by chave and tpEvento. Lote cStat 128 is processed; per evento, `IsRegistered` (135/136) and `IsAlreadyDone` (573); everything else, including 655, is a rejection. `ProcEventoNFe` builds the stored `procEventoNFe`.
- Signing (`xmldsig.go`): `NewSigner(tls.Certificate)` needs an RSA key and parsed leaf; `SignEvento` writes `infEvento` already in C14N 1.0 form (`buildInfEvento`, `escapeText`, `checkEventoText`) and signs with SHA-1/RSA-SHA1, the NF-e profile. No C14N library. A different event schema (e.g. `eventoCTe` 4.00) means generalizing `buildInfEvento`/`SignEvento`, not copying them.
- TLS check (`tls_check.go`): `CheckTLS`/`CheckTLSCTe` handshake and close without a request; they spend no budget and cannot prove SEFAZ accepts the certificate.

### Existing methods to extend, not duplicate

`ConsNSU` and `ConsChNFe` (`distribuicao.go`) and `ConsCTeNSU` (`cte_distribuicao.go`) exist with no caller, kept for ROADMAP Fase 2 (fetch by key, NSU gap recovery). Wire them; do not write a second `distDFeInt` builder. `ConsChNFe` answers 632/640/641/653/654 come back as `*RejectionError`; give them meaning in the caller. Add the method to the narrow interface the caller uses (`sefazClient` in `internal/app/nfe.go`, `nfeFetcher` in `internal/sync/source_nfe.go`) so tests can fake it.

### UF services (planned 610110)

`docs/CTE_SEFAZ.md`, "Próximos passos": the prestação do serviço em desacordo goes to the SEFAZ autorizadora of the CT-e (`CTeRecepcaoEventoV4`), not the Ambiente Nacional, routed by the UF in the key (SVRS for most, own endpoints for SP, MT, MS, MG, PR). A UF → URL table per environment belongs in `internal/sefaz`, next to `endpoints.go` and keyed by IBGE code from `internal/foundation/uf`; `Endpoints`/`EndpointsFor` stay for Ambiente Nacional services. It needs an audit table like `nfe_manifestacoes`. Design it when the work starts, with sources.

## ADN: `internal/adn`

- REST/JSON over the same `httpclient`, `LogLabel: "ADN API"`, `RedactURL: sanitizeURL` (masks `cnpjConsulta`), default 3 retries (ADN has no hourly budget; `requestsPerHour` returns 0 for NFS-e).
- Base URLs `BaseURLProduction` / `BaseURLRestrictedProduction` in `client.go`, chosen by `sync.ResolveEnvironmentURL(env)`.
- `request()` sends JSON headers and `Expect`s 404; `notFound` turns the `NENHUM_DOCUMENTO_LOCALIZADO`/`E2220` envelope into `ErrNoDocumentsLocated` and anything else into a `StatusError`. `FetchDocuments` (`GET DFe/{NSU}?cnpjConsulta=`) returns `LoteDFe` items whose `ArquivoXml` is base64 of gzip; decode with `gzipxml.Decode` and its size `Limits`, as SEFAZ `docZip`s are. `RawGet` backs the direct query (`NFSe/{chave}/Eventos` in `internal/app/query.go`). Add a typed method per new route; keep `cnpjConsulta` as the query parameter name.

## Redaction

`internal/foundation/redact`: `MaskIdentifier` keeps the first and last two characters; `MaskXMLIdentifiers` masks CNPJ, CPF, IE, chNFe, chCTe, chNFSe, chave, xNome, email and similar elements. SEFAZ sets `RedactBody: redact.MaskXMLIdentifiers`; a new identifier in a URL needs a `RedactURL`, and a new XML element carrying one needs adding to `identifierElement`. Never log a raw request body or the certificate.

## SEFAZ rules to respect (`docs/pesquisa/2026-10-07-sefaz-dfe.md`)

- Reuse the returned `ultNSU`; all software consulting the same CNPJ shares one ascending sequence. Out of sequence gets 656, and querying while blocked restarts the hour.
- Wait 1 hour after 137 or `ultNSU == maxNSU`. 656 blocks 1 hour and, since NT v1.14, returns the server's `ultNSU`.
- `consNSU` and `consChNFe` together: 20 per hour. nanci budgets every distribution request (distNSU too) per company, source and environment in `sync_requests`: `sync.SyncService.spendRequest` (`internal/sync/loop.go`) records before sending, so failures count, and blocks with `rate_budget` when spent. A call outside the loop must use the same `sync.Store` (`RequestsSince`, `RecordRequest`, `SetBlockedUntil`) with the same source and respect `company_sync_sources.blocked_until`. CT-e has its own budget by assumption.
- 60-day keep-alive: the AN only generates NSU for a CNPJ root that called distNSU in the last 60 days (third-party reports); nanci warns at 45 (`sync.DistIdleWarningDays`). Only a successful distNSU resets it.
- 90-day window (CT-e: 3 months) for distribution and consChNFe; `ultNSU` 0 returns only the last 3 months.
- The emitente gets 641 from consChNFe and never receives its own NF-e by distribution.
- CNPJ fields are alphanumeric (NT 2014.002 v1.40, NT 2025.001): validate with `cnpj.Validate` and send `cnpj.Clean` (uppercase, letters kept); keys through `dfe.ParseAccessKey`. Never strip letters or parse a CNPJ as a number.

## Events need explicit confirmation

Anything that sends an event (manifestação, 610110, any future evento) or otherwise changes state at SEFAZ is irreversible. It needs a plan/dry-run path that asks no password and a confirmed path: `--confirmar` in the CLI (`internal/cli/nfe_manifest.go`) and a confirmation dialog in the desktop (`NFeCienciaConfirmDialog.vue`, mandatory acknowledgement). Record every attempt, failures included, in an audit table (`nfe_manifestacoes` via `NFeRepository.RecordManifestacoes`, written by `eventSender.send` in `internal/app/nfe_manifest.go`), and keep it out of local resets (`nfe reset` keeps it).

## Tests

- SEFAZ: `newFakeClient(t, status, body, cfg)` in `helpers_test.go` starts an `httptest` `fakeSEFAZ` that records requests; assert the exact request bytes and headers (`TestDistribuicao_RequestBytes`), that invalid input sends nothing (`TestDistribuicao_InvalidInputSendsNothing`), every cStat class, and SOAP faults. `mustDocZip` builds docZips from fixtures.
- Fixtures: `internal/sefaz/testdata` (fictitious `retdist-cte-*.xml`, golden `evento-ciencia-signed.xml`, regenerate with `go test ./internal/sefaz -run TestSignEvento_Golden -update`), document fixtures in `internal/nfe/testdata` and `internal/cte/testdata`. Mock A1: `internal/foundation/cert/testdata/cert_a1_mock_70860312000150.pfx`, password `mockdata` (`newMockSigner`). Never commit a real XML or certificate.
- Signatures: `TestInfEventoDigest_RealEvent` checks the digest against an accepted real event; `xmlsec_test.go` (build tag `xmlsec`, `go test -tags xmlsec ./internal/sefaz/`) verifies with `xmlsec1`.
- ADN: `httptest` servers in `client_test.go`, `endpoints_test.go`, `logging_test.go`; assert identifiers never reach logs or error text (`TestClient_ErrorLogsAreBoundedAndMasked`).
- App/sync: swap `newSEFAZClient` (`internal/app/nfe_manifest_test.go`) or a fetcher fake (`fakeCTeDistribuicao` in `internal/app/cte_test.go`); never hit the network in tests.

## Checklist

- [ ] Endpoint constants per environment, `Endpoints`/`EndpointsFor` or a UF table, and the test fake updated.
- [ ] Request built compact, CNPJ via `cnpj.Validate`/`Clean`, key via `dfe.ParseAccessKey`, `tpAmb` from the client.
- [ ] Every cStat the service can return classified; unknown ones are `*RejectionError`.
- [ ] No transport retry for anything that spends the hourly budget; budget recorded before sending.
- [ ] Identifiers masked in URLs, bodies and errors.
- [ ] Event sending behind CLI `--confirmar` and a desktop confirmation; every attempt audited.
- [ ] Fake-server tests for request bytes, invalid input, each cStat and faults.
- [ ] `docs/<SOURCE>_SEFAZ.md` or `docs/NFSE_ADN.md` updated, each external fact with its source URL and date.
