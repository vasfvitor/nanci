package sefaz

import (
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestEndpointsFor(t *testing.T) {
	tests := []struct {
		env   nfse.Environment
		want  Endpoints
		tpAmb string
	}{
		{
			env: nfse.EnvironmentProduction,
			want: Endpoints{
				Distribuicao:    "https://www1.nfe.fazenda.gov.br/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx",
				DistribuicaoCTe: "https://www1.cte.fazenda.gov.br/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx",
				RecepcaoEvento:  "https://www.nfe.fazenda.gov.br/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx",
			},
			tpAmb: "1",
		},
		{
			env: nfse.EnvironmentRestricted,
			want: Endpoints{
				Distribuicao:    "https://hom1.nfe.fazenda.gov.br/NFeDistribuicaoDFe/NFeDistribuicaoDFe.asmx",
				DistribuicaoCTe: "https://hom1.cte.fazenda.gov.br/CTeDistribuicaoDFe/CTeDistribuicaoDFe.asmx",
				RecepcaoEvento:  "https://hom1.nfe.fazenda.gov.br/NFeRecepcaoEvento4/NFeRecepcaoEvento4.asmx",
			},
			tpAmb: "2",
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.env), func(t *testing.T) {
			got, err := EndpointsFor(tt.env)
			if err != nil || got != tt.want {
				t.Errorf("EndpointsFor = %+v, %v; want %+v", got, err, tt.want)
			}
			tpAmb, err := TpAmb(tt.env)
			if err != nil || tpAmb != tt.tpAmb {
				t.Errorf("TpAmb = %q, %v; want %q", tpAmb, err, tt.tpAmb)
			}
		})
	}
}

func TestEndpointsFor_InvalidEnvironment(t *testing.T) {
	if _, err := EndpointsFor("homologacao"); !errors.Is(err, dfe.ErrInvalidEnum) {
		t.Errorf("EndpointsFor: err = %v, want ErrInvalidEnum", err)
	}
	if _, err := TpAmb(""); !errors.Is(err, dfe.ErrInvalidEnum) {
		t.Errorf("TpAmb: err = %v, want ErrInvalidEnum", err)
	}
}
