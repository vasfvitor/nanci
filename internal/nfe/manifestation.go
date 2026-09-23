package nfe

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// ManifestationType is a manifestação do destinatário nanci can send.
type ManifestationType string

const (
	ManifestationCiencia         ManifestationType = "ciencia"
	ManifestationConfirmacao     ManifestationType = "confirmacao"
	ManifestationDesconhecimento ManifestationType = "desconhecimento"
	ManifestationNaoRealizada    ManifestationType = "nao_realizada"
)

func ParseManifestationType(val string) (ManifestationType, error) {
	t := ManifestationType(val)
	if !t.Valid() {
		return "", fmt.Errorf("invalid manifestation type %q: %w", val, nfse.ErrInvalidEnum)
	}
	return t, nil
}

func (t ManifestationType) Valid() bool {
	return t.TpEvento() != ""
}

func (t ManifestationType) String() string {
	return string(t)
}

// TpEvento returns the tpEvento code, or "" for an invalid type.
func (t ManifestationType) TpEvento() string {
	switch t {
	case ManifestationCiencia:
		return TpEventoCiencia
	case ManifestationConfirmacao:
		return TpEventoConfirmacao
	case ManifestationDesconhecimento:
		return TpEventoDesconhecimento
	case ManifestationNaoRealizada:
		return TpEventoNaoRealizada
	default:
		return ""
	}
}

// DescEvento returns the descEvento text required by the event schema. The
// schema enumerates these strings without accents, so they must be sent
// exactly like this.
func (t ManifestationType) DescEvento() string {
	switch t {
	case ManifestationCiencia:
		return "Ciencia da Operacao"
	case ManifestationConfirmacao:
		return "Confirmacao da Operacao"
	case ManifestationDesconhecimento:
		return "Desconhecimento da Operacao"
	case ManifestationNaoRealizada:
		return "Operacao nao Realizada"
	default:
		return ""
	}
}

// ManifestacaoFromEvents derives the manifestação state of one company from
// the events of a document. Only registered events authored by companyCNPJ
// count. The latest conclusive event (confirmação, desconhecimento, operação
// não realizada) wins; without one, any ciência gives ManifestacaoCiencia.
func ManifestacaoFromEvents(events []Event, companyCNPJ string) Manifestacao {
	m, _ := ManifestacaoAndTimeFromEvents(events, companyCNPJ)
	return m
}

// ManifestacaoAndTimeFromEvents is ManifestacaoFromEvents plus the time of
// the event that set the state: the latest conclusive event, or else the
// earliest ciência. The time is nil for ManifestacaoNenhuma and for an event
// without registration or event time.
func ManifestacaoAndTimeFromEvents(events []Event, companyCNPJ string) (Manifestacao, *time.Time) {
	company := cnpj.Clean(companyCNPJ)
	result := ManifestacaoNenhuma
	var latestConclusive, earliestCiencia time.Time
	hasConclusive := false

	for _, e := range events {
		if !e.Registered || !sameParty(company, e.AutorCNPJ) {
			continue
		}
		var conclusive Manifestacao
		switch e.Type {
		case EventTypeCiencia:
			if at := eventTime(e); earliestCiencia.IsZero() || (!at.IsZero() && at.Before(earliestCiencia)) {
				earliestCiencia = at
			}
			if !hasConclusive {
				result = ManifestacaoCiencia
			}
			continue
		case EventTypeConfirmacao:
			conclusive = ManifestacaoConfirmada
		case EventTypeDesconhecimento:
			conclusive = ManifestacaoDesconhecida
		case EventTypeNaoRealizada:
			conclusive = ManifestacaoNaoRealizada
		default:
			continue
		}
		at := eventTime(e)
		if !hasConclusive || !at.Before(latestConclusive) {
			result = conclusive
			latestConclusive = at
			hasConclusive = true
		}
	}

	var at time.Time
	switch {
	case hasConclusive:
		at = latestConclusive
	case result == ManifestacaoCiencia:
		at = earliestCiencia
	}
	if at.IsZero() {
		return result, nil
	}
	return result, &at
}

