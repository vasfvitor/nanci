package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// maxItemAttempts is how many runs in a row may fail to decode or parse the
// same NSU before the loop skips it as unsupported.
const maxItemAttempts = 3

// SyncService walks one Source by NSU and records the run.
type SyncService struct {
	store  *Store
	source Source
	log    *slog.Logger
}

// NewSyncService creates a new SyncService for the source.
func NewSyncService(syncRepo *Store, source Source, log *slog.Logger) *SyncService {
	return &SyncService{
		store:  syncRepo,
		source: source,
		log:    log,
	}
}

// Sync starts the synchronization process for a specific company.
func (s *SyncService) Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error {
	if mode == "" {
		mode = nfse.SyncModeNormal
	}
	kind := s.source.Kind()

	state, err := s.store.GetOrCreateState(ctx, nfse.GetOrCreateSyncStateParams{
		CompanyID:        company.ID,
		Source:           kind,
		Environment:      company.Environment,
		ConsultationCNPJ: company.CNPJ,
	})
	if err != nil {
		return fmt.Errorf("failed to load sync state: %w", err)
	}
	sourceState, err := s.store.SourceState(ctx, company.ID, kind)
	if err != nil {
		return fmt.Errorf("failed to load source state: %w", err)
	}
	if err := checkBlocked(kind, sourceState, time.Now()); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "Iniciando processo de sincronização",
		slog.String("cnpj", company.CNPJ),
		slog.String("source", string(kind)),
		slog.String("mode", string(mode)),
		slog.Int64("from_nsu", state.LastProcessedNSU))

	syncRun, err := s.store.StartRun(ctx, nfse.StartRunParams{
		CompanyID:         company.ID,
		Source:            kind,
		CredentialID:      credential.ID,
		Environment:       company.Environment,
		CredentialCNPJ:    credential.OwnerCNPJ,
		ConsultationCNPJ:  company.CNPJ,
		ConsultationBasis: nfse.ConsultationBasis(consultationBasis),
		Mode:              mode,
		FromNSU:           state.LastProcessedNSU,
		ToNSU:             state.LastProcessedNSU,
	})
	if err != nil {
		return fmt.Errorf("failed to create sync run: %w", err)
	}

	runState := syncRuntimeState{
		runID:            syncRun.ID,
		lastProcessedNSU: state.LastProcessedNSU,
		lastFoundNSU:     state.LastFoundNSU,
		source:           sourceState,
	}
	finalStatus := nfse.SyncStatusCompleted
	stopReason := nfse.SyncStopReasonEmptyLimit
	errorCode := ""
	errorMsg := ""

	defer func() {
		_ = s.finishRun(ctx, nfse.FinishRunParams{
			RunID:                 syncRun.ID,
			Status:                finalStatus,
			StopReason:            stopReason,
			ErrorCode:             errorCode,
			ErrorMsg:              errorMsg,
			CheckedCount:          runState.checkedCount,
			DocumentsFound:        runState.documentsInserted,
			EmptyCount:            runState.emptyCount,
			ConsecutiveEmptyCount: runState.consecutiveEmpty,
			ErrorsCount:           runState.errorsCount,
			LastFoundNSU:          runState.lastFoundNSU,
		})
	}()

	cursor := state.LastProcessedNSU
	if mode == nfse.SyncModeFirstSetup {
		cursor = 0
	}

	for {
		select {
		case <-ctx.Done():
			finalStatus = nfse.SyncStatusInterrupted
			stopReason = nfse.SyncStopReasonContextCanceled
			return ctx.Err()
		default:
		}

		budgetLeft, err := s.spendRequest(ctx, company)
		if err != nil {
			finalStatus, stopReason, errorCode, errorMsg = classifySyncError(err)
			return err
		}
		if !budgetLeft {
			stopReason = nfse.SyncStopReasonRateBudget
			break
		}

		batch, err := s.processBatch(ctx, company, cursor, &runState, progress)
		if err != nil {
			finalStatus, stopReason, errorCode, errorMsg = classifySyncError(err)
			return err
		}

		if batch.Done {
			if batch.StopReason != "" {
				stopReason = batch.StopReason
			}
			break
		}

		if batch.NextCursor <= cursor {
			s.log.WarnContext(ctx, "sync batch did not advance cursor; stopping to avoid loop",
				slog.Int64("cursor_last_nsu", cursor),
				slog.Int("docs_in_batch", len(batch.Items)),
				slog.Int("skipped_stale", runState.documentsSkippedStale),
				slog.Int("skipped_duplicate", runState.documentsSkippedDup))
			break
		}

		if err := waitRequestDelay(ctx, s.source.Policy().RequestDelay); err != nil {
			finalStatus = nfse.SyncStatusInterrupted
			stopReason = nfse.SyncStopReasonContextCanceled
			return err
		}

		cursor = batch.NextCursor
	}

	s.log.InfoContext(ctx, "Sync completed",
		slog.String("source", string(kind)),
		slog.Int64("last_processed_nsu", runState.lastProcessedNSU),
		slog.Int("documents_returned", runState.documentsReturned),
		slog.Int("documents_inserted", runState.documentsInserted),
		slog.Int("events_inserted", runState.eventsInserted),
		slog.Int("documents_skipped_stale", runState.documentsSkippedStale),
		slog.Int("documents_skipped_duplicate", runState.documentsSkippedDup),
		slog.Int("documents_skipped_policy", runState.documentsSkippedPolicy),
		slog.Int("events_skipped_policy", runState.eventsSkippedPolicy),
		slog.Int("unsupported", runState.unsupported))

	return nil
}

