package uf

import "testing"

func TestCodeAndSigla(t *testing.T) {
	tests := []struct {
		sigla string
		code  int
	}{
		{"RO", 11}, {"SP", 35}, {"RS", 43}, {"DF", 53}, {"BA", 29},
	}
	for _, tt := range tests {
		code, ok := Code(tt.sigla)
		if !ok || code != tt.code {
			t.Errorf("Code(%q) = %d, %v, want %d", tt.sigla, code, ok, tt.code)
		}
		sigla, ok := Sigla(tt.code)
		if !ok || sigla != tt.sigla {
			t.Errorf("Sigla(%d) = %q, %v, want %q", tt.code, sigla, ok, tt.sigla)
		}
	}
}

func TestValid(t *testing.T) {
	for _, sigla := range []string{"", "sp", "XX", "S", " SP"} {
		if Valid(sigla) {
			t.Errorf("Valid(%q) = true", sigla)
		}
		if _, ok := Code(sigla); ok {
			t.Errorf("Code(%q) found a code", sigla)
		}
	}
	if _, ok := Sigla(99); ok {
		t.Error("Sigla(99) found a sigla")
	}
}
