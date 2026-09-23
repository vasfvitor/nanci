// Package uf lists the 27 Brazilian federative units (26 states and the
// Distrito Federal) with their IBGE codes, as used by fiscal documents
// (cUF, cUFAutor).
package uf

import "slices"

// codeBySigla maps each sigla to its IBGE code.
var codeBySigla = map[string]int{
	"RO": 11, "AC": 12, "AM": 13, "RR": 14, "PA": 15, "AP": 16, "TO": 17,
	"MA": 21, "PI": 22, "CE": 23, "RN": 24, "PB": 25, "PE": 26, "AL": 27, "SE": 28, "BA": 29,
	"MG": 31, "ES": 32, "RJ": 33, "SP": 35,
	"PR": 41, "SC": 42, "RS": 43,
	"MS": 50, "MT": 51, "GO": 52, "DF": 53,
}

// Code returns the IBGE code of an uppercase sigla such as "SP" (35).
func Code(sigla string) (int, bool) {
	code, ok := codeBySigla[sigla]
	return code, ok
}

// Valid reports whether sigla is one of the 27 uppercase siglas.
func Valid(sigla string) bool {
	_, ok := codeBySigla[sigla]
	return ok
}

// Sigla returns the sigla of an IBGE code such as 35 ("SP").
func Sigla(code int) (string, bool) {
	for sigla, c := range codeBySigla {
		if c == code {
			return sigla, true
		}
	}
	return "", false
}

// Siglas returns the 27 siglas in alphabetical order.
func Siglas() []string {
	siglas := make([]string, 0, len(codeBySigla))
	for sigla := range codeBySigla {
		siglas = append(siglas, sigla)
	}
	slices.Sort(siglas)
	return siglas
}
