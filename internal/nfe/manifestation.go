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

// ParseManifestationType reads a manifestação type by name or by tpEvento
// code, ignoring case and surrounding spaces. "nao-realizada" is accepted as
// nao_realizada.
func ParseManifestationType(val string) (ManifestationType, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case string(ManifestationCiencia), TpEventoCiencia:
		return ManifestationCiencia, nil
	case string(ManifestationConfirmacao), TpEventoConfirmacao:
		return ManifestationConfirmacao, nil
	case string(ManifestationDesconhecimento), TpEventoDesconhecimento:
		return ManifestationDesconhecimento, nil
	case string(ManifestationNaoRealizada), "nao-realizada", TpEventoNaoRealizada:
		return ManifestationNaoRealizada, nil
	default:
		return "", fmt.Errorf("invalid manifestation type %q: %w", val, nfse.ErrInvalidEnum)
	}
}

func (t ManifestationType) Valid() bool {
	return t.TpEvento() != ""
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

// Label returns the name of the event shown to the user, or "" for an
// invalid type.
func (t ManifestationType) Label() string {
	switch t {
	case ManifestationCiencia:
		return "Ciência da Operação"
	case ManifestationConfirmacao:
		return "Confirmação da Operação"
	case ManifestationDesconhecimento:
		return "Desconhecimento da Operação"
	case ManifestationNaoRealizada:
		return "Operação não Realizada"
	default:
		return ""
	}
}

// ManifestacaoAndTimeFromEvents derives the manifestação state of one
// company from the events of a document, and the time of the event that set
// it. Only registered events authored by companyCNPJ count. The latest
// conclusive event (confirmação, desconhecimento, operação não realizada)
// wins; without one, any ciência gives ManifestacaoCiencia, timed by the
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
// nanci never blocks a manifestação locally because of them: SEFAZ has the
// final word and rejects a late event with cStat 596. They only drive
// warnings.
const (
	// CienciaWarningDays is when an NF-e without any manifestação starts to
	// show a warning.
	CienciaWarningDays = 10
	// ConclusiveDeadlineDays is the deadline for a conclusive manifestação
	// (Confirmação, Desconhecimento or Operação não Realizada). Cláusula
	// 15ª-C of Ajuste SINIEF 07/05, as amended by Ajuste SINIEF 14/2026 (in
	// force since 2026-06-01), sets it at 90 days from the authorization;
	// after that, without a conclusive event, the operation is deemed
	// confirmed (§ 6º). Ciência does not stop that presumption.
	ConclusiveDeadlineDays = 90
)

// Deadlines are the manifestação dates of one NF-e. Both are zero when the
// document has neither an authorization nor an issue date.
type Deadlines struct {
	CienciaDue    time.Time
	ConclusiveDue time.Time
}

// ManifestationDeadlines computes the deadlines from AuthorizedAt, falling
// back to IssueDate.
func ManifestationDeadlines(doc Document) Deadlines {
	var start time.Time
	switch {
	case doc.AuthorizedAt != nil:
		start = *doc.AuthorizedAt
	case !doc.IssueDate.IsZero():
		start = doc.IssueDate
	default:
		return Deadlines{}
	}
	return Deadlines{
		CienciaDue:    start.AddDate(0, 0, CienciaWarningDays),
		ConclusiveDue: start.AddDate(0, 0, ConclusiveDeadlineDays),
	}
}

// CienciaWarning reports whether now is on or after CienciaDue.
func (d Deadlines) CienciaWarning(now time.Time) bool {
	return !d.CienciaDue.IsZero() && !now.Before(d.CienciaDue)
}

// TacitlyConfirmed reports whether the operation of doc is deemed confirmed
// by law: the company is the destinatário of an authorized NF-e, it has no
// conclusive manifestação, and now is past ConclusiveDue. A ciência does not
// prevent it.
func TacitlyConfirmed(doc CompanyDocument, now time.Time) bool {
	if doc.CompanyRole != CompanyRoleDestinatario || doc.Situacao != SituacaoAutorizada {
		return false
	}
	if doc.Manifestacao != ManifestacaoNenhuma && doc.Manifestacao != ManifestacaoCiencia {
		return false
	}
	due := ManifestationDeadlines(doc.Document).ConclusiveDue
	return !due.IsZero() && now.After(due)
}

// ConclusiveBlockReason says why a conclusive manifestação cannot be sent
// on a document whose manifestação is m, or "" when it can. nanci sends at
// most one conclusive manifestação per NF-e, whatever its type.
func ConclusiveBlockReason(m Manifestacao) string {
	if m.Conclusive() {
		return "NF-e já possui manifestação conclusiva (" + m.Label() + ")"
	}
	return ""
}

// Justificativa length limits for Operação não Realizada (xJust).
const (
	JustificativaMinLength = 15
	JustificativaMaxLength = 255
)

// ValidateJustificativa cleans xJust (control characters become spaces,
// whitespace runs collapse to one space, ends are trimmed) and checks it
// against tipo. Operação não Realizada requires 15 to 255 characters; every
// other type rejects a non-empty justificativa. It returns the cleaned text,
// or an error whose message can be shown to the user.
func ValidateJustificativa(tipo ManifestationType, xJust string) (string, error) {
	if !tipo.Valid() {
		return "", fmt.Errorf("invalid manifestation type %q: %w", tipo, nfse.ErrInvalidEnum)
	}
	cleaned := cleanText(xJust)
	length := utf8.RuneCountInString(cleaned)

	if tipo != ManifestationNaoRealizada {
		if length > 0 {
			return "", errors.New("justificativa só é aceita para operação não realizada")
		}
		return "", nil
	}
	if length < JustificativaMinLength || length > JustificativaMaxLength {
		return "", fmt.Errorf("justificativa inválida: informe de %d a %d caracteres", JustificativaMinLength, JustificativaMaxLength)
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
