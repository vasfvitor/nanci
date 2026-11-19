package app

import (
	"slices"
	"testing"
)

func TestParseNFSeAccessKeysUppercasesTheChave(t *testing.T) {
	// cMun(7) ambGer(1) tpInsc(1) an alphanumeric inscrição federal(14)
	// nNFSe(13) AAMM(4) cNum(9) DV(1).
	const chave = "3550308" + "1" + "2" + "12ABCD34000195" + "0000000000001" + "2609" + "000000001" + "0"

	got, err := parseNFSeAccessKeys([]string{" 3550308121" + "2abcd34000195" + "0000000000001" + "2609" + "000000001" + "0 ", "nfs" + chave})
	if err != nil {
		t.Fatalf("parseNFSeAccessKeys: %v", err)
	}
	if want := []string{chave, "NFS" + chave}; !slices.Equal(got, want) {
		t.Errorf("chaves = %q, want %q", got, want)
	}
}
