package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dacte"
	"github.com/vasfvitor/nanci/internal/report"
)

// CTeExportInput selects the CT-e to export. Empty filters select all the
// documents of the company's current environment.
type CTeExportInput struct {
	CNPJ       string
	Competence string // "YYYY-MM" of the issue date
	// Role matches the primary role or any other role the company plays.
	Role         string
	ChavesAcesso []string
	// Incremental exports only documents not exported before, or whose XML
	// changed since.
	Incremental bool
	OutPath     string
}

// CTeExportResult is ExportResult plus the documents a DACTE export left
// out because their kind has no DACTE.
type CTeExportResult struct {
	ExportResult
	// SkippedSemDACTE counts the CT-e OS, GTV-e and CT-e Simplificado a
	// DACTE export left out. They are not marked as exported, so the next
	// incremental export counts them again.
	SkippedSemDACTE int
}

// CTeExportXMLInput identifies one of the company's CT-e and the destination
// XML path.
type CTeExportXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// CTeExportDACTEInput identifies one of the company's CT-e and the
// destination PDF path.
type CTeExportDACTEInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// errSemDACTE is the error of a single DACTE export of a document kind the
// renderer has no layout for.
var errSemDACTE = errors.New("DACTE disponível só para o CT-e modelo 57; CT-e OS, GTV-e e CT-e Simplificado não têm DACTE")

// ExportXMLZip packs the XML of the matching CT-e and their events into a ZIP
// at in.OutPath and marks the documents as exported. With nothing to export
// it writes no file and returns an empty OutPath.
func (s *CTeService) ExportXMLZip(ctx context.Context, in CTeExportInput) (CTeExportResult, error) {
	return s.bulkExport(ctx, in, cte.ExportKindXML, nil, func(docs []cte.CompanyDocument, tempPath string) error {
		events, err := s.eventsByChave(ctx, docs)
		if err != nil {
			return err
		}
		return report.GenerateZIP(report.CTeZipEntries(docs, events), s.XMLStore, tempPath)
	})
}

// ExportCSV writes a CSV report of the matching CT-e to in.OutPath.
func (s *CTeService) ExportCSV(ctx context.Context, in CTeExportInput) (CTeExportResult, error) {
	return s.bulkExport(ctx, in, cte.ExportKindCSV, nil, func(docs []cte.CompanyDocument, tempPath string) error {
		return report.WriteCSV(report.CTeSpreadsheet(report.BuildCTeRows(docs)), tempPath)
	})
}

// ExportXLSX writes an Excel report of the matching CT-e to in.OutPath.
func (s *CTeService) ExportXLSX(ctx context.Context, in CTeExportInput) (CTeExportResult, error) {
	return s.bulkExport(ctx, in, cte.ExportKindXLSX, nil, func(docs []cte.CompanyDocument, tempPath string) error {
		return report.WriteXLSX(report.CTeSpreadsheet(report.BuildCTeRows(docs)), tempPath)
	})
}

// ExportDACTEZip writes one DACTE PDF per matching CT-e modelo 57 into a ZIP
// at in.OutPath. CT-e OS, GTV-e and CT-e Simplificado are left out and
// counted in SkippedSemDACTE.
func (s *CTeService) ExportDACTEZip(ctx context.Context, in CTeExportInput) (CTeExportResult, error) {
	if s.DACTERenderer == nil {
		return CTeExportResult{}, fmt.Errorf("DACTE não configurado")
	}
	return s.bulkExport(ctx, in, cte.ExportKindDACTE, cte.CompanyDocument.HasDACTE, func(docs []cte.CompanyDocument, tempPath string) error {
		return report.GeneratePDFZIP(report.CTePDFEntries(docs), s.XMLStore, s.renderDACTE, tempPath)
	})
}

