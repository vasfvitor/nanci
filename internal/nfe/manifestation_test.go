package nfe

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestManifestationType(t *testing.T) {
	tests := []struct {
		tipo       ManifestationType
		tpEvento   string
		descEvento string
	}{
		{ManifestationCiencia, "210210", "Ciencia da Operacao"},
		{ManifestationConfirmacao, "210200", "Confirmacao da Operacao"},
		{ManifestationDesconhecimento, "210220", "Desconhecimento da Operacao"},
		{ManifestationNaoRealizada, "210240", "Operacao nao Realizada"},
	}
	for _, tt := range tests {
		t.Run(tt.tipo.String(), func(t *testing.T) {
			parsed, err := ParseManifestationType(string(tt.tipo))
			if err != nil || parsed != tt.tipo {
				t.Fatalf("ParseManifestationType(%q) = %q, %v", tt.tipo, parsed, err)
			}
			if got := tt.tipo.TpEvento(); got != tt.tpEvento {
				t.Errorf("TpEvento() = %q, want %q", got, tt.tpEvento)
			}
			if got := tt.tipo.DescEvento(); got != tt.descEvento {
				t.Errorf("DescEvento() = %q, want %q", got, tt.descEvento)
			}
			// The parser maps the code back to the same kind of event.
			if EventTypeFromTpEvento(tt.tpEvento) == EventTypeUnknown {
				t.Errorf("EventTypeFromTpEvento(%q) is unknown", tt.tpEvento)
			}
		})
	}

	if _, err := ParseManifestationType("cancelamento"); err == nil {
		t.Error("ParseManifestationType(cancelamento) should fail")
	}
	if ManifestationType("x").TpEvento() != "" || ManifestationType("x").DescEvento() != "" {
		t.Error("invalid type should have empty TpEvento and DescEvento")
	}
}

