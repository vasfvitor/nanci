package sefaz

import (
	"fmt"
	"strconv"

	"github.com/vasfvitor/nanci/internal/foundation/uf"
)

// cteAutorizadora is the CTeRecepcaoEventoV4 of one SEFAZ that authorizes
// CT-e, in production and in homologação.
type cteAutorizadora struct {
	producao, homologacao string
}

// The autorizadoras of the CT-e and their event service, from the national
// CT-e portal's list of web services (webServices.aspx, production and
// homologação), checked on 2026-11-19. RS uses the SVRS host and SP the SVSP
// one. Sources: docs/pesquisa/2026-11-19-cte-desacordo.md.
var (
	cteSVRS = cteAutorizadora{
		producao:    "https://cte.svrs.rs.gov.br/ws/CTeRecepcaoEventoV4/CTeRecepcaoEventoV4.asmx",
		homologacao: "https://cte-homologacao.svrs.rs.gov.br/ws/CTeRecepcaoEventoV4/CTeRecepcaoEventoV4.asmx",
	}
	cteSVSP = cteAutorizadora{
		producao:    "https://nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoEventoV4.asmx",
		homologacao: "https://homologacao.nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoEventoV4.asmx",
	}
	cteMG = cteAutorizadora{
		producao:    "https://cte.fazenda.mg.gov.br/cte/services/CTeRecepcaoEventoV4",
		homologacao: "https://hcte.fazenda.mg.gov.br/cte/services/CTeRecepcaoEventoV4",
	}
	cteMS = cteAutorizadora{
		producao:    "https://producao.cte.ms.gov.br/ws/CTeRecepcaoEventoV4",
		homologacao: "https://homologacao.cte.ms.gov.br/ws/CTeRecepcaoEventoV4",
	}
	cteMT = cteAutorizadora{
		producao:    "https://cte.sefaz.mt.gov.br/ctews2/services/CTeRecepcaoEventoV4",
		homologacao: "https://homologacao.sefaz.mt.gov.br/ctews2/services/CTeRecepcaoEventoV4",
	}
	ctePR = cteAutorizadora{
		producao:    "https://cte.fazenda.pr.gov.br/cte4/CTeRecepcaoEventoV4",
		homologacao: "https://homologacao.cte.fazenda.pr.gov.br/cte4/CTeRecepcaoEventoV4",
	}
)

// cteAutorizadoraBySigla maps every UF to the SEFAZ that authorizes its CT-e:
// SVRS for 18 UF, SVSP for AP, PE and RR, and their own for the other six.
var cteAutorizadoraBySigla = map[string]cteAutorizadora{
	"AC": cteSVRS, "AL": cteSVRS, "AM": cteSVRS, "BA": cteSVRS, "CE": cteSVRS, "DF": cteSVRS,
	"ES": cteSVRS, "GO": cteSVRS, "MA": cteSVRS, "PA": cteSVRS, "PB": cteSVRS, "PI": cteSVRS,
	"RJ": cteSVRS, "RN": cteSVRS, "RO": cteSVRS, "SC": cteSVRS, "SE": cteSVRS, "TO": cteSVRS,
	"RS": cteSVRS,
	"AP": cteSVSP, "PE": cteSVSP, "RR": cteSVSP,
	"SP": cteSVSP,
	"MG": cteMG, "MS": cteMS, "MT": cteMT, "PR": ctePR,
}

// CTeRecepcaoEventoURL returns the CTeRecepcaoEventoV4 that takes the events
// of a CT-e issued in cUF (the first two digits of the chave), for tpAmb.
func CTeRecepcaoEventoURL(cUF, tpAmb string) (string, error) {
	code, err := strconv.Atoi(cUF)
	if err != nil {
		return "", fmt.Errorf("invalid cUF %q", cUF)
	}
	sigla, ok := uf.Sigla(code)
	if !ok {
		return "", fmt.Errorf("invalid cUF %q", cUF)
	}
	autorizadora, ok := cteAutorizadoraBySigla[sigla]
	if !ok {
		return "", fmt.Errorf("no CT-e autorizadora for UF %s", sigla)
	}
	switch tpAmb {
	case TpAmbProducao:
		return autorizadora.producao, nil
	case TpAmbHomologacao:
		return autorizadora.homologacao, nil
	default:
		return "", fmt.Errorf("invalid tpAmb %q", tpAmb)
	}
}
