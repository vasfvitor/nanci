package cert

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
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

	// Decode the PFX file using the provided password
	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		if errors.Is(err, pkcs12.ErrIncorrectPassword) {
			return LoadedCertificate{}, ErrInvalidPass
		}
		return LoadedCertificate{}, err
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
	// ICP-Brasil CNPJ OID: 2.16.76.1.3.3
	oidCNPJ := []int{2, 16, 76, 1, 3, 3}

	for _, name := range certificate.Subject.Names {
		if name.Type.Equal(oidCNPJ) {
			val, ok := name.Value.(string)
			if ok && len(val) >= 14 {
				// The CNPJ in the OID is 14 digits long, sometimes prefixed/suffixed depending on exact issuer format, but generally exact.
				// We normalize it and extract just the numbers.
				normalized := normalizeCNPJToken(val)
				if len(normalized) == 14 {
					return normalized, nil
				}
			}
		}
	}

	return "", ErrCNPJNotFound
}

func normalizeCNPJToken(value string) string {
	replacer := strings.NewReplacer(".", "", "/", "", "-", "", " ", "", ":", "")
	return replacer.Replace(strings.TrimSpace(value))
}
