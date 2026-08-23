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

## Byte-slice password entry point

Nanci clears certificate passwords from memory after use (issue #10). Go
strings are immutable and cannot be zeroed, so the password must never be
converted to a `string`. Upstream only exposes string passwords.

`password-bytes.go` adds `DecodeChainBytes(pfxData, password []byte)`, the
`[]byte` counterpart of `DecodeChain`. It encodes the password with
`bmpStringBytesZeroTerminated`, the `[]byte` counterpart of
`bmpStringZeroTerminated`, zeroes that intermediate BMP buffer before
returning, and never modifies the caller's password slice — zeroing the
password itself stays the caller's job.

The only change this requires in upstream files is in `pkcs12.go`: the body of
`DecodeChain` was moved verbatim into an unexported `decodeChain(pfxData,
encodedPassword []byte)`, which both entry points call once the password is
BMP-encoded. `DecodeChain` keeps its upstream signature and behaviour.

Key derivation inside the package (`pbkdf`, `pbDecrypt`) still leaves derived
keys and IVs in memory; only the password path is covered.

When updating upstream:

1. Replace the upstream source files while retaining `ber.go`, `password-bytes.go`,
   their tests, this file, the `unmarshal` normalization call in `pkcs12.go`,
   and the `DecodeChain`/`decodeChain` split.
2. Review upstream changes to `unmarshal`, `getSafeContents`, `verifyMac`,
   `DecodeChain`, and `bmpString`. If `bmpString` changes, mirror the change in
   `bmpStringBytes` — `password-bytes_test.go` asserts the two agree.
3. Run the root certificate tests, the fork tests, and the optional external
   certificate acceptance test.