func TestManifestacaoFromEvents(t *testing.T) {
	at := func(day int) *time.Time {
		v := time.Date(2026, 9, day, 10, 0, 0, 0, time.UTC)
		return &v
	}
	event := func(tp EventType, author string, day int) Event {
		return Event{Type: tp, AutorCNPJ: author, RegisteredAt: at(day), Registered: true}
	}

	tests := []struct {
		name   string
		events []Event
		want   Manifestacao
	}{
		{"no events", nil, ManifestacaoNenhuma},
		{"only other events", []Event{event(EventTypeCancelamento, cnpjEmitente, 1), event(EventTypeCartaCorrecao, cnpjEmitente, 2)}, ManifestacaoNenhuma},
		{"ciencia", []Event{event(EventTypeCiencia, cnpjMock, 2)}, ManifestacaoCiencia},
		{"ciencia by another company is ignored", []Event{event(EventTypeCiencia, "70860312000231", 2)}, ManifestacaoNenhuma},
		{"conclusive beats ciencia", []Event{event(EventTypeCiencia, cnpjMock, 2), event(EventTypeConfirmacao, cnpjMock, 3)}, ManifestacaoConfirmada},
		{"conclusive beats a later ciencia", []Event{event(EventTypeDesconhecimento, cnpjMock, 3), event(EventTypeCiencia, cnpjMock, 4)}, ManifestacaoDesconhecida},
		{
			"latest conclusive wins regardless of order",
			[]Event{event(EventTypeNaoRealizada, cnpjMock, 9), event(EventTypeConfirmacao, cnpjMock, 5)},
			ManifestacaoNaoRealizada,
		},
		{
			"event time is used when registration time is missing",
			[]Event{
				{Type: EventTypeConfirmacao, AutorCNPJ: cnpjMock, EventAt: at(8), Registered: true},
				event(EventTypeDesconhecimento, cnpjMock, 6),
			},
			ManifestacaoConfirmada,
		},
		{
			"unregistered events are ignored",
			[]Event{event(EventTypeCiencia, cnpjMock, 2), {Type: EventTypeConfirmacao, AutorCNPJ: cnpjMock, RegisteredAt: at(3)}},
			ManifestacaoCiencia,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ManifestacaoFromEvents(tt.events, "70.860.312/0001-50"); got != tt.want {
				t.Errorf("ManifestacaoFromEvents = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestManifestacaoAndTimeFromEvents(t *testing.T) {
	at := func(day int) *time.Time {
		v := time.Date(2026, 9, day, 10, 0, 0, 0, time.UTC)
		return &v
	}
	event := func(tp EventType, day int) Event {
		return Event{Type: tp, AutorCNPJ: cnpjMock, RegisteredAt: at(day), Registered: true}
	}

	tests := []struct {
		name    string
		events  []Event
		want    Manifestacao
		wantDay int // 0 means nil
	}{
		{"no events", nil, ManifestacaoNenhuma, 0},
		{"earliest ciencia", []Event{event(EventTypeCiencia, 4), event(EventTypeCiencia, 2)}, ManifestacaoCiencia, 2},
		{"latest conclusive", []Event{event(EventTypeCiencia, 2), event(EventTypeConfirmacao, 5), event(EventTypeDesconhecimento, 7)}, ManifestacaoDesconhecida, 7},
		{"ciencia without time", []Event{{Type: EventTypeCiencia, AutorCNPJ: cnpjMock, Registered: true}}, ManifestacaoCiencia, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotAt := ManifestacaoAndTimeFromEvents(tt.events, cnpjMock)
			if got != tt.want {
				t.Errorf("manifestacao = %q, want %q", got, tt.want)
			}
			switch {
			case tt.wantDay == 0 && gotAt != nil:
				t.Errorf("time = %s, want nil", gotAt)
			case tt.wantDay != 0 && (gotAt == nil || !gotAt.Equal(*at(tt.wantDay))):
				t.Errorf("time = %v, want day %d", gotAt, tt.wantDay)
			}
		})
	}
}

func TestManifestationDeadlines(t *testing.T) {
	loc := time.FixedZone("-03", -3*60*60)
	authorized := time.Date(2026, 9, 1, 9, 15, 42, 0, loc)
	issued := time.Date(2026, 8, 31, 18, 0, 0, 0, loc)

	d := ManifestationDeadlines(Document{AuthorizedAt: &authorized, IssueDate: issued})
	if want := time.Date(2026, 9, 11, 9, 15, 42, 0, loc); !d.CienciaDue.Equal(want) {
		t.Errorf("CienciaDue = %s, want %s", d.CienciaDue, want)
	}
	if want := time.Date(2026, 11, 30, 9, 15, 42, 0, loc); !d.ConclusiveDue.Equal(want) {
		t.Errorf("ConclusiveDue = %s, want %s", d.ConclusiveDue, want)
	}
	if d.FromIssueDate {
		t.Error("FromIssueDate should be false when AuthorizedAt is set")
	}

	fallback := ManifestationDeadlines(Document{IssueDate: issued})
	if !fallback.FromIssueDate || !fallback.CienciaDue.Equal(issued.AddDate(0, 0, CienciaWarningDays)) {
		t.Errorf("fallback deadlines = %+v", fallback)
	}

	if none := ManifestationDeadlines(Document{}); !none.CienciaDue.IsZero() || !none.ConclusiveDue.IsZero() {
		t.Errorf("deadlines without dates = %+v, want zero", none)
	}
}

func TestDeadlineWarnings(t *testing.T) {
	authorized := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	d := ManifestationDeadlines(Document{AuthorizedAt: &authorized})

	tests := []struct {
		name           string
		now            time.Time
		wantCiencia    bool
		wantConclusive bool
	}{
		{"just authorized", authorized, false, false},
		{"one second before ciência warning", d.CienciaDue.Add(-time.Second), false, false},
		{"ciência warning", d.CienciaDue, true, false},
		{"one second before conclusive warning", d.ConclusiveDue.AddDate(0, 0, -ConclusiveWarningDays).Add(-time.Second), true, false},
		{"conclusive warning", d.ConclusiveDue.AddDate(0, 0, -ConclusiveWarningDays), true, true},
		{"past conclusive deadline", d.ConclusiveDue.AddDate(0, 0, 1), true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.CienciaWarning(tt.now); got != tt.wantCiencia {
				t.Errorf("CienciaWarning = %v, want %v", got, tt.wantCiencia)
			}
			if got := d.ConclusiveWarning(tt.now); got != tt.wantConclusive {
				t.Errorf("ConclusiveWarning = %v, want %v", got, tt.wantConclusive)
			}
		})
	}

	var zero Deadlines
	if zero.CienciaWarning(authorized) || zero.ConclusiveWarning(authorized) {
		t.Error("zero deadlines should never warn")
	}
}

func TestTacitlyConfirmed(t *testing.T) {
	authorized := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	due := authorized.AddDate(0, 0, ConclusiveDeadlineDays)
	base := CompanyDocument{
		Document:     Document{AuthorizedAt: &authorized, Situacao: SituacaoAutorizada},
		CompanyRole:  CompanyRoleDestinatario,
		Manifestacao: ManifestacaoCiencia,
	}
	with := func(change func(*CompanyDocument)) CompanyDocument {
		doc := base
		change(&doc)
		return doc
	}

	tests := []struct {
		name string
		doc  CompanyDocument
		now  time.Time
		want bool
	}{
		{"ciência, at the deadline", base, due, false},
		{"ciência, past the deadline", base, due.Add(time.Second), true},
		{"no manifestação, past the deadline", with(func(d *CompanyDocument) { d.Manifestacao = ManifestacaoNenhuma }), due.Add(time.Second), true},
		{"confirmada", with(func(d *CompanyDocument) { d.Manifestacao = ManifestacaoConfirmada }), due.Add(time.Second), false},
		{"desconhecida", with(func(d *CompanyDocument) { d.Manifestacao = ManifestacaoDesconhecida }), due.Add(time.Second), false},
		{"emitente", with(func(d *CompanyDocument) { d.CompanyRole = CompanyRoleEmitente }), due.Add(time.Second), false},
		{"cancelada", with(func(d *CompanyDocument) { d.Situacao = SituacaoCancelada }), due.Add(time.Second), false},
		{"no dates", with(func(d *CompanyDocument) { d.AuthorizedAt = nil }), due.Add(time.Second), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TacitlyConfirmed(tt.doc, tt.now); got != tt.want {
				t.Errorf("TacitlyConfirmed = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateJustificativa(t *testing.T) {
	tests := []struct {
		name    string
		tipo    ManifestationType
		xJust   string
		want    string
		wantErr bool
	}{
		{"14 characters", ManifestationNaoRealizada, strings.Repeat("a", 14), "", true},
		{"15 characters", ManifestationNaoRealizada, strings.Repeat("a", 15), strings.Repeat("a", 15), false},
		{"255 characters", ManifestationNaoRealizada, strings.Repeat("a", 255), strings.Repeat("a", 255), false},
		{"256 characters", ManifestationNaoRealizada, strings.Repeat("a", 256), "", true},
		{"accented characters count once", ManifestationNaoRealizada, strings.Repeat("ç", 15), strings.Repeat("ç", 15), false},
		{"empty for nao_realizada", ManifestationNaoRealizada, "   ", "", true},
		{
			"whitespace padding does not count",
			ManifestationNaoRealizada,
			"   " + strings.Repeat("a", 14) + "   ",
			"", true,
		},
		{
			"cleans control characters and whitespace",
			ManifestationNaoRealizada,
			"  Mercadoria\tnao\r\nentregue \x00 pelo   fornecedor  ",
			"Mercadoria nao entregue pelo fornecedor", false,
		},
		{"empty for ciencia", ManifestationCiencia, "", "", false},
		{"blank for confirmacao", ManifestationConfirmacao, " \t ", "", false},
		{"rejected for desconhecimento", ManifestationDesconhecimento, "Nao reconheco esta operacao", "", true},
		{"invalid type", ManifestationType("x"), "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateJustificativa(tt.tipo, tt.xJust)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ValidateJustificativa = %q, want %q", got, tt.want)
			}
		})
	}

	_, err := ValidateJustificativa(ManifestationNaoRealizada, "curta")
	if !errors.Is(err, ErrInvalidJustificativa) {
		t.Errorf("error = %v, want ErrInvalidJustificativa", err)
	}
}
