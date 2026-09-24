package cte

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
)

const (
	keyProcCTe   = "35260912345678000195570010000001011123456784" // toma3 = 3
	keyToma4     = "35260912345678000195570010000001021234567891"
	keyV200      = "35260912345678000195570010000001031345678907"
	keyCTeOS     = "35260912345678000195670010000001041456789014"
	keyGTVe      = "35260912345678000195640010000001051567890127"
	keyCTeSimp   = "35260912345678000195570020000001061678901234"
	nfeKeyA      = "35260911222333000181550010000012341123456787"
	nfeKeyB      = "35260911222333000181550010000012351234567894"
	nfeKeyToma4  = "35260911222333000181550010000012361345678900"
	nfeKeyV200   = "35260970860312000150550010000000101000000106"
	nfeKeyCTeSim = "35260911222333000181550010000012401240000019"
)

var (
	emitente = Party{CNPJ: cnpjTransportador, Name: "TRANSPORTADORA FICTICIA LTDA", IE: "123456789012", UF: "SP"}
	mockDest = Party{CNPJ: cnpjMock, Name: "EMPRESA MOCK LTDA", IE: "555666777888", UF: "RJ"}
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

func parseFixture(t *testing.T, name string) Document {
	t.Helper()
	doc, err := ParseProcCTe(readFixture(t, name))
	if err != nil {
		t.Fatalf("ParseProcCTe(%s): %v", name, err)
	}
	return doc
}

// TestFixturesHaveValidCheckDigits keeps the fixtures honest: every CNPJ and
// access key in them must pass validation, except the keys masked with 9s.
func TestFixturesHaveValidCheckDigits(t *testing.T) {
	cnpjRe := regexp.MustCompile(`<CNPJ>([^<]+)<`)
	keyRe := regexp.MustCompile(`<(?:chCTe|chave|chNFe)>([^<]+)<|Id="CTe([^"]+)"`)

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
			if strings.Trim(key, "9") == "" {
				continue
			}
			if _, err := dfe.ParseAccessKey(key); err != nil {
				t.Errorf("%s: key %s: %v", file, key, err)
			}
		}
	}
}

func TestParseProcCTe(t *testing.T) {
	doc := parseFixture(t, "procte.xml")

	want := Document{
		ChaveAcesso:      keyProcCTe,
		TpAmb:            "1",
		Modelo:           "57",
		TipoDocumento:    TipoDocumentoCTe,
		Serie:            "1",
		Numero:           "101",
		CFOP:             "6353",
		NatOp:            "PRESTACAO DE SERVICO DE TRANSPORTE",
		Competence:       "2026-09",
		Protocolo:        "135260000000101",
		TpCTe:            "0",
		TpServ:           "0",
		Modal:            "01",
		MunIni:           Municipio{Codigo: "3550308", Nome: "SAO PAULO", UF: "SP"},
		MunFim:           Municipio{Codigo: "3304557", Nome: "RIO DE JANEIRO", UF: "RJ"},
		Emitente:         emitente,
		Remetente:        Party{CNPJ: cnpjRemetente, Name: "DISTRIBUIDORA FICTICIA DE PECAS LTDA", IE: "111222333444", UF: "SP"},
		Expedidor:        Party{CNPJ: cnpjExpedidor, Name: "ARMAZEM FICTICIO LTDA", IE: "135792468000", UF: "SP"},
		Recebedor:        Party{CNPJ: cnpjRecebedor, Name: "CENTRO DE DISTRIBUICAO FICTICIO LTDA", IE: "24681357", UF: "RJ"},
		Destinatario:     mockDest,
		Tomador:          mockDest,
		TomadorIndicador: "3",
		AutorizadosCNPJ:  []string{cnpjAutorizado},
		// Comp values (1200.00 and 300.00) must not land in any total.
		TotalValue:          dfe.NewMoneyFromCents(150000),
		ReceivableValue:     dfe.NewMoneyFromCents(145000),
		ICMSValue:           dfe.NewMoneyFromCents(18000),
		TotTribValue:        dfe.NewMoneyFromCents(25035),
		CargaValue:          dfe.NewMoneyFromCents(2500000),
		ProdutoPredominante: "PECAS AUTOMOTIVAS",
		NFeChaves:           []string{nfeKeyA, nfeKeyB},
		Situacao:            SituacaoAutorizada,
		LayoutVersion:       "4.00",
	}
	assertDocument(t, doc, want)
	if !doc.IssueDate.Equal(mustTime(t, "2026-09-02T08:30:00-03:00")) {
		t.Errorf("IssueDate = %s", doc.IssueDate)
	}
	assertTimePtr(t, "AuthorizedAt", doc.AuthorizedAt, "2026-09-02T08:30:41-03:00")
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
}

