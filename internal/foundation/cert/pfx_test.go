package cert

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractOwnerCNPJFromSubject(t *testing.T) {
	certificate := &x509.Certificate{
		Subject: pkix.Name{
			Names: []pkix.AttributeTypeAndValue{
				{Type: oidCNPJ, Value: "45.723.174/0001-10"},
			},
		},
	}

	got, err := extractOwnerCNPJ(certificate)
	if err != nil {
		t.Fatalf("extractOwnerCNPJ failed: %v", err)
	}
	if got != "45723174000110" {
		t.Fatalf("extractOwnerCNPJ() = %q, want %q", got, "45723174000110")
	}
}

func TestExtractOwnerCNPJFromSubjectAltNameOtherName(t *testing.T) {
	value, err := asn1.MarshalWithParams("45.723.174/0001-10", "utf8")
	if err != nil {
		t.Fatal(err)
	}
	otherName, err := asn1.MarshalWithParams(struct {
		TypeID asn1.ObjectIdentifier
		Value  asn1.RawValue `asn1:"tag:0,explicit"`
	}{
		TypeID: oidCNPJ,
		Value: asn1.RawValue{
			FullBytes: value,
		},
	}, "tag:0")
	if err != nil {
		t.Fatal(err)
	}
	subjectAltName, err := asn1.Marshal([]asn1.RawValue{{FullBytes: otherName}})
	if err != nil {
		t.Fatal(err)
	}

	certificate := &x509.Certificate{
		Extensions: []pkix.Extension{
			{Id: oidSubjectAltName, Value: subjectAltName},
		},
	}

	got, err := extractOwnerCNPJ(certificate)
	if err != nil {
		t.Fatalf("extractOwnerCNPJ failed: %v", err)
	}
	if got != "45723174000110" {
		t.Fatalf("extractOwnerCNPJ() = %q, want %q", got, "45723174000110")
	}
}

