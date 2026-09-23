package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// Outcomes of one manifestação, per chave. The first three are also the
// statuses stored for answered events.
const (
	NFeOutcomeRegistrada   = nfe.ManifestationStatusRegistrada   // SEFAZ registered the event (135/136)
	NFeOutcomeJaRegistrada = nfe.ManifestationStatusJaRegistrada // nothing left to do (573, 655)
	NFeOutcomeRejeitada    = nfe.ManifestationStatusRejeitada    // SEFAZ refused the event; see CStat and XMotivo
	NFeOutcomeNaoEnviada   = "nao_enviada"                       // the lote got no SEFAZ answer, or was never sent
)

// NFeCienciaInput selects the NF-e for Ciência da Operação: either the
// listed chaves or, with AllResumos, every resumo still waiting for it.
type NFeCienciaInput struct {
	CNPJ         string
	ChavesAcesso []string
	// AllResumos selects every resumo addressed to the company, authorized
	// and without manifestação.
	AllResumos bool
}

// NFeCandidate is an NF-e that can receive Ciência da Operação.
type NFeCandidate struct {
	ChaveAcesso   string
	Serie         string
	Numero        string
	EmitenteCNPJ  string
	EmitenteName  string
	IssueDate     time.Time
	TotalValue    nfse.Money
	CienciaDue    time.Time
	ConclusiveDue time.Time
}

// NFeSkipped is a requested chave that will not be sent, and why.
type NFeSkipped struct {
	ChaveAcesso string
	Reason      string
}

// NFeCienciaPlan is what RegisterCiencia would send.
type NFeCienciaPlan struct {
	Eligible []NFeCandidate
	Skipped  []NFeSkipped
	Lotes    int // lotes of up to sefaz.MaxEventosPorLote eventos
}

// NFeEventOutcome is the result of one manifestação event.
type NFeEventOutcome struct {
	ChaveAcesso  string
	TpEvento     string
	Status       string // registrada | ja_registrada | rejeitada | nao_enviada
	CStat        string // empty when SEFAZ did not answer
	XMotivo      string
	Protocolo    string
	RegisteredAt *time.Time
}

// NFeManifestationSummary is the result of RegisterCiencia.
type NFeManifestationSummary struct {
	Requested         int // eligible chaves
	Registered        int
	AlreadyRegistered int
	Rejected          int
	NotSent           int
	Outcomes          []NFeEventOutcome
	Skipped           []NFeSkipped
	// Interrupted is the error that stopped the sending, such as a transport
	// failure; the lotes after it were not sent. Empty when every lote was
	// sent.
	Interrupted string
}

// NFeManifestationInput is one conclusive manifestação.
type NFeManifestationInput struct {
	CNPJ        string
	ChaveAcesso string
	// Tipo is confirmacao, desconhecimento or nao_realizada (nao-realizada
	// also works), or the tpEvento code 210200, 210220 or 210240.
	Tipo string
	// Justificativa is required (15 to 255 characters) for nao_realizada and
	// must be empty otherwise.
	Justificativa string
}

// PlanCiencia checks which NF-e can receive Ciência da Operação. It makes no
// network call and asks for no password.
func (s *NFeService) PlanCiencia(ctx context.Context, in NFeCienciaInput) (NFeCienciaPlan, error) {
	_, plan, err := s.planCiencia(ctx, in)
	return plan, err
}

