package fixtures

import "github.com/vasfvitor/nanci/internal/nfse"

func Company() nfse.Company {
	return nfse.Company{
		ID:           "dev-company-70860312000150",
		CNPJ:         "70860312000150",
		CNPJRoot:     "70860312",
		Name:         "Empresa Mock Teste",
		CredentialID: "dev-credential-70860312000150",
		Environment:  nfse.EnvironmentRestricted,
		LastNSU:      0,
	}
}