type syncRuntimeState struct {
	runID                  nfse.SyncRunID
	lastProcessedNSU       int64
	lastFoundNSU           *int64
	maxNSU                 *int64
	checkedCount           int
	documentsInserted      int
	eventsInserted         int
	documentsReturned      int
	documentsSkippedStale  int
	documentsSkippedDup    int
	documentsSkippedPolicy int
	eventsSkippedPolicy    int
	unsupported            int
	completasSaved         int
	resumosSaved           int
	emptyCount             int
	consecutiveEmpty       int
	errorsCount            int
	source                 SourceState
}

type syncFailure struct {
	err        error
	status     nfse.SyncStatus
	stopReason nfse.SyncStopReason
	code       string
}

func (e *syncFailure) Error() string {
	return e.err.Error()
}

func (e *syncFailure) Unwrap() error {
	return e.err
}

func classifySyncError(err error) (nfse.SyncStatus, nfse.SyncStopReason, string, string) {
	var syncErr *syncFailure
	if errors.As(err, &syncErr) {
		return syncErr.status, syncErr.stopReason, syncErr.code, syncErr.err.Error()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nfse.SyncStatusInterrupted, nfse.SyncStopReasonContextCanceled, "context_canceled", err.Error()
	}
	return nfse.SyncStatusFailed, nfse.SyncStopReasonProcessError, "process_error", err.Error()
}

// persistFailure marks a failed checkpoint write as a failed run.
func persistFailure(err error) *syncFailure {
	return &syncFailure{
		err:        err,
		status:     nfse.SyncStatusFailed,
		stopReason: nfse.SyncStopReasonProcessError,
		code:       "persist_error",
	}
}

// spendRequest enforces the source's hourly request budget before a fetch.
// It records the request before it is sent, so failed attempts count too.
// It returns false, and blocks the source until the oldest request in the
// window expires, when the budget is exhausted.
func (s *SyncService) spendRequest(ctx context.Context, company *nfse.Company) (bool, error) {
	limit := s.source.Policy().RequestsPerHour
	if limit <= 0 {
		return true, nil
	}
	kind := s.source.Kind()

	now := time.Now().UTC()
	count, oldest, err := s.store.RequestsSince(ctx, company.ID, kind, now.Add(-time.Hour))
	if err != nil {
		return false, persistFailure(fmt.Errorf("failed to read request budget: %w", err))
	}
	if count >= limit {
		until := now.Add(time.Hour)
		if oldest != nil {
			until = oldest.Add(time.Hour)
		}
		if err := s.store.SetBlockedUntil(ctx, company.ID, kind, until, nfse.SyncStopReasonRateBudget); err != nil {
			return false, persistFailure(fmt.Errorf("failed to block source after request budget: %w", err))
		}
		s.log.WarnContext(ctx, "Limite de consultas por hora atingido",
			slog.String("source", string(kind)),
			slog.Int("requests_last_hour", count),
			slog.Int("limit", limit),
			slog.Time("next_allowed_at", until))
		return false, nil
	}

	if err := s.store.RecordRequest(ctx, company.ID, kind, now); err != nil {
		return false, persistFailure(fmt.Errorf("failed to record request: %w", err))
	}
	return true, nil
}

