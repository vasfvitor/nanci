package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// ImportXML stores CT-e (CT-e, CT-e OS, GTV-e, CT-e Simplificado) and CT-e
// event XML files the company holds, without contacting SEFAZ, with the
// same rules as NFeService.ImportXML.
func (s *CTeService) ImportXML(ctx context.Context, in ImportXMLInput) (ImportXMLResult, error) {
	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return ImportXMLResult{}, err
	}
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return ImportXMLResult{}, err
	}

	var result ImportXMLResult
	for _, path := range in.Files {
		file := ImportedFile{Path: path}
		if err := s.importFile(ctx, comp, tpAmb, path, &file); err != nil {
			file.Outcome = ImportOutcomeRejected
			file.Message = err.Error()
		}
		result.add(file)
	}
	return result, nil
}

// importFile stores one file and fills Kind, ChaveAcesso and Outcome. An
// error rejects the file.
func (s *CTeService) importFile(ctx context.Context, comp *company.Company, tpAmb, path string, file *ImportedFile) error {
	data, err := readImportFile(path)
	if err != nil {
		return err
	}
	root, err := dfe.RootElement(data)
	if err != nil {
		return err
	}
	kind, ok := cte.KindOfRoot(root)
	if !ok {
		return fmt.Errorf("não é um XML de CT-e: raiz <%s>", root)
	}

	if kind == cte.KindProcCTe {
		doc, err := cte.ParseProcCTe(data)
		if err != nil {
			return err
		}
		file.ChaveAcesso = string(doc.ChaveAcesso)
		file.Kind = ImportKindCompleta
		if doc.TpAmb, err = importTpAmb(doc.TpAmb, tpAmb, &doc.ParseWarnings); err != nil {
			return err
		}
		if cte.ClassifyParticipation(&doc, comp.CNPJ).CompanyRole == cte.CompanyRoleNone {
			return fmt.Errorf("a empresa %s não é parte deste CT-e", cnpj.Format(comp.CNPJ))
		}
		if doc.RawHash, err = storeXML(s.XMLStore, data); err != nil {
			return err
		}
		inserted, err := s.CTeRepo.ApplyDocument(ctx, store.ApplyCTeDocumentParams{
			Document:    doc,
			CompanyID:   comp.ID,
			CompanyCNPJ: comp.CNPJ,
		})
		if err != nil {
			return fmt.Errorf("gravar CT-e: %w", err)
		}
		file.Outcome = importOutcome(inserted)
		return nil
	}

	ev, err := cte.ParseProcEventoCTe(data)
	if err != nil {
		return err
	}
	file.ChaveAcesso = string(ev.ChaveAcesso)
	file.Kind = ImportKindEvento
	if ev.TpAmb, err = importTpAmb(ev.TpAmb, tpAmb, &ev.ParseWarnings); err != nil {
		return err
	}
	// As in the distribution: an event for a CT-e the company does not hold
	// is dropped, unless the company authored it.
	if cnpj.Clean(ev.AutorCNPJ) != comp.CNPJ {
		exists, err := s.CTeRepo.CompanyDocumentExists(ctx, comp.ID, string(ev.ChaveAcesso))
		if err != nil {
			return fmt.Errorf("buscar CT-e do evento: %w", err)
		}
		if !exists {
			return errors.New("CT-e desconhecido para a empresa; importe o XML do CT-e primeiro")
		}
	}
	if ev.RawHash, err = storeXML(s.XMLStore, data); err != nil {
		return err
	}
	outcome, err := s.SyncRepo.Apply(ctx, func(tx *sql.Tx) (sync.ItemOutcome, error) {
		inserted, err := s.CTeRepo.ApplyEventTx(ctx, tx, store.ApplyCTeEventParams{Event: ev})
		return sync.ItemOutcome{Inserted: inserted}, err
	})
	if err != nil {
		return fmt.Errorf("gravar evento do CT-e: %w", err)
	}
	file.Outcome = importOutcome(outcome.Inserted)
	return nil
}