// eventTime is the moment used to order events: the registration time when
// known, else the event time.
func eventTime(e Event) time.Time {
	if e.RegisteredAt != nil {
		return *e.RegisteredAt
	}
	if e.EventAt != nil {
		return *e.EventAt
	}
	return time.Time{}
}

// Deadline constants, in days counted from the authorization of the NF-e.
// nanci never blocks a manifestação locally because of them: sources disagree
// on the conclusive deadline and SEFAZ has the final word (cStat 596). They
// only drive warnings.
const (
	// CienciaWarningDays is when an NF-e without any manifestação starts to
	// show a warning.
	CienciaWarningDays = 10
	// ConclusiveDeadlineDays is the deadline shown for a conclusive
	// manifestação.
	ConclusiveDeadlineDays = 180
	// ConclusiveWarningDays is how many days before ConclusiveDue the
	// warning starts.
	ConclusiveWarningDays = 30
)

// Deadlines are the manifestação dates of one NF-e. Both are zero when the
// document has neither an authorization nor an issue date.
type Deadlines struct {
	CienciaDue    time.Time
	ConclusiveDue time.Time
	// FromIssueDate is true when AuthorizedAt was missing and the dates
	// count from IssueDate instead.
	FromIssueDate bool
}

// ManifestationDeadlines computes the deadlines from AuthorizedAt, falling
// back to IssueDate.
func ManifestationDeadlines(doc Document) Deadlines {
	var start time.Time
	fromIssueDate := false
	switch {
	case doc.AuthorizedAt != nil:
		start = *doc.AuthorizedAt
	case !doc.IssueDate.IsZero():
		start = doc.IssueDate
		fromIssueDate = true
	default:
		return Deadlines{}
	}
	return Deadlines{
		CienciaDue:    start.AddDate(0, 0, CienciaWarningDays),
		ConclusiveDue: start.AddDate(0, 0, ConclusiveDeadlineDays),
		FromIssueDate: fromIssueDate,
	}
}

// CienciaWarning reports whether now is on or after CienciaDue.
func (d Deadlines) CienciaWarning(now time.Time) bool {
	return !d.CienciaDue.IsZero() && !now.Before(d.CienciaDue)
}

// ConclusiveWarning reports whether now is within ConclusiveWarningDays of
// ConclusiveDue, or past it.
func (d Deadlines) ConclusiveWarning(now time.Time) bool {
	return !d.ConclusiveDue.IsZero() && !now.Before(d.ConclusiveDue.AddDate(0, 0, -ConclusiveWarningDays))
}

// Justificativa length limits for Operação não Realizada (xJust).
const (
	JustificativaMinLength = 15
	JustificativaMaxLength = 255
)

// ErrInvalidJustificativa is returned by ValidateJustificativa.
var ErrInvalidJustificativa = errors.New("invalid justificativa")

// ValidateJustificativa cleans xJust (control characters become spaces,
// whitespace runs collapse to one space, ends are trimmed) and checks it
// against tipo. Operação não Realizada requires 15 to 255 characters; every
// other type rejects a non-empty justificativa. It returns the cleaned text.
func ValidateJustificativa(tipo ManifestationType, xJust string) (string, error) {
	if !tipo.Valid() {
		return "", fmt.Errorf("invalid manifestation type %q: %w", tipo, nfse.ErrInvalidEnum)
	}
	cleaned := cleanText(xJust)
	length := utf8.RuneCountInString(cleaned)

	if tipo != ManifestationNaoRealizada {
		if length > 0 {
			return "", fmt.Errorf("%w: only %s accepts a justificativa", ErrInvalidJustificativa, ManifestationNaoRealizada)
		}
		return "", nil
	}
	if length < JustificativaMinLength || length > JustificativaMaxLength {
		return "", fmt.Errorf("%w: must have %d to %d characters, got %d",
			ErrInvalidJustificativa, JustificativaMinLength, JustificativaMaxLength, length)
	}
	return cleaned, nil
}

func cleanText(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}