// progressParams is the run checkpoint as it stands in runState.
func (s *SyncService) progressParams(company *nfse.Company, runState *syncRuntimeState) nfse.PersistSyncProgressParams {
	return nfse.PersistSyncProgressParams{
		CompanyID:             company.ID,
		Source:                s.source.Kind(),
		RunID:                 runState.runID,
		Environment:           company.Environment,
		ConsultationCNPJ:      company.CNPJ,
		LastProcessedNSU:      runState.lastProcessedNSU,
		LastFoundNSU:          runState.lastFoundNSU,
		MaxNSU:                runState.maxNSU,
		LastEmptyStreak:       runState.consecutiveEmpty,
		CheckedCount:          runState.checkedCount,
		DocumentsFound:        runState.documentsInserted,
		EmptyCount:            runState.emptyCount,
		ConsecutiveEmptyCount: runState.consecutiveEmpty,
		ErrorsCount:           runState.errorsCount,
		MarkSuccess:           true,
	}
}

// processBatch fetches one batch after cursor, processes its fresh items in
// NSU order and checkpoints the run.
func (s *SyncService) processBatch(ctx context.Context, company *nfse.Company, cursor int64, runState *syncRuntimeState, progress nfse.ProgressFunc) (Batch, error) {
	batch, err := s.source.Fetch(ctx, company, cursor)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Batch{}, err
		}
		return Batch{}, &syncFailure{
			err:        fmt.Errorf("failed to fetch documents at NSU %d: %w", cursor, err),
			status:     nfse.SyncStatusFailed,
			stopReason: nfse.SyncStopReasonFetchError,
			code:       "fetch_error",
		}
	}

	runState.checkedCount++
	runState.documentsReturned += len(batch.Items)
	if batch.MaxNSU > 0 {
		maxNSU := batch.MaxNSU
		runState.maxNSU = &maxNSU
	}

	processedAny := false
	for _, item := range batch.Items {
		if item.NSU <= cursor {
			runState.documentsSkippedStale++
			s.log.DebugContext(ctx, "Ignoring stale document", slog.Int64("nsu", item.NSU))
			continue
		}
		if err := s.processItem(ctx, company, item, runState); err != nil {
			return Batch{}, err
		}
		processedAny = true
	}

	// Item commits already checkpointed the run. An empty or all-stale batch
	// still counts, and a source may move the cursor past its last item
	// (NF-e returns ultNSU).
	needsCheckpoint := !processedAny
	if len(batch.Items) == 0 {
		runState.emptyCount++
		runState.consecutiveEmpty++
	} else {
		runState.consecutiveEmpty = 0
	}
	if batch.NextCursor > runState.lastProcessedNSU {
		runState.lastProcessedNSU = batch.NextCursor
		needsCheckpoint = true
	}
	if needsCheckpoint {
		if err := s.store.PersistProgress(ctx, s.progressParams(company, runState)); err != nil {
			return Batch{}, persistFailure(fmt.Errorf("failed to persist batch checkpoint: %w", err))
		}
	}

	if batch.WaitUntil != nil {
		if err := s.store.SetBlockedUntil(ctx, company.ID, s.source.Kind(), *batch.WaitUntil, batch.StopReason); err != nil {
			return Batch{}, persistFailure(fmt.Errorf("failed to record source wait: %w", err))
		}
	}

	if batch.Done && runState.source.InitialSyncDoneAt == nil &&
		(batch.StopReason == nfse.SyncStopReasonEmptyLimit || batch.StopReason == nfse.SyncStopReasonCaughtUp) {
		if err := s.store.MarkInitialSyncCompleted(ctx, company.ID, s.source.Kind()); err != nil {
			return Batch{}, persistFailure(fmt.Errorf("failed to mark initial sync completed: %w", err))
		}
		now := time.Now().UTC()
		runState.source.InitialSyncDoneAt = &now
	}

	s.reportProgress(progress, runState, cursor, batch)
	s.log.DebugContext(ctx, "Distribution response observed",
		slog.String("source", string(s.source.Kind())),
		slog.Int64("requested_last_nsu", cursor),
		slog.Int64("ult_nsu", batch.UltNSU),
		slog.Int64("max_nsu", batch.MaxNSU),
		slog.Int("docs_in_batch", len(batch.Items)),
		slog.Int("documents_inserted", runState.documentsInserted),
		slog.Int("events_inserted", runState.eventsInserted),
		slog.Int("documents_skipped_stale", runState.documentsSkippedStale),
		slog.Int("documents_skipped_duplicate", runState.documentsSkippedDup),
		slog.Int("documents_skipped_policy", runState.documentsSkippedPolicy),
		slog.Int("events_skipped_policy", runState.eventsSkippedPolicy))

	return batch, nil
}

