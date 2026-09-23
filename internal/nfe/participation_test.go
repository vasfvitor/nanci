package nfe

import "testing"

func TestClassifyParticipation(t *testing.T) {
	const filialMock = "70860312000231" // same root as cnpjMock

	completa := Document{
		EmitenteCNPJ:      cnpjEmitente,
		DestinatarioCNPJ:  cnpjMock,
		TransportadorCNPJ: cnpjTransportador,
		AutorizadosCNPJ:   []string{"98765432000198", cnpjAutorizado},
		Completeness:      CompletenessCompleta,
	}
	resumo := Document{EmitenteCNPJ: cnpjEmitente, Completeness: CompletenessResumo}

	tests := []struct {
		name    string
		doc     Document
		company string
		want    Participation
	}{
		{"destinatário", completa, cnpjMock, Participation{CompanyRoleDestinatario, VisibilityReasonExactDestinatario}},
		{"destinatário with punctuation", completa, "70.860.312/0001-50", Participation{CompanyRoleDestinatario, VisibilityReasonExactDestinatario}},
		{"emitente", completa, cnpjEmitente, Participation{CompanyRoleEmitente, VisibilityReasonExactEmitente}},
		{"transportador", completa, cnpjTransportador, Participation{CompanyRoleTransportador, VisibilityReasonExactTransportador}},
		{"autorizado", completa, cnpjAutorizado, Participation{CompanyRoleAutorizado, VisibilityReasonExactAutorizado}},
		{
			"destinatário wins over emitente",
			Document{EmitenteCNPJ: cnpjMock, DestinatarioCNPJ: cnpjMock, Completeness: CompletenessCompleta},
			cnpjMock,
			Participation{CompanyRoleDestinatario, VisibilityReasonExactDestinatario},
		},
		{"resumo from another emitente", resumo, cnpjMock, Participation{CompanyRoleDestinatario, VisibilityReasonResumoDestinatario}},
		{"resumo emitted by the company", resumo, cnpjEmitente, Participation{CompanyRoleEmitente, VisibilityReasonExactEmitente}},
		{"same root only", completa, filialMock, Participation{CompanyRoleNone, VisibilityReasonSameRootOnly}},
		{"unrelated", completa, "11444777000161", Participation{CompanyRoleNone, VisibilityReasonUnknown}},
		{
			"empty parties never match an empty company",
			Document{Completeness: CompletenessCompleta},
			"",
			Participation{CompanyRoleNone, VisibilityReasonUnknown},
		},
		{
			"alphanumeric destinatário",
			Document{EmitenteCNPJ: cnpjEmitente, DestinatarioCNPJ: "12ABC34501DE35", Completeness: CompletenessCompleta},
			"12.abc.345/01de-35",
			Participation{CompanyRoleDestinatario, VisibilityReasonExactDestinatario},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyParticipation(&tt.doc, tt.company); got != tt.want {
				t.Errorf("ClassifyParticipation = %+v, want %+v", got, tt.want)
			}
		})
	}
}
