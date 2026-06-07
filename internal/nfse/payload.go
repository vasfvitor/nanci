package nfse

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

type PayloadLimits struct {
	CompressedBytes   int64
	UncompressedBytes int64
}

type DecodedPayload struct {
	XML    []byte
	SHA256 string
}

var (
	ErrPayloadTooLarge = errors.New("payload exceeds configured size limits")
)

// DecodePayload decodes the base64-gzipped payload into raw XML bytes, respecting size limits.
func DecodePayload(payloadBase64 string, limits PayloadLimits) (DecodedPayload, error) {
	// 1. Base64 Size Check
	// A base64 string length gives a direct upper bound on decoded bytes: (len * 3) / 4
	estimatedDecodedSize := int64(len(payloadBase64)) * 3 / 4
	if estimatedDecodedSize > limits.CompressedBytes {
		return DecodedPayload{}, fmt.Errorf("%w: compressed payload estimated at %d bytes (limit %d)", 
			ErrPayloadTooLarge, estimatedDecodedSize, limits.CompressedBytes)
	}

	gzippedData, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return DecodedPayload{}, fmt.Errorf("failed to decode base64: %w", err)
	}

	if int64(len(gzippedData)) > limits.CompressedBytes {
		return DecodedPayload{}, fmt.Errorf("%w: compressed payload is %d bytes (limit %d)", 
			ErrPayloadTooLarge, len(gzippedData), limits.CompressedBytes)
	}

	// 2. Un-Gzip with limits
	gzipReader, err := gzip.NewReader(bytes.NewReader(gzippedData))
	if err != nil {
		return DecodedPayload{}, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	limitedReader := io.LimitReader(gzipReader, limits.UncompressedBytes+1) // +1 to detect overflow
	
	xmlData, err := io.ReadAll(limitedReader)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return DecodedPayload{}, fmt.Errorf("failed to read uncompressed xml: %w", err)
	}

	if int64(len(xmlData)) > limits.UncompressedBytes {
		return DecodedPayload{}, fmt.Errorf("%w: uncompressed payload exceeded limit of %d bytes", 
			ErrPayloadTooLarge, limits.UncompressedBytes)
	}

	// 3. Compute SHA-256 hash
	hash := sha256.Sum256(xmlData)
	hashHex := hex.EncodeToString(hash[:])

	return DecodedPayload{
		XML:    xmlData,
		SHA256: hashHex,
	}, nil
}
