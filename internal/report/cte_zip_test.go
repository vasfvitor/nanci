package report

import (
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
)

func TestCTeZipEntriesLayout(t *testing.T) {
	const proc, cteOS, gtve, simp = "35260912345678000195570010000001011123456784", "35260912345678000195670010000001041456789014",
		"35260912345678000195640010000001051567890127", "35260912345678000195570020000001061678901234"
	doc := func(chave string, tipo cte.TipoDocumento, competence string, role cte.CompanyRole) cte.CompanyDocument {
		return cte.CompanyDocument{
			Document:    cte.Document{ChaveAcesso: dfe.AccessKey(chave), TipoDocumento: tipo, Competence: competence, RawHash: "hash-" + chave[40:]},
			CompanyRole: role,
		}
	}
	docs := []cte.CompanyDocument{
		doc(proc, cte.TipoDocumentoCTe, "2026-09", cte.CompanyRoleTomador),
		doc(cteOS, cte.TipoDocumentoCTeOS, "2026-09", cte.CompanyRoleTomador),
		doc(gtve, cte.TipoDocumentoGTVe, "", cte.CompanyRoleDestinatario),
		doc(simp, cte.TipoDocumentoCTeSimplificado, "2026-09", cte.CompanyRoleNone),
	}
	events := map[string][]cte.Event{
		proc: {
			{TpEvento: "110111", NSeqEvento: 1, RawHash: "hash-canc"},
			{TpEvento: "110180", NSeqEvento: 2},
		},
	}

	got := CTeZipEntries(docs, events)
	want := []ZipEntry{
		{Path: "2026-09/tomador/" + proc + "-procCTe.xml", RawHash: "hash-" + proc[40:]},
		{Path: "2026-09/tomador/eventos/" + proc + "-110111-1.xml", RawHash: "hash-canc"},
		{Path: "2026-09/tomador/" + cteOS + "-procCTeOS.xml", RawHash: "hash-" + cteOS[40:]},
		{Path: "destinatario/" + gtve + "-procGTVe.xml", RawHash: "hash-" + gtve[40:]},
		{Path: "2026-09/sem-papel-fiscal/" + simp + "-procCTeSimp.xml", RawHash: "hash-" + simp[40:]},
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries =\n%+v\nwant\n%+v", got, want)
	}
}
