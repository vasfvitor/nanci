package sefaz

import "context"

const (
	cteNamespace          = "http://www.portalfiscal.inf.br/cte"
	wsdlDistribuicaoCTe   = "http://www.portalfiscal.inf.br/cte/wsdl/CTeDistribuicaoDFe"
	actionDistribuicaoCTe = wsdlDistribuicaoCTe + "/cteDistDFeInteresse"
	versaoDistDFeCTe      = "1.00"
)

// cteDist is CTeDistribuicaoDFe (NT 2015.002). The answer is the same
// retDistDFeInt as the NF-e one, in the CT-e namespace. The action and the
// envelope were accepted by the production service on 2026-09-24 (cStat 137).
var cteDist = distService{
	name:      "CT-e",
	namespace: cteNamespace,
	wsdl:      wsdlDistribuicaoCTe,
	action:    actionDistribuicaoCTe,
	versao:    versaoDistDFeCTe,
	wrapper:   "cteDistDFeInteresse",
	dadosMsg:  "cteDadosMsg",
	endpoint:  func(e Endpoints) string { return e.DistribuicaoCTe },
}

// DistCTeNSU asks CTeDistribuicaoDFe for the documents after ultNSU
// (distNSU). cUFAutor is the IBGE code of the company's UF. The CT-e service
// has no query by access key.
func (c *Client) DistCTeNSU(ctx context.Context, cnpjValue string, cUFAutor int, ultNSU int64) (DistResult, error) {
	query, err := nsuQuery("distNSU", "ultNSU", ultNSU)
	if err != nil {
		return DistResult{}, err
	}
	return c.distribuicao(ctx, cteDist, cnpjValue, cUFAutor, query)
}

// ConsCTeNSU asks CTeDistribuicaoDFe for the single document with the given
// NSU (consNSU).
func (c *Client) ConsCTeNSU(ctx context.Context, cnpjValue string, cUFAutor int, nsu int64) (DistResult, error) {
	query, err := nsuQuery("consNSU", "NSU", nsu)
	if err != nil {
		return DistResult{}, err
	}
	return c.distribuicao(ctx, cteDist, cnpjValue, cUFAutor, query)
}
