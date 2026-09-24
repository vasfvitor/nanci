package app

import (
	"context"
	"fmt"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
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

// ExportXMLZip packs the XML of the matching NF-e and their complete events
// into a ZIP at in.OutPath and marks the documents as exported. With nothing
// to export it writes no file and returns an empty OutPath.
func (s *NFeService) ExportXMLZip(ctx context.Context, in NFeExportInput) (NFeExportResult, error) {
	res := NFeExportResult{ExportResult: ExportResult{
		OutPath:     in.OutPath,
		Format:      nfe.ExportKindXML,
		Incremental: in.Incremental,
	}}
	if in.OutPath == "" {
		return res, fmt.Errorf("caminho de saída não especificado")
	}

	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return res, err
	}
	docs, skipped, err := s.exportDocuments(ctx, comp, in)
	if err != nil {
		return res, err
	}
	res.SkippedResumos = skipped
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
	events, err := s.NFeRepo.ListEventsByChaves(ctx, withEvents)
	if err != nil {
		return res, fmt.Errorf("listar eventos das NF-e: %w", err)
	}
	eventsByChave := make(map[string][]nfe.Event, len(withEvents))
	for _, e := range events {
		eventsByChave[string(e.ChaveAcesso)] = append(eventsByChave[string(e.ChaveAcesso)], e)
	}

	err = writeViaTemp(in.OutPath, func(tempPath string) error {
		return report.GenerateZIP(report.NFeZipEntries(docs, eventsByChave), s.XMLStore, tempPath)
	})
	if err != nil {
		return res, err
	}
	if err := s.NFeRepo.MarkExported(ctx, comp.ID, nfe.ExportKindXML, docs); err != nil {
		return res, fmt.Errorf("marcar NF-e como exportadas: %w", err)
	}
	return res, nil
}

// ExportXML writes the stored XML of one of the company's NF-e (procNFe, or
// resNFe for a resumo) to in.OutPath and marks it as exported.
func (s *NFeService) ExportXML(ctx context.Context, in NFeExportXMLInput) error {
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
	if err := s.NFeRepo.MarkExported(ctx, comp.ID, nfe.ExportKindXML, []nfe.CompanyDocument{doc}); err != nil {
		return fmt.Errorf("marcar NF-e como exportada: %w", err)
	}
	return nil
}

// exportDocuments lists the documents to export and how many resumos were
// left out because in.IncludeResumos is false.
func (s *NFeService) exportDocuments(ctx context.Context, comp *nfse.Company, in NFeExportInput) ([]nfe.CompanyDocument, int, error) {
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return nil, 0, err
	}
	filter := nfe.DocumentFilter{
		Competence:   in.Competence,
		ChavesAcesso: in.ChavesAcesso,
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
		docs, err = s.NFeRepo.ListPendingExport(ctx, comp.ID, filter, nfe.ExportKindXML)
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
