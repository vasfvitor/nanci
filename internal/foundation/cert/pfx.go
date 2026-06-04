package cert

import (
	"crypto/tls"
	"errors"
	"os"

	"software.sslmate.com/src/go-pkcs12"
)

var (
	ErrFileNotFound = errors.New("certificate file not found")
	ErrInvalidPass  = errors.New("invalid password for certificate")
)

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
