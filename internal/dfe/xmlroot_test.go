package dfe

import (
	"strings"
	"testing"
)

func TestRootElement(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"plain", `<nfeProc versao="4.00"><NFe/></nfeProc>`, "nfeProc"},
		{"declaration", `<?xml version="1.0" encoding="UTF-8"?><cteProc/>`, "cteProc"},
		{"bom and whitespace", "\xEF\xBB\xBF \r\n\t<?xml version=\"1.0\"?>\n<resNFe/>", "resNFe"},
		{"comment first", `<!-- exported --><procEventoNFe/>`, "procEventoNFe"},
		{"namespace prefix", `<nfe:nfeProc xmlns:nfe="http://www.portalfiscal.inf.br/nfe"/>`, "nfeProc"},
		{"default namespace", `<cteOSProc xmlns="http://www.portalfiscal.inf.br/cte"/>`, "cteOSProc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RootElement([]byte(tt.data))
			if err != nil || got != tt.want {
				t.Errorf("RootElement = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestRootElementErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"nil", "", "XML vazio"},
		{"whitespace", " \n\t ", "XML vazio"},
		{"bom only", "\xEF\xBB\xBF\n", "XML vazio"},
		{"text", "This is not an XML file.", "XML inválido"},
		{"declaration only", `<?xml version="1.0"?>`, "XML inválido"},
		{"broken tag", `<nfeProc`, "XML inválido"},
		{"text before root", `garbage<nfeProc/>`, "XML inválido"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RootElement([]byte(tt.data))
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("RootElement = %q, %v; want an error starting with %q", got, err, tt.want)
			}
		})
	}
}
