package xmlwalk_test

import (
	"encoding/xml"
	"errors"
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/foundation/xmlwalk"
)

const sample = `<?xml version="1.0"?>
<nfeProc xmlns="http://www.portalfiscal.inf.br/nfe">
  <NFe><infNFe Id="NFe123" versao="4.00">
    <ide><nNF> 42 </nNF></ide>
    <det nItem="1"><imposto><vICMS>1.00</vICMS></imposto></det>
    <total><ICMSTot><vICMS>1.00</vICMS></ICMSTot></total>
    <empty>  </empty>
  </infNFe></NFe>
</nfeProc>`

func TestWalk(t *testing.T) {
	var starts []string
	var id string
	texts := map[string]string{}
	err := xmlwalk.Walk([]byte(sample),
		func(path string, attrs []xml.Attr) {
			starts = append(starts, path)
			if path == "/nfeProc/NFe/infNFe" {
				id = xmlwalk.AttrValue(attrs, "Id")
			}
		},
		func(path, text string) error {
			texts[path] = text
			return nil
		})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	if !slices.Contains(starts, "/nfeProc/NFe/infNFe/empty") || starts[0] != "/nfeProc" {
		t.Errorf("start paths = %v", starts)
	}
	if id != "NFe123" {
		t.Errorf("infNFe Id = %q, want NFe123", id)
	}
	want := map[string]string{
		"/nfeProc/NFe/infNFe/ide/nNF":             "42",
		"/nfeProc/NFe/infNFe/det/imposto/vICMS":   "1.00",
		"/nfeProc/NFe/infNFe/total/ICMSTot/vICMS": "1.00",
	}
	if len(texts) != len(want) {
		t.Errorf("text paths = %v, want %v", texts, want)
	}
	for path, value := range want {
		if texts[path] != value {
			t.Errorf("text at %s = %q, want %q", path, texts[path], value)
		}
	}
}

func TestWalkErrors(t *testing.T) {
	noText := func(string, string) error { return nil }
	if err := xmlwalk.Walk([]byte("  \n"), nil, noText); err == nil {
		t.Error("Walk of a blank document succeeded")
	}
	if err := xmlwalk.Walk([]byte("<a><b></a>"), nil, noText); err == nil {
		t.Error("Walk of malformed XML succeeded")
	}

	stop := errors.New("stop")
	calls := 0
	err := xmlwalk.Walk([]byte(sample), nil, func(string, string) error {
		calls++
		return stop
	})
	if !errors.Is(err, stop) || calls != 1 {
		t.Errorf("Walk = %v after %d calls, want the onText error after 1", err, calls)
	}
}

func TestHasAnySuffix(t *testing.T) {
	path := "/nfeProc/NFe/infNFe/total/ICMSTot/vICMS"
	if !xmlwalk.HasAnySuffix(path, "/det/imposto/vICMS", "/ICMSTot/vICMS") {
		t.Error("HasAnySuffix missed /ICMSTot/vICMS")
	}
	if xmlwalk.HasAnySuffix(path, "/imposto/vICMS") || xmlwalk.HasAnySuffix(path) {
		t.Error("HasAnySuffix matched a suffix the path lacks")
	}
}

func TestAttrValueAbsent(t *testing.T) {
	attrs := []xml.Attr{{Name: xml.Name{Local: "versao"}, Value: " 4.00 "}}
	if got := xmlwalk.AttrValue(attrs, "versao"); got != "4.00" {
		t.Errorf("AttrValue(versao) = %q, want 4.00", got)
	}
	if got := xmlwalk.AttrValue(attrs, "Id"); got != "" {
		t.Errorf("AttrValue(Id) = %q, want empty", got)
	}
}
