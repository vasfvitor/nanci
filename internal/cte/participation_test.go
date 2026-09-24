package cte

import (
	"reflect"
	"testing"
)

const (
	cnpjMock          = "70860312000150" // the managed company in most fixtures
	cnpjTransportador = "12345678000195" // emitente of every fixture
	cnpjRemetente     = "11222333000181"
	cnpjExpedidor     = "13579246000101"
	cnpjRecebedor     = "24681357000140"
	cnpjTerceiro      = "11223344000186" // tomador named in toma4
	cnpjAutorizado    = "45678901000175"
)

func TestClassifyParticipation(t *testing.T) {
	const filialMock = "70860312000231" // same root as cnpjMock

	// Each party is a different company; the tomador is a third party.
	separate := Document{
		Emitente:        Party{CNPJ: cnpjTransportador},
		Remetente:       Party{CNPJ: cnpjRemetente},
		Expedidor:       Party{CNPJ: cnpjExpedidor},
		Recebedor:       Party{CNPJ: cnpjRecebedor},
		Destinatario:    Party{CNPJ: cnpjMock},
		Tomador:         Party{CNPJ: cnpjTerceiro},
		AutorizadosCNPJ: []string{"98765432000198", cnpjAutorizado},
	}
	// toma3 = 0: the remetente pays the freight.
	remetenteTomador := Document{
		Emitente:     Party{CNPJ: cnpjTransportador},
		Remetente:    Party{CNPJ: cnpjMock},
		Destinatario: Party{CNPJ: cnpjRemetente},
		Tomador:      Party{CNPJ: cnpjMock},
	}

	tests := []struct {
		name    string
		doc     Document
		company string
		want    Participation
	}{
		{"tomador (toma4)", separate, cnpjTerceiro, exact(CompanyRoleTomador)},
		{"destinatário", separate, cnpjMock, exact(CompanyRoleDestinatario)},
		{"destinatário with punctuation", separate, "70.860.312/0001-50", exact(CompanyRoleDestinatario)},
		{"remetente", separate, cnpjRemetente, exact(CompanyRoleRemetente)},
		{"expedidor", separate, cnpjExpedidor, exact(CompanyRoleExpedidor)},
		{"recebedor", separate, cnpjRecebedor, exact(CompanyRoleRecebedor)},
		{"emitente", separate, cnpjTransportador, exact(CompanyRoleEmitente)},
		{"autorizado", separate, cnpjAutorizado, exact(CompanyRoleAutorizado)},
		{
			"tomador and remetente at once",
			remetenteTomador,
			cnpjMock,
			Participation{
				CompanyRole:      CompanyRoleTomador,
				Papeis:           []CompanyRole{CompanyRoleTomador, CompanyRoleRemetente},
				VisibilityReason: VisibilityReasonExactTomador,
			},
		},
		{
			"every role in priority order",
			Document{
				Emitente:        Party{CNPJ: cnpjMock},
				Remetente:       Party{CNPJ: cnpjMock},
				Expedidor:       Party{CNPJ: cnpjMock},
				Recebedor:       Party{CNPJ: cnpjMock},
				Destinatario:    Party{CNPJ: cnpjMock},
				Tomador:         Party{CNPJ: cnpjMock},
				AutorizadosCNPJ: []string{cnpjMock},
			},
			cnpjMock,
			Participation{
				CompanyRole: CompanyRoleTomador,
				Papeis: []CompanyRole{
					CompanyRoleTomador, CompanyRoleDestinatario, CompanyRoleRemetente, CompanyRoleExpedidor,
					CompanyRoleRecebedor, CompanyRoleEmitente, CompanyRoleAutorizado,
				},
				VisibilityReason: VisibilityReasonExactTomador,
			},
		},
		{
			"CPF destinatário",
			Document{Emitente: Party{CNPJ: cnpjTransportador}, Destinatario: Party{CNPJ: "12345678909"}},
			"123.456.789-09",
			exact(CompanyRoleDestinatario),
		},
		{
			"alphanumeric tomador",
			Document{Emitente: Party{CNPJ: cnpjTransportador}, Tomador: Party{CNPJ: "12ABC34501DE35"}},
			"12.abc.345/01de-35",
			exact(CompanyRoleTomador),
		},
		{"same root only", separate, filialMock, Participation{CompanyRole: CompanyRoleNone, VisibilityReason: VisibilityReasonSameRootOnly}},
		{
			"same root as an autorizado",
			Document{Emitente: Party{CNPJ: cnpjTransportador}, AutorizadosCNPJ: []string{cnpjMock}},
			filialMock,
			Participation{CompanyRole: CompanyRoleNone, VisibilityReason: VisibilityReasonSameRootOnly},
		},
		{"unrelated", separate, "11444777000161", Participation{CompanyRole: CompanyRoleNone, VisibilityReason: VisibilityReasonUnknown}},
		{
			"empty parties never match an empty company",
			Document{},
			"",
			Participation{CompanyRole: CompanyRoleNone, VisibilityReason: VisibilityReasonUnknown},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyParticipation(&tt.doc, tt.company); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClassifyParticipation = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// exact is the participation of a company that plays only role.
func exact(role CompanyRole) Participation {
	return Participation{CompanyRole: role, Papeis: []CompanyRole{role}, VisibilityReason: exactVisibility[role]}
}
