package report

import (
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

type ReportRow struct {
	CompanyRole       nfse.CompanyRole
	IssueDate         time.Time
	PrestadorCNPJ     string
	PrestadorName     string
	TomadorCNPJ       string
	TomadorName       string
	CounterpartyCNPJ  string
	CounterpartyName  string
	NFSeNumber        string
	Competence        string
	ChaveAcesso       string
	ServiceValue      float64
	ISSValue          float64
	IRRFValue         float64
	INSSValue         float64
	PISValue          float64
	COFINSValue       float64
	CSLLValue         float64
	TotalRetentions   float64
	EstimatedNetValue float64
	Status            nfse.DocumentStatus
	Description       string
	RawHash           string
	WarningsCount     int
}

func BuildRows(docs []nfse.CompanyDocument) []ReportRow {
	var reportRows []ReportRow

	for _, doc := range docs {
		isEmitida := doc.CompanyRole == nfse.CompanyRolePrestada

		docContraparte := doc.PrestadorCNPJ
		nomeContraparte := doc.PrestadorName
		if isEmitida {
			docContraparte = doc.TomadorCNPJ
			nomeContraparte = doc.TomadorName
		}

		serviceValue := float64(doc.ServiceValue) / 100.0
		totalRetentions := float64(doc.TotalRetentions) / 100.0
		netValue := serviceValue - totalRetentions

		reportRows = append(reportRows, ReportRow{
			CompanyRole:       doc.CompanyRole,
			IssueDate:         doc.IssueDate,
			PrestadorCNPJ:     doc.PrestadorCNPJ,
			PrestadorName:     doc.PrestadorName,
			TomadorCNPJ:       doc.TomadorCNPJ,
			TomadorName:       doc.TomadorName,
			CounterpartyCNPJ:  docContraparte,
			CounterpartyName:  nomeContraparte,
			NFSeNumber:        doc.NFSeNumber,
			Competence:        doc.Competence,
			ChaveAcesso:       string(doc.ChaveAcesso),
			ServiceValue:      serviceValue,
			ISSValue:          float64(doc.ISSValue) / 100.0,
			IRRFValue:         float64(doc.IRRFValue) / 100.0,
			INSSValue:         float64(doc.INSSValue) / 100.0,
			PISValue:          float64(doc.PISValue) / 100.0,
			COFINSValue:       float64(doc.COFINSValue) / 100.0,
			CSLLValue:         float64(doc.CSLLValue) / 100.0,
			TotalRetentions:   totalRetentions,
			EstimatedNetValue: netValue,
			Status:            doc.Status,
			Description:       doc.ServiceDescription,
			RawHash:           doc.RawHash,
			WarningsCount:     0,
		})
	}

	return reportRows
}