// RegisterCiencia sends Ciência da Operação for the eligible NF-e, in lotes
// of up to 20, asking for the certificate password once. It returns an error
// only when nothing was sent (invalid input, no eligible NF-e, password
// canceled, certificate problem). Once sending starts, failures are reported
// per chave, and a lote without answer stops the sending and sets
// Interrupted.
func (s *NFeService) RegisterCiencia(ctx context.Context, in NFeCienciaInput) (NFeManifestationSummary, error) {
	comp, plan, err := s.planCiencia(ctx, in)
	if err != nil {
		return NFeManifestationSummary{}, err
	}
	if len(plan.Eligible) == 0 {
		return NFeManifestationSummary{}, fmt.Errorf("nenhuma NF-e elegível para ciência (%d ignoradas)", len(plan.Skipped))
	}

	sender, err := s.newSender(ctx, comp, fmt.Sprintf("Assinatura: Ciência da Operação (%d notas)", len(plan.Eligible)))
	if err != nil {
		return NFeManifestationSummary{}, err
	}

	summary := NFeManifestationSummary{
		Requested: len(plan.Eligible),
		Skipped:   plan.Skipped,
	}
	for lote := range slices.Chunk(plan.Eligible, sefaz.MaxEventosPorLote) {
		eventos := make([]sefaz.Evento, 0, len(lote))
		for _, c := range lote {
			eventos = append(eventos, s.evento(comp, c.ChaveAcesso, nfe.ManifestationCiencia, ""))
		}

		var outcomes []NFeEventOutcome
		if summary.Interrupted == "" {
			outcomes, err = sender.send(ctx, eventos)
			if err != nil {
				summary.Interrupted = err.Error()
			}
		} else {
			outcomes = notSentOutcomes(eventos)
		}
		for _, o := range outcomes {
			switch o.Status {
			case NFeOutcomeRegistrada:
				summary.Registered++
			case NFeOutcomeJaRegistrada:
				summary.AlreadyRegistered++
			case NFeOutcomeRejeitada:
				summary.Rejected++
			default:
				summary.NotSent++
			}
		}
		summary.Outcomes = append(summary.Outcomes, outcomes...)
	}
	return summary, nil
}

// RegisterManifestation sends one conclusive manifestação (confirmação,
// desconhecimento or operação não realizada). Tipo, justificativa, the
// company's role and the NF-e situação are checked before the password
// prompt. The deadline is not checked: SEFAZ decides (cStat 596). An event
// SEFAZ answered is returned without error, whatever its outcome. A request
// without answer, or an answer that could not be recorded, is an error; the
// outcome is returned with it.
func (s *NFeService) RegisterManifestation(ctx context.Context, in NFeManifestationInput) (NFeEventOutcome, error) {
	tipo, err := parseConclusiveManifestation(in.Tipo)
	if err != nil {
		return NFeEventOutcome{}, err
	}
	xJust, err := nfe.ValidateJustificativa(tipo, in.Justificativa)
	if err != nil {
		if tipo == nfe.ManifestationNaoRealizada {
			return NFeEventOutcome{}, fmt.Errorf("justificativa inválida: informe de %d a %d caracteres", nfe.JustificativaMinLength, nfe.JustificativaMaxLength)
		}
		return NFeEventOutcome{}, fmt.Errorf("justificativa só é aceita para operação não realizada")
	}

	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return NFeEventOutcome{}, err
	}
	doc, err := s.companyDocument(ctx, comp.ID, in.ChaveAcesso)
	if err != nil {
		return NFeEventOutcome{}, err
	}
	if reason := manifestationBlockReason(doc); reason != "" {
		return NFeEventOutcome{}, fmt.Errorf("NF-e %s: %s", doc.ChaveAcesso, reason)
	}
	if doc.Manifestacao == manifestacaoAfter(tipo) {
		return NFeEventOutcome{}, fmt.Errorf("NF-e %s já tem %s registrada", doc.ChaveAcesso, manifestationLabel(tipo))
	}

	sender, err := s.newSender(ctx, comp, "Assinatura: "+manifestationLabel(tipo))
	if err != nil {
		return NFeEventOutcome{}, err
	}
	outcomes, err := sender.send(ctx, []sefaz.Evento{s.evento(comp, string(doc.ChaveAcesso), tipo, xJust)})
	if err != nil {
		return outcomes[0], fmt.Errorf("enviar manifestação: %w", err)
	}
	return outcomes[0], nil
}

