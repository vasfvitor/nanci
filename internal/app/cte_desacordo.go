package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// DesacordoPrazoDias is the deadline of the prestação em desacordo, counted
// from the CT-e authorization (Ajuste SINIEF 09/07, cláusula 17ª, § 6º).
// nanci shows it and never enforces it: SEFAZ decides (cStat 787).
const DesacordoPrazoDias = 45

// CTeDesacordoInput is one prestação do serviço em desacordo to plan or send.
type CTeDesacordoInput struct {
	CNPJ        string
	ChaveAcesso string
	// Observacao is the xObs: 15 to 255 characters within Latin-1.
	Observacao string
}

// CTeDesacordoPlan is what RegisterDesacordo would send. Nothing is sent to
// build it and no password is asked.
type CTeDesacordoPlan struct {
	CompanyName string
	CNPJ        string
	TpAmb       string // 1 produção, 2 homologação
	Document    cte.CompanyDocument
	Observacao  string
	NSeqEvento  int
	// Due is the end of the 45 days, zero when the CT-e has no authorization
	// date. DaysLeft counts calendar days to it: 0 on the due day, negative
	// once it passed; it is 0 when Due is zero.
	Due      time.Time
	DaysLeft int
	// BlockReason says why the event cannot be sent, or "" when it can. A
	// passed deadline is not a block.
	BlockReason string
}

// CTeDesacordoOutcome is the answer to one prestação em desacordo.
type CTeDesacordoOutcome struct {
	ChaveAcesso  string
	Status       string // cte.DesacordoStatus* or NFeOutcomeNaoEnviada
	CStat        string // empty when SEFAZ did not answer
	XMotivo      string
	Protocolo    string
	RegisteredAt *time.Time
}

// PlanDesacordo checks the prestação em desacordo and returns what
// RegisterDesacordo would send. Invalid input is an error; a CT-e that
// cannot receive the event is reported in BlockReason.
func (s *CTeService) PlanDesacordo(ctx context.Context, in CTeDesacordoInput) (CTeDesacordoPlan, error) {
	_, plan, err := s.planDesacordo(ctx, in)
	return plan, err
}

