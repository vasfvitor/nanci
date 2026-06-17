package app

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"

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

// ExportDANFSeInput identifies one company-visible NFS-e and the destination PDF path.
type ExportDANFSeInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
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
	return report.GenerateZIP(report.BuildRows(docs), a.XMLStore, input.OutPath)
}

// ExportDANFSe writes a DANFSe PDF for one company-visible NFS-e.
func (a *App) ExportDANFSe(ctx context.Context, input ExportDANFSeInput) error {
	if input.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	if input.ChaveAcesso == "" {
		return fmt.Errorf("chave de acesso não especificada")
	}

	company, err := a.companyByCNPJ(ctx, input.CNPJ)
	if err != nil {
		return err
	}
	doc, err := a.DocumentReader.CompanyDocumentByChave(ctx, company.ID, input.ChaveAcesso)
	if err != nil {
		return fmt.Errorf("localizar documento: %w", err)
	}

	pdf, err := a.renderDANFSe(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(input.OutPath, pdf, 0o644); err != nil { // #nosec G306 -- destination is explicitly selected by the local user.
		return fmt.Errorf("gravar DANFSe: %w", err)
	}
	return nil
}

// ExportDANFSeZIP writes one DANFSe PDF per matching document into a ZIP archive.
func (a *App) ExportDANFSeZIP(ctx context.Context, input ExportInput) (err error) {
	docs, err := a.queryExportDocs(ctx, input)
	if err != nil {
		return err
	}

	zipFile, err := os.Create(input.OutPath) // #nosec G304 -- destination is explicitly selected by the local user.
	if err != nil {
		return fmt.Errorf("criar arquivo ZIP de DANFSes: %w", err)
	}
	defer func() {
		if cerr := zipFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("fechar arquivo ZIP de DANFSes: %w", cerr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if cerr := zipWriter.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("fechar ZIP de DANFSes: %w", cerr)
		}
	}()

	for _, doc := range docs {
		pdf, err := a.renderDANFSe(&doc)
		if err != nil {
			return fmt.Errorf("gerar DANFSe %s: %w", doc.ChaveAcesso, err)
		}

		roleFolder := string(doc.CompanyRole)
		if roleFolder == "" || roleFolder == "none" {
			roleFolder = "sem-papel-fiscal"
		}
		entryPath := filepath.ToSlash(filepath.Join(doc.Competence, roleFolder, string(doc.ChaveAcesso)+".pdf"))

		writer, err := zipWriter.Create(entryPath)
		if err != nil {
			return fmt.Errorf("criar entrada DANFSe %s: %w", doc.ChaveAcesso, err)
		}
		if _, err := writer.Write(pdf); err != nil {
			return fmt.Errorf("escrever DANFSe %s: %w", doc.ChaveAcesso, err)
		}
	}

	return nil
}

func (a *App) renderDANFSe(doc *nfse.CompanyDocument) ([]byte, error) {
	if a.DANFSeRenderer == nil {
		return nil, fmt.Errorf("DANFSe não configurado")
	}
	if doc.RawHash == "" {
		return nil, fmt.Errorf("XML original não encontrado para a chave %s", doc.ChaveAcesso)
	}

	xmlData, err := a.XMLStore.Get(doc.RawHash)
	if err != nil {
		return nil, fmt.Errorf("ler XML original da chave %s: %w", doc.ChaveAcesso, err)
	}

	pdf, err := a.DANFSeRenderer.Render(xmlData)
	if err != nil {
		return nil, fmt.Errorf("renderizar DANFSe: %w", err)
	}
	return pdf, nil
}

// queryExportDocs validates input and returns the matching documents from the store.
func (a *App) queryExportDocs(ctx context.Context, input ExportInput) ([]nfse.CompanyDocument, error) {
	if input.OutPath == "" {
		return nil, fmt.Errorf("caminho de saída não especificado")
	}

	company, err := a.companyByCNPJ(ctx, input.CNPJ)
	if err != nil {
		return nil, err
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
