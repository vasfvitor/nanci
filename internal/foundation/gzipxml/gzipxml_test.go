package gzipxml_test

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/foundation/gzipxml"
)

func gzipBase64(t *testing.T, data []byte) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecode(t *testing.T) {
	xmlData := []byte("<xml>test</xml>")

	decoded, err := gzipxml.Decode(gzipBase64(t, xmlData), gzipxml.Limits{
		CompressedBytes:   1024,
		UncompressedBytes: 1024,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if string(decoded.XML) != string(xmlData) {
		t.Errorf("Expected XML %s, got %s", string(xmlData), string(decoded.XML))
	}
	sum := sha256.Sum256(xmlData)
	if want := hex.EncodeToString(sum[:]); decoded.SHA256 != want {
		t.Errorf("Expected SHA256 %s, got %s", want, decoded.SHA256)
	}
}

func TestDecode_ExceedsLimits(t *testing.T) {
	payload := gzipBase64(t, []byte("<xml>test data that exceeds the limit</xml>"))

	_, err := gzipxml.Decode(payload, gzipxml.Limits{
		CompressedBytes:   10, // Intentionally small
		UncompressedBytes: 10,
	})
	if !errors.Is(err, gzipxml.ErrTooLarge) {
		t.Fatalf("Expected ErrTooLarge, got %v", err)
	}
}

func TestDecode_UncompressedLimit(t *testing.T) {
	payload := gzipBase64(t, bytes.Repeat([]byte("a"), 100))

	_, err := gzipxml.Decode(payload, gzipxml.Limits{
		CompressedBytes:   1024,
		UncompressedBytes: 99,
	})
	if !errors.Is(err, gzipxml.ErrTooLarge) {
		t.Fatalf("Expected ErrTooLarge, got %v", err)
	}
}

func TestDecode_InvalidBase64(t *testing.T) {
	if _, err := gzipxml.Decode("not base64!", gzipxml.Limits{CompressedBytes: 1024, UncompressedBytes: 1024}); err == nil {
		t.Fatal("Expected error for invalid base64")
	}
}
