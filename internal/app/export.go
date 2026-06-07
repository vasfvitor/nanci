package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/report"
)

// ExportInput is shared by all export formats.
type ExportInput struct {
	CNPJ       string
	Competence string // "YYYY-MM", optional
	Direction  string // "tomada" | "prestada" | "intermediario", optional
	OutPath    string // destination file path
}

// ExportCSV writes a CSV report for the matching documents to input.OutPath.
func (a *App) ExportCSV(ctx context.Context, input ExportInput) error {
	docs, err := a.queryExportDocs(ctx, input)
	if err != nil {
		return err
	}
	return report.GenerateCSV(report.BuildRows(docs), input.OutPath)
}

// ExportXLSX writes an Excel report for the matching documents to input.OutPath.
func (a *App) ExportXLSX(ctx context.Context, input ExportInput) error {
	docs, err := a.queryExportDocs(ctx, input)
	if err != nil {
		return err
	}
	return report.GenerateXLSX(report.BuildRows(docs), input.OutPath)
}

// ExportZIP packs the raw XML files for the matching documents into input.OutPath.
func (a *App) ExportZIP(ctx context.Context, input ExportInput) error {
	docs, err := a.queryExportDocs(ctx, input)
	if err != nil {
		return err
	}
	return report.GenerateZIP(report.BuildRows(docs), files.NewBlobStore(a.DataDir), input.OutPath)
}

// queryExportDocs validates input and returns the matching documents from the store.
func (a *App) queryExportDocs(ctx context.Context, input ExportInput) ([]nfse.CompanyDocument, error) {
	if input.OutPath == "" {
		return nil, fmt.Errorf("caminho de saída não especificado")
	}

	if err := cnpj.Validate(input.CNPJ); err != nil {
		return nil, fmt.Errorf("CNPJ inválido: %w", err)
	}

	cleanedCNPJ := cnpj.Clean(input.CNPJ)

	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return nil, fmt.Errorf("buscar empresa: %w", err)
	}
	if company == nil {
		return nil, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
	}

	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
	}

	docs, err := a.DocumentReader.ListCompanyDocuments(ctx, company.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("listar documentos: %w", err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("nenhum documento encontrado para exportar")
	}

	return docs, nil
}
