package app

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/danfse"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/report"
	"github.com/vasfvitor/nanci/internal/store"
)

// ExportInput is shared by all export formats.
type ExportInput struct {
	CNPJ         string
	Competence   string // "YYYY-MM", optional
	Direction    string // "tomada" | "prestada" | "intermediario", optional
	OutPath      string // destination file path
	Incremental  bool
	ChavesAcesso []string
}

// ExportResult contains structured info about the export operation.
type ExportResult struct {
	OutPath       string
	Format        string
	Incremental   bool
	ExportedCount int
}

// ExportDANFSeInput identifies one company-visible NFS-e and the destination PDF path.
type ExportDANFSeInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// ExportXMLInput identifies one company-visible NFS-e and the destination XML path.
type ExportXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// ExportService owns the export use cases: bulk formats (CSV, XLSX, ZIP of
// raw XML, ZIP of DANFSe PDFs), single-document exports (DANFSe, XML), and
// the pending-export counter.
type ExportService struct {
	CompanyStore   *company.Store
	DocumentRepo   *store.DocumentRepository
	XMLStore       files.XMLStore
	DANFSeRenderer danfse.Renderer
}

func NewExportService(d Dependencies) *ExportService {
	return &ExportService{

		CompanyStore: d.CompanyStore,
		DocumentRepo: d.DocumentRepo,

		XMLStore:       d.XMLStore,
		DANFSeRenderer: d.DANFSeRenderer,
	}
}

// ExportCSV writes a CSV report for the matching documents to input.OutPath.
func (s *ExportService) ExportCSV(ctx context.Context, input ExportInput) (ExportResult, error) {
	return s.bulkExport(ctx, input, "csv", func(docs []nfse.CompanyDocument, tempPath string) error {
		return report.GenerateCSV(report.BuildRows(docs), tempPath)
	})
}

// ExportXLSX writes an Excel report for the matching documents to input.OutPath.
func (s *ExportService) ExportXLSX(ctx context.Context, input ExportInput) (ExportResult, error) {
	return s.bulkExport(ctx, input, "xlsx", func(docs []nfse.CompanyDocument, tempPath string) error {
		return report.GenerateXLSX(report.BuildRows(docs), tempPath)
	})
}

// ExportZIP packs the raw XML files for the matching documents into input.OutPath.
func (s *ExportService) ExportZIP(ctx context.Context, input ExportInput) (ExportResult, error) {
	return s.bulkExport(ctx, input, "xml", func(docs []nfse.CompanyDocument, tempPath string) error {
		return report.GenerateZIP(report.BuildRows(docs), s.XMLStore, tempPath)
	})
}

// ExportDANFSeZIP writes one DANFSe PDF per matching document into a ZIP archive.
func (s *ExportService) ExportDANFSeZIP(ctx context.Context, input ExportInput) (ExportResult, error) {
	return s.bulkExport(ctx, input, "danfse", func(docs []nfse.CompanyDocument, tempPath string) error {
		zipFile, err := os.Create(tempPath) //nolint:gosec // intentional: creating temp export file in user directory
		if err != nil {
			return fmt.Errorf("criar arquivo ZIP temporário: %w", err)
		}
		defer func() {
			if cerr := zipFile.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("fechar ZIP temporário: %w", cerr)
			}
		}()

		zipWriter := zip.NewWriter(zipFile)
		defer func() {
			if cerr := zipWriter.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("fechar zip writer: %w", cerr)
			}
		}()

		for _, doc := range docs {
			pdf, err := s.renderDANFSe(&doc)
			if err != nil {
				return fmt.Errorf("gerar DANFSe %s: %w", doc.ChaveAcesso, err)
			}

			roleFolder := string(doc.CompanyRole)
			if roleFolder == "" || roleFolder == "none" {
				roleFolder = "sem-papel-fiscal"
			}
			var entryPath string
			if doc.Competence != "" {
				entryPath = filepath.ToSlash(filepath.Join(doc.Competence, roleFolder, string(doc.ChaveAcesso)+".pdf"))
			} else {
				entryPath = filepath.ToSlash(filepath.Join(roleFolder, string(doc.ChaveAcesso)+".pdf"))
			}

			writer, err := zipWriter.Create(entryPath)
			if err != nil {
				return fmt.Errorf("criar entrada DANFSe %s: %w", doc.ChaveAcesso, err)
			}
			if _, err := writer.Write(pdf); err != nil {
				return fmt.Errorf("escrever DANFSe %s: %w", doc.ChaveAcesso, err)
			}
		}

		return nil
	})
}