// processItem hands one fresh item to the source and counts the outcome. On
// failure it checkpoints the error without moving the cursor.
func (s *SyncService) processItem(ctx context.Context, company *nfse.Company, item Item, runState *syncRuntimeState) error {
	nextLastFoundNSU := runState.lastFoundNSU
	if runState.lastFoundNSU == nil || item.NSU > *runState.lastFoundNSU {
		nsu := item.NSU
		nextLastFoundNSU = &nsu
	}

	itemProgress := s.progressParams(company, runState)
	itemProgress.LastProcessedNSU = item.NSU
	itemProgress.LastFoundNSU = nextLastFoundNSU
	commit := func(ctx context.Context, write func(tx *sql.Tx) (ItemOutcome, error)) (ItemOutcome, error) {
		return s.store.ApplyWithProgress(ctx, itemProgress, write)
	}

	outcome, err := s.source.ProcessItem(ctx, company, runState.source, item, commit)
	if err != nil {
		runState.errorsCount++
		outcome, err = s.skipPoisonItem(ctx, company, item, err, commit)
	}
	if err != nil {
		failed := s.progressParams(company, runState)
		failed.MarkSuccess = false
		failed.ErrorCode = "process_error"
		failed.ErrorMessage = err.Error()
		if persistErr := s.store.PersistProgress(ctx, failed); persistErr != nil {
			return &syncFailure{
				err:        fmt.Errorf("failed to persist checkpoint after processing error: %w", persistErr),
				status:     nfse.SyncStatusFailed,
				stopReason: nfse.SyncStopReasonProcessError,
				code:       "process_error",
			}
		}
		return &syncFailure{
			err:        fmt.Errorf("failed to process NSU %d: %w", item.NSU, err),
			status:     nfse.SyncStatusFailed,
			stopReason: nfse.SyncStopReasonProcessError,
			code:       "process_error",
		}
	}

	switch {
	case outcome.SkippedByPolicy && outcome.IsEvent:
		runState.eventsSkippedPolicy++
	case outcome.SkippedByPolicy:
		runState.documentsSkippedPolicy++
	case outcome.Unsupported:
		runState.unsupported++
	case outcome.Inserted && outcome.IsEvent:
		runState.eventsInserted++
	case outcome.Inserted:
		runState.documentsInserted++
	default:
		runState.documentsSkippedDup++
	}
	switch outcome.Completeness {
	case nfe.CompletenessCompleta:
		runState.completasSaved++
	case nfe.CompletenessResumo:
		runState.resumosSaved++
	}
	runState.lastProcessedNSU = item.NSU
	runState.lastFoundNSU = nextLastFoundNSU
	return nil
}

// skipPoisonItem gives up on an item that failed to decode or parse
// maxItemAttempts runs in a row: it commits the item as unsupported so one
// bad document cannot stall the source. Any other failure, or an earlier
// attempt, is returned unchanged.
func (s *SyncService) skipPoisonItem(ctx context.Context, company *nfse.Company, item Item, processErr error, commit CommitFunc) (ItemOutcome, error) {
	var parseErr *ProcessingError
	if !errors.As(processErr, &parseErr) {
		return ItemOutcome{}, processErr
	}

	attempts, err := s.store.RecordItemFailure(ctx, nfse.GetOrCreateSyncStateParams{
		CompanyID:        company.ID,
		Source:           s.source.Kind(),
		Environment:      company.Environment,
		ConsultationCNPJ: company.CNPJ,
	}, item.NSU)
	if err != nil {
		return ItemOutcome{}, errors.Join(processErr, fmt.Errorf("record item failure: %w", err))
	}
	if attempts < maxItemAttempts {
		return ItemOutcome{}, processErr
	}

	outcome, err := commit(ctx, func(*sql.Tx) (ItemOutcome, error) {
		return ItemOutcome{Unsupported: true, IsEvent: item.IsEvent}, nil
	})
	if err != nil {
		return ItemOutcome{}, fmt.Errorf("persist unsupported item progress failed: %w", err)
	}
	s.log.WarnContext(ctx, "Documento ignorado após falhas repetidas de leitura",
		slog.String("source", string(s.source.Kind())),
		slog.Int64("nsu", item.NSU),
		slog.Int("attempts", attempts),
		slog.String("raw_hash", parseErr.RawHash),
		slog.Any("err", parseErr))
	return outcome, nil
}