func TestParseProcCTeToma4AndMaskedChaves(t *testing.T) {
	doc := parseFixture(t, "procte-toma4.xml")

	terceiro := Party{CNPJ: cnpjTerceiro, Name: "OPERADOR LOGISTICO FICTICIO LTDA", IE: "112233440000", UF: "PR"}
	if doc.Tomador != terceiro || doc.TomadorIndicador != "4" {
		t.Errorf("Tomador = %+v (toma %q), want %+v (toma 4)", doc.Tomador, doc.TomadorIndicador, terceiro)
	}
	if !reflect.DeepEqual(doc.AutorizadosCNPJ, []string{cnpjMock, cnpjAutorizado}) {
		t.Errorf("AutorizadosCNPJ = %v", doc.AutorizadosCNPJ)
	}
	if !reflect.DeepEqual(doc.NFeChaves, []string{nfeKeyToma4}) {
		t.Errorf("NFeChaves = %v, want only the unmasked key", doc.NFeChaves)
	}
	if !doc.MaskedKeys {
		t.Error("MaskedKeys = false, want true for a copy with masked keys")
	}
	if want := []string{"2 NF-e chaves masked with 9s were dropped"}; !reflect.DeepEqual(doc.ParseWarnings, want) {
		t.Errorf("ParseWarnings = %v, want %v", doc.ParseWarnings, want)
	}
	if doc.ICMSValue.Cents() != 4560 {
		t.Errorf("ICMSValue from ICMS90 = %d, want 4560", doc.ICMSValue.Cents())
	}
	if got := ClassifyParticipation(&doc, cnpjMock); got.CompanyRole != CompanyRoleAutorizado {
		t.Errorf("mock participation = %+v, want autorizado", got)
	}
}

func TestParseProcCTeLayout200Toma03(t *testing.T) {
	doc := parseFixture(t, "procte-v200-toma03.xml")

	mockRem := Party{CNPJ: cnpjMock, Name: "EMPRESA MOCK LTDA", IE: "555666777888", UF: "SP"}
	if doc.Tomador != mockRem || doc.TomadorIndicador != "0" {
		t.Errorf("Tomador = %+v (toma %q), want the remetente", doc.Tomador, doc.TomadorIndicador)
	}
	if doc.ChaveAcesso != keyV200 || doc.LayoutVersion != "2.00" || doc.TpAmb != "2" {
		t.Errorf("(chave, layout, tpAmb) = (%q, %q, %q), want (%s, 2.00, 2)", doc.ChaveAcesso, doc.LayoutVersion, doc.TpAmb, keyV200)
	}
	if !reflect.DeepEqual(doc.NFeChaves, []string{nfeKeyV200}) {
		t.Errorf("NFeChaves = %v, want the key under rem", doc.NFeChaves)
	}
	if doc.ICMSValue != 0 || doc.TotalValue.Cents() != 9510 {
		t.Errorf("(ICMS, total) = (%d, %d), want (0, 9510)", doc.ICMSValue.Cents(), doc.TotalValue.Cents())
	}
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
	got := ClassifyParticipation(&doc, cnpjMock)
	if got.CompanyRole != CompanyRoleTomador || !reflect.DeepEqual(got.Papeis, []CompanyRole{CompanyRoleTomador, CompanyRoleRemetente}) {
		t.Errorf("mock participation = %+v, want tomador and remetente", got)
	}
}

