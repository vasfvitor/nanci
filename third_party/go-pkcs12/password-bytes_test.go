// This file is a local nanci addition; it does not exist upstream.
// See README.nanci.md.

package pkcs12

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

// newTestPFX builds a self-signed certificate and encodes it as a PKCS#12 file
// protected by password.
func newTestPFX(t *testing.T, password string) (pfxData []byte, want *x509.Certificate) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "nanci decode chain bytes test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	pfxData, err = Modern.Encode(key, certificate, nil, password)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return pfxData, certificate
}

func TestDecodeChainBytesMatchesDecodeChain(t *testing.T) {
	for _, password := range []string{"", "mockdata", "sênha çom acentos"} {
		t.Run(password, func(t *testing.T) {
			pfxData, want := newTestPFX(t, password)

			wantKey, wantCert, wantCACerts, err := DecodeChain(pfxData, password)
			if err != nil {
				t.Fatalf("DecodeChain: %v", err)
			}
			gotKey, gotCert, gotCACerts, err := DecodeChainBytes(pfxData, []byte(password))
			if err != nil {
				t.Fatalf("DecodeChainBytes: %v", err)
			}

			if !gotCert.Equal(want) || !wantCert.Equal(want) {
				t.Fatal("decoded certificate does not match the encoded one")
			}
			if len(gotCACerts) != len(wantCACerts) {
				t.Fatalf("caCerts = %d, want %d", len(gotCACerts), len(wantCACerts))
			}
			gotRSA, ok := gotKey.(*rsa.PrivateKey)
			if !ok {
				t.Fatalf("DecodeChainBytes key type = %T, want *rsa.PrivateKey", gotKey)
			}
			wantRSA, ok := wantKey.(*rsa.PrivateKey)
			if !ok {
				t.Fatalf("DecodeChain key type = %T, want *rsa.PrivateKey", wantKey)
			}
			if !gotRSA.Equal(wantRSA) {
				t.Fatal("DecodeChainBytes returned a different private key than DecodeChain")
			}
		})
	}
}

func TestDecodeChainBytesWrongPassword(t *testing.T) {
	pfxData, _ := newTestPFX(t, "mockdata")

	if _, _, _, err := DecodeChainBytes(pfxData, []byte("wrong")); err != ErrIncorrectPassword {
		t.Fatalf("err = %v, want ErrIncorrectPassword", err)
	}
}

func TestDecodeChainBytesLeavesPasswordIntact(t *testing.T) {
	pfxData, _ := newTestPFX(t, "mockdata")

	password := []byte("mockdata")
	if _, _, _, err := DecodeChainBytes(pfxData, password); err != nil {
		t.Fatalf("DecodeChainBytes: %v", err)
	}
	if !bytes.Equal(password, []byte("mockdata")) {
		t.Fatalf("password buffer = %q, want it untouched", password)
	}
}

func TestBMPStringBytesMatchesBMPString(t *testing.T) {
	for _, s := range []string{"", "a", "mockdata", "sênha çom acentos", "日本語", "\xff\xfe"} {
		want, wantErr := bmpStringZeroTerminated(s)
		got, gotErr := bmpStringBytesZeroTerminated([]byte(s))
		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("%q: errors differ: string=%v bytes=%v", s, wantErr, gotErr)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%q: bmpStringBytesZeroTerminated = %x, want %x", s, got, want)
		}
	}
}

func TestBMPStringBytesRejectsNonBMP(t *testing.T) {
	if _, err := bmpStringBytesZeroTerminated([]byte("🌎")); err == nil {
		t.Fatal("expected an error for a rune outside the BMP")
	}
}

func TestZeroBytes(t *testing.T) {
	b := []byte("senha")
	zeroBytes(b)
	if !bytes.Equal(b, make([]byte, 5)) {
		t.Fatalf("zeroBytes left %x", b)
	}
	zeroBytes(nil)
}
