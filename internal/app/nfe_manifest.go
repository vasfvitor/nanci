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

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// NFeOutcomeNaoEnviada is the outcome of an event whose lote got no SEFAZ
// answer, or was never sent because an earlier lote failed. The other
// outcomes are the stored nfe.ManifestacaoStatus* values. The two sets
// differ here on purpose: a lote without answer is stored as
// nfe.ManifestacaoStatusErro, a lote never sent is not stored at all, and
// both are reported to the user as nao_enviada.
const NFeOutcomeNaoEnviada = "nao_enviada"

// NFeCienciaInput selects the NF-e for Ciência da Operação: either the
// listed chaves or, with AllResumos, every resumo still waiting for it.
type NFeCienciaInput struct {
	CNPJ         string
	ChavesAcesso []string
	// AllResumos selects every resumo addressed to the company, authorized
	// and without manifestação.
	AllResumos bool
}

// NFeSkipped is a requested chave that will not be sent, and why.
type NFeSkipped struct {
	ChaveAcesso string
	Reason      string
}

// NFeCienciaPlan is what RegisterCiencia would send.
type NFeCienciaPlan struct {
	Eligible []NFeDocument
	Skipped  []NFeSkipped
}

// NFeEventOutcome is the result of one manifestação event.
type NFeEventOutcome struct {
	ChaveAcesso  string
	TpEvento     string
	Status       string // nfe.ManifestacaoStatusRegistrada, JaRegistrada, Rejeitada, or NFeOutcomeNaoEnviada
	CStat        string // empty when SEFAZ did not answer
	XMotivo      string
	Protocolo    string
	RegisteredAt *time.Time
}

// NFeManifestacaoSummary is the result of RegisterCiencia: one outcome per
// eligible chave, in the order sent.
type NFeManifestacaoSummary struct {
	Outcomes []NFeEventOutcome
	Skipped  []NFeSkipped
	// Interrupted is the error that stopped the sending, such as a transport
	// failure; the lotes after it were not sent. Empty when every lote was
	// sent.
	Interrupted string
}

// NFeManifestacaoInput is one conclusive manifestação.
type NFeManifestacaoInput struct {
	CNPJ        string
	ChaveAcesso string
	// Tipo is confirmacao, desconhecimento or nao_realizada (nao-realizada
	// also works), or the tpEvento code 210200, 210220 or 210240.
	Tipo string
	// Justificativa is required (15 to 255 characters) for nao_realizada and
	// must be empty otherwise.
	Justificativa string
}