// planCiencia resolves the company and splits the requested chaves into
// eligible and skipped.
func (s *NFeService) planCiencia(ctx context.Context, in NFeCienciaInput) (*nfse.Company, NFeCienciaPlan, error) {
	switch {
	case in.AllResumos && len(in.ChavesAcesso) > 0:
		return nil, NFeCienciaPlan{}, errors.New("informe as chaves ou todas as pendentes, não ambos")
	case !in.AllResumos && len(in.ChavesAcesso) == 0:
		return nil, NFeCienciaPlan{}, errors.New("informe ao menos uma chave de acesso")
	}
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return nil, NFeCienciaPlan{}, err
	}

	var plan NFeCienciaPlan
	if in.AllResumos {
		docs, err := s.NFeRepo.ListCompanyDocuments(ctx, comp.ID, nfe.DocumentFilter{
			Role:         nfe.CompanyRoleDestinatario,
			Situacao:     nfe.SituacaoAutorizada,
			Completeness: nfe.CompletenessResumo,
			Manifestacao: nfe.ManifestacaoNenhuma,
		})
		if err != nil {
			return nil, NFeCienciaPlan{}, fmt.Errorf("listar NF-e: %w", err)
		}
		for _, doc := range docs {
			plan.Eligible = append(plan.Eligible, candidateFrom(doc))
		}
	} else {
		if err := s.planChaves(ctx, comp.ID, in.ChavesAcesso, &plan); err != nil {
			return nil, NFeCienciaPlan{}, err
		}
	}
	plan.Lotes = (len(plan.Eligible) + sefaz.MaxEventosPorLote - 1) / sefaz.MaxEventosPorLote
	return comp, plan, nil
}

func (s *NFeService) planChaves(ctx context.Context, companyID nfse.CompanyID, rawChaves []string, plan *NFeCienciaPlan) error {
	var chaves []string
	seen := make(map[string]bool, len(rawChaves))
	for _, raw := range rawChaves {
		chave, err := nfe.ParseAccessKey(raw)
		if err != nil {
			plan.Skipped = append(plan.Skipped, NFeSkipped{ChaveAcesso: strings.TrimSpace(raw), Reason: "chave de acesso inválida"})
			continue
		}
		if seen[string(chave)] {
			plan.Skipped = append(plan.Skipped, NFeSkipped{ChaveAcesso: string(chave), Reason: "chave repetida"})
			continue
		}
		seen[string(chave)] = true
		chaves = append(chaves, string(chave))
	}
	if len(chaves) == 0 {
		return nil
	}

	docs, err := s.NFeRepo.ListCompanyDocuments(ctx, companyID, nfe.DocumentFilter{ChavesAcesso: chaves})
	if err != nil {
		return fmt.Errorf("listar NF-e: %w", err)
	}
	byChave := make(map[string]nfe.CompanyDocument, len(docs))
	for _, doc := range docs {
		byChave[string(doc.ChaveAcesso)] = doc
	}

	for _, chave := range chaves {
		doc, ok := byChave[chave]
		if !ok {
			plan.Skipped = append(plan.Skipped, NFeSkipped{ChaveAcesso: chave, Reason: "não encontrada para a empresa"})
			continue
		}
		reason := manifestationBlockReason(doc)
		if reason == "" && doc.Manifestacao != nfe.ManifestacaoNenhuma {
			reason = "já manifestada (" + string(doc.Manifestacao) + ")"
		}
		if reason != "" {
			plan.Skipped = append(plan.Skipped, NFeSkipped{ChaveAcesso: chave, Reason: reason})
			continue
		}
		plan.Eligible = append(plan.Eligible, candidateFrom(doc))
	}
	return nil
}

// manifestationBlockReason says why the company cannot manifest on doc at
// all, or "" when it can.
func manifestationBlockReason(doc nfe.CompanyDocument) string {
	if doc.CompanyRole != nfe.CompanyRoleDestinatario {
		return "a empresa não é a destinatária"
	}
	if doc.Situacao != nfe.SituacaoAutorizada {
		return "NF-e " + string(doc.Situacao)
	}
	return ""
}

func candidateFrom(doc nfe.CompanyDocument) NFeCandidate {
	deadlines := nfe.ManifestationDeadlines(doc.Document)
	return NFeCandidate{
		ChaveAcesso:   string(doc.ChaveAcesso),
		Serie:         doc.Serie,
		Numero:        doc.Numero,
		EmitenteCNPJ:  doc.EmitenteCNPJ,
		EmitenteName:  doc.EmitenteName,
		IssueDate:     doc.IssueDate,
		TotalValue:    doc.TotalValue,
		CienciaDue:    deadlines.CienciaDue,
		ConclusiveDue: deadlines.ConclusiveDue,
	}
}

