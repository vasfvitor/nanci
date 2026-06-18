package pkcs12

import (
	"bytes"
	"crypto/x509/pkix"
	"encoding/asn1"
	"strings"
	"testing"
)

func TestEncryptedContentInfoDataPrimitive(t *testing.T) {
	info := encryptedContentInfo{
		EncryptedContent: asn1.RawValue{
			Class: 2,
			Tag:   0,
			Bytes: []byte{1, 2, 3},
		},
	}

	got, err := info.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("Data() = %x, want 010203", got)
	}
}

func TestEncryptedContentInfoDataConstructed(t *testing.T) {
	info := encryptedContentInfo{
		EncryptedContent: asn1.RawValue{
			Class:      2,
			Tag:        0,
			IsCompound: true,
			Bytes: []byte{
				0x04, 0x02, 0x01, 0x02,
				0x04, 0x01, 0x03,
			},
		},
	}

	got, err := info.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("Data() = %x, want 010203", got)
	}
}

func TestEncryptedContentInfoDataRejectsInvalidConstructedChunk(t *testing.T) {
	info := encryptedContentInfo{
		EncryptedContent: asn1.RawValue{
			Class:      2,
			Tag:        0,
			IsCompound: true,
			Bytes:      []byte{0x02, 0x01, 0x01},
		},
	}

	_, err := info.Data()
	if err == nil || !strings.Contains(err.Error(), "non-OCTET STRING") {
		t.Fatalf("Data() error = %v, want non-OCTET STRING error", err)
	}
}

func TestEncryptedContentInfoSetDataMarshalsImplicitTag(t *testing.T) {
	info := encryptedContentInfo{
		ContentType: oidDataContentType,
		ContentEncryptionAlgorithm: pkix.AlgorithmIdentifier{
			Algorithm: oidPBEWithSHAAnd3KeyTripleDESCBC,
		},
	}
	info.SetData([]byte{1, 2, 3})

	der, err := asn1.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded encryptedContentInfo
	if err := unmarshal(der, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	got, err := decoded.Data()
	if err != nil {
		t.Fatalf("Data failed: %v", err)
	}
	if !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("round-trip data = %x, want 010203", got)
	}
}
