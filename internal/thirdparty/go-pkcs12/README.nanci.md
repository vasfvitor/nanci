# Local go-pkcs12 fork

This directory is based on `software.sslmate.com/src/go-pkcs12` v0.7.1.
The upstream BSD license is preserved in `LICENSE`.

Nanci needs to decode valid PKCS#12 files that use BER indefinite lengths,
including BER inside the MAC-authenticated `AuthenticatedSafe`. Upstream uses
Go's DER-only `encoding/asn1` decoder.

The local change normalizes supported BER forms immediately before each ASN.1
unmarshal. The original authenticated payload remains unchanged until
`verifyMac` validates it. MAC failures are never ignored, and `MacData` is
never removed.

When updating upstream:

1. Replace the upstream source files while retaining `ber.go`, its tests, this
   file, and the `unmarshal` normalization call in `pkcs12.go`.
2. Review upstream changes to `unmarshal`, `getSafeContents`, and `verifyMac`.
3. Run the root certificate tests, the fork tests, and the optional external
   certificate acceptance test documented in `docs/CERTIFICATES.md`.
