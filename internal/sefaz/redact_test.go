package sefaz

import "testing"

func TestRedactForLog(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "identifiers are masked",
			in:   `<resNFe><chNFe>35260911222333000181550010000012341123456787</chNFe><CNPJ>11222333000181</CNPJ><xNome>DISTRIBUIDORA FICTICIA</xNome><IE>111222333444</IE><vNF>10.00</vNF></resNFe>`,
			want: `<resNFe><chNFe>35****************************************87</chNFe><CNPJ>11**********81</CNPJ><xNome>DI******************IA</xNome><IE>11********44</IE><vNF>10.00</vNF></resNFe>`,
		},
		{
			name: "prefixed elements and event answers",
			in:   `<ns2:CPF>12345678909</ns2:CPF><CNPJDest>70860312000150</CNPJDest><CPFDest>12345678909</CPFDest>`,
			want: `<ns2:CPF>12*******09</ns2:CPF><CNPJDest>70**********50</CNPJDest><CPFDest>12*******09</CPFDest>`,
		},
		{
			name: "similar names are kept",
			in:   `<IEST>123456</IEST><CNPJBase>12345678</CNPJBase><cStat>138</cStat>`,
			want: `<IEST>123456</IEST><CNPJBase>12345678</CNPJBase><cStat>138</cStat>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(RedactForLog([]byte(tt.in))); got != tt.want {
				t.Errorf("RedactForLog:\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}
