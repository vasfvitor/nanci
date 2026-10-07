// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Upstream tests from v0.7.2 and v0.7.3 for the PBMAC1 key length checks.
// The rest of upstream mac_test.go is not carried. See README.nanci.md.

package pkcs12

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func loadTestData(t *testing.T, filename string) []byte {
	base64data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to load test data: %v", err)
	}
	rawData, err := base64.StdEncoding.DecodeString(string(base64data))
	if err != nil {
		t.Fatalf("failed to decode test data %q: %v", filename, err)
	}
	return rawData
}

// pbmac1MacData builds a PBMAC1 macData whose PBKDF2 parameters request the
// given derived key length.
func pbmac1MacData(t *testing.T, keyLength int) *macData {
	t.Helper()
	kdfParams := pbkdf2Params{
		Salt:       asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		Iterations: 1000,
		KeyLength:  keyLength,
		Prf:        pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
	}
	kdfParamsBytes, err := asn1.Marshal(kdfParams)
	if err != nil {
		t.Fatalf("Failed to marshal KDF params: %v", err)
	}
	params := pbmac1Params{
		Kdf:    pkix.AlgorithmIdentifier{Algorithm: oidPBKDF2, Parameters: asn1.RawValue{FullBytes: kdfParamsBytes}},
		MacAlg: pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
	}
	paramsBytes, err := asn1.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal PBMAC1 params: %v", err)
	}
	return &macData{
		Mac: digestInfo{
			Algorithm: pkix.AlgorithmIdentifier{
				Algorithm:  oidPBMAC1,
				Parameters: asn1.RawValue{FullBytes: paramsBytes},
			},
		},
	}
}

func TestPBMAC1RejectsShortKeyLength(t *testing.T) {
	message := []byte{11, 12, 13, 14, 15}
	password, err := bmpStringZeroTerminated("test-password")
	if err != nil {
		t.Fatalf("Failed to encode password to BMP string: %v", err)
	}

	// RFC 9579 recommends rejecting key lengths shorter than 20 octets to
	// prevent MAC-forgery/authentication-bypass attacks (e.g. CVE-2026-34181).
	for _, keyLength := range []int{1, 8, 16, 19} {
		wantErr := fmt.Sprintf("pkcs12: PBMAC1 key length %d is too short to be secure (minimum 20 octets)", keyLength)
		if _, err := doMac(pbmac1MacData(t, keyLength), message, password); err == nil || err.Error() != wantErr {
			t.Errorf("KeyLength %d: got error %v, want %q", keyLength, err, wantErr)
		}
	}

	// A key length of exactly 20 octets is the minimum allowed and must succeed.
	if _, err := doMac(pbmac1MacData(t, 20), message, password); err != nil {
		t.Errorf("KeyLength 20: unexpected error: %v", err)
	}
}

func TestPBMAC1RejectsLongKeyLength(t *testing.T) {
	message := []byte{11, 12, 13, 14, 15}
	password, err := bmpStringZeroTerminated("test-password")
	if err != nil {
		t.Fatalf("Failed to encode password to BMP string: %v", err)
	}

	// A KeyLength larger than the maximum HMAC output size (64 octets, SHA-512) is rejected so it can't
	// cause a huge PBKDF2 allocation (same as OpenSSL's EVP_MAX_MD_SIZE cap)
	for _, keyLength := range []int{65, 128, 1 << 20, 1 << 30} {
		wantErr := fmt.Sprintf("pkcs12: PBMAC1 key length %d is too large (maximum 64 octets)", keyLength)
		if _, err := doMac(pbmac1MacData(t, keyLength), message, password); err == nil || err.Error() != wantErr {
			t.Errorf("KeyLength %d: got error %v, want %q", keyLength, err, wantErr)
		}
	}

	// A key length of exactly 64 octets is the maximum allowed and must succeed.
	if _, err := doMac(pbmac1MacData(t, 64), message, password); err != nil {
		t.Errorf("KeyLength 64: unexpected error: %v", err)
	}
}

// TestPBMAC1ShortKeyAuthenticationBypass demonstrates the attack that the
// short-key check prevents: a forged trust store whose 1-octet PBMAC1 key is
// derived from one password can be "opened" with a different password whenever
// the two passwords' PBKDF2 outputs collide in that single octet (~1/256).
func TestPBMAC1ShortKeyAuthenticationBypass(t *testing.T) {
	// pbmac1-short-key-bypass.txt is a complete PKCS#12 trust store whose PBMAC1 MAC
	// uses a 1-octet PBKDF2 key. Its salt was chosen by brute force -- feasible only
	// because a 1-octet key has just 256 values -- so that the key derived from the
	// password used to compute the MAC ("fakepassword") collides with the key derived
	// from a different password ("realpassword"). The file therefore authenticates
	// under "realpassword" even though that password was never used to create it.
	//
	// A correct short-key check rejects the file outright; without it,
	// DecodeTrustStore authenticates the file under the wrong password, which is an
	// authentication bypass (cf. OpenSSL CVE-2026-34181).
	pfxData := loadTestData(t, "pbmac1-short-key-bypass.txt")

	_, err := DecodeTrustStore(pfxData, "realpassword")
	if err == nil {
		t.Fatal("authentication bypass: a 1-octet-key PBMAC1 trust store decoded under the wrong password")
	}
	// Guard against the file silently becoming undecodable for some unrelated reason,
	// which would make the assertion above vacuous: the only acceptable failure is the
	// short-key rejection. (The testdata file uses a 1-octet key.)
	if want := "pkcs12: PBMAC1 key length 1 is too short to be secure (minimum 20 octets)"; err.Error() != want {
		t.Fatalf("expected the short-key check to reject the file, got a different error: %v", err)
	}
}