func TestParseProcCTeOS(t *testing.T) {
	doc := parseFixture(t, "procteos.xml")

	want := Document{
		ChaveAcesso:     keyCTeOS,
		TpAmb:           "1",
		Modelo:          "67",
		TipoDocumento:   TipoDocumentoCTeOS,
		Serie:           "1",
		Numero:          "104",
		CFOP:            "5357",
		NatOp:           "TRANSPORTE DE PESSOAS",
		Competence:      "2026-09",
		Protocolo:       "135260000000104",
		TpCTe:           "0",
		TpServ:          "6",
		Modal:           "01",
		MunIni:          Municipio{Codigo: "3550308", Nome: "SAO PAULO", UF: "SP"},
		MunFim:          Municipio{Codigo: "3518800", Nome: "GUARULHOS", UF: "SP"},
		Emitente:        emitente,
		Tomador:         mockDest,
		TotalValue:      dfe.NewMoneyFromCents(200000),
		ReceivableValue: dfe.NewMoneyFromCents(200000),
		ICMSValue:       dfe.NewMoneyFromCents(24000),
		TotTribValue:    dfe.NewMoneyFromCents(31000),
		Situacao:        SituacaoAutorizada,
		LayoutVersion:   "4.00",
	}
	assertDocument(t, doc, want)
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
}

func TestParseProcGTVe(t *testing.T) {
	doc := parseFixture(t, "procgtve.xml")

	want := Document{
		ChaveAcesso:      keyGTVe,
		TpAmb:            "1",
		Modelo:           "64",
		TipoDocumento:    TipoDocumentoGTVe,
		Serie:            "1",
		Numero:           "105",
		CFOP:             "5359",
		NatOp:            "TRANSPORTE DE VALORES",
		Competence:       "2026-09",
		Protocolo:        "135260000000105",
		TpCTe:            "4",
		TpServ:           "9",
		Modal:            "01",
		MunIni:           Municipio{Codigo: "3550308", Nome: "SAO PAULO", UF: "SP"},
		MunFim:           Municipio{Codigo: "3509502", Nome: "CAMPINAS", UF: "SP"},
		Emitente:         emitente,
		Remetente:        Party{CNPJ: cnpjRemetente, Name: "DISTRIBUIDORA FICTICIA DE PECAS LTDA", IE: "111222333444", UF: "SP"},
		Destinatario:     Party{CNPJ: cnpjMock, Name: "EMPRESA MOCK LTDA", IE: "555666777888", UF: "SP"},
		Tomador:          Party{CNPJ: cnpjMock, Name: "EMPRESA MOCK LTDA", IE: "555666777888", UF: "SP"},
		TomadorIndicador: "1", // destinatário on a GTV-e
		CargaValue:       dfe.NewMoneyFromCents(5125075),
		Situacao:         SituacaoAutorizada,
		LayoutVersion:    "4.00",
	}
	assertDocument(t, doc, want)
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
}

func TestParseProcCTeSimp(t *testing.T) {
	doc := parseFixture(t, "proctesimp.xml")

	want := Document{
		ChaveAcesso:      keyCTeSimp,
		TpAmb:            "1",
		Modelo:           "57",
		TipoDocumento:    TipoDocumentoCTeSimplificado,
		Serie:            "2",
		Numero:           "106",
		CFOP:             "6353",
		NatOp:            "PRESTACAO DE SERVICO DE TRANSPORTE",
		Competence:       "2026-09",
		Protocolo:        "135260000000106",
		TpCTe:            "5",
		TpServ:           "0",
		Modal:            "01",
		MunIni:           Municipio{Codigo: "3550308", Nome: "SAO PAULO", UF: "SP"}, // first det
		MunFim:           Municipio{Codigo: "3304557", Nome: "RIO DE JANEIRO", UF: "RJ"},
		Emitente:         emitente,
		Tomador:          mockDest,
		TomadorIndicador: "3",
		// det vPrest (600.00 and 400.00) must not land in the totals.
		TotalValue:          dfe.NewMoneyFromCents(100000),
		ReceivableValue:     dfe.NewMoneyFromCents(100000),
		ICMSValue:           dfe.NewMoneyFromCents(12000), // ICMSOutraUF
		CargaValue:          dfe.NewMoneyFromCents(780000),
		ProdutoPredominante: "CAIXAS",
		NFeChaves:           []string{nfeKeyCTeSim}, // listed in both det, kept once
		Situacao:            SituacaoAutorizada,
		LayoutVersion:       "4.00",
	}
	assertDocument(t, doc, want)
	if len(doc.ParseWarnings) != 0 {
		t.Errorf("unexpected warnings: %v", doc.ParseWarnings)
	}
}

