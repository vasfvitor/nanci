package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// importMaxBytes bounds one imported XML file, the same decoded size the
// distribution accepts for one document.
const importMaxBytes = 20 * 1024 * 1024

// Kinds of an imported file. The CT-e has no resumo.
const (
	ImportKindCompleta = "completa"
	ImportKindResumo   = "resumo"
	ImportKindEvento   = "evento"
)

// Outcomes of an imported file.
const (
	ImportOutcomeImported = "imported" // the company did not hold it before
	ImportOutcomeUpdated  = "updated"  // merged with what the company held
	ImportOutcomeRejected = "rejected" // not stored; Message says why
)

// ImportXMLInput lists the XML files to import for the company.
type ImportXMLInput struct {
	CNPJ  string
	Files []string
}

// ImportedFile is the result of one file. Kind and ChaveAcesso are empty
// when the file was rejected before they were known.
type ImportedFile struct {
	Path        string
	Kind        string // completa | resumo | evento
	ChaveAcesso string
	Outcome     string // imported | updated | rejected
	Message     string
}

// ImportXMLResult is one ImportedFile per input file, in input order, and
// the counts per outcome.
type ImportXMLResult struct {
	Files    []ImportedFile
	Imported int
	Updated  int
	Rejected int
}

func (r *ImportXMLResult) add(f ImportedFile) {
	r.Files = append(r.Files, f)
	switch f.Outcome {
	case ImportOutcomeImported:
		r.Imported++
	case ImportOutcomeUpdated:
		r.Updated++
	default:
		r.Rejected++
	}
}

// ImportXML stores NF-e and NF-e event XML files the company holds, such as
// the ones its own system issued or a supplier sent, without contacting
// SEFAZ. A file is rejected, and the others still imported, when it cannot
// be read or parsed, belongs to the other environment, names the company in
// no role, or is an event for an NF-e the company does not hold and did not
// author. Imported documents have no NSU. Only a missing company fails the
// call.
func (s *NFeService) ImportXML(ctx context.Context, in ImportXMLInput) (ImportXMLResult, error) {
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
func (s *NFeService) importFile(ctx context.Context, comp *company.Company, tpAmb, path string, file *ImportedFile) error {
	data, err := readImportFile(path)
	if err != nil {
		return err
	}
	root, err := dfe.RootElement(data)
	if err != nil {
		return err
	}
	kind, ok := nfe.KindOfRoot(root)
	if !ok {
		return fmt.Errorf("não é um XML de NF-e: raiz <%s>", root)
	}

	switch kind {
	case nfe.KindProcNFe, nfe.KindResNFe:
		parse := nfe.ParseProcNFe
		if kind == nfe.KindResNFe {
			parse = nfe.ParseResNFe
		}
		doc, err := parse(data)
		if err != nil {
			return err
		}
		file.ChaveAcesso = string(doc.ChaveAcesso)
		file.Kind = ImportKindCompleta
		if doc.Completeness == nfe.CompletenessResumo {
			file.Kind = ImportKindResumo
		}
		if doc.TpAmb, err = importTpAmb(doc.TpAmb, tpAmb, &doc.ParseWarnings); err != nil {
			return err
		}
		if nfe.ClassifyParticipation(&doc, comp.CNPJ).CompanyRole == nfe.CompanyRoleNone {
			return fmt.Errorf("a empresa %s não é parte desta NF-e", cnpj.Format(comp.CNPJ))
		}
		if doc.RawHash, err = storeXML(s.XMLStore, data); err != nil {
			return err
		}
		inserted, err := s.NFeRepo.ApplyDocument(ctx, store.ApplyNFeDocumentParams{
			Document:    doc,
			CompanyID:   comp.ID,
			CompanyCNPJ: comp.CNPJ,
		})
		if err != nil {
			return fmt.Errorf("gravar NF-e: %w", err)
		}
		file.Outcome = importOutcome(inserted)
		return nil

	default: // nfe.KindProcEventoNFe, nfe.KindResEvento
		parse := nfe.ParseProcEventoNFe
		if kind == nfe.KindResEvento {
			parse = nfe.ParseResEvento
		}
		ev, err := parse(data)
		if err != nil {
			return err
		}
		file.ChaveAcesso = string(ev.ChaveAcesso)
		file.Kind = ImportKindEvento
		if ev.TpAmb, err = importTpAmb(ev.TpAmb, tpAmb, &ev.ParseWarnings); err != nil {
			return err
		}
		// As in the distribution: an event for an NF-e the company does not
		// hold is dropped, unless the company authored it.
		if cnpj.Clean(ev.AutorCNPJ) != comp.CNPJ {
			exists, err := s.NFeRepo.CompanyDocumentExists(ctx, comp.ID, string(ev.ChaveAcesso))
			if err != nil {
				return fmt.Errorf("buscar NF-e do evento: %w", err)
			}
			if !exists {
				return errors.New("NF-e desconhecida para a empresa; importe o XML da nota primeiro")
			}
		}
		if ev.RawHash, err = storeXML(s.XMLStore, data); err != nil {
			return err
		}
		outcome, err := s.SyncRepo.Apply(ctx, func(tx *sql.Tx) (sync.ItemOutcome, error) {
			inserted, err := s.NFeRepo.ApplyEventTx(ctx, tx, store.ApplyNFeEventParams{Event: ev})
			return sync.ItemOutcome{Inserted: inserted}, err
		})
		if err != nil {
			return fmt.Errorf("gravar evento da NF-e: %w", err)
		}
		file.Outcome = importOutcome(outcome.Inserted)
		return nil
	}
}

// storeXML keeps data in the blob store under its sha256 and returns the
// hash.
func storeXML(xmlStore files.XMLStore, data []byte) (string, error) {
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if err := xmlStore.Store(hash, data); err != nil {
		return "", fmt.Errorf("salvar XML: %w", err)
	}
	return hash, nil
}

// readImportFile reads one file, refusing one larger than importMaxBytes.
func readImportFile(path string) ([]byte, error) {
	f, err := os.Open(path) // #nosec G304 -- the user picked the file to import.
	if err != nil {
		return nil, fmt.Errorf("ler arquivo: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, importMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("ler arquivo: %w", err)
	}
	if len(data) > importMaxBytes {
		return nil, fmt.Errorf("arquivo maior que %d MiB", importMaxBytes/(1024*1024))
	}
	return data, nil
}

// importTpAmb is the tpAmb to store for an imported XML of a company whose
// environment is tpAmb, as sync.CheckTpAmb decides it; an XML of the other
// environment is refused instead of stored with a warning.
func importTpAmb(xmlTpAmb, tpAmb string, warnings *[]string) (string, error) {
	got := sync.CheckTpAmb(xmlTpAmb, tpAmb, warnings)
	switch got {
	case tpAmb:
		return got, nil
	case sefaz.TpAmbHomologacao:
		return "", errors.New("XML de homologação em empresa de produção")
	default:
		return "", errors.New("XML de produção em empresa de homologação")
	}
}

func importOutcome(inserted bool) string {
	if inserted {
		return ImportOutcomeImported
	}
	return ImportOutcomeUpdated
}
