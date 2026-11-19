package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

// RecoverNSUReserve is how many requests of the hour a recovery without a
// limit leaves for the next pull.
const RecoverNSUReserve = 2

// ErrNoRequestsLeft means the hour's budget has nothing left for a recovery.
var ErrNoRequestsLeft = errors.New("sem consultas sobrando nesta hora para recuperar NSUs")

// Why a recovery stopped before asking for every missing NSU.
const (
	RecoverStopLimit           = "limit"            // it spent what it was allowed to
	RecoverStopRateBudget      = "rate_budget"      // the hour's budget ran out
	RecoverStopConsumoIndevido = "consumo_indevido" // SEFAZ answered 656
	RecoverStopCanceled        = "canceled"         // the caller cancelled after some NSUs were asked
)

// RecoverNSUInput names the company and source whose gaps to recover. Limit
// caps the requests of this run; 0 spends what the hour has left minus
// RecoverNSUReserve.
type RecoverNSUInput struct {
	CNPJ   string
	Source syncstate.SyncSource
	Limit  int
}

// RecoverNSUResult counts what one recovery did.
type RecoverNSUResult struct {
	Source      syncstate.SyncSource
	CompanyName string
	CNPJ        string
	Asked       int   // consNSU requests answered
	Recovered   int   // NSUs whose document was stored
	Empty       int   // NSUs with nothing to store: never generated (137), rejected alone (589, 632) or skipped by policy
	Unreadable  int   // NSUs whose document could not be read; the XML is in the log
	Expired     int64 // NSUs closed unasked because they are past the window
	Missing     int64 // NSUs still missing after the run
	// StopReason is set when the run stopped with NSUs left to ask for:
	// RecoverStopLimit, RecoverStopRateBudget, RecoverStopConsumoIndevido or
	// RecoverStopCanceled.
	StopReason       string
	NextAllowedAt    *time.Time
	RequestsLastHour int
	RequestBudget    int
}

// nsuFetcher is a Source that can ask for a single NSU (consNSU).
type nsuFetcher interface {
	Source
	FetchNSU(ctx context.Context, company *company.Company, nsu int64) (sefaz.DistResult, []Item, error)
}

// RecoverNSU asks SEFAZ for each NSU of the company's open gaps in the NF-e
// or CT-e distribution, oldest first, and stores what comes back as the
// distribution does, without moving the cursor. Each request spends one of
// the source's 20 per hour, recorded before it is sent. As in FetchByChave, a
// consumo_indevido or rate_budget block refuses the run before the password
// prompt and the wait after caught_up does not. Gaps whose NSUs date from
// longer ago than NSUGapWindow are closed without asking. A gap keeps the NSU it reached, so
// a run cut short by a cancel, the limit or a 656 continues where it stopped.
func (m *Manager) RecoverNSU(ctx context.Context, input RecoverNSUInput) (RecoverNSUResult, error) {
	cleanedCNPJ, err := cnpj.Normalize(input.CNPJ)
	if err != nil {
		return RecoverNSUResult{}, err
	}
	source := input.Source
	if source != syncstate.SyncSourceNFe && source != syncstate.SyncSourceCTe {
		return RecoverNSUResult{}, fmt.Errorf("a recuperação de NSU só existe para NF-e e CT-e, não para %q", source)
	}
	if input.Limit < 0 {
		return RecoverNSUResult{}, errors.New("o limite de consultas não pode ser negativo")
	}

	comp, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return RecoverNSUResult{}, err
	}
	if err := m.checkSource(comp, source); err != nil {
		return RecoverNSUResult{}, err
	}
	release, err := m.ReserveSource(ctx, comp.ID, source)
	if err != nil {
		return RecoverNSUResult{}, err
	}
	defer release()
	if err := m.BlockedFor(ctx, comp, source, recoveryBlocks...); err != nil {
		return RecoverNSUResult{}, err
	}

	result := RecoverNSUResult{Source: source, CompanyName: comp.Name, CNPJ: comp.CNPJ}
	gaps, err := m.openGapsInWindow(ctx, comp, source, &result)
	if err != nil {
		return RecoverNSUResult{}, err
	}
	if len(gaps) > 0 {
		err := m.recoverGaps(ctx, comp, source, input.Limit, gaps, &result)
		switch {
		case errors.Is(err, context.Canceled) && result.Asked > 0:
			// What was asked is stored and spent: report it instead of only
			// the cancel.
			result.StopReason = RecoverStopCanceled
			ctx = context.WithoutCancel(ctx)
		case err != nil:
			return RecoverNSUResult{}, err
		}
	}

	if err := m.fillRecoverTotals(ctx, comp, source, &result); err != nil {
		return RecoverNSUResult{}, err
	}
	return result, nil
}

