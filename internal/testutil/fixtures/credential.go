package fixtures

import "github.com/vasfvitor/nanci/internal/nfse"

func Credential() nfse.Credential {
	return nfse.Credential{
		ID:                "dev-credential-70860312000150",
		Label:             "Certificado Mock 70860312000150",
		CertPath:          "devdata/certs/cert_a1_mock_70860312000150.pfx",
		Environment:       nfse.EnvironmentRestricted,
		OwnerCNPJ:         "70860312000150",
		OwnerCNPJRoot:     "70860312",
		FingerprintSHA256: "mock-fingerprint",
	}
}