// TestResolveTomador covers each toma code against a document that has every
// party, and a pointer to a party missing from the document.
func TestResolveTomador(t *testing.T) {
	rem := Party{CNPJ: cnpjRemetente}
	exped := Party{CNPJ: cnpjExpedidor}
	receb := Party{CNPJ: cnpjRecebedor}
	dest := Party{CNPJ: cnpjMock}
	named := Party{CNPJ: cnpjTerceiro}

	tests := []struct {
		name  string
		tipo  TipoDocumento
		toma  string
		noExp bool
		want  Party
	}{
		{"CT-e remetente", TipoDocumentoCTe, "0", false, rem},
		{"CT-e expedidor", TipoDocumentoCTe, "1", false, exped},
		{"CT-e recebedor", TipoDocumentoCTe, "2", false, receb},
		{"CT-e destinatário", TipoDocumentoCTe, "3", false, dest},
		{"CT-e toma4", TipoDocumentoCTe, "4", false, named},
		{"CT-e pointer to a missing expedidor", TipoDocumentoCTe, "1", true, Party{}},
		{"GTV-e remetente", TipoDocumentoGTVe, "0", false, rem},
		{"GTV-e destinatário", TipoDocumentoGTVe, "1", false, dest},
		{"GTV-e tomaTerceiro", TipoDocumentoGTVe, "4", false, named},
		{"CT-e OS", TipoDocumentoCTeOS, "", false, named},
		{"CT-e Simplificado", TipoDocumentoCTeSimplificado, "0", false, named},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Document{
				TipoDocumento:    tt.tipo,
				TomadorIndicador: tt.toma,
				Remetente:        rem,
				Expedidor:        exped,
				Recebedor:        receb,
				Destinatario:     dest,
			}
			if tt.noExp {
				doc.Expedidor = Party{}
			}
			var warnings []string
			resolveTomador(&doc, named, &warnings)
			if doc.Tomador != tt.want {
				t.Errorf("Tomador = %+v, want %+v", doc.Tomador, tt.want)
			}
			if wantWarning := tt.want.CNPJ == ""; (len(warnings) == 1) != wantWarning {
				t.Errorf("warnings = %v, want a warning only when unresolved", warnings)
			}
		})
	}
}

func TestParseProcCTeKeySources(t *testing.T) {
	valid := string(readFixture(t, "procte.xml"))
	protChave := "<chCTe>" + keyProcCTe + "</chCTe>"

	t.Run("falls back to infCte Id", func(t *testing.T) {
		doc, err := ParseProcCTe([]byte(strings.Replace(valid, protChave, "", 1)))
		if err != nil {
			t.Fatalf("ParseProcCTe: %v", err)
		}
		if doc.ChaveAcesso != keyProcCTe {
			t.Errorf("ChaveAcesso = %q", doc.ChaveAcesso)
		}
		if len(doc.ParseWarnings) != 1 || !strings.Contains(doc.ParseWarnings[0], "infCte Id") {
			t.Errorf("ParseWarnings = %v, want one about the Id fallback", doc.ParseWarnings)
		}
	})
	t.Run("protocol and Id must agree", func(t *testing.T) {
		data := strings.Replace(valid, protChave, "<chCTe>"+keyToma4+"</chCTe>", 1)
		if _, err := ParseProcCTe([]byte(data)); err == nil {
			t.Fatal("expected an error for mismatched keys")
		}
	})
	t.Run("tpAmb falls back to the protocol", func(t *testing.T) {
		doc, err := ParseProcCTe([]byte(strings.Replace(valid, "<tpAmb>1</tpAmb>\n        <tpCTe>", "<tpCTe>", 1)))
		if err != nil {
			t.Fatalf("ParseProcCTe: %v", err)
		}
		if doc.TpAmb != "1" || len(doc.ParseWarnings) != 1 {
			t.Errorf("(tpAmb, warnings) = (%q, %v), want (1, one warning)", doc.TpAmb, doc.ParseWarnings)
		}
	})
}