func (s *ExportService) bulkExport(ctx context.Context, input ExportInput, kind string, generator func([]nfse.CompanyDocument, string) error) (ExportResult, error) {
	res := ExportResult{
		OutPath:     input.OutPath,
		Format:      kind,
		Incremental: input.Incremental,
	}

	if input.OutPath == "" {
		return res, fmt.Errorf("caminho de saída não especificado")
	}

	company, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return res, err
	}

	filter := nfse.DocumentFilter{
		Competence:   input.Competence,
		Direction:    input.Direction,
		ChavesAcesso: input.ChavesAcesso,
	}

	if company.SyncStartPolicy != "" && company.SyncStartPolicy != nfse.SyncStartPolicyAll && company.SyncStartDate != nil {
		filter.IssueDateGTE = company.SyncStartDate
	}

	var docs []nfse.CompanyDocument
	if input.Incremental {
		docs, err = s.DocumentRepo.ListPendingExportDocuments(ctx, company.ID, filter, kind)
	} else {
		docs, err = s.DocumentRepo.ListCompanyDocuments(ctx, company.ID, filter)
	}
	if err != nil {
		return res, fmt.Errorf("listar documentos: %w", err)
	}

	res.ExportedCount = len(docs)
	if res.ExportedCount == 0 {
		res.OutPath = ""
		return res, nil
	}

	ext := filepath.Ext(input.OutPath)
	tempPath := strings.TrimSuffix(input.OutPath, ext) + ".tmp" + ext
	defer func() { _ = os.Remove(tempPath) }()

	if err := generator(docs, tempPath); err != nil {
		return res, fmt.Errorf("gerar arquivo: %w", err)
	}

	if err := os.Rename(tempPath, input.OutPath); err != nil {
		return res, fmt.Errorf("mover arquivo temporário para destino final: %w", err)
	}

	marks := make([]nfse.DocumentExportMark, len(docs))
	for i, doc := range docs {
		marks[i] = nfse.DocumentExportMark{
			DocumentID: string(doc.DocumentID),
			ExportKind: kind,
			Hash:       doc.RawHash,
		}
	}

	if err := s.DocumentRepo.MarkDocumentsExported(ctx, company.ID, kind, marks); err != nil {
		return res, fmt.Errorf("marcar documentos como exportados: %w", err)
	}

	return res, nil
}

// ExportDANFSe writes a DANFSe PDF for one company-visible NFS-e.
func (s *ExportService) ExportDANFSe(ctx context.Context, input ExportDANFSeInput) error {
	if input.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	if input.ChaveAcesso == "" {
		return fmt.Errorf("chave de acesso não especificada")
	}

	company, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.DocumentRepo.CompanyDocumentByChave(ctx, company.ID, input.ChaveAcesso)
	if err != nil {
		return fmt.Errorf("localizar documento: %w", err)
	}

	pdf, err := s.renderDANFSe(doc)
	if err != nil {
		return err
	}

	tempPath := input.OutPath + ".tmp"
	if err := os.WriteFile(tempPath, pdf, 0o644); err != nil { // #nosec G306
		_ = os.Remove(tempPath)
		return fmt.Errorf("gravar DANFSe temp: %w", err)
	}
	if err := os.Rename(tempPath, input.OutPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("mover DANFSe temp: %w", err)
	}

	mark := nfse.DocumentExportMark{
		DocumentID: string(doc.DocumentID),
		ExportKind: "danfse",
		Hash:       doc.RawHash,
	}
	if err := s.DocumentRepo.MarkDocumentsExported(ctx, company.ID, "danfse", []nfse.DocumentExportMark{mark}); err != nil {
		return fmt.Errorf("marcar danfse exportado: %w", err)
	}

	return nil
}

// ExportXML writes the raw XML for one company-visible NFS-e.
func (s *ExportService) ExportXML(ctx context.Context, input ExportXMLInput) error {
	if input.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	if input.ChaveAcesso == "" {
		return fmt.Errorf("chave de acesso não especificada")
	}

	company, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.DocumentRepo.CompanyDocumentByChave(ctx, company.ID, input.ChaveAcesso)
	if err != nil {
		return fmt.Errorf("localizar documento: %w", err)
	}

	if doc.RawHash == "" {
		return fmt.Errorf("XML original não encontrado para a chave %s", doc.ChaveAcesso)
	}

	xmlData, err := s.XMLStore.Get(doc.RawHash)
	if err != nil {
		return fmt.Errorf("ler XML original da chave %s: %w", doc.ChaveAcesso, err)
	}

	tempPath := input.OutPath + ".tmp"
	if err := os.WriteFile(tempPath, xmlData, 0o644); err != nil { // #nosec G306
		_ = os.Remove(tempPath)
		return fmt.Errorf("gravar XML temp: %w", err)
	}
	if err := os.Rename(tempPath, input.OutPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("mover XML temp: %w", err)
	}

	mark := nfse.DocumentExportMark{
		DocumentID: string(doc.DocumentID),
		ExportKind: "xml",
		Hash:       doc.RawHash,
	}
	if err := s.DocumentRepo.MarkDocumentsExported(ctx, company.ID, "xml", []nfse.DocumentExportMark{mark}); err != nil {
		return fmt.Errorf("marcar xml exportado: %w", err)
	}

	return nil
}

// CountPendingExportDocuments counts the documents that are pending export for the given format.
func (s *ExportService) CountPendingExportDocuments(ctx context.Context, input ExportInput, kind string) (int, error) {
	company, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return 0, err
	}
	filter := nfse.DocumentFilter{
		Competence: input.Competence,
		Direction:  input.Direction,
	}

	if company.SyncStartPolicy != "" && company.SyncStartPolicy != nfse.SyncStartPolicyAll && company.SyncStartDate != nil {
		filter.IssueDateGTE = company.SyncStartDate
	}
	return s.DocumentRepo.CountPendingExportDocuments(ctx, company.ID, filter, kind)
}

func (s *ExportService) renderDANFSe(doc *nfse.CompanyDocument) ([]byte, error) {
	if s.DANFSeRenderer == nil {
		return nil, fmt.Errorf("DANFSe não configurado")
	}
	if doc.RawHash == "" {
		return nil, fmt.Errorf("XML original não encontrado para a chave %s", doc.ChaveAcesso)
	}

	xmlData, err := s.XMLStore.Get(doc.RawHash)
	if err != nil {
		return nil, fmt.Errorf("ler XML original da chave %s: %w", doc.ChaveAcesso, err)
	}

	pdf, err := s.DANFSeRenderer.Render(xmlData)
	if err != nil {
		return nil, fmt.Errorf("renderizar DANFSe: %w", err)
	}
	return pdf, nil
}