// parseConclusiveManifestation reads a conclusive type by name or tpEvento
// code. Ciência is refused: it has its own bulk entry point.
func parseConclusiveManifestation(raw string) (nfe.ManifestationType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(nfe.ManifestationConfirmacao), nfe.TpEventoConfirmacao:
		return nfe.ManifestationConfirmacao, nil
	case string(nfe.ManifestationDesconhecimento), nfe.TpEventoDesconhecimento:
		return nfe.ManifestationDesconhecimento, nil
	case string(nfe.ManifestationNaoRealizada), "nao-realizada", nfe.TpEventoNaoRealizada:
		return nfe.ManifestationNaoRealizada, nil
	case string(nfe.ManifestationCiencia), nfe.TpEventoCiencia:
		return "", errors.New("a ciência da operação é enviada pela ciência em lote")
	default:
		return "", fmt.Errorf("tipo de manifestação inválido %q: use confirmacao, desconhecimento ou nao_realizada", raw)
	}
}

// manifestacaoAfter is the manifestação state a registered event of tipo
// leads to.
func manifestacaoAfter(tipo nfe.ManifestationType) nfe.Manifestacao {
	switch tipo {
	case nfe.ManifestationConfirmacao:
		return nfe.ManifestacaoConfirmada
	case nfe.ManifestationDesconhecimento:
		return nfe.ManifestacaoDesconhecida
	case nfe.ManifestationNaoRealizada:
		return nfe.ManifestacaoNaoRealizada
	default:
		return nfe.ManifestacaoCiencia
	}
}

func manifestationLabel(tipo nfe.ManifestationType) string {
	switch tipo {
	case nfe.ManifestationConfirmacao:
		return "Confirmação da Operação"
	case nfe.ManifestationDesconhecimento:
		return "Desconhecimento da Operação"
	case nfe.ManifestationNaoRealizada:
		return "Operação não Realizada"
	default:
		return "Ciência da Operação"
	}
}

func (s *NFeService) evento(comp *nfse.Company, chave string, tipo nfe.ManifestationType, xJust string) sefaz.Evento {
	return sefaz.Evento{
		ChaveAcesso: chave,
		CNPJ:        comp.CNPJ,
		TpEvento:    tipo.TpEvento(),
		DescEvento:  tipo.DescEvento(),
		NSeqEvento:  1,
		DhEvento:    s.now(),
		XJust:       xJust,
	}
}

// eventSender signs and sends manifestação lotes for one company and
// records every answer.
type eventSender struct {
	service *NFeService
	company *nfse.Company
	client  sefazClient
	signer  *sefaz.Signer
}

// newSender loads the company certificate once, asking for its password
// with purpose.
func (s *NFeService) newSender(ctx context.Context, comp *nfse.Company, purpose string) (*eventSender, error) {
	loaded, err := s.Certificates.LoadForCompany(ctx, comp, purpose)
	if err != nil {
		return nil, err
	}
	signer, err := sefaz.NewSigner(loaded.TLS)
	if err != nil {
		return nil, fmt.Errorf("preparar assinatura: %w", err)
	}
	client, err := newSEFAZClient(sefaz.ClientConfig{
		Environment: comp.Environment,
		Certificate: &loaded.TLS,
		Log:         s.Log,
	})
	if err != nil {
		return nil, fmt.Errorf("configurar cliente SEFAZ: %w", err)
	}
	return &eventSender{service: s, company: comp, client: client, signer: signer}, nil
}