func (s *SyncService) reportProgress(progress nfse.ProgressFunc, runState *syncRuntimeState, cursor int64, batch Batch) {
	if progress == nil {
		return
	}
	docsInBatch := len(batch.Items)
	progress(nfse.ProgressEvent{
		Source:                   s.source.Kind(),
		CurrentNSU:               cursor,
		MaxNSU:                   batch.MaxNSU,
		LastProcessedNSU:         runState.lastProcessedNSU,
		LastFoundNSU:             runState.lastFoundNSU,
		EmptyStreak:              runState.consecutiveEmpty,
		DocsFound:                runState.documentsInserted,
		DocumentsSaved:           runState.documentsInserted,
		EventsSaved:              runState.eventsInserted,
		DocumentsSkippedByPolicy: runState.documentsSkippedPolicy,
		EventsSkippedByPolicy:    runState.eventsSkippedPolicy,
		CompletasSaved:           runState.completasSaved,
		ResumosSaved:             runState.resumosSaved,
		DocsInBatch:              docsInBatch,
		Errors:                   runState.errorsCount,
		Message:                  fmt.Sprintf("cursor=%d fetched=%d ultNSU=%d maxNSU=%d inserted=%d events=%d stale=%d duplicate=%d skipped_policy=%d/%d", cursor, docsInBatch, batch.UltNSU, batch.MaxNSU, runState.documentsInserted, runState.eventsInserted, runState.documentsSkippedStale, runState.documentsSkippedDup, runState.documentsSkippedPolicy, runState.eventsSkippedPolicy),
	})
}

func waitRequestDelay(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *SyncService) finishRun(ctx context.Context, params nfse.FinishRunParams) error {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.store.FinishRun(finishCtx, params)
}

// ProcessingError carries the structured context that used to be emitted via a
// duplicate ErrorContext log call inside processDocument/processEvent. The
// service layer now returns it instead of logging, so a single boundary (CLI
// main, desktop) renders the failure once. NSU/Schema/etc. survive in the
// returned error via Error() and via slog.LogValue for any slog-aware boundary.
type ProcessingError struct {
	Op         string // "decode document" | "parse document" | "decode event" | "parse event"
	NSU        int64
	Schema     string
	DocType    string
	EventType  string
	XMLPreview string
	RawHash    string // blob of the raw XML, saved when the payload decoded
	Err        error
}

func (e *ProcessingError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s failed (nsu=%d", e.Op, e.NSU)
	if e.Schema != "" {
		fmt.Fprintf(&b, ", schema=%s", e.Schema)
	}
	if e.DocType != "" {
		fmt.Fprintf(&b, ", tipo_documento=%s", e.DocType)
	}
	if e.EventType != "" {
		fmt.Fprintf(&b, ", tipo_evento=%s", e.EventType)
	}
	if e.XMLPreview != "" {
		fmt.Fprintf(&b, ", xml_preview=%s", e.XMLPreview)
	}
	b.WriteString("): ")
	b.WriteString(e.Err.Error())
	return b.String()
}

func (e *ProcessingError) Unwrap() error { return e.Err }

// LogValue lets any slog-aware boundary render the structured fields by logging
// the error via slog.Any("err", err) without re-deriving them.
func (e *ProcessingError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("op", e.Op),
		slog.Int64("nsu", e.NSU),
		slog.String("error", e.Err.Error()),
	}
	if e.Schema != "" {
		attrs = append(attrs, slog.String("schema", e.Schema))
	}
	if e.DocType != "" {
		attrs = append(attrs, slog.String("tipo_documento", e.DocType))
	}
	if e.EventType != "" {
		attrs = append(attrs, slog.String("tipo_evento", e.EventType))
	}
	if e.XMLPreview != "" {
		attrs = append(attrs, slog.String("xml_preview", e.XMLPreview))
	}
	if e.RawHash != "" {
		attrs = append(attrs, slog.String("raw_hash", e.RawHash))
	}
	return slog.GroupValue(attrs...)
}