// RegisterDesacordo sends the prestação em desacordo to the SEFAZ that
// authorized the CT-e. The observation, the company being the tomador, the
// model, the situação and the absence of a registered desacordo are checked
// before the password prompt; the deadline is not (SEFAZ answers 787). An
// event SEFAZ answered is returned without error, whatever its outcome. A
// request without answer, or an answer that could not be recorded, is an
// error; the outcome is returned with it.
func (s *CTeService) RegisterDesacordo(ctx context.Context, in CTeDesacordoInput) (CTeDesacordoOutcome, error) {
	comp, plan, err := s.planDesacordo(ctx, in)
	if err != nil {
		return CTeDesacordoOutcome{}, err
	}
	if plan.BlockReason != "" {
		return CTeDesacordoOutcome{}, errors.New(plan.BlockReason)
	}

	loaded, err := s.Certificates.LoadForCompany(ctx, comp, "Assinatura: Prestação do Serviço em Desacordo")
	if err != nil {
		return CTeDesacordoOutcome{}, err
	}
	signer, err := sefaz.NewSigner(loaded.TLS)
	if err != nil {
		return CTeDesacordoOutcome{}, fmt.Errorf("preparar assinatura: %w", err)
	}
	client, err := companySEFAZClient(comp, &loaded.TLS, s.Log, s.SyncManager)
	if err != nil {
		return CTeDesacordoOutcome{}, fmt.Errorf("configurar cliente SEFAZ: %w", err)
	}

	evento := sefaz.EventoDesacordo{
		ChaveAcesso: string(plan.Document.ChaveAcesso),
		CNPJ:        comp.CNPJ,
		NSeqEvento:  plan.NSeqEvento,
		DhEvento:    s.now(),
		XObs:        plan.Observacao,
	}
	record := cte.DesacordoRecord{
		CompanyID:   comp.ID,
		ChaveAcesso: evento.ChaveAcesso,
		TpAmb:       plan.TpAmb,
		NSeqEvento:  evento.NSeqEvento,
		Observacao:  evento.XObs,
		EventAt:     evento.DhEvento,
	}

	result, sendErr := client.EnviarEventoDesacordo(ctx, signer, evento)
	if sendErr != nil {
		record.Status = cte.DesacordoStatusErro
		record.XMotivo = sendErr.Error()
		if err := s.CTeRepo.RecordDesacordo(ctx, record); err != nil {
			s.Log.WarnContext(ctx, "Falha ao registrar prestação em desacordo não enviada", slog.Any("err", err))
		}
		outcome := CTeDesacordoOutcome{ChaveAcesso: evento.ChaveAcesso, Status: NFeOutcomeNaoEnviada, XMotivo: sendErr.Error()}
		return outcome, fmt.Errorf("enviar prestação em desacordo: %w", sendErr)
	}

	record.Status = desacordoStatus(result.CStat)
	record.CStat = strconv.Itoa(result.CStat)
	record.XMotivo = result.XMotivo
	record.Protocolo = result.Protocolo
	record.RegisteredAt = result.DhRegEvento
	record.RequestRawHash = s.keepEventXML(ctx, result.SignedEvento)
	record.ResponseRawHash = s.keepEventXML(ctx, result.RetEvento)
	if record.Status == cte.DesacordoStatusRegistrada {
		event, err := s.registeredEvent(ctx, result)
		if err != nil {
			s.Log.WarnContext(ctx, "Prestação em desacordo registrada, mas o evento não foi lido", slog.Any("err", err))
		} else {
			record.Event = &event
		}
	}

	outcome := CTeDesacordoOutcome{
		ChaveAcesso:  record.ChaveAcesso,
		Status:       record.Status,
		CStat:        record.CStat,
		XMotivo:      record.XMotivo,
		Protocolo:    record.Protocolo,
		RegisteredAt: record.RegisteredAt,
	}
	if err := s.CTeRepo.RecordDesacordo(ctx, record); err != nil {
		return outcome, fmt.Errorf("gravar resultado da prestação em desacordo: %w", err)
	}
	return outcome, nil
}

// planDesacordo validates the input, resolves the company and the CT-e and
// says whether the event can be sent.
func (s *CTeService) planDesacordo(ctx context.Context, in CTeDesacordoInput) (*company.Company, CTeDesacordoPlan, error) {
	if err := sefaz.CheckXObsDesacordo(in.Observacao); err != nil {
		return nil, CTeDesacordoPlan{}, fmt.Errorf("%w: %w", ErrInvalidObservacao, err)
	}
	comp, err := company.LookupByCNPJ(ctx, s.CompanyStore, in.CNPJ)
	if err != nil {
		return nil, CTeDesacordoPlan{}, err
	}
	doc, err := s.companyDocument(ctx, comp, in.ChaveAcesso)
	if err != nil {
		return nil, CTeDesacordoPlan{}, err
	}
	tpAmb, err := environmentTpAmb(comp)
	if err != nil {
		return nil, CTeDesacordoPlan{}, err
	}
	events, err := s.CTeRepo.ListEventsByChave(ctx, string(doc.ChaveAcesso))
	if err != nil {
		return nil, CTeDesacordoPlan{}, err
	}

	plan := CTeDesacordoPlan{
		CompanyName: comp.Name,
		CNPJ:        comp.CNPJ,
		TpAmb:       tpAmb,
		Document:    doc,
		Observacao:  in.Observacao,
	}
	if doc.AuthorizedAt != nil {
		plan.Due = doc.AuthorizedAt.AddDate(0, 0, DesacordoPrazoDias)
		plan.DaysLeft = daysLeft(plan.Due, s.now())
	}
	registered, protocolo := desacordoState(events)
	plan.NSeqEvento = registered + 1
	plan.BlockReason = desacordoBlockReason(comp, doc, protocolo)
	return comp, plan, nil
}