// send sends one lote and records it. The error is set when SEFAZ gave no
// answer for the lote, or its answer could not be recorded; the outcomes are
// returned either way.
func (e *eventSender) send(ctx context.Context, eventos []sefaz.Evento) ([]NFeEventOutcome, error) {
	idLote := sefaz.NewIDLote(e.service.now())
	lote, sendErr := e.client.EnviarEventos(ctx, e.signer, idLote, eventos)
	if sendErr != nil {
		records := make([]nfe.ManifestationRecord, 0, len(eventos))
		for _, ev := range eventos {
			record := e.record(idLote, ev)
			record.Status = nfe.ManifestationStatusErro
			record.XMotivo = sendErr.Error()
			records = append(records, record)
		}
		if err := e.service.NFeRepo.RecordManifestations(ctx, records); err != nil {
			e.service.Log.WarnContext(ctx, "Falha ao registrar lote de manifestação não enviado", slog.Any("err", err))
		}
		outcomes := notSentOutcomes(eventos)
		for i := range outcomes {
			outcomes[i].XMotivo = sendErr.Error()
		}
		return outcomes, sendErr
	}

	records := make([]nfe.ManifestationRecord, 0, len(eventos))
	outcomes := make([]NFeEventOutcome, 0, len(eventos))
	for i, ev := range eventos {
		// EnviarEventos answers every evento in order; a missing answer is
		// recorded as a rejection without cStat.
		var result sefaz.EventoResult
		if i < len(lote.Eventos) {
			result = lote.Eventos[i]
		}
		record := e.record(idLote, ev)
		record.Status = manifestationStatus(result.CStat)
		if result.CStat != 0 {
			record.CStat = strconv.Itoa(result.CStat)
		}
		record.XMotivo = result.XMotivo
		record.Protocolo = result.Protocolo
		record.RegisteredAt = result.DhRegEvento
		record.RequestRawHash = e.keepXML(ctx, result.SignedEvento)
		if result.RetEvento != nil {
			record.ResponseRawHash = e.keepXML(ctx, result.RetEvento)
			if record.Status != nfe.ManifestationStatusRejeitada {
				record.ProcEventoRawHash = e.keepXML(ctx, sefaz.ProcEventoNFe(result.SignedEvento, result.RetEvento))
			}
		}
		records = append(records, record)

		outcomes = append(outcomes, NFeEventOutcome{
			ChaveAcesso:  record.ChaveAcesso,
			TpEvento:     record.TpEvento,
			Status:       record.Status,
			CStat:        record.CStat,
			XMotivo:      record.XMotivo,
			Protocolo:    record.Protocolo,
			RegisteredAt: record.RegisteredAt,
		})
	}
	if err := e.service.NFeRepo.RecordManifestations(ctx, records); err != nil {
		return outcomes, fmt.Errorf("gravar resultado do lote %s: %w", idLote, err)
	}
	return outcomes, nil
}

func (e *eventSender) record(idLote string, ev sefaz.Evento) nfe.ManifestationRecord {
	eventAt := ev.DhEvento
	return nfe.ManifestationRecord{
		CompanyID:     e.company.ID,
		CompanyCNPJ:   e.company.CNPJ,
		IDLote:        idLote,
		ChaveAcesso:   ev.ChaveAcesso,
		TpEvento:      ev.TpEvento,
		NSeqEvento:    ev.NSeqEvento,
		EventAt:       &eventAt,
		Description:   ev.DescEvento,
		Justificativa: ev.XJust,
	}
}

// keepXML stores data in the blob store and returns its hash, or "" when it
// could not be stored: the answer is still recorded without the blob.
func (e *eventSender) keepXML(ctx context.Context, data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if err := e.service.XMLStore.Store(hash, data); err != nil {
		e.service.Log.WarnContext(ctx, "Falha ao salvar XML de manifestação", slog.Any("err", err))
		return ""
	}
	return hash
}

// manifestationStatus maps an evento cStat to the stored status, which is
// also the outcome reported for it.
func manifestationStatus(cStat int) string {
	switch {
	case sefaz.IsRegistered(cStat):
		return nfe.ManifestationStatusRegistrada
	case sefaz.IsAlreadyDone(cStat):
		return nfe.ManifestationStatusJaRegistrada
	default:
		return nfe.ManifestationStatusRejeitada
	}
}

func notSentOutcomes(eventos []sefaz.Evento) []NFeEventOutcome {
	outcomes := make([]NFeEventOutcome, 0, len(eventos))
	for _, ev := range eventos {
		outcomes = append(outcomes, NFeEventOutcome{
			ChaveAcesso: ev.ChaveAcesso,
			TpEvento:    ev.TpEvento,
			Status:      NFeOutcomeNaoEnviada,
		})
	}
	return outcomes
}
