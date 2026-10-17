package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/danfse"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/report"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/syncstate"
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
	return s.bulkExport(ctx, input, nfse.ExportKindCSV, func(docs []nfse.CompanyDocument, tempPath string) error {
		return report.WriteCSV(report.NFSeCSV(report.BuildNFSeRows(docs)), tempPath)
	})
}

// ExportXLSX writes an Excel report for the matching documents to input.OutPath.
func (s *ExportService) ExportXLSX(ctx context.Context, input ExportInput) (ExportResult, error) {
	return s.bulkExport(ctx, input, nfse.ExportKindXLSX, func(docs []nfse.CompanyDocument, tempPath string) error {
		return report.WriteXLSX(report.NFSeXLSX(report.BuildNFSeRows(docs)), tempPath)
	})
}

// ExportZIP packs the raw XML files for the matching documents into input.OutPath.
func (s *ExportService) ExportZIP(ctx context.Context, input ExportInput) (ExportResult, error) {
	return s.bulkExport(ctx, input, nfse.ExportKindXML, func(docs []nfse.CompanyDocument, tempPath string) error {
		return report.GenerateZIP(report.NFSeZipEntries(report.BuildNFSeRows(docs)), s.XMLStore, tempPath)
	})
}

// ExportDANFSeZIP writes one DANFSe PDF per matching document into a ZIP
// archive. Documents without a stored XML are left out of the archive.
func (s *ExportService) ExportDANFSeZIP(ctx context.Context, input ExportInput) (ExportResult, error) {
	if s.DANFSeRenderer == nil {
		return ExportResult{OutPath: input.OutPath, Format: nfse.ExportKindDANFSe, Incremental: input.Incremental}, fmt.Errorf("DANFSe não configurado")
	}
	return s.bulkExport(ctx, input, nfse.ExportKindDANFSe, func(docs []nfse.CompanyDocument, tempPath string) error {
		render := func(xml []byte, _ bool) ([]byte, error) { return s.DANFSeRenderer.Render(xml) }
		return report.GeneratePDFZIP(report.NFSePDFEntries(docs), s.XMLStore, render, tempPath)
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

	chaves, err := parseNFSeAccessKeys(input.ChavesAcesso)
	if err != nil {
		return res, err
	}

	company, err := company.LookupByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return res, err
	}

	filter := nfse.DocumentFilter{
		Competence:   input.Competence,
		Direction:    input.Direction,
		ChavesAcesso: chaves,
	}

	if company.SyncStartPolicy != "" && company.SyncStartPolicy != syncstate.SyncStartPolicyAll && company.SyncStartDate != nil {
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

	err = writeViaTemp(input.OutPath, func(tempPath string) error {
		return generator(docs, tempPath)
	})
	if err != nil {
		return res, err
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
	chave, err := parseNFSeAccessKey(input.ChaveAcesso)
	if err != nil {
		return err
	}

	company, err := company.LookupByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.DocumentRepo.CompanyDocumentByChave(ctx, company.ID, chave)
	if err != nil {
		return fmt.Errorf("localizar documento: %w", err)
	}

	pdf, err := s.renderDANFSe(doc)
	if err != nil {
		return err
	}

	if err := writeFileAtomic(input.OutPath, pdf, "DANFSe"); err != nil {
		return err
	}

	mark := nfse.DocumentExportMark{
		DocumentID: string(doc.DocumentID),
		ExportKind: nfse.ExportKindDANFSe,
		Hash:       doc.RawHash,
	}
	if err := s.DocumentRepo.MarkDocumentsExported(ctx, company.ID, nfse.ExportKindDANFSe, []nfse.DocumentExportMark{mark}); err != nil {
		return fmt.Errorf("marcar danfse exportado: %w", err)
	}

	return nil
}

// ExportXML writes the raw XML for one company-visible NFS-e.
func (s *ExportService) ExportXML(ctx context.Context, input ExportXMLInput) error {
	if input.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	chave, err := parseNFSeAccessKey(input.ChaveAcesso)
	if err != nil {
		return err
	}

	company, err := company.LookupByCNPJ(ctx, s.CompanyStore, input.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.DocumentRepo.CompanyDocumentByChave(ctx, company.ID, chave)
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

	if err := writeFileAtomic(input.OutPath, xmlData, "XML"); err != nil {
		return err
	}

	mark := nfse.DocumentExportMark{
		DocumentID: string(doc.DocumentID),
		ExportKind: nfse.ExportKindXML,
		Hash:       doc.RawHash,
	}
	if err := s.DocumentRepo.MarkDocumentsExported(ctx, company.ID, nfse.ExportKindXML, []nfse.DocumentExportMark{mark}); err != nil {
		return fmt.Errorf("marcar xml exportado: %w", err)
	}

	return nil
}

// parseNFSeAccessKey validates the chave of a single-document export and
// returns it trimmed.
func parseNFSeAccessKey(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("chave de acesso não especificada")
	}
	chaves, err := parseNFSeAccessKeys([]string{raw})
	if err != nil {
		return "", err
	}
	return chaves[0], nil
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
