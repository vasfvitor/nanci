package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/cte"
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

// CTeExportXMLInput identifies one of the company's CT-e and the destination
// XML path.
type CTeExportXMLInput struct {
	CNPJ        string
	ChaveAcesso string
	OutPath     string
}

// ExportXMLZip packs the XML of the matching CT-e and their events into a ZIP
// at in.OutPath and marks the documents as exported. With nothing to export
// it writes no file and returns an empty OutPath.
func (s *CTeService) ExportXMLZip(ctx context.Context, in CTeExportInput) (ExportResult, error) {
	res := ExportResult{
		OutPath:     in.OutPath,
		Format:      cte.ExportKindXML,
		Incremental: in.Incremental,
	}
	if in.OutPath == "" {
		return res, fmt.Errorf("caminho de saída não especificado")
	}

	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return res, err
	}
	filter, err := cteFilter(comp, ListCTeInput{Competence: in.Competence, Role: in.Role, ChavesAcesso: in.ChavesAcesso})
	if err != nil {
		return res, err
	}
	var docs []cte.CompanyDocument
	if in.Incremental {
		docs, err = s.CTeRepo.ListPendingExport(ctx, comp.ID, filter, cte.ExportKindXML)
	} else {
		docs, err = s.CTeRepo.ListCompanyDocuments(ctx, comp.ID, filter)
	}
	if err != nil {
		return res, fmt.Errorf("listar CT-e: %w", err)
	}
	res.ExportedCount = len(docs)
	if len(docs) == 0 {
		res.OutPath = ""
		return res, nil
	}

	var withEvents []string
	for _, doc := range docs {
		if doc.EventCount > 0 {
			withEvents = append(withEvents, string(doc.ChaveAcesso))
		}
	}
	events, err := s.CTeRepo.ListEventsByChaves(ctx, withEvents)
	if err != nil {
		return res, fmt.Errorf("listar eventos dos CT-e: %w", err)
	}
	eventsByChave := make(map[string][]cte.Event, len(withEvents))
	for _, e := range events {
		eventsByChave[string(e.ChaveAcesso)] = append(eventsByChave[string(e.ChaveAcesso)], e)
	}

	err = writeViaTemp(in.OutPath, func(tempPath string) error {
		return report.GenerateZIP(report.CTeZipEntries(docs, eventsByChave), s.XMLStore, tempPath)
	})
	if err != nil {
		return res, err
	}
	if err := s.CTeRepo.MarkExported(ctx, comp.ID, cte.ExportKindXML, docs); err != nil {
		return res, fmt.Errorf("marcar CT-e como exportados: %w", err)
	}
	return res, nil
}

// ExportXML writes the stored XML of one of the company's CT-e (procCTe,
// procCTeOS, procGTVe or procCTeSimp) to in.OutPath and marks it as exported.
func (s *CTeService) ExportXML(ctx context.Context, in CTeExportXMLInput) error {
	if in.OutPath == "" {
		return fmt.Errorf("caminho de saída não especificado")
	}
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
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
