package httpclient

import "testing"

func TestTruncateForLog(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int
		want  string
	}{
		{name: "under limit", body: "abc", limit: 10, want: "abc"},
		{name: "at limit", body: "abcde", limit: 5, want: "abcde"},
		{name: "over limit", body: "abcdefgh", limit: 5, want: "abcde... (truncated)"},
		// "ação" is a(1) ç(2) ã(2) o(1) bytes; a cut inside ç or ã backs up.
		{name: "does not split rune", body: "ação", limit: 2, want: "a... (truncated)"},
		{name: "cut lands on rune start", body: "ação", limit: 3, want: "aç... (truncated)"},
		{name: "cut inside second rune", body: "ação", limit: 4, want: "aç... (truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateForLog([]byte(tt.body), tt.limit); got != tt.want {
				t.Errorf("TruncateForLog(%q, %d) = %q, want %q", tt.body, tt.limit, got, tt.want)
			}
		})
	}
}

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
			name: "access keys and prefixed elements",
			in:   `<chNFSe>35503082211222333000181000000000000126090000000001</chNFSe><ns2:chNFe>35260911222333000181550010000012341123456787</ns2:chNFe>`,
			want: `<chNFSe>35**********************************************01</chNFSe><ns2:chNFe>35****************************************87</ns2:chNFe>`,
		},
		{
			name: "similar names and values are kept",
			in:   `<IEST>123456</IEST><IMunic>1</IMunic><CNPJBase>12345678</CNPJBase><vServ>10.00</vServ>`,
			want: `<IEST>123456</IEST><IMunic>1</IMunic><CNPJBase>12345678</CNPJBase><vServ>10.00</vServ>`,
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