// NFeManifestacaoPlan is what RegisterManifestacao would send.
type NFeManifestacaoPlan struct {
	Tipo nfe.TipoManifestacao
	// Justificativa is the cleaned text sent as xJust; empty unless Tipo is
	// nao_realizada.
	Justificativa string
	Document      nfe.CompanyDocument
	CienciaDue    time.Time
	// ConclusiveDue is zero when the NF-e has neither an authorization nor
	// an issue date.
	ConclusiveDue time.Time
	// DaysLeft is how many calendar days are left until ConclusiveDue: 0 on
	// the due day, negative once it passed.
	DaysLeft int
	// TacitlyConfirmed is true when ConclusiveDue passed without a
	// conclusive manifestação: SEFAZ should reject the event (cStat 596).
	TacitlyConfirmed bool
	// BlockReason says why the manifestação cannot be sent, or "" when it
	// can.
	BlockReason string
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
func (s *NFeService) RegisterCiencia(ctx context.Context, in NFeCienciaInput) (NFeManifestacaoSummary, error) {
	comp, plan, err := s.planCiencia(ctx, in)
	if err != nil {
		return NFeManifestacaoSummary{}, err
	}
	if len(plan.Eligible) == 0 {
		return NFeManifestacaoSummary{}, fmt.Errorf("nenhuma NF-e elegível para ciência (%d ignoradas)", len(plan.Skipped))
	}

	sender, err := s.newSender(ctx, comp, fmt.Sprintf("Assinatura: Ciência da Operação (%d notas)", len(plan.Eligible)))
	if err != nil {
		return NFeManifestacaoSummary{}, err
	}

	summary := NFeManifestacaoSummary{Skipped: plan.Skipped}
	for lote := range slices.Chunk(plan.Eligible, sefaz.MaxEventosPorLote) {
		eventos := make([]sefaz.Evento, 0, len(lote))
		for _, c := range lote {
			eventos = append(eventos, s.evento(comp, string(c.ChaveAcesso), nfe.TipoManifestacaoCiencia, ""))
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
		summary.Outcomes = append(summary.Outcomes, outcomes...)
	}
	return summary, nil
}

// PlanManifestacao checks one conclusive manifestação and returns what
// RegisterManifestacao would send. It makes no network call and asks for no
// password. Invalid input is an error; a document that cannot receive the
// manifestação is reported in BlockReason.
func (s *NFeService) PlanManifestacao(ctx context.Context, in NFeManifestacaoInput) (NFeManifestacaoPlan, error) {
	_, plan, err := s.planManifestacao(ctx, in)
	return plan, err
}

// RegisterManifestacao sends one conclusive manifestação (confirmação,
// desconhecimento or operação não realizada). Tipo, justificativa, the
// company's role, the NF-e situação and the absence of an earlier conclusive
// manifestação are checked before the password prompt. The deadline is not
// checked: SEFAZ decides (cStat 596). An event SEFAZ answered is returned
// without error, whatever its outcome. A request without answer, or an
// answer that could not be recorded, is an error; the outcome is returned
// with it.
func (s *NFeService) RegisterManifestacao(ctx context.Context, in NFeManifestacaoInput) (NFeEventOutcome, error) {
	comp, plan, err := s.planManifestacao(ctx, in)
	if err != nil {
		return NFeEventOutcome{}, err
	}
	if plan.BlockReason != "" {
		return NFeEventOutcome{}, errors.New(plan.BlockReason)
	}

	sender, err := s.newSender(ctx, comp, "Assinatura: "+plan.Tipo.Label())
	if err != nil {
		return NFeEventOutcome{}, err
	}
	evento := s.evento(comp, string(plan.Document.ChaveAcesso), plan.Tipo, plan.Justificativa)
	outcomes, err := sender.send(ctx, []sefaz.Evento{evento})
	if err != nil {
		return outcomes[0], fmt.Errorf("enviar manifestação: %w", err)
	}
	return outcomes[0], nil
}

// planManifestacao validates the input, resolves the company and the NF-e
// and says whether the manifestação can be sent.
func (s *NFeService) planManifestacao(ctx context.Context, in NFeManifestacaoInput) (*nfse.Company, NFeManifestacaoPlan, error) {
	tipo, err := parseConclusiveManifestacao(in.Tipo)
	if err != nil {
		return nil, NFeManifestacaoPlan{}, err
	}
	xJust, err := nfe.ValidateJustificativa(tipo, in.Justificativa)
	if err != nil {
		return nil, NFeManifestacaoPlan{}, err
	}
	comp, err := lookupCompanyByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return nil, NFeManifestacaoPlan{}, err
	}
	doc, err := s.companyDocument(ctx, comp.ID, in.ChaveAcesso)
	if err != nil {
		return nil, NFeManifestacaoPlan{}, err
	}

	now := s.now()
	deadlines := nfe.ManifestacaoPrazos(doc.Document)
	plan := NFeManifestacaoPlan{
		Tipo:             tipo,
		Justificativa:    xJust,
		Document:         doc,
		CienciaDue:       deadlines.CienciaDue,
		ConclusiveDue:    deadlines.ConclusiveDue,
		DaysLeft:         daysLeft(deadlines.ConclusiveDue, now),
		TacitlyConfirmed: nfe.TacitlyConfirmed(doc, now),
	}
	if reason := manifestacaoBlockReason(doc); reason != "" {
		plan.BlockReason = fmt.Sprintf("NF-e %s: %s", doc.ChaveAcesso, reason)
	} else {
		plan.BlockReason = nfe.ConclusiveBlockReason(doc.Manifestacao)
	}
	return comp, plan, nil
}

// daysLeft counts the calendar days from now until due, in now's location:
// 0 on the due day, negative once it passed.
func daysLeft(due, now time.Time) int {
	due = due.In(now.Location())
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return int(dueDay.Sub(today).Hours() / 24)
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

	now := s.now()
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
			plan.Eligible = append(plan.Eligible, newNFeDocument(doc, now))
		}
	} else {
		if err := s.planChaves(ctx, comp.ID, in.ChavesAcesso, now, &plan); err != nil {
			return nil, NFeCienciaPlan{}, err
		}
	}
	return comp, plan, nil
}

