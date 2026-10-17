package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/danfe"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/report"
)

// NFeExportInput selects the NF-e to export. Empty filters select all.
type NFeExportInput struct {
	CNPJ         string
	Competence   string // "YYYY-MM" of the issue date
	Role         string // destinatario | emitente | transportador | autorizado | none
	ChavesAcesso []string
	// IncludeResumos also exports resNFe files. A resumo is not the fiscal
	// document, so by default it is left out and counted in SkippedResumos.
	IncludeResumos bool
	// Incremental exports only documents not exported before, or whose XML
	// changed since (a resumo upgraded to completa).
	Incremental bool
	OutPath     string
}

// NFeExportResult is ExportResult plus the resumos left out of the archive.
type NFeExportResult struct {
	ExportResult
	SkippedResumos int
}

// NFeExportXMLInput identifies one of the company's NF-e and the destination
// XML path.
type NFeExportXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// NFeExportDANFEInput identifies one of the company's NF-e and the
// destination PDF path.
type NFeExportDANFEInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// ExportXMLZip packs the XML of the matching NF-e and their complete events
// into a ZIP at in.OutPath and marks the documents as exported. With nothing
// to export it writes no file and returns an empty OutPath.
func (s *NFeService) ExportXMLZip(ctx context.Context, in NFeExportInput) (NFeExportResult, error) {
	return s.bulkExport(ctx, in, nfe.ExportKindXML, func(docs []nfe.CompanyDocument, tempPath string) error {
		events, err := s.eventsByChave(ctx, docs)
		if err != nil {
			return err
		}
		return report.GenerateZIP(report.NFeZipEntries(docs, events), s.XMLStore, tempPath)
	})
}

// ExportCSV writes the matching NF-e as CSV rows at in.OutPath and marks
// them as exported to CSV.
func (s *NFeService) ExportCSV(ctx context.Context, in NFeExportInput) (NFeExportResult, error) {
	return s.bulkExport(ctx, in, nfe.ExportKindCSV, func(docs []nfe.CompanyDocument, tempPath string) error {
		return report.WriteCSV(report.NFeSpreadsheet(report.BuildNFeRows(docs)), tempPath)
	})
}

// ExportXLSX writes the matching NF-e as an Excel workbook at in.OutPath and
// marks them as exported to XLSX.
func (s *NFeService) ExportXLSX(ctx context.Context, in NFeExportInput) (NFeExportResult, error) {
	return s.bulkExport(ctx, in, nfe.ExportKindXLSX, func(docs []nfe.CompanyDocument, tempPath string) error {
		return report.WriteXLSX(report.NFeSpreadsheet(report.BuildNFeRows(docs)), tempPath)
	})
}

// ExportDANFEZip packs the DANFE of the matching NF-e into a ZIP at
// in.OutPath and marks them as exported to DANFE. A resumo has no DANFE, so
// resumos are always left out and counted in SkippedResumos, whatever
// in.IncludeResumos says.
func (s *NFeService) ExportDANFEZip(ctx context.Context, in NFeExportInput) (NFeExportResult, error) {
	if s.DANFERenderer == nil {
		return NFeExportResult{}, fmt.Errorf("DANFE não configurado")
	}
	in.IncludeResumos = false
	return s.bulkExport(ctx, in, nfe.ExportKindDANFE, func(docs []nfe.CompanyDocument, tempPath string) error {
		render := func(xml []byte, cancelada bool) ([]byte, error) {
			return s.DANFERenderer.Render(xml, danfe.Options{Cancelada: cancelada})
		}
		return report.GeneratePDFZIP(report.NFePDFEntries(docs), s.XMLStore, render, tempPath)
	})
}

// bulkExport runs one export of the NF-e that in selects: gen writes the
// file at a temp path, which then replaces in.OutPath, and the documents are
// marked as exported with kind. With nothing to export it writes no file and
// returns an empty OutPath.
func (s *NFeService) bulkExport(ctx context.Context, in NFeExportInput, kind string, gen func(docs []nfe.CompanyDocument, tempPath string) error) (NFeExportResult, error) {
	res := NFeExportResult{ExportResult: ExportResult{
		OutPath:     in.OutPath,
		Format:      kind,
		Incremental: in.Incremental,
	}}
	if in.OutPath == "" {
		return res, fmt.Errorf("caminho de saída não especificado")
	}

	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return res, err
	}
	docs, skipped, err := s.exportDocuments(ctx, comp, in, kind)
	if err != nil {
		return res, err
	}
	res.SkippedResumos = skipped
	res.ExportedCount = len(docs)
	if len(docs) == 0 {
		res.OutPath = ""
		return res, nil
	}

	err = writeViaTemp(in.OutPath, func(tempPath string) error {
		return gen(docs, tempPath)
	})
	if err != nil {
		return res, err
	}
	if err := s.NFeRepo.MarkExported(ctx, comp.ID, kind, docs); err != nil {
		return res, fmt.Errorf("marcar NF-e como exportadas: %w", err)
	}
	return res, nil
}

