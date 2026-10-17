package desktopapi

import (
	"reflect"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
)

func TestImportXMLResultFrom(t *testing.T) {
	got := ImportXMLResultFrom(app.ImportXMLResult{
		Files: []app.ImportedFile{
			{Path: "a.xml", Kind: "resumo", ChaveAcesso: "35260912345678000195550010000123451000123456", Outcome: "imported"},
			{Path: "b.xml", Outcome: "rejected", Message: "XML ilegível"},
		},
		Imported: 1,
		Rejected: 1,
	})
	want := ImportXMLResult{
		Files: []ImportedFile{
			{Path: "a.xml", Kind: "resumo", ChaveAcesso: "35260912345678000195550010000123451000123456", Outcome: "imported"},
			{Path: "b.xml", Outcome: "rejected", Message: "XML ilegível"},
		},
		Imported: 1,
		Rejected: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ImportXMLResultFrom = %+v, want %+v", got, want)
	}
}

func TestImportXMLResultFromEmptyFilesIsNotNil(t *testing.T) {
	got := ImportXMLResultFrom(app.ImportXMLResult{})
	if got.Files == nil || len(got.Files) != 0 {
		t.Errorf("Files = %v, want an empty slice", got.Files)
	}
}

func TestFetchByChaveResultFrom(t *testing.T) {
	got := FetchByChaveResultFrom(app.FetchByChaveResult{
		ChaveAcesso:      "35260912345678000195550010000123451000123456",
		Outcome:          "not_found",
		Message:          "A SEFAZ não encontrou a NF-e.",
		RequestsLastHour: 5,
		RequestBudget:    20,
	})
	want := FetchByChaveResult{
		ChaveAcesso:      "35260912345678000195550010000123451000123456",
		Outcome:          "not_found",
		Message:          "A SEFAZ não encontrou a NF-e.",
		RequestsLastHour: 5,
		RequestBudget:    20,
	}
	if got != want {
		t.Errorf("FetchByChaveResultFrom = %+v, want %+v", got, want)
	}
}