func (s *NFeService) planChaves(ctx context.Context, companyID nfse.CompanyID, rawChaves []string, now time.Time, plan *NFeCienciaPlan) error {
	var chaves []string
	seen := make(map[string]bool, len(rawChaves))
	for _, raw := range rawChaves {
		chave, err := dfe.ParseAccessKey(raw)
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
		if reason := cienciaBlockReason(doc); reason != "" {
			plan.Skipped = append(plan.Skipped, NFeSkipped{ChaveAcesso: chave, Reason: reason})
			continue
		}
		plan.Eligible = append(plan.Eligible, newNFeDocument(doc, now))
	}
	return nil
}

// manifestacaoBlockReason says why the company cannot manifest on doc at
// all, or "" when it can.
func manifestacaoBlockReason(doc nfe.CompanyDocument) string {
	if doc.CompanyRole != nfe.CompanyRoleDestinatario {
		return "a empresa não é a destinatária"
	}
	if doc.Situacao != nfe.SituacaoAutorizada {
		return "NF-e " + string(doc.Situacao)
	}
	return ""
}

// cienciaBlockReason says why doc cannot receive Ciência da Operação, or ""
// when it can.
func cienciaBlockReason(doc nfe.CompanyDocument) string {
	if reason := manifestacaoBlockReason(doc); reason != "" {
		return reason
	}
	if doc.Manifestacao != nfe.ManifestacaoNenhuma {
		return "já manifestada (" + string(doc.Manifestacao) + ")"
	}
	return ""
}

// conclusiveBlockReason says why doc cannot receive a conclusive
// manifestação, or "" when it can.
func conclusiveBlockReason(doc nfe.CompanyDocument) string {
	if reason := manifestacaoBlockReason(doc); reason != "" {
		return reason
	}
	return nfe.ConclusiveBlockReason(doc.Manifestacao)
}

// parseConclusiveManifestacao reads a conclusive type by name or tpEvento
// code. Ciência is refused: it has its own bulk entry point.
func parseConclusiveManifestacao(raw string) (nfe.TipoManifestacao, error) {
	tipo, err := nfe.ParseTipoManifestacao(raw)
	if err != nil {
		return "", fmt.Errorf("tipo de manifestação inválido (use confirmacao, desconhecimento ou nao_realizada): %w", err)
	}
	if tipo == nfe.TipoManifestacaoCiencia {
		return "", errors.New("a ciência da operação é enviada pela ciência em lote")
	}
	return tipo, nil
}

func (s *NFeService) evento(comp *nfse.Company, chave string, tipo nfe.TipoManifestacao, xJust string) sefaz.Evento {
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
	tpAmb   string
}

// newSender loads the company certificate once, asking for its password
// with purpose.
func (s *NFeService) newSender(ctx context.Context, comp *nfse.Company, purpose string) (*eventSender, error) {
	tpAmb, err := sefaz.TpAmb(comp.Environment)
	if err != nil {
		return nil, err
	}
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
	return &eventSender{service: s, company: comp, client: client, signer: signer, tpAmb: tpAmb}, nil
}

