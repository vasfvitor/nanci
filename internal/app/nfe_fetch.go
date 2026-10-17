package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/uf"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/sync"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// Outcomes of FetchByChave.
const (
	FetchOutcomeCompleta = "completa"  // the procNFe was stored
	FetchOutcomeResumo   = "resumo"    // only the resNFe came; Message says what to do
	FetchOutcomeNotFound = "not_found" // cStat 137
)

// fetchBlockWait is how long SEFAZ blocks the CNPJ after cStat 656.
const fetchBlockWait = time.Hour

// FetchByChaveInput names the NF-e to ask SEFAZ for.
type FetchByChaveInput struct {
	CNPJ        string
	ChaveAcesso string
}

// FetchByChaveResult is what SEFAZ delivered for the chave and how much of
// the hourly budget is spent after the request.
type FetchByChaveResult struct {
	ChaveAcesso      string
	Outcome          string // completa | resumo | not_found
	Message          string
	RequestsLastHour int
	RequestBudget    int
}

// FetchByChave asks the NF-e distribution for one NF-e by its chave
// (consChNFe) and stores what comes back as the distribution does, without
// moving the NSU cursor. The request spends one of the NF-e budget's 20 per
// hour. A consumo_indevido or rate_budget block refuses it before the
// password prompt; the hour of wait after the queue caught up does not,
// since it only applies to distNSU. A SEFAZ rejection is an error with its
// meaning spelled out.
func (s *NFeService) FetchByChave(ctx context.Context, in FetchByChaveInput) (FetchByChaveResult, error) {
	chave, err := dfe.ParseAccessKey(in.ChaveAcesso)
	if err != nil {
		return FetchByChaveResult{}, fmt.Errorf("chave de acesso inválida: %w", err)
	}
	if chave.Modelo() != "55" {
		return FetchByChaveResult{}, errors.New("só NF-e (modelo 55) pode ser buscada pela chave")
	}
	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return FetchByChaveResult{}, err
	}
	cUFAutor, err := companyUFCode(comp)
	if err != nil {
		return FetchByChaveResult{}, err
	}

	source := syncstate.SyncSourceNFe
	release, err := s.SyncManager.ReserveSource(comp.ID, source)
	if err != nil {
		return FetchByChaveResult{}, err
	}
	defer release()

	if err := s.SyncManager.BlockedFor(ctx, comp, source, syncstate.SyncStopReasonConsumoIndevido, syncstate.SyncStopReasonRateBudget); err != nil {
		return FetchByChaveResult{}, err
	}
	spent, err := s.SyncManager.SpendRequest(ctx, comp, source)
	if err != nil {
		return FetchByChaveResult{}, err
	}
	if !spent {
		// SpendRequest has just blocked the source with rate_budget.
		if err := s.SyncManager.BlockedFor(ctx, comp, source, syncstate.SyncStopReasonRateBudget); err != nil {
			return FetchByChaveResult{}, err
		}
		return FetchByChaveResult{}, errors.New("limite de consultas por hora de NF-e atingido")
	}

	loaded, err := s.Certificates.LoadForCompany(ctx, comp, "Busca de NF-e pela chave")
	if err != nil {
		return FetchByChaveResult{}, err
	}
	client, err := newSEFAZClient(sefaz.ClientConfig{
		Environment: comp.Environment,
		Certificate: &loaded.TLS,
		Log:         s.Log,
		Endpoints:   s.SyncManager.SEFAZEndpoints,
	})
	if err != nil {
		return FetchByChaveResult{}, fmt.Errorf("configurar cliente SEFAZ: %w", err)
	}
	resp, err := client.ConsChNFe(ctx, comp.CNPJ, cUFAutor, chave.String())
	var rejection *sefaz.RejectionError
	if errors.As(err, &rejection) {
		return FetchByChaveResult{}, consChNFeRejection(rejection)
	}
	if err != nil {
		return FetchByChaveResult{}, fmt.Errorf("consultar NF-e pela chave: %w", err)
	}

	result := FetchByChaveResult{ChaveAcesso: chave.String()}
	switch resp.CStat {
	case sefaz.CStatConsumoIndevido:
		until := s.now().UTC().Add(fetchBlockWait)
		if err := s.SyncRepo.SetBlockedUntil(ctx, comp.ID, source, comp.Environment, until, syncstate.SyncStopReasonConsumoIndevido); err != nil {
			return FetchByChaveResult{}, fmt.Errorf("bloquear consultas de NF-e: %w", err)
		}
		return FetchByChaveResult{}, &sync.BlockedError{Source: source, Until: until, Reason: syncstate.SyncStopReasonConsumoIndevido}
	case sefaz.CStatNenhumDocumento:
		result.Outcome = FetchOutcomeNotFound
		result.Message = "NF-e não localizada ou sem acesso para a empresa"
	default: // sefaz.CStatDocumentoLocalizado
		if err := s.storeFetched(ctx, comp, cUFAutor, resp.Docs, &result); err != nil {
			return FetchByChaveResult{}, err
		}
	}

	limits, err := s.SyncManager.SourceLimits(ctx, comp, source)
	if err != nil {
		return FetchByChaveResult{}, err
	}
	result.RequestsLastHour = limits.RequestsLastHour
	result.RequestBudget = limits.RequestBudget
	return result, nil
}

