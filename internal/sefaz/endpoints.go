// Portions adapted from github.com/mschunke/gonfe (MIT License). See third_party/gonfe/LICENSE.

// Package sefaz is the SOAP client for the NF-e web services of the Ambiente
// Nacional: NFeDistribuicaoDFe (documents addressed to a CNPJ) and
// NFeRecepcaoEvento4 (manifestação do destinatário). It only speaks the
// protocol; storage and sync rules belong to the caller.
package sefaz

import (
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// Web service URLs of the Ambiente Nacional. Distribution only exists on
// www1 in production; events are served by www (www1 also answers).
const (
	DistribuicaoProducao      = "https://www1.nfe.fazenda.gov.br/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx"
	DistribuicaoHomologacao   = "https://hom1.nfe.fazenda.gov.br/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx"
	RecepcaoEventoProducao    = "https://www.nfe.fazenda.gov.br/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx"
	RecepcaoEventoHomologacao = "https://hom1.nfe.fazenda.gov.br/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx"
)

// tpAmb values sent in every request.
const (
	TpAmbProducao    = "1"
	TpAmbHomologacao = "2"
)

// Endpoints are the service URLs a Client talks to.
type Endpoints struct {
	Distribuicao   string
	RecepcaoEvento string
}

// EndpointsFor returns the Ambiente Nacional URLs for env: producao uses the
// production hosts and producao_restrita uses homologação.
func EndpointsFor(env nfse.Environment) (Endpoints, error) {
	switch env {
	case nfse.EnvironmentProduction:
		return Endpoints{Distribuicao: DistribuicaoProducao, RecepcaoEvento: RecepcaoEventoProducao}, nil
	case nfse.EnvironmentRestricted:
		return Endpoints{Distribuicao: DistribuicaoHomologacao, RecepcaoEvento: RecepcaoEventoHomologacao}, nil
	default:
		return Endpoints{}, fmt.Errorf("invalid environment %q: %w", env, nfse.ErrInvalidEnum)
	}
}

// TpAmb maps env to the tpAmb code: producao is 1, producao_restrita is 2
// (homologação).
func TpAmb(env nfse.Environment) (string, error) {
	switch env {
	case nfse.EnvironmentProduction:
		return TpAmbProducao, nil
	case nfse.EnvironmentRestricted:
		return TpAmbHomologacao, nil
	default:
		return "", fmt.Errorf("invalid environment %q: %w", env, nfse.ErrInvalidEnum)
	}
}