// openGapsInWindow returns the source's open gaps, closing as expired the
// ones whose NSUs date from before the distribution window.
func (m *Manager) openGapsInWindow(ctx context.Context, comp *company.Company, source syncstate.SyncSource, result *RecoverNSUResult) ([]NSUGap, error) {
	expired, err := m.SyncRepo.ExpireNSUGaps(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return nil, fmt.Errorf("encerrar NSUs fora da janela: %w", err)
	}
	result.Expired = expired
	gaps, err := m.SyncRepo.OpenNSUGaps(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return nil, fmt.Errorf("carregar NSUs faltantes: %w", err)
	}
	return gaps, nil
}

// recoveryBlocks are the blocks that hold back consNSU. The hour of wait
// after the queue caught up only holds back distNSU (decision of 17/10/2026,
// as for FetchByChave).
var recoveryBlocks = []syncstate.SyncStopReason{syncstate.SyncStopReasonConsumoIndevido, syncstate.SyncStopReasonRateBudget}

// RecoveryLimits is what a recovery started now could do.
type RecoveryLimits struct {
	Missing int64 // NSUs a recovery can still ask for (MissingNSUs)
	// Allowance is how many NSUs a run without a limit would ask for now:
	// the missing ones, up to the hour's budget left after RecoverNSUReserve.
	Allowance int
	// ReadyAt is set while NSUs are missing and a run cannot start: the end
	// of a block in recoveryBlocks, or, with the budget spent, when the
	// oldest request of the hour leaves the window. Ask again then.
	ReadyAt *time.Time
}

// RecoveryLimits reports what a recovery of the company's source could do
// now. It is the rule RecoverNSU applies, for the status to show.
func (m *Manager) RecoveryLimits(ctx context.Context, comp *company.Company, source syncstate.SyncSource) (RecoveryLimits, error) {
	missing, err := m.SyncRepo.MissingNSUs(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return RecoveryLimits{}, fmt.Errorf("contar NSUs faltantes: %w", err)
	}
	limits := RecoveryLimits{Missing: missing}
	if missing == 0 {
		return limits, nil
	}
	var blocked *BlockedError
	if err := m.BlockedFor(ctx, comp, source, recoveryBlocks...); errors.As(err, &blocked) {
		limits.ReadyAt = &blocked.Until
		return limits, nil
	} else if err != nil {
		return RecoveryLimits{}, err
	}
	now := time.Now().UTC()
	count, oldest, err := m.SyncRepo.RequestsSince(ctx, comp.ID, source, comp.Environment, now.Add(-time.Hour))
	if err != nil {
		return RecoveryLimits{}, fmt.Errorf("contar consultas da última hora: %w", err)
	}
	left := requestsPerHour(source) - count - RecoverNSUReserve
	limits.Allowance = int(min(missing, int64(max(left, 0))))
	if limits.Allowance == 0 && oldest != nil {
		readyAt := oldest.Add(time.Hour)
		limits.ReadyAt = &readyAt
	}
	return limits, nil
}

// recoverAllowance is how many requests this run may spend: up to limit
// when set, reserve included, or else what RecoveryLimits allows.
func (m *Manager) recoverAllowance(ctx context.Context, comp *company.Company, source syncstate.SyncSource, limit int) (int, error) {
	if limit > 0 {
		sourceLimits, err := m.SourceLimits(ctx, comp, source)
		if err != nil {
			return 0, err
		}
		if allowed := min(limit, sourceLimits.RequestBudget-sourceLimits.RequestsLastHour); allowed > 0 {
			return allowed, nil
		}
		return 0, ErrNoRequestsLeft
	}
	limits, err := m.RecoveryLimits(ctx, comp, source)
	if err != nil {
		return 0, err
	}
	if limits.Allowance > 0 {
		return limits.Allowance, nil
	}
	return 0, fmt.Errorf("%w (%d ficam para a sincronização)", ErrNoRequestsLeft, RecoverNSUReserve)
}