func TestLoadPKCS12_FileNotFound(t *testing.T) {
	_, err := LoadPKCS12("non_existent_file.pfx", "password")
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestLoadPKCS12_InvalidData(t *testing.T) {
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.pfx")

	err := os.WriteFile(invalidFile, []byte("this is not a valid pfx"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadPKCS12(invalidFile, "password")
	if err == nil {
		t.Error("expected error for invalid PFX data, got nil")
	}
}

func TestLoadPKCS12_ValidMockCert(t *testing.T) {
	mockPfxPath := filepath.Join("testdata", "cert_a1_mock_70860312000150.pfx")
	if _, err := os.Stat(mockPfxPath); os.IsNotExist(err) {
		t.Skip("Mock cert not found, skipping. Run 'go run gen/mock_cert.go' to generate it.")
	}

	loaded, err := LoadPKCS12(mockPfxPath, "mockdata")
	if err != nil {
		t.Fatalf("LoadPKCS12 failed on valid mock cert: %v", err)
	}

	if loaded.TLS.PrivateKey == nil {
		t.Error("expected PrivateKey to be populated, got nil")
	}

	if loaded.Inspection.OwnerCNPJ != "70860312000150" {
		t.Errorf("expected OwnerCNPJ to be 70860312000150, got %q", loaded.Inspection.OwnerCNPJ)
	}

	if loaded.Inspection.OwnerCNPJRoot != "70860312" {
		t.Errorf("expected OwnerCNPJRoot to be 70860312, got %q", loaded.Inspection.OwnerCNPJRoot)
	}

	if loaded.Inspection.FingerprintSHA256 == "" {
		t.Error("expected non-empty FingerprintSHA256")
	}
}

func TestLoadPKCS12_ValidMockCert_BER(t *testing.T) {
	mockPfxPath := filepath.Join("testdata", "cert_a1_mock_70860312000150.pfx")
	pfxData, err := os.ReadFile(mockPfxPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("Mock cert not found, skipping.")
		}
		t.Fatal(err)
	}

	berData, tamperOffset, err := makeChunkedBERPFX(pfxData, 1000)
	if err != nil {
		t.Fatalf("failed to create BER mock certificate: %v", err)
	}

	berFile := filepath.Join(t.TempDir(), "mock_ber.pfx")
	if err := os.WriteFile(berFile, berData, 0600); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPKCS12(berFile, "mockdata")
	if err != nil {
		t.Fatalf("LoadPKCS12 failed on BER mock cert: %v", err)
	}

	if loaded.TLS.PrivateKey == nil {
		t.Error("expected PrivateKey to be populated, got nil")
	}

	if loaded.Inspection.OwnerCNPJ != "70860312000150" {
		t.Errorf("expected OwnerCNPJ to be 70860312000150, got %q", loaded.Inspection.OwnerCNPJ)
	}

	if _, err := LoadPKCS12(berFile, "wrong-password"); !errors.Is(err, ErrInvalidPass) {
		t.Fatalf("expected ErrInvalidPass for wrong password, got %v", err)
	}

	tampered := bytes.Clone(berData)
	tampered[tamperOffset] ^= 0x01
	tamperedFile := filepath.Join(t.TempDir(), "mock_ber_tampered.pfx")
	if err := os.WriteFile(tamperedFile, tampered, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPKCS12(tamperedFile, "mockdata"); !errors.Is(err, ErrInvalidPass) {
		t.Fatalf("expected ErrInvalidPass for tampered authenticated content, got %v", err)
	}
}

func TestLoadPKCS12_ExternalAcceptance(t *testing.T) {
	path := os.Getenv("NANCI_TEST_PFX_PATH")
	password := os.Getenv("NANCI_CERT_PASSWORD")
	if path == "" || password == "" {
		t.Skip("set NANCI_TEST_PFX_PATH and NANCI_CERT_PASSWORD to test an external certificate")
	}

	loaded, err := LoadPKCS12(path, password)
	if err != nil {
		t.Fatalf("LoadPKCS12 failed on external certificate: %v", err)
	}
	if loaded.TLS.PrivateKey == nil {
		t.Fatal("expected external certificate private key to be populated")
	}
	if loaded.Inspection.OwnerCNPJ == "" {
		t.Fatal("expected external certificate owner CNPJ to be populated")
	}

	if _, err := LoadPKCS12(path, password+"-wrong"); !errors.Is(err, ErrInvalidPass) {
		t.Fatalf("expected ErrInvalidPass for external certificate wrong password, got %v", err)
	}
}

func makeChunkedBERPFX(der []byte, chunkSize int) ([]byte, int, error) {
	if chunkSize <= 0 {
		return nil, 0, errors.New("chunk size must be positive")
	}

	parser := berParser{data: der}
	obj, offset, err := parser.readObject(0, 0)
	if err != nil {
		return nil, 0, err
	}
	if offset != len(der) {
		return nil, 0, errors.New("PFX has trailing data")
	}

	root, ok := obj.(asn1Structured)
	if !ok || len(root.content) != 3 {
		return nil, 0, errors.New("expected PFX sequence with MacData")
	}
	authSafe, ok := root.content[1].(asn1Structured)
	if !ok || len(authSafe.content) != 2 {
		return nil, 0, errors.New("expected AuthSafe ContentInfo")
	}
	explicitContent, ok := authSafe.content[1].(asn1Structured)
	if !ok || len(explicitContent.content) != 1 {
		return nil, 0, errors.New("expected explicit AuthSafe content")
	}
	octets, ok := explicitContent.content[0].(asn1Primitive)
	if !ok || len(octets.tagBytes) != 1 || octets.tagBytes[0] != 0x04 {
		return nil, 0, errors.New("expected primitive AuthSafe OCTET STRING")
	}

	out := new(bytes.Buffer)
	out.Write([]byte{0x30, 0x80})
	if err := root.content[0].encodeTo(out); err != nil {
		return nil, 0, err
	}

	out.Write([]byte{0x30, 0x80})
	if err := authSafe.content[0].encodeTo(out); err != nil {
		return nil, 0, err
	}

	out.Write([]byte{0xa0, 0x80, 0x24, 0x80})
	tamperOffset := 0
	for start := 0; start < len(octets.content); start += chunkSize {
		end := min(start+chunkSize, len(octets.content))
		if err := out.WriteByte(0x04); err != nil {
			return nil, 0, err
		}
		if err := encodeLength(out, end-start); err != nil {
			return nil, 0, err
		}
		if tamperOffset == 0 {
			tamperOffset = out.Len()
		}
		if _, err := out.Write(octets.content[start:end]); err != nil {
			return nil, 0, err
		}
	}
	out.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	if err := root.content[2].encodeTo(out); err != nil {
		return nil, 0, err
	}
	out.Write([]byte{0x00, 0x00})
	return out.Bytes(), tamperOffset, nil
}
