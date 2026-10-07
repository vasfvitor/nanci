# Local go-pkcs12 fork

This directory is based on `software.sslmate.com/src/go-pkcs12` v0.7.1 plus
the v0.7.2 and v0.7.3 security backports listed below.
The upstream BSD license is preserved in `LICENSE`.

## Security backports from v0.7.2 and v0.7.3

The local `replace` hides this module from `govulncheck`, so upstream security
fixes must be ported by hand. Ported so far, from
https://github.com/SSLMate/go-pkcs12:

- v0.7.2 (GO-2026-5052): `03c441f` "[Security fix] Reject PBMAC1 keys that
  are too short", `be1f487` "Improve the too-short PBMAC1 key error message"
  and the tests from `8d284f9` "Add tests for rejecting short PBMAC1 keys".
  `doMac` rejects a PBMAC1 `KeyLength` below 20 octets, which otherwise
  allows MAC forgery (same class as OpenSSL CVE-2026-34181).
- v0.7.3: `c0472ed` "Reject files with invalid IV lengths or
  excessively-long key lengths". `pbDecrypterFor` rejects a PBES2 IV whose
  length is not the AES block size (it used to panic in
  `cipher.NewCBCDecrypter`), and `doMac` rejects a PBMAC1 `KeyLength` above
  64 octets.

`mac.go` and `crypto.go` match v0.7.3 except for the fork's
`Data() ([]byte, error)` change, which lets `encryptedContentInfo` return the
constructed (BER) encrypted content it reassembles. `mac_test.go`, `crypto_test.go` and
`testdata/pbmac1-short-key-bypass.txt` carry only the upstream tests for these
fixes; `testDecryptable` follows the fork's `Data() ([]byte, error)` signature.

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
   the `Data() ([]byte, error)` change, and the `DecodeChain`/`decodeChain`
   split. Once the new base includes v0.7.3, drop the backport section above.
2. Review upstream changes to `unmarshal`, `getSafeContents`, `verifyMac`,
   `DecodeChain`, and `bmpString`. If `bmpString` changes, mirror the change in
   `bmpStringBytes` — `password-bytes_test.go` asserts the two agree.
3. Run the root certificate tests, the fork tests, and the optional external
   certificate acceptance test.

Because `govulncheck` cannot see this fork, check upstream releases and the
Go vulnerability database for `software.sslmate.com/src/go-pkcs12` by hand.