// recoverGaps walks the gaps NSU by NSU until they end or the run must stop.
func (m *Manager) recoverGaps(ctx context.Context, comp *company.Company, source syncstate.SyncSource, limit int, gaps []NSUGap, result *RecoverNSUResult) error {
	allowance, err := m.recoverAllowance(ctx, comp, source, limit)
	if err != nil {
		return err
	}
	loaded, err := m.Certificates.LoadForCompany(ctx, comp, "Recuperação de NSUs de "+sourceLabel(source))
	if err != nil {
		return err
	}
	src, err := m.newSource(comp, source, loaded.TLS)
	if err != nil {
		return err
	}
	fetcher, ok := src.(nsuFetcher)
	if !ok {
		return fmt.Errorf("a origem %s não consulta NSU avulso", source)
	}
	state, err := m.SyncRepo.SourceState(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return fmt.Errorf("carregar estado da origem: %w", err)
	}

	for _, gap := range gaps {
		for nsu := gap.Next; nsu <= gap.Last; nsu++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			if result.Asked >= allowance {
				result.StopReason = RecoverStopLimit
				return nil
			}
			if result.Asked > 0 {
				if err := waitRequestDelay(ctx, src.Policy().RequestDelay); err != nil {
					return err
				}
			}
			spent, err := m.SpendRequest(ctx, comp, source)
			if err != nil {
				return err
			}
			if !spent {
				result.StopReason = RecoverStopRateBudget
				return nil
			}
			stop, err := m.recoverOne(ctx, comp, fetcher, state, gap, nsu, result)
			if err != nil || stop {
				return err
			}
		}
	}
	return nil
}

// recoverOne asks for nsu and stores the answer. It reports stop when SEFAZ
// blocked the CNPJ (656). Once SEFAZ has answered, the request is spent, so
// storing the answer and moving the gap ignore a cancel: asking again would
// spend another request for the same NSU.
func (m *Manager) recoverOne(ctx context.Context, comp *company.Company, fetcher nsuFetcher, state SourceState, gap NSUGap, nsu int64, result *RecoverNSUResult) (bool, error) {
	source := fetcher.Kind()
	resp, items, err := fetcher.FetchNSU(ctx, comp, nsu)
	if err != nil {
		return false, fmt.Errorf("consultar NSU %d: %w", nsu, err)
	}
	result.Asked++
	ctx = context.WithoutCancel(ctx)

	switch resp.CStat {
	case sefaz.CStatConsumoIndevido:
		until := time.Now().UTC().Add(distWaitAfterStop)
		blockCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := m.SyncRepo.SetBlockedUntil(blockCtx, comp.ID, source, comp.Environment, until, syncstate.SyncStopReasonConsumoIndevido)
		cancel()
		if err != nil {
			return false, fmt.Errorf("bloquear consultas de %s: %w", sourceLabel(source), err)
		}
		result.StopReason = RecoverStopConsumoIndevido
		return true, nil
	case sefaz.CStatDocumentoLocalizado:
		return false, m.storeRecovered(ctx, comp, fetcher, state, gap, nsu, items, result)
	case sefaz.CStatNenhumDocumento, sefaz.CStatNSUAcimaDoMaximo, sefaz.CStatForaDePrazo:
		// 137: the Ambiente Nacional did not generate the NSU, and the NT
		// says to disregard it. 589 and 632 reject this NSU alone: asking
		// again would get them again and keep every later NSU waiting. Other
		// rejections (certificate, CNPJ, service down) stop the run below.
		if resp.CStat != sefaz.CStatNenhumDocumento {
			m.Log.WarnContext(ctx, "SEFAZ recusou o NSU pedido por consNSU",
				slog.Int64("nsu", nsu), slog.Int("cstat", resp.CStat), slog.String("xmotivo", resp.XMotivo))
		}
		result.Empty++
		return false, m.SyncRepo.AdvanceNSUGap(ctx, gap, nsu)
	default:
		return false, fmt.Errorf("consultar NSU %d: %w", nsu, &sefaz.RejectionError{CStat: resp.CStat, XMotivo: resp.XMotivo})
	}
}

