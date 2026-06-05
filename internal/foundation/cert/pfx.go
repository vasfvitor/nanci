package cert

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
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

// LoadPKCS12 reads a .pfx or .p12 file and parses it into a tls.Certificate.
func LoadPKCS12(path string, password string) (*tls.Certificate, error) {
	pfxData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	// Decode the PFX file using the provided password
	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		if errors.Is(err, pkcs12.ErrIncorrectPassword) {
			return nil, ErrInvalidPass
		}
		return nil, err
	}

	// Construct the tls.Certificate
	tlsCert := tls.Certificate{
		PrivateKey: privateKey,
		Leaf:       certificate,
	}

	// The first certificate in the Certificate block is the leaf certificate,
	// followed by any intermediate CAs that were included in the PFX.
	tlsCert.Certificate = append(tlsCert.Certificate, certificate.Raw)
	for _, ca := range caCerts {
		tlsCert.Certificate = append(tlsCert.Certificate, ca.Raw)
	}

	return &tlsCert, nil
}

// LoadPKCS12WithInspection reads a PKCS#12 file and returns the TLS certificate plus
// metadata derived from the leaf certificate for same-root validation and auditing.
func LoadPKCS12WithInspection(path string, password string) (*tls.Certificate, *Inspection, error) {
	pfxData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrFileNotFound
		}
		return nil, nil, err
	}

	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		if errors.Is(err, pkcs12.ErrIncorrectPassword) {
			return nil, nil, ErrInvalidPass
		}
		return nil, nil, err
	}

	ownerCNPJ, err := extractOwnerCNPJ(certificate)
	if err != nil {
		return nil, nil, err
	}

	tlsCert := tls.Certificate{
		PrivateKey: privateKey,
		Leaf:       certificate,
	}
	tlsCert.Certificate = append(tlsCert.Certificate, certificate.Raw)
	for _, ca := range caCerts {
		tlsCert.Certificate = append(tlsCert.Certificate, ca.Raw)
	}

	sum := sha256.Sum256(certificate.Raw)
	inspection := &Inspection{
		OwnerCNPJ:         ownerCNPJ,
		OwnerCNPJRoot:     ownerCNPJ[:8],
		FingerprintSHA256: hex.EncodeToString(sum[:]),
		SubjectName:       certificate.Subject.String(),
		NotBefore:         certificate.NotBefore.UTC(),
		NotAfter:          certificate.NotAfter.UTC(),
	}

	return &tlsCert, inspection, nil
}

func extractOwnerCNPJ(certificate *x509.Certificate) (string, error) {
	candidates := []string{
		certificate.Subject.String(),
		certificate.Issuer.String(),
		certificate.Subject.CommonName,
		certificate.Subject.SerialNumber,
	}

	for _, name := range certificate.Subject.Names {
		candidates = append(candidates, fmt.Sprint(name.Value))
	}
	for _, san := range certificate.DNSNames {
		candidates = append(candidates, san)
	}
	for _, email := range certificate.EmailAddresses {
		candidates = append(candidates, email)
	}
	for _, uri := range certificate.URIs {
		candidates = append(candidates, uri.String())
	}

	re := regexp.MustCompile(`[A-Za-z0-9./-]{14,18}`)
	for _, candidate := range candidates {
		for _, match := range re.FindAllString(candidate, -1) {
			cleaned := normalizeCNPJToken(match)
			if len(cleaned) == 14 {
				return cleaned, nil
			}
		}
	}

	return "", ErrCNPJNotFound
}

func normalizeCNPJToken(value string) string {
	replacer := strings.NewReplacer(".", "", "/", "", "-", "", " ", "", ":", "")
	return replacer.Replace(strings.TrimSpace(value))
}
