// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Upstream tests from v0.7.3 for the PBES2 IV length check. The rest of
// upstream crypto_test.go is not carried. testDecryptable and the
// EncryptedContent assignment follow the fork's Data() ([]byte, error) and
// asn1.RawValue EncryptedContent. See README.nanci.md.

package pkcs12

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"testing"
)

type testDecryptable struct {
	data      []byte
	algorithm pkix.AlgorithmIdentifier
}

func (d testDecryptable) Algorithm() pkix.AlgorithmIdentifier { return d.algorithm }
func (d testDecryptable) Data() ([]byte, error)               { return d.data, nil }

// pbes2AlgorithmWithIV builds a PBES2 (PBKDF2-HMAC-SHA256 + AES-256-CBC)
// AlgorithmIdentifier whose encryption-scheme IV is ivLen bytes long.
func pbes2AlgorithmWithIV(t *testing.T, ivLen int) pkix.AlgorithmIdentifier {
	t.Helper()

	var kdfparams pbkdf2Params
	saltBytes, err := asn1.Marshal([]byte("salt-salt"))
	if err != nil {
		t.Fatal(err)
	}
	kdfparams.Salt.FullBytes = saltBytes
	kdfparams.Iterations = 2048
	kdfparams.Prf.Algorithm = oidHmacWithSHA256

	var params pbes2Params
	params.Kdf.Algorithm = oidPBKDF2
	if params.Kdf.Parameters.FullBytes, err = asn1.Marshal(kdfparams); err != nil {
		t.Fatal(err)
	}
	params.EncryptionScheme.Algorithm = oidAES256CBC
	if params.EncryptionScheme.Parameters.FullBytes, err = asn1.Marshal(make([]byte, ivLen)); err != nil {
		t.Fatal(err)
	}

	alg := pkix.AlgorithmIdentifier{Algorithm: oidPBES2}
	if alg.Parameters.FullBytes, err = asn1.Marshal(params); err != nil {
		t.Fatal(err)
	}
	return alg
}

// A PBES2 encryption scheme can carry an IV whose length doesn't match the AES block size; make sure it's rejected.
func TestPbDecryptBadPBES2IVLength(t *testing.T) {
	password, _ := bmpStringZeroTerminated("")

	for _, ivLen := range []int{0, 4, 8, 15, 17, 32} {
		td := testDecryptable{
			data:      make([]byte, 16),
			algorithm: pbes2AlgorithmWithIV(t, ivLen),
		}
		_, err := pbDecrypt(&td, password)
		if err == nil {
			t.Errorf("ivLen=%d: expected an error, got nil", ivLen)
		}
	}

	// Sanity check: the correct (16-byte) IV length is still accepted by
	// pbDecrypterFor (it fails later, on padding, not on the IV).
	if _, _, err := pbDecrypterFor(pbes2AlgorithmWithIV(t, 16), password); err != nil {
		t.Errorf("ivLen=16: unexpected error %v", err)
	}
}

// Ensure PKCS#12 file with invalid IV length is rejected with an error.
func TestDecodeTrustStoreBadPBES2IVLength(t *testing.T) {
	var ed encryptedData
	ed.Version = 0
	ed.EncryptedContentInfo.ContentType = oidDataContentType
	ed.EncryptedContentInfo.ContentEncryptionAlgorithm = pbes2AlgorithmWithIV(t, 4)
	ed.EncryptedContentInfo.SetData(make([]byte, 16))

	var ci contentInfo
	ci.ContentType = oidEncryptedDataContentType
	ci.Content.Class = 2
	ci.Content.Tag = 0
	ci.Content.IsCompound = true
	ci.Content.Bytes, _ = asn1.Marshal(ed)

	authenticatedSafeBytes, _ := asn1.Marshal([]contentInfo{ci})

	var pfx pfxPdu
	pfx.Version = 3
	pfx.AuthSafe.ContentType = oidDataContentType
	pfx.AuthSafe.Content.Class = 2
	pfx.AuthSafe.Content.Tag = 0
	pfx.AuthSafe.Content.IsCompound = true
	pfx.AuthSafe.Content.Bytes, _ = asn1.Marshal(authenticatedSafeBytes)

	pfxData, err := asn1.Marshal(pfx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodeTrustStore(pfxData, ""); err == nil {
		t.Error("expected an error decoding a file with a malformed PBES2 IV, got nil")
	}
}
