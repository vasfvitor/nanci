package redact

import "testing"

func TestMaskIdentifier(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "abc", want: "***"},
		{in: "abcd", want: "****"},
		{in: "abcde", want: "ab*de"},
		{in: "12345678000195", want: "12**********95"},
	}
	for _, tt := range tests {
		if got := MaskIdentifier(tt.in); got != tt.want {
			t.Errorf("MaskIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMaskXMLIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "NFS-e parties",
			in:   `<prest><CNPJ>11222333000181</CNPJ><IM>12345678</IM><xNome>PRESTADORA FICTICIA</xNome><fone>11999998888</fone><email>contato@ficticia.com.br</email></prest><toma><CPF>12345678909</CPF><NIF>ABC123456</NIF><xFant>FANTASIA</xFant></toma>`,
			want: `<prest><CNPJ>11**********81</CNPJ><IM>12****78</IM><xNome>PR***************IA</xNome><fone>11*******88</fone><email>co*******************br</email></prest><toma><CPF>12*******09</CPF><NIF>AB*****56</NIF><xFant>FA****IA</xFant></toma>`,
		},
		{
			name: "NF-e resumo",
			in:   `<resNFe><chNFe>35260911222333000181550010000012341123456787</chNFe><CNPJ>11222333000181</CNPJ><xNome>DISTRIBUIDORA FICTICIA</xNome><IE>111222333444</IE><vNF>10.00</vNF></resNFe>`,
			want: `<resNFe><chNFe>35****************************************87</chNFe><CNPJ>11**********81</CNPJ><xNome>DI******************IA</xNome><IE>11********44</IE><vNF>10.00</vNF></resNFe>`,
		},
		{
			name: "access keys and prefixed elements",
			in:   `<chNFSe>35503082211222333000181000000000000126090000000001</chNFSe><ns2:chNFe>35260911222333000181550010000012341123456787</ns2:chNFe>`,
			want: `<chNFSe>35**********************************************01</chNFSe><ns2:chNFe>35****************************************87</ns2:chNFe>`,
		},
		{
			name: "event answers",
			in:   `<ns2:CPF>12345678909</ns2:CPF><CNPJDest>70860312000150</CNPJDest><CPFDest>12345678909</CPFDest>`,
			want: `<ns2:CPF>12*******09</ns2:CPF><CNPJDest>70**********50</CNPJDest><CPFDest>12*******09</CPFDest>`,
		},
		{
			name: "similar names and values are kept",
			in:   `<IEST>123456</IEST><IMunic>1</IMunic><CNPJBase>12345678</CNPJBase><vServ>10.00</vServ><cStat>138</cStat>`,
			want: `<IEST>123456</IEST><IMunic>1</IMunic><CNPJBase>12345678</CNPJBase><vServ>10.00</vServ><cStat>138</cStat>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(MaskXMLIdentifiers([]byte(tt.in))); got != tt.want {
				t.Errorf("MaskXMLIdentifiers:\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}