// eventsByChave returns the stored events of the documents, by chave.
func (s *NFeService) eventsByChave(ctx context.Context, docs []nfe.CompanyDocument) (map[string][]nfe.Event, error) {
	var withEvents []string
	for _, doc := range docs {
		if doc.EventCount > 0 {
			withEvents = append(withEvents, string(doc.ChaveAcesso))
		}
	}
	events, err := s.NFeRepo.ListEventsByChaves(ctx, withEvents)
	if err != nil {
		return nil, fmt.Errorf("listar eventos das NF-e: %w", err)
	}
	byChave := make(map[string][]nfe.Event, len(withEvents))
	for _, e := range events {
		byChave[string(e.ChaveAcesso)] = append(byChave[string(e.ChaveAcesso)], e)
	}
	return byChave, nil
}

// ExportXML writes the stored XML of one of the company's NF-e (procNFe, or
// resNFe for a resumo) to in.OutPath and marks it as exported.
func (s *NFeService) ExportXML(ctx context.Context, in NFeExportXMLInput) error {
	if in.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.companyDocument(ctx, comp, in.ChaveAcesso)
	if err != nil {
		return err
	}
	xmlData, err := s.storedXML(doc)
	if err != nil {
		return err
	}

	if err := writeFileAtomic(in.OutPath, xmlData, "XML"); err != nil {
		return err
	}
	if err := s.NFeRepo.MarkExported(ctx, comp.ID, nfe.ExportKindXML, []nfe.CompanyDocument{doc}); err != nil {
		return fmt.Errorf("marcar NF-e como exportada: %w", err)
	}
	return nil
}

// ExportDANFE renders the DANFE of one of the company's NF-e to in.OutPath
// and marks it as exported to DANFE. A resumo has no DANFE: it needs the
// complete XML. A cancelled NF-e is printed with the cancellation watermark.
func (s *NFeService) ExportDANFE(ctx context.Context, in NFeExportDANFEInput) error {
	if s.DANFERenderer == nil {
		return fmt.Errorf("DANFE não configurado")
	}
	if in.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.companyDocument(ctx, comp, in.ChaveAcesso)
	if err != nil {
		return err
	}
	if doc.Completeness == nfe.CompletenessResumo {
		return fmt.Errorf("a NF-e %s é um resumo: o DANFE precisa do XML completo", doc.ChaveAcesso)
	}
	xmlData, err := s.storedXML(doc)
	if err != nil {
		return err
	}
	pdf, err := s.DANFERenderer.Render(xmlData, danfe.Options{Cancelada: doc.Situacao == nfe.SituacaoCancelada})
	if err != nil {
		return fmt.Errorf("renderizar DANFE: %w", err)
	}

	if err := writeFileAtomic(in.OutPath, pdf, "DANFE"); err != nil {
		return err
	}
	if err := s.NFeRepo.MarkExported(ctx, comp.ID, nfe.ExportKindDANFE, []nfe.CompanyDocument{doc}); err != nil {
		return fmt.Errorf("marcar NF-e como exportada: %w", err)
	}
	return nil
}

// storedXML reads the current XML blob of doc.
func (s *NFeService) storedXML(doc nfe.CompanyDocument) ([]byte, error) {
	if doc.RawHash == "" {
		return nil, fmt.Errorf("XML original não encontrado para a chave %s", doc.ChaveAcesso)
	}
	xmlData, err := s.XMLStore.Get(doc.RawHash)
	if err != nil {
		return nil, fmt.Errorf("ler XML original da chave %s: %w", doc.ChaveAcesso, err)
	}
	return xmlData, nil
}

// exportDocuments lists the documents to export, with Incremental against
// the marks of kind, and how many resumos were left out because
// in.IncludeResumos is false.
func (s *NFeService) exportDocuments(ctx context.Context, comp *company.Company, in NFeExportInput, kind string) ([]nfe.CompanyDocument, int, error) {
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return nil, 0, err
	}
	chaves, err := parseAccessKeys(in.ChavesAcesso)
	if err != nil {
		return nil, 0, err
	}
	filter := nfe.DocumentFilter{
		Competence:   in.Competence,
		ChavesAcesso: chaves,
		TpAmb:        tpAmb,
	}
	if in.Role != "" {
		role, err := nfe.ParseCompanyRole(in.Role)
		if err != nil {
			return nil, 0, fmt.Errorf("papel inválido: %w", err)
		}
		filter.Role = role
	}

	var docs []nfe.CompanyDocument
	if in.Incremental {
		docs, err = s.NFeRepo.ListPendingExport(ctx, comp.ID, filter, kind)
	} else {
		docs, err = s.NFeRepo.ListCompanyDocuments(ctx, comp.ID, filter)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("listar NF-e: %w", err)
	}
	if in.IncludeResumos {
		return docs, 0, nil
	}

	var completas []nfe.CompanyDocument
	skipped := 0
	for _, doc := range docs {
		if doc.Completeness == nfe.CompletenessResumo {
			skipped++
			continue
		}
		completas = append(completas, doc)
	}
	return completas, skipped, nil
}