// bulkExport lists the matching CT-e, keeps those keep accepts (all of them
// when keep is nil), writes the kept ones with gen through a temporary file
// and marks them as exported under kind. Documents keep rejects are counted
// in SkippedSemDACTE and left unmarked. With nothing to write it writes no
// file and returns an empty OutPath.
func (s *CTeService) bulkExport(ctx context.Context, in CTeExportInput, kind string, keep func(cte.CompanyDocument) bool, gen func([]cte.CompanyDocument, string) error) (CTeExportResult, error) {
	res := CTeExportResult{ExportResult: ExportResult{
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
	filter, err := cteFilter(comp, ListCTeInput{Competence: in.Competence, Role: in.Role, ChavesAcesso: in.ChavesAcesso})
	if err != nil {
		return res, err
	}
	var listed []cte.CompanyDocument
	if in.Incremental {
		listed, err = s.CTeRepo.ListPendingExport(ctx, comp.ID, filter, kind)
	} else {
		listed, err = s.CTeRepo.ListCompanyDocuments(ctx, comp.ID, filter)
	}
	if err != nil {
		return res, fmt.Errorf("listar CT-e: %w", err)
	}

	docs := listed
	if keep != nil {
		docs = make([]cte.CompanyDocument, 0, len(listed))
		for _, doc := range listed {
			if keep(doc) {
				docs = append(docs, doc)
			}
		}
		res.SkippedSemDACTE = len(listed) - len(docs)
	}
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
	if err := s.CTeRepo.MarkExported(ctx, comp.ID, kind, docs); err != nil {
		return res, fmt.Errorf("marcar CT-e como exportados: %w", err)
	}
	return res, nil
}

// eventsByChave loads the stored events of the documents that have any,
// grouped by chave.
func (s *CTeService) eventsByChave(ctx context.Context, docs []cte.CompanyDocument) (map[string][]cte.Event, error) {
	var withEvents []string
	for _, doc := range docs {
		if doc.EventCount > 0 {
			withEvents = append(withEvents, string(doc.ChaveAcesso))
		}
	}
	events, err := s.CTeRepo.ListEventsByChaves(ctx, withEvents)
	if err != nil {
		return nil, fmt.Errorf("listar eventos dos CT-e: %w", err)
	}
	byChave := make(map[string][]cte.Event, len(withEvents))
	for _, e := range events {
		byChave[string(e.ChaveAcesso)] = append(byChave[string(e.ChaveAcesso)], e)
	}
	return byChave, nil
}

// ExportXML writes the stored XML of one of the company's CT-e (procCTe,
// procCTeOS, procGTVe or procCTeSimp) to in.OutPath and marks it as exported.
func (s *CTeService) ExportXML(ctx context.Context, in CTeExportXMLInput) error {
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
	if doc.RawHash == "" {
		return fmt.Errorf("XML original não encontrado para a chave %s", doc.ChaveAcesso)
	}
	xmlData, err := s.XMLStore.Get(doc.RawHash)
	if err != nil {
		return fmt.Errorf("ler XML original da chave %s: %w", doc.ChaveAcesso, err)
	}

	if err := writeFileAtomic(in.OutPath, xmlData, "XML"); err != nil {
		return err
	}
	if err := s.CTeRepo.MarkExported(ctx, comp.ID, cte.ExportKindXML, []cte.CompanyDocument{doc}); err != nil {
		return fmt.Errorf("marcar CT-e como exportado: %w", err)
	}
	return nil
}

// ExportDACTE writes the DACTE PDF of one of the company's CT-e modelo 57 to
// in.OutPath and marks it as exported. A cancelled CT-e gets the
// cancellation watermark.
func (s *CTeService) ExportDACTE(ctx context.Context, in CTeExportDACTEInput) error {
	if in.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	if s.DACTERenderer == nil {
		return fmt.Errorf("DACTE não configurado")
	}
	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return err
	}
	doc, err := s.companyDocument(ctx, comp, in.ChaveAcesso)
	if err != nil {
		return err
	}
	if !doc.HasDACTE() {
		return errSemDACTE
	}
	if doc.RawHash == "" {
		return fmt.Errorf("XML original não encontrado para a chave %s", doc.ChaveAcesso)
	}
	xmlData, err := s.XMLStore.Get(doc.RawHash)
	if err != nil {
		return fmt.Errorf("ler XML original da chave %s: %w", doc.ChaveAcesso, err)
	}
	pdf, err := s.renderDACTE(xmlData, doc.Situacao == cte.SituacaoCancelada)
	if err != nil {
		return err
	}

	if err := writeFileAtomic(in.OutPath, pdf, "DACTE"); err != nil {
		return err
	}
	if err := s.CTeRepo.MarkExported(ctx, comp.ID, cte.ExportKindDACTE, []cte.CompanyDocument{doc}); err != nil {
		return fmt.Errorf("marcar CT-e como exportado: %w", err)
	}
	return nil
}

// renderDACTE renders one DACTE, with the cancellation watermark when
// cancelada.
func (s *CTeService) renderDACTE(xml []byte, cancelada bool) ([]byte, error) {
	pdf, err := s.DACTERenderer.Render(xml, dacte.Options{Cancelada: cancelada})
	if err != nil {
		return nil, fmt.Errorf("renderizar DACTE: %w", err)
	}
	return pdf, nil
}
