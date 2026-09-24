package nfe

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const (
	keyProcNFe      = "35260911222333000181550010000012341123456787"
	keyCancelada    = "35260911222333000181550010000012351234567894"
	keyDenegada     = "35260911222333000181550010000012361345678900"
	keyAlfanumerico = "41260912ABC34501DE35550020000000771456789019"

	cnpjEmitente      = "11222333000181"
	cnpjMock          = "70860312000150"
	cnpjTransportador = "12345678000195"
	cnpjAutorizado    = "45678901000175"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func assertTimePtr(t *testing.T, field string, got *time.Time, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %s", field, want)
		return
	}
	if !got.Equal(mustTime(t, want)) {
		t.Errorf("%s = %s, want %s", field, got.Format(time.RFC3339), want)
	}
}

// TestFixturesHaveValidCheckDigits keeps the fixtures honest: every CNPJ and
// access key in them must pass validation.
func TestFixturesHaveValidCheckDigits(t *testing.T) {
	cnpjRe := regexp.MustCompile(`<(?:CNPJ|CNPJDest)>([^<]+)<`)
	keyRe := regexp.MustCompile(`<chNFe>([^<]+)<|Id="NFe([^"]+)"`)

	files, err := filepath.Glob(filepath.Join("testdata", "*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file) // #nosec G304 -- path comes from the testdata glob.
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range cnpjRe.FindAllSubmatch(data, -1) {
			if err := cnpj.Validate(string(m[1])); err != nil {
				t.Errorf("%s: CNPJ %s: %v", file, m[1], err)
			}
		}
		for _, m := range keyRe.FindAllSubmatch(data, -1) {
			key := string(m[1]) + string(m[2])
			if _, err := dfe.ParseAccessKey(key); err != nil {
				t.Errorf("%s: key %s: %v", file, key, err)
			}
		}
	}
}

func TestParseResNFe(t *testing.T) {
	doc, err := ParseResNFe(readFixture(t, "resnfe.xml"))
	if err != nil {
		t.Fatalf("ParseResNFe: %v", err)
	}

	want := Document{
		ChaveAcesso:   keyProcNFe,
		Modelo:        "55",
		Serie:         "1",
		Numero:        "1234",
		Competence:    "2026-09",
		Protocolo:     "135260000000001",
		EmitenteCNPJ:  cnpjEmitente,
		EmitenteName:  "DISTRIBUIDORA FICTICIA DE PECAS LTDA",
		EmitenteIE:    "111222333444",
		EmitenteUF:    "SP",
		TpNF:          "1",
		TotalValue:    nfse.NewMoneyFromCents(125050),
		Situacao:      SituacaoAutorizada,
		Completeness:  CompletenessResumo,
		LayoutVersion: "1.01",
	}
	assertDocumentFields(t, doc, want)
	if !doc.IssueDate.Equal(mustTime(t, "2026-09-01T09:15:00-03:00")) {
		t.Errorf("IssueDate = %s", doc.IssueDate)
	}
	assertTimePtr(t, "AuthorizedAt", doc.AuthorizedAt, "2026-09-01T09:15:42-03:00")
	if doc.DestinatarioCNPJ != "" || doc.ICMSValue != 0 {
		t.Errorf("resumo should not carry destinatário or ICMS, got %q / %d", doc.DestinatarioCNPJ, doc.ICMSValue)
	}
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
}

func TestParseResNFeCancelada(t *testing.T) {
	doc, err := ParseResNFe(readFixture(t, "resnfe-cancelada.xml"))
	if err != nil {
		t.Fatalf("ParseResNFe: %v", err)
	}
	if doc.Situacao != SituacaoCancelada {
		t.Errorf("Situacao = %q, want cancelada", doc.Situacao)
	}
	// dhEmi 2026-08-31T23:40-03:00 is 2026-09-01 in UTC; competence follows
	// the offset written in the document.
	if doc.Competence != "2026-08" {
		t.Errorf("Competence = %q, want 2026-08", doc.Competence)
	}
}

func TestParseResNFeRejects(t *testing.T) {
	valid := string(readFixture(t, "resnfe.xml"))
	tests := []struct {
		name string
		data string
	}{
		{"invalid xml fixture", string(readFixture(t, "invalid.xml"))},
		{"empty", "  "},
		{"malformed", "<resNFe><chNFe>"},
		{"missing chNFe", strings.Replace(valid, "<chNFe>"+keyProcNFe+"</chNFe>", "", 1)},
		{"bad key check digit", strings.Replace(valid, keyProcNFe, keyProcNFe[:43]+"8", 1)},
		{"unknown cSitNFe", strings.Replace(valid, "<cSitNFe>1</cSitNFe>", "<cSitNFe>9</cSitNFe>", 1)},
		{"missing cSitNFe", strings.Replace(valid, "<cSitNFe>1</cSitNFe>", "", 1)},
		{"bad vNF", strings.Replace(valid, "<vNF>1250.50</vNF>", "<vNF>1.250,50</vNF>", 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseResNFe([]byte(tt.data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseProcNFe(t *testing.T) {
	doc, err := ParseProcNFe(readFixture(t, "procnfe.xml"))
	if err != nil {
		t.Fatalf("ParseProcNFe: %v", err)
	}

	want := Document{
		ChaveAcesso:       keyProcNFe,
		Modelo:            "55",
		Serie:             "1",
		Numero:            "1234",
		Competence:        "2026-09",
		Protocolo:         "135260000000001",
		EmitenteCNPJ:      cnpjEmitente,
		EmitenteName:      "DISTRIBUIDORA FICTICIA DE PECAS LTDA",
		EmitenteIE:        "111222333444",
		EmitenteUF:        "SP",
		DestinatarioCNPJ:  cnpjMock,
		DestinatarioName:  "EMPRESA MOCK LTDA",
		TransportadorCNPJ: cnpjTransportador,
		TpNF:              "1",
		FinNFe:            "1",
		NatOp:             "VENDA DE MERCADORIA",
		// Items carry vICMS 180.00 and 24.00 and vIPI 50.00 and 0.50; only
		// the ICMSTot values may land here.
		TotalValue:    nfse.NewMoneyFromCents(125050),
		ICMSValue:     nfse.NewMoneyFromCents(20400),
		IPIValue:      nfse.NewMoneyFromCents(5050),
		Situacao:      SituacaoAutorizada,
		Completeness:  CompletenessCompleta,
		LayoutVersion: "4.00",
	}
	assertDocumentFields(t, doc, want)
	if !slices.Equal(doc.AutorizadosCNPJ, []string{cnpjAutorizado}) {
		t.Errorf("AutorizadosCNPJ = %v, want [%s]", doc.AutorizadosCNPJ, cnpjAutorizado)
	}
	assertTimePtr(t, "AuthorizedAt", doc.AuthorizedAt, "2026-09-01T09:15:42-03:00")
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
}

// TestParseProcNFeItemTaxesDoNotOverwriteTotals puts total before det, so a
// parser matching a bare "/vICMS" suffix would end with the last item's value.
func TestParseProcNFeItemTaxesDoNotOverwriteTotals(t *testing.T) {
	data := `<nfeProc><NFe><infNFe Id="NFe` + keyProcNFe + `" versao="4.00">
<ide><mod>55</mod><serie>1</serie><nNF>1234</nNF><dhEmi>2026-09-01T09:15:00-03:00</dhEmi></ide>
<total><ICMSTot><vICMS>204.00</vICMS><vIPI>50.50</vIPI><vNF>1250.50</vNF></ICMSTot></total>
<det nItem="1"><imposto><ICMS><ICMS00><vICMS>180.00</vICMS></ICMS00></ICMS><IPI><IPITrib><vIPI>50.00</vIPI></IPITrib></IPI></imposto></det>
<det nItem="2"><imposto><ICMS><ICMS00><vICMS>24.00</vICMS></ICMS00></ICMS><IPI><IPITrib><vIPI>0.50</vIPI></IPITrib></IPI></imposto></det>
</infNFe></NFe><protNFe><infProt><chNFe>` + keyProcNFe + `</chNFe><cStat>100</cStat></infProt></protNFe></nfeProc>`

	doc, err := ParseProcNFe([]byte(data))
	if err != nil {
		t.Fatalf("ParseProcNFe: %v", err)
	}
	if doc.ICMSValue.Cents() != 20400 || doc.IPIValue.Cents() != 5050 || doc.TotalValue.Cents() != 125050 {
		t.Errorf("totals = ICMS %d, IPI %d, NF %d; want 20400, 5050, 125050",
			doc.ICMSValue.Cents(), doc.IPIValue.Cents(), doc.TotalValue.Cents())
	}
}

func TestParseProcNFeDenegada(t *testing.T) {
	doc, err := ParseProcNFe(readFixture(t, "procnfe-denegada.xml"))
	if err != nil {
		t.Fatalf("ParseProcNFe: %v", err)
	}
	if doc.Situacao != SituacaoDenegada {
		t.Errorf("Situacao = %q, want denegada", doc.Situacao)
	}
	if doc.ChaveAcesso != keyDenegada {
		t.Errorf("ChaveAcesso = %q", doc.ChaveAcesso)
	}
	if doc.TransportadorCNPJ != "" || len(doc.AutorizadosCNPJ) != 0 {
		t.Errorf("unexpected transportador/autorizados: %q %v", doc.TransportadorCNPJ, doc.AutorizadosCNPJ)
	}
}

func TestParseProcNFeAlfanumerico(t *testing.T) {
	doc, err := ParseProcNFe(readFixture(t, "procnfe-alfanumerico.xml"))
	if err != nil {
		t.Fatalf("ParseProcNFe: %v", err)
	}
	if doc.ChaveAcesso != keyAlfanumerico {
		t.Errorf("ChaveAcesso = %q", doc.ChaveAcesso)
	}
	if doc.EmitenteCNPJ != "12ABC34501DE35" || doc.EmitenteCNPJ != doc.ChaveAcesso.EmitenteCNPJ() {
		t.Errorf("EmitenteCNPJ = %q, key slot %q", doc.EmitenteCNPJ, doc.ChaveAcesso.EmitenteCNPJ())
	}
	if doc.EmitenteUF != "PR" || doc.Serie != "2" || doc.Numero != "77" {
		t.Errorf("UF/serie/numero = %q/%q/%q", doc.EmitenteUF, doc.Serie, doc.Numero)
	}
}

func TestParseProcNFeKeySources(t *testing.T) {
	valid := string(readFixture(t, "procnfe.xml"))
	protChave := "<chNFe>" + keyProcNFe + "</chNFe>"

	t.Run("falls back to infNFe Id", func(t *testing.T) {
		doc, err := ParseProcNFe([]byte(strings.Replace(valid, protChave, "", 1)))
		if err != nil {
			t.Fatalf("ParseProcNFe: %v", err)
		}
		if doc.ChaveAcesso != keyProcNFe {
			t.Errorf("ChaveAcesso = %q", doc.ChaveAcesso)
		}
		if len(doc.ParseWarnings) != 1 {
			t.Errorf("expected one fallback warning, got %v", doc.ParseWarnings)
		}
	})

	t.Run("rejects mismatched keys", func(t *testing.T) {
		data := strings.Replace(valid, protChave, "<chNFe>"+keyDenegada+"</chNFe>", 1)
		if _, err := ParseProcNFe([]byte(data)); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestParseProcNFeRejects(t *testing.T) {
	valid := string(readFixture(t, "procnfe.xml"))
	tests := []struct {
		name string
		data string
	}{
		{"invalid xml fixture", string(readFixture(t, "invalid.xml"))},
		{"resumo is not a procNFe", string(readFixture(t, "resnfe.xml"))},
		{"missing protNFe cStat", strings.Replace(valid, "<cStat>100</cStat>", "", 1)},
		{"unsupported protNFe cStat", strings.Replace(valid, "<cStat>100</cStat>", "<cStat>539</cStat>", 1)},
		{"bad total", strings.Replace(valid, "<vNF>1250.50</vNF>", "<vNF>abc</vNF>", 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseProcNFe([]byte(tt.data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseResEvento(t *testing.T) {
	ev, err := ParseResEvento(readFixture(t, "resevento-cancelamento.xml"))
	if err != nil {
		t.Fatalf("ParseResEvento: %v", err)
	}
	want := Event{
		ChaveAcesso:  keyCancelada,
		TpEvento:     TpEventoCancelamento,
		Type:         EventTypeCancelamento,
		NSeqEvento:   1,
		Protocolo:    "135260000000010",
		AutorCNPJ:    cnpjEmitente,
		Description:  "Cancelamento",
		Completeness: CompletenessResumo,
		Registered:   true,
	}
	assertEventFields(t, ev, want)
	assertTimePtr(t, "EventAt", ev.EventAt, "2026-09-01T08:00:00-03:00")
	assertTimePtr(t, "RegisteredAt", ev.RegisteredAt, "2026-09-01T08:00:03-03:00")
}

func TestParseProcEventoNFe(t *testing.T) {
	tests := []struct {
		fixture        string
		want           Event
		wantEventAt    string
		wantRegistered string
	}{
		{
			fixture: "proceventonfe-cancelamento.xml",
			want: Event{
				ChaveAcesso:   keyCancelada,
				TpEvento:      TpEventoCancelamento,
				Type:          EventTypeCancelamento,
				NSeqEvento:    1,
				Protocolo:     "135260000000010", // retEvento nProt, not the NF-e nProt inside detEvento
				AutorCNPJ:     cnpjEmitente,
				Description:   "Cancelamento",
				Justificativa: "Erro na digitacao dos valores da nota",
				Completeness:  CompletenessCompleta,
				Registered:    true,
				CStat:         "135",
				XMotivo:       "Evento registrado e vinculado a NF-e",
			},
			wantEventAt:    "2026-09-01T08:00:00-03:00",
			wantRegistered: "2026-09-01T08:00:03-03:00",
		},
		{
			fixture: "proceventonfe-ciencia.xml",
			want: Event{
				ChaveAcesso:  keyProcNFe,
				TpEvento:     TpEventoCiencia,
				Type:         EventTypeCiencia,
				NSeqEvento:   1,
				Protocolo:    "891260000000001",
				AutorCNPJ:    cnpjMock,
				Description:  "Ciencia da Operacao",
				Completeness: CompletenessCompleta,
				Registered:   true,
				CStat:        "135",
				XMotivo:      "Evento registrado e vinculado a NF-e",
			},
			wantEventAt:    "2026-09-02T10:00:00-03:00",
			wantRegistered: "2026-09-02T10:00:05-03:00",
		},
		{
			fixture: "proceventonfe-cce.xml",
			want: Event{
				ChaveAcesso:  keyProcNFe,
				TpEvento:     TpEventoCCe,
				Type:         EventTypeCartaCorrecao,
				NSeqEvento:   1,
				Protocolo:    "135260000000020",
				AutorCNPJ:    cnpjEmitente,
				Description:  "Carta de Correcao",
				Correcao:     "Corrigir o endereco de entrega para Rua Ficticia, 150",
				Completeness: CompletenessCompleta,
				Registered:   true,
				CStat:        "135",
				XMotivo:      "Evento registrado e vinculado a NF-e",
			},
			wantEventAt:    "2026-09-03T14:20:00-03:00",
			wantRegistered: "2026-09-03T14:20:04-03:00",
		},
		{
			// Public SEFAZ-accepted sample from the sped-nfe docs.
			fixture: "proceventonfe-ciencia-real.xml",
			want: Event{
				ChaveAcesso:  "35170349607369000156550010000229481398694060",
				TpEvento:     TpEventoCiencia,
				Type:         EventTypeCiencia,
				NSeqEvento:   1,
				Protocolo:    "891170419030368",
				AutorCNPJ:    "69161982000108",
				Description:  "Ciencia da Operacao",
				Completeness: CompletenessCompleta,
				Registered:   true,
				CStat:        "135",
				XMotivo:      "Evento registrado e vinculado a NF-e",
			},
			wantEventAt:    "2017-03-18T10:30:58-03:00",
			wantRegistered: "2017-03-18T10:36:00-03:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			ev, err := ParseProcEventoNFe(readFixture(t, tt.fixture))
			if err != nil {
				t.Fatalf("ParseProcEventoNFe: %v", err)
			}
			assertEventFields(t, ev, tt.want)
			assertTimePtr(t, "EventAt", ev.EventAt, tt.wantEventAt)
			assertTimePtr(t, "RegisteredAt", ev.RegisteredAt, tt.wantRegistered)
			if len(ev.ParseWarnings) != 0 {
				t.Errorf("unexpected warnings: %v", ev.ParseWarnings)
			}
		})
	}
}

func TestParseProcEventoNFeNotRegistered(t *testing.T) {
	data := strings.Replace(string(readFixture(t, "proceventonfe-ciencia.xml")),
		"<cStat>135</cStat><xMotivo>Evento registrado e vinculado a NF-e</xMotivo>",
		"<cStat>573</cStat><xMotivo>Rejeicao: Duplicidade de Evento</xMotivo>", 1)

	ev, err := ParseProcEventoNFe([]byte(data))
	if err != nil {
		t.Fatalf("ParseProcEventoNFe: %v", err)
	}
	if ev.Registered || ev.Protocolo != "" || ev.RegisteredAt != nil {
		t.Errorf("rejected event should not look registered: %+v", ev)
	}
	if len(ev.ParseWarnings) != 1 || !strings.Contains(ev.ParseWarnings[0], "573") {
		t.Errorf("ParseWarnings = %v, want one naming cStat 573", ev.ParseWarnings)
	}
}

func TestParseEventoRejects(t *testing.T) {
	valid := string(readFixture(t, "proceventonfe-ciencia.xml"))
	tests := []struct {
		name string
		data string
	}{
		{"invalid xml fixture", string(readFixture(t, "invalid.xml"))},
		{"missing tpEvento", strings.Replace(valid, "<tpEvento>210210</tpEvento><nSeqEvento>1</nSeqEvento><verEvento>", "<nSeqEvento>1</nSeqEvento><verEvento>", 1)},
		{"bad nSeqEvento", strings.Replace(valid, "<nSeqEvento>1</nSeqEvento><verEvento>", "<nSeqEvento>x</nSeqEvento><verEvento>", 1)},
		{"bad key", strings.ReplaceAll(valid, keyProcNFe, keyProcNFe[:43]+"0")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseProcEventoNFe([]byte(tt.data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	if _, err := ParseResEvento(readFixture(t, "invalid.xml")); err == nil {
		t.Error("ParseResEvento: expected an error for invalid.xml")
	}
}

// assertDocumentFields compares everything except the times and slices,
// which the callers check.
func assertDocumentFields(t *testing.T, got, want Document) {
	t.Helper()
	got.IssueDate, want.IssueDate = time.Time{}, time.Time{}
	got.AuthorizedAt, want.AuthorizedAt = nil, nil
	got.AutorizadosCNPJ, want.AutorizadosCNPJ = nil, nil
	got.ParseWarnings, want.ParseWarnings = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("document mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

// assertEventFields compares everything except EventAt, RegisteredAt and
// ParseWarnings, which the callers check.
func assertEventFields(t *testing.T, got, want Event) {
	t.Helper()
	got.EventAt, want.EventAt = nil, nil
	got.RegisteredAt, want.RegisteredAt = nil, nil
	got.ParseWarnings, want.ParseWarnings = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("event mismatch\n got: %+v\nwant: %+v", got, want)
	}
}