// send sends one lote and records it. The error is set when SEFAZ gave no
// answer for the lote, or its answer could not be recorded; the outcomes are
// returned either way.
func (e *eventSender) send(ctx context.Context, eventos []sefaz.Evento) ([]NFeEventOutcome, error) {
	idLote := sefaz.NewIDLote(e.service.now())
	lote, sendErr := e.client.EnviarEventos(ctx, e.signer, idLote, eventos)
	if sendErr != nil {
		records := make([]nfe.ManifestacaoRecord, 0, len(eventos))
		for _, ev := range eventos {
			record := e.record(idLote, ev)
			record.Status = nfe.ManifestacaoStatusErro
			record.XMotivo = sendErr.Error()
			records = append(records, record)
		}
		if err := e.service.NFeRepo.RecordManifestacoes(ctx, records); err != nil {
			e.service.Log.WarnContext(ctx, "Falha ao registrar lote de manifestação não enviado", slog.Any("err", err))
		}
		outcomes := notSentOutcomes(eventos)
		for i := range outcomes {
			outcomes[i].XMotivo = sendErr.Error()
		}
		return outcomes, sendErr
	}

	records := make([]nfe.ManifestacaoRecord, 0, len(eventos))
	outcomes := make([]NFeEventOutcome, 0, len(eventos))
	for i, ev := range eventos {
		// EnviarEventos answers every evento in order; a missing answer is
		// recorded as a rejection without cStat.
		var result sefaz.EventoResult
		if i < len(lote.Eventos) {
			result = lote.Eventos[i]
		}
		record := e.record(idLote, ev)
		record.Status = manifestacaoStatus(result.CStat)
		if result.CStat != 0 {
			record.CStat = strconv.Itoa(result.CStat)
		}
		record.XMotivo = result.XMotivo
		record.Protocolo = result.Protocolo
		record.RegisteredAt = result.DhRegEvento
		record.RequestRawHash = e.keepXML(ctx, result.SignedEvento)
		if result.RetEvento != nil {
			record.ResponseRawHash = e.keepXML(ctx, result.RetEvento)
			if record.Status != nfe.ManifestacaoStatusRejeitada {
				record.ProcEventoRawHash = e.keepXML(ctx, sefaz.ProcEventoNFe(result.SignedEvento, result.RetEvento))
			}
		}
		records = append(records, record)

		outcomes = append(outcomes, NFeEventOutcome{
			ChaveAcesso:  record.ChaveAcesso,
			TpEvento:     record.TpEvento,
			Status:       record.Status,
			CStat:        record.CStat,
			XMotivo:      outcomeMotivo(result.CStat, record.XMotivo),
			Protocolo:    record.Protocolo,
			RegisteredAt: record.RegisteredAt,
		})
	}
	if err := e.service.NFeRepo.RecordManifestacoes(ctx, records); err != nil {
		return outcomes, fmt.Errorf("gravar resultado do lote %s: %w", idLote, err)
	}
	return outcomes, nil
}

func (e *eventSender) record(idLote string, ev sefaz.Evento) nfe.ManifestacaoRecord {
	eventAt := ev.DhEvento
	return nfe.ManifestacaoRecord{
		CompanyID:     e.company.ID,
		CompanyCNPJ:   e.company.CNPJ,
		IDLote:        idLote,
		TpAmb:         e.tpAmb,
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

// outcomeMotivo is the XMotivo reported for an evento. A ciência answered
// with 655 is a rejection whose reason is spelled out: SEFAZ already holds a
// conclusive manifestação that nanci may not have pulled yet.
func outcomeMotivo(cStat int, xMotivo string) string {
	if cStat == sefaz.CStatCienciaAposManifestacao {
		return "NF-e já possui manifestação conclusiva: " + xMotivo
	}
	return xMotivo
}

// manifestacaoStatus maps an evento cStat to the stored status, which is
// also the outcome reported for it. Only 135/136 and 573 store an event;
// everything else, 655 included, is a rejection and leaves the
// manifestação unchanged.
func manifestacaoStatus(cStat int) string {
	switch {
	case sefaz.IsRegistered(cStat):
		return nfe.ManifestacaoStatusRegistrada
	case sefaz.IsAlreadyDone(cStat):
		return nfe.ManifestacaoStatusJaRegistrada
	default:
		return nfe.ManifestacaoStatusRejeitada
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