// desacordoState counts the registered desacordos of a CT-e and returns
// the protocol of the one still in force, or "" when every one was
// cancelled (610111) or there is none.
func desacordoState(events []cte.Event) (registered int, protocolo string) {
	cancelled := 0
	for _, e := range events {
		if !e.Registered {
			continue
		}
		switch e.Type {
		case cte.EventTypePrestacaoDesacordo:
			registered++
			protocolo = e.Protocolo
		case cte.EventTypeCancelamentoDesacordo:
			cancelled++
		}
	}
	if registered <= cancelled {
		protocolo = ""
	}
	return registered, protocolo
}

// desacordoBlockReason says why the company cannot send the prestação em
// desacordo on doc, or "" when it can (MOC CT-e 4.00, 6.5).
func desacordoBlockReason(comp *company.Company, doc cte.CompanyDocument, protocoloEmVigor string) string {
	chave := doc.ChaveAcesso
	switch {
	case doc.Modelo != "57" && doc.Modelo != "67":
		return fmt.Sprintf("CT-e %s: a prestação em desacordo vale só para o CT-e (modelo 57) e o CT-e OS (67), não para o modelo %s", chave, doc.Modelo)
	case cnpj.Clean(doc.Tomador.CNPJ) != cnpj.Clean(comp.CNPJ):
		return fmt.Sprintf("CT-e %s: a empresa não é a tomadora (o tomador é %s)", chave, tomadorLabel(doc.Tomador))
	case doc.Situacao != cte.SituacaoAutorizada:
		return fmt.Sprintf("CT-e %s %s", chave, doc.Situacao)
	case protocoloEmVigor != "":
		return fmt.Sprintf("CT-e %s já tem prestação em desacordo registrada (protocolo %s)", chave, protocoloEmVigor)
	}
	return ""
}

func tomadorLabel(p cte.Party) string {
	if p.CNPJ == "" {
		return "desconhecido"
	}
	if len(p.CNPJ) == 14 {
		return cnpj.Format(p.CNPJ)
	}
	return p.CNPJ
}

// desacordoStatus maps the cStat of the answer to the stored status. Only
// 134, 135 and 136 register the event; 631 means an earlier send did.
func desacordoStatus(cStat int) string {
	switch {
	case sefaz.IsRegisteredCTe(cStat):
		return cte.DesacordoStatusRegistrada
	case sefaz.IsAlreadyDoneCTe(cStat):
		return cte.DesacordoStatusJaRegistrada
	default:
		return cte.DesacordoStatusRejeitada
	}
}

// registeredEvent stores the procEventoCTe of a registered desacordo and
// reads it back with the distribution's parser, so the event is stored like
// one that arrives by the distribution.
func (s *CTeService) registeredEvent(ctx context.Context, result sefaz.EventoCTeResult) (cte.Event, error) {
	proc := sefaz.ProcEventoCTe(result.SignedEvento, result.RetEvento)
	event, err := cte.ParseProcEventoCTe(proc)
	if err != nil {
		return cte.Event{}, err
	}
	event.RawHash = s.keepEventXML(ctx, proc)
	if event.RawHash == "" {
		return cte.Event{}, errors.New("o XML do evento não foi salvo")
	}
	return event, nil
}

// keepEventXML stores data in the blob store and returns its hash, or ""
// when it could not be stored: the answer is still recorded without it.
func (s *CTeService) keepEventXML(ctx context.Context, data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if err := s.XMLStore.Store(hash, data); err != nil {
		s.Log.WarnContext(ctx, "Falha ao salvar XML da prestação em desacordo", slog.Any("err", err))
		return ""
	}
	return hash
}