func TestParseProcCTeSituacao(t *testing.T) {
	valid := string(readFixture(t, "procte.xml"))
	tests := []struct {
		cStat string
		want  Situacao
	}{
		{"100", SituacaoAutorizada},
		{"150", SituacaoAutorizada},
		{"110", SituacaoDenegada},
		{"301", SituacaoDenegada},
	}
	for _, tt := range tests {
		data := strings.Replace(valid, "<cStat>100</cStat>", "<cStat>"+tt.cStat+"</cStat>", 1)
		doc, err := ParseProcCTe([]byte(data))
		if err != nil {
			t.Fatalf("cStat %s: %v", tt.cStat, err)
		}
		if doc.Situacao != tt.want {
			t.Errorf("cStat %s: Situacao = %q, want %q", tt.cStat, doc.Situacao, tt.want)
		}
	}
}

func TestParseProcCTeRejects(t *testing.T) {
	valid := string(readFixture(t, "procte.xml"))
	tests := []struct {
		name string
		data string
	}{
		{"invalid xml fixture", string(readFixture(t, "invalid.xml"))},
		{"empty", "  "},
		{"malformed", "<cteProc><CTe><infCte>"},
		{"unknown wrapper", strings.NewReplacer("<CTe xmlns", "<MDFe xmlns", "</CTe>", "</MDFe>").Replace(valid)},
		{"missing both keys", strings.Replace(strings.Replace(valid, "<chCTe>"+keyProcCTe+"</chCTe>", "", 1), `Id="CTe`+keyProcCTe+`"`, "", 1)},
		{"bad key check digit", strings.ReplaceAll(valid, keyProcCTe, keyProcCTe[:43]+"5")},
		{"unknown cStat", strings.Replace(valid, "<cStat>100</cStat>", "<cStat>999</cStat>", 1)},
		{"missing cStat", strings.Replace(valid, "<cStat>100</cStat>", "", 1)},
		{"missing tpAmb", strings.ReplaceAll(valid, "<tpAmb>1</tpAmb>", "")},
		{"invalid tpAmb", strings.Replace(valid, "<tpAmb>1</tpAmb>\n        <tpCTe>", "<tpAmb>3</tpAmb><tpCTe>", 1)},
		{"bad vTPrest", strings.Replace(valid, "<vTPrest>1500.00</vTPrest>", "<vTPrest>1.500,00</vTPrest>", 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseProcCTe([]byte(tt.data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseProcEventoCTe(t *testing.T) {
	tests := []struct {
		fixture        string
		want           Event
		wantEventAt    string
		wantRegistered string
	}{
		{
			fixture: "proceventocte-cancelamento.xml",
			want: Event{
				ChaveAcesso:   keyProcCTe,
				TpAmb:         "1",
				COrgao:        "35",
				TpEvento:      TpEventoCancelamento,
				Type:          EventTypeCancelamento,
				NSeqEvento:    1,
				Protocolo:     "135260000000201",
				AutorCNPJ:     cnpjTransportador,
				Description:   "Cancelamento",
				Justificativa: "Erro na emissao do documento de transporte",
				Registered:    true,
				CStat:         "135",
				XMotivo:       "Evento registrado e vinculado a CT-e",
			},
			wantEventAt:    "2026-09-02T10:00:00-03:00",
			wantRegistered: "2026-09-02T10:00:04-03:00",
		},
		{
			fixture: "proceventocte-comprovante.xml",
			want: Event{
				ChaveAcesso: keyToma4,
				TpAmb:       "1",
				COrgao:      "35",
				TpEvento:    TpEventoComprovanteEntrega,
				Type:        EventTypeComprovanteEntrega,
				NSeqEvento:  1,
				Protocolo:   "135260000000202",
				AutorCNPJ:   cnpjTransportador,
				Description: "Comprovante de Entrega do CT-e",
				Registered:  true,
				CStat:       "135",
				XMotivo:     "Evento registrado e vinculado a CT-e",
			},
			wantEventAt:    "2026-09-04T16:20:00-03:00",
			wantRegistered: "2026-09-04T16:20:03-03:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			ev, err := ParseProcEventoCTe(readFixture(t, tt.fixture))
			if err != nil {
				t.Fatalf("ParseProcEventoCTe: %v", err)
			}
			assertEvent(t, ev, tt.want)
			assertTimePtr(t, "EventAt", ev.EventAt, tt.wantEventAt)
			assertTimePtr(t, "RegisteredAt", ev.RegisteredAt, tt.wantRegistered)
			if len(ev.ParseWarnings) != 0 {
				t.Errorf("unexpected warnings: %v", ev.ParseWarnings)
			}
		})
	}
}

// TestParseProcEventoCTeCartaCorrecao follows the layout of the 3.00
// procEventoCTe (no namespace on the root, two-digit nSeqEvento in
// retEventoCTe).
func TestParseProcEventoCTeCartaCorrecao(t *testing.T) {
	data := `<procEventoCTe versao="3.00">
<eventoCTe xmlns="http://www.portalfiscal.inf.br/cte" versao="3.00"><infEvento Id="ID1101103526091234567800019557001000000101112345678401">
<cOrgao>35</cOrgao><tpAmb>2</tpAmb><CNPJ>12345678000195</CNPJ><chCTe>` + keyProcCTe + `</chCTe>
<dhEvento>2026-09-03T13:30:09-03:00</dhEvento><tpEvento>110110</tpEvento><nSeqEvento>1</nSeqEvento>
<detEvento versaoEvento="3.00"><evCCeCTe><descEvento>Carta de Correcao</descEvento>
<infCorrecao><grupoAlterado>rem</grupoAlterado><campoAlterado>nro</campoAlterado><valorAlterado>170</valorAlterado></infCorrecao>
<infCorrecao><grupoAlterado>infQ</grupoAlterado><campoAlterado>qCarga</campoAlterado><valorAlterado>900.0000</valorAlterado><nroItemAlterado>2</nroItemAlterado></infCorrecao>
<xCondUso>A Carta de Correcao e disciplinada pelo Art. 58-B do CONVENIO/SINIEF 06/89</xCondUso>
</evCCeCTe></detEvento></infEvento></eventoCTe>
<retEventoCTe versao="3.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento><tpAmb>2</tpAmb><cOrgao>35</cOrgao>
<cStat>135</cStat><xMotivo>Evento registrado e vinculado a CT-e</xMotivo><chCTe>` + keyProcCTe + `</chCTe>
<tpEvento>110110</tpEvento><xEvento>Carta de Correcao</xEvento><nSeqEvento>01</nSeqEvento>
<dhRegEvento>2026-09-03T13:30:15-03:00</dhRegEvento><nProt>135260000000203</nProt></infEvento></retEventoCTe>
</procEventoCTe>`

	ev, err := ParseProcEventoCTe([]byte(data))
	if err != nil {
		t.Fatalf("ParseProcEventoCTe: %v", err)
	}
	if want := "rem.nro=170; infQ.qCarga[2]=900.0000"; ev.Correcao != want {
		t.Errorf("Correcao = %q, want %q", ev.Correcao, want)
	}
	if ev.Type != EventTypeCartaCorrecao || ev.TpAmb != "2" || !ev.Registered {
		t.Errorf("(type, tpAmb, registered) = (%q, %q, %v)", ev.Type, ev.TpAmb, ev.Registered)
	}
	if !strings.HasPrefix(ev.CondicaoUso, "A Carta de Correcao") || ev.Description != "Carta de Correcao" {
		t.Errorf("(CondicaoUso, Description) = (%q, %q)", ev.CondicaoUso, ev.Description)
	}
}

func TestParseProcEventoCTeObservacao(t *testing.T) {
	data := strings.Replace(string(readFixture(t, "proceventocte-cancelamento.xml")),
		"<xJust>Erro na emissao do documento de transporte</xJust>",
		"<indDesacordoOper>1</indDesacordoOper><xObs>Mercadoria nao foi entregue ao destinatario</xObs>", 1)
	data = strings.ReplaceAll(data, "110111", TpEventoPrestacaoDesacordo)

	ev, err := ParseProcEventoCTe([]byte(data))
	if err != nil {
		t.Fatalf("ParseProcEventoCTe: %v", err)
	}
	if ev.Type != EventTypePrestacaoDesacordo || ev.Observacao != "Mercadoria nao foi entregue ao destinatario" || ev.Justificativa != "" {
		t.Errorf("(type, xObs, xJust) = (%q, %q, %q)", ev.Type, ev.Observacao, ev.Justificativa)
	}
}

func TestParseProcEventoCTeNotRegistered(t *testing.T) {
	data := strings.Replace(string(readFixture(t, "proceventocte-cancelamento.xml")),
		"<cStat>135</cStat>", "<cStat>631</cStat>", 1)

	ev, err := ParseProcEventoCTe([]byte(data))
	if err != nil {
		t.Fatalf("ParseProcEventoCTe: %v", err)
	}
	if ev.Registered || ev.Protocolo != "" || ev.RegisteredAt != nil {
		t.Errorf("rejected event should not look registered: %+v", ev)
	}
	if len(ev.ParseWarnings) != 1 || !strings.Contains(ev.ParseWarnings[0], "631") {
		t.Errorf("ParseWarnings = %v, want one naming cStat 631", ev.ParseWarnings)
	}
}

func TestParseProcEventoCTeRejects(t *testing.T) {
	valid := string(readFixture(t, "proceventocte-cancelamento.xml"))
	tests := []struct {
		name string
		data string
	}{
		{"invalid xml fixture", string(readFixture(t, "invalid.xml"))},
		{"missing tpEvento", strings.Replace(valid, "<tpEvento>110111</tpEvento>\n      <nSeqEvento>", "<nSeqEvento>", 1)},
		{"bad nSeqEvento", strings.Replace(valid, "<nSeqEvento>1</nSeqEvento>\n      <detEvento", "<nSeqEvento>x</nSeqEvento><detEvento", 1)},
		{"tpEvento with a path", strings.ReplaceAll(valid, "<tpEvento>110111</tpEvento>", "<tpEvento>../../x</tpEvento>")},
		{"short tpEvento", strings.ReplaceAll(valid, "<tpEvento>110111</tpEvento>", "<tpEvento>11011</tpEvento>")},
		{"tpEvento with a letter", strings.ReplaceAll(valid, "<tpEvento>110111</tpEvento>", "<tpEvento>11011a</tpEvento>")},
		{"bad key", strings.ReplaceAll(valid, keyProcCTe, keyProcCTe[:43]+"0")},
		{"missing tpAmb", strings.ReplaceAll(valid, "<tpAmb>1</tpAmb>", "")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseProcEventoCTe([]byte(tt.data)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// assertDocument compares everything except the times and the warnings,
// which the callers check.
func assertDocument(t *testing.T, got, want Document) {
	t.Helper()
	got.IssueDate, want.IssueDate = time.Time{}, time.Time{}
	got.AuthorizedAt, want.AuthorizedAt = nil, nil
	got.ParseWarnings, want.ParseWarnings = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("document mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

// assertEvent compares everything except EventAt, RegisteredAt and
// ParseWarnings, which the callers check.
func assertEvent(t *testing.T, got, want Event) {
	t.Helper()
	got.EventAt, want.EventAt = nil, nil
	got.RegisteredAt, want.RegisteredAt = nil, nil
	got.ParseWarnings, want.ParseWarnings = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("event mismatch\n got: %+v\nwant: %+v", got, want)
	}
}
