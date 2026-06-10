package cert

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

var (
	ErrFileNotFound = errors.New("certificate file not found")
	ErrInvalidPass  = errors.New("invalid password for certificate")
	ErrCNPJNotFound = errors.New("certificate owner cnpj not found")

	oidCNPJ           = asn1.ObjectIdentifier{2, 16, 76, 1, 3, 3}
	oidSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}
)

type Inspection struct {
	OwnerCNPJ         string
	OwnerCNPJRoot     string
	FingerprintSHA256 string
	SubjectName       string
	NotBefore         time.Time
	NotAfter          time.Time
}

type LoadedCertificate struct {
	TLS        tls.Certificate
	Inspection Inspection
}

// LoadPKCS12 reads a .pfx or .p12 file, parses it into a tls.Certificate, and extracts inspection metadata.
func LoadPKCS12(path string, password string) (LoadedCertificate, error) {
	pfxData, err := os.ReadFile(path) // #nosec G304 -- path is explicitly selected by the local user.
	if err != nil {
		if os.IsNotExist(err) {
			return LoadedCertificate{}, ErrFileNotFound
		}
		return LoadedCertificate{}, err
	}

	// Decode DER directly. If the input uses BER forms unsupported by Go's
	// encoding/asn1 package, normalize them and retry with MAC verification intact.
	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		if errors.Is(err, pkcs12.ErrIncorrectPassword) {
			return LoadedCertificate{}, ErrInvalidPass
		}

		derData, changed, normalizeErr := normalizePKCS12BER(pfxData)
		if normalizeErr == nil && changed {
			privateKey, certificate, caCerts, err = pkcs12.DecodeChain(derData, password)
		}

		if err != nil {
			if errors.Is(err, pkcs12.ErrIncorrectPassword) {
				return LoadedCertificate{}, ErrInvalidPass
			}
			return LoadedCertificate{}, err
		}
	}

	if privateKey == nil {
		return LoadedCertificate{}, errors.New("certificado não possui chave privada (necessário para mTLS)")
	}

	ownerCNPJ, err := extractOwnerCNPJ(certificate)
	if err != nil {
		return LoadedCertificate{}, err
	}

	// Construct the tls.Certificate
	tlsCert := tls.Certificate{
		PrivateKey: privateKey,
		Leaf:       certificate,
	}

	tlsCert.Certificate = append(tlsCert.Certificate, certificate.Raw)
	for _, ca := range caCerts {
		tlsCert.Certificate = append(tlsCert.Certificate, ca.Raw)
	}

	sum := sha256.Sum256(certificate.Raw)
	inspection := Inspection{
		OwnerCNPJ:         ownerCNPJ,
		OwnerCNPJRoot:     ownerCNPJ[:8],
		FingerprintSHA256: hex.EncodeToString(sum[:]),
		SubjectName:       certificate.Subject.String(),
		NotBefore:         certificate.NotBefore.UTC(),
		NotAfter:          certificate.NotAfter.UTC(),
	}

	return LoadedCertificate{
		TLS:        tlsCert,
		Inspection: inspection,
	}, nil
}

func extractOwnerCNPJ(certificate *x509.Certificate) (string, error) {
	for _, name := range certificate.Subject.Names {
		if name.Type.Equal(oidCNPJ) {
			if cnpj := cnpjFromValue(name.Value); cnpj != "" {
				return cnpj, nil
			}
		}
	}

	for _, extension := range certificate.Extensions {
		if !extension.Id.Equal(oidSubjectAltName) {
			continue
		}
		cnpj, err := cnpjFromSubjectAltName(extension.Value)
		if err != nil {
			return "", err
		}
		if cnpj != "" {
			return cnpj, nil
		}
	}

	return "", ErrCNPJNotFound
}

func cnpjFromSubjectAltName(der []byte) (string, error) {
	var names []asn1.RawValue
	trailing, err := asn1.Unmarshal(der, &names)
	if err != nil {
		return "", err
	}
	if len(trailing) != 0 {
		return "", errors.New("certificate subject alternative name has trailing data")
	}

	for _, name := range names {
		if name.Class != 2 || name.Tag != 0 || !name.IsCompound {
			continue
		}

		var otherName struct {
			TypeID asn1.ObjectIdentifier
			Value  asn1.RawValue `asn1:"tag:0,explicit"`
		}
		trailing, err := asn1.UnmarshalWithParams(name.FullBytes, &otherName, "tag:0")
		if err != nil {
			return "", err
		}
		if len(trailing) != 0 {
			return "", errors.New("certificate otherName has trailing data")
		}
		if otherName.TypeID.Equal(oidCNPJ) {
			if cnpj := cnpjFromRawValue(otherName.Value); cnpj != "" {
				return cnpj, nil
			}
		}
	}

	return "", nil
}

func cnpjFromRawValue(value asn1.RawValue) string {
	if value.Class == 2 && value.Tag == 0 && value.IsCompound {
		var inner asn1.RawValue
		if trailing, err := asn1.Unmarshal(value.Bytes, &inner); err == nil && len(trailing) == 0 {
			return cnpjFromRawValue(inner)
		}
	}

	return cnpjFromValue(value.Bytes)
}

func cnpjFromValue(value any) string {
	var text string
	switch value := value.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return ""
	}

	normalized := normalizeCNPJToken(text)
	if len(normalized) == 14 {
		return normalized
	}
	return ""
}

func normalizeCNPJToken(value string) string {
	var normalized strings.Builder
	normalized.Grow(14)
	for _, r := range strings.TrimSpace(value) {
		if r >= '0' && r <= '9' {
			normalized.WriteRune(r)
		}
	}
	return normalized.String()
}