// storeFetched stores the docZips of a consChNFe answer through the NF-e
// source, as the distribution would, and sets the outcome from the
// document schema. The answer's NSUs are recorded on the document but do
// not move the cursor.
func (s *NFeService) storeFetched(ctx context.Context, comp *company.Company, cUFAutor int, docs []sefaz.DocZip, result *FetchByChaveResult) error {
	src := sync.NewNFeSource(nil, s.NFeRepo, s.XMLStore, s.Log, cUFAutor)
	for _, dz := range docs {
		item := sync.Item{NSU: dz.NSU, Schema: dz.Schema, Payload: dz.Content}
		if _, err := src.ProcessItem(ctx, comp, sync.SourceState{}, item, s.SyncRepo.Apply); err != nil {
			return fmt.Errorf("gravar NF-e recebida: %w", err)
		}
		switch nfe.ClassifySchema(dz.Schema) {
		case nfe.SchemaProcNFe:
			result.Outcome = FetchOutcomeCompleta
			result.Message = ""
		case nfe.SchemaResNFe:
			if result.Outcome != FetchOutcomeCompleta {
				result.Outcome = FetchOutcomeResumo
				result.Message = "a SEFAZ entregou só o resumo: registre a Ciência da Operação para receber o XML completo"
			}
		}
	}
	if result.Outcome == "" {
		result.Outcome = FetchOutcomeNotFound
		result.Message = "a SEFAZ não entregou o documento da NF-e"
	}
	return nil
}

// consChNFeRejection spells out a consChNFe rejection.
func consChNFeRejection(rejection *sefaz.RejectionError) error {
	var meaning string
	switch rejection.CStat {
	case 618:
		meaning = "modelo não aceito pela consulta por chave"
	case 632:
		meaning = "chave fora da janela de 90 dias da distribuição"
	case 640:
		meaning = "a empresa não é destinatária, transportadora nem autorizada nesta NF-e"
	case 641:
		meaning = "a SEFAZ não entrega ao emitente a própria NF-e; importe o XML"
	case 653:
		meaning = "NF-e cancelada"
	case 654:
		meaning = "NF-e denegada"
	default:
		return fmt.Errorf("cStat %d: %s", rejection.CStat, rejection.XMotivo)
	}
	return fmt.Errorf("%s (cStat %d)", meaning, rejection.CStat)
}

// companyUFCode is the IBGE code of the company's UF, sent as cUFAutor.
func companyUFCode(comp *company.Company) (int, error) {
	if comp.UF == "" {
		return 0, errors.New("empresa sem UF cadastrada")
	}
	code, ok := uf.Code(comp.UF)
	if !ok {
		return 0, fmt.Errorf("UF %q da empresa é inválida", comp.UF)
	}
	return code, nil
}