// storeRecovered stores the items of a consNSU answer and moves the gap past
// nsu together with the last one. A document that cannot be read still moves
// the gap: asking again would only spend another request. An item the source
// skipped by policy counts as empty, as nothing was stored.
func (m *Manager) storeRecovered(ctx context.Context, comp *company.Company, fetcher nsuFetcher, state SourceState, gap NSUGap, nsu int64, items []Item, result *RecoverNSUResult) error {
	if len(items) == 0 {
		result.Empty++
		return m.SyncRepo.AdvanceNSUGap(ctx, gap, nsu)
	}
	var outcome ItemOutcome
	for i, item := range items {
		commit := m.SyncRepo.Apply
		if i == len(items)-1 {
			commit = m.SyncRepo.ApplyAndAdvanceNSUGap(gap, nsu)
		}
		var err error
		outcome, err = fetcher.ProcessItem(ctx, comp, state, item, commit)
		var parseErr *ProcessingError
		if errors.As(err, &parseErr) {
			m.Log.WarnContext(ctx, "Documento recuperado por consNSU não pôde ser lido", slog.Any("error", parseErr))
			result.Unreadable++
			return m.SyncRepo.AdvanceNSUGap(ctx, gap, nsu)
		}
		if err != nil {
			return fmt.Errorf("gravar documento do NSU %d: %w", nsu, err)
		}
	}
	if outcome.SkippedByPolicy {
		result.Empty++
	} else {
		result.Recovered++
	}
	return nil
}

// fillRecoverTotals sets what is still missing and the hour's budget.
func (m *Manager) fillRecoverTotals(ctx context.Context, comp *company.Company, source syncstate.SyncSource, result *RecoverNSUResult) error {
	missing, err := m.SyncRepo.MissingNSUs(ctx, comp.ID, source, comp.Environment)
	if err != nil {
		return fmt.Errorf("contar NSUs faltantes: %w", err)
	}
	result.Missing = missing
	if missing == 0 {
		result.StopReason = ""
	}
	limits, err := m.SourceLimits(ctx, comp, source)
	if err != nil {
		return err
	}
	result.NextAllowedAt = limits.NextAllowedAt
	result.RequestsLastHour = limits.RequestsLastHour
	result.RequestBudget = limits.RequestBudget
	return nil
}

// DiscardNSUInput names the company and source whose open gaps to discard.
// Without Confirm nothing changes.
type DiscardNSUInput struct {
	CNPJ    string
	Source  syncstate.SyncSource
	Confirm bool
}

// DiscardNSU closes every open gap of the company's source, so recovery
// never asks for those NSUs; their documents can still come in by XML
// import. It returns how many NSUs the gaps held: with Confirm, the ones
// discarded; without it, the ones that would be. It holds the source like a
// pull, so it never races a recovery, and asks SEFAZ nothing.
func (m *Manager) DiscardNSU(ctx context.Context, input DiscardNSUInput) (int64, error) {
	cleanedCNPJ, err := cnpj.Normalize(input.CNPJ)
	if err != nil {
		return 0, err
	}
	source := input.Source
	if source != syncstate.SyncSourceNFe && source != syncstate.SyncSourceCTe {
		return 0, fmt.Errorf("a recuperação de NSU só existe para NF-e e CT-e, não para %q", source)
	}
	comp, err := m.CompanyProvider.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		return 0, err
	}
	release, err := m.ReserveSource(ctx, comp.ID, source)
	if err != nil {
		return 0, err
	}
	defer release()
	if !input.Confirm {
		return m.SyncRepo.OpenNSUs(ctx, comp.ID, source, comp.Environment)
	}
	return m.SyncRepo.DiscardNSUGaps(ctx, comp.ID, source, comp.Environment)
}
