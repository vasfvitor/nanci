package syncservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
)

const (
	requestDelay = 500 * time.Millisecond
)

var syncRequestDelay = requestDelay

// SyncService orchestrates the synchronization of documents from the ADN API.
type SyncService struct {
	store      nfse.SyncRepository
	apiClient  documentFetcher
	fileWriter files.XMLStore
	log        *slog.Logger
}

type documentFetcher interface {
	FetchDocuments(ctx context.Context, req adn.DistributionRequest) (*adn.DocumentResponse, error)
}

// NewSyncService creates a new SyncService.
func NewSyncService(syncRepo nfse.SyncRepository, adnClient documentFetcher, xmlStore files.XMLStore, log *slog.Logger) *SyncService {
	return &SyncService{
		store:      syncRepo,
		apiClient:  adnClient,
		fileWriter: xmlStore,
		log:        log,
	}
}

// Sync starts the synchronization process for a specific company.
func (s *SyncService) Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, mode nfse.SyncMode, progress nfse.ProgressFunc) error {
	if mode == "" {
		mode = nfse.SyncModeNormal
	}

	state, err := s.store.GetOrCreateState(ctx, nfse.GetOrCreateSyncStateParams{
		CompanyID:        company.ID,
		Environment:      company.Environment,
		ConsultationCNPJ: company.CNPJ,
		LegacyLastNSU:    company.LastNSU,
	})
	if err != nil {
		return fmt.Errorf("failed to load sync state: %w", err)
	}
	company.LastNSU = state.LastProcessedNSU

	s.log.InfoContext(ctx, "Iniciando processo de sincronização",
		slog.String("cnpj", company.CNPJ),
		slog.String("mode", string(mode)),
		slog.Int64("from_nsu", state.LastProcessedNSU))

	syncRun, err := s.store.StartRun(ctx, nfse.StartRunParams{
		CompanyID:         company.ID,
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
		lastProcessedNSU:       state.LastProcessedNSU,
		lastFoundNSU:           state.LastFoundNSU,
		lastFoundNSUValid:      state.LastFoundNSUValid,
		checkedCount:           0,
		documentsInserted:      0,
		documentsReturned:      0,
		documentsSkippedStale:  0,
		documentsSkippedDup:    0,
		emptyCount:             0,
		consecutiveEmpty:       0,
		errorsCount:            0,
		initialEmptyStreak:     state.LastEmptyStreak,
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
			LastFoundNSUValid:     runState.lastFoundNSUValid,
		})
	}()

	cursorLastNSU := state.LastProcessedNSU
	if mode == nfse.SyncModeFirstSetup {
		cursorLastNSU = 0
	}

	for {
		select {
		case <-ctx.Done():
			finalStatus = nfse.SyncStatusInterrupted
			stopReason = nfse.SyncStopReasonContextCanceled
			return ctx.Err()
		default:
		}

		batchResult, err := s.processNSU(ctx, company, syncRun.ID, cursorLastNSU, &runState, progress)
		if err != nil {
			finalStatus, stopReason, errorCode, errorMsg = classifySyncError(err)
			return err
		}

		if batchResult.docsInBatch == 0 {
			break
		}

		if !batchResult.advanced {
			s.log.WarnContext(ctx, "sync batch did not advance cursor; stopping to avoid loop",
				slog.Int64("cursor_last_nsu", cursorLastNSU),
				slog.Int("docs_in_batch", batchResult.docsInBatch),
				slog.Int("skipped_stale", runState.documentsSkippedStale),
				slog.Int("skipped_duplicate", runState.documentsSkippedDup))
			break
		}

		if err := waitRequestDelay(ctx); err != nil {
			finalStatus = nfse.SyncStatusInterrupted
			stopReason = nfse.SyncStopReasonContextCanceled
			return err
		}

		cursorLastNSU = runState.lastProcessedNSU
	}

	s.log.InfoContext(ctx, "Sync completed",
		slog.Int64("last_processed_nsu", runState.lastProcessedNSU),
		slog.Int("documents_returned", runState.documentsReturned),
		slog.Int("documents_inserted", runState.documentsInserted),
		slog.Int("documents_skipped_stale", runState.documentsSkippedStale),
		slog.Int("documents_skipped_duplicate", runState.documentsSkippedDup))

	return nil
}

type syncRuntimeState struct {
	lastProcessedNSU      int64
	lastFoundNSU          int64
	lastFoundNSUValid     bool
	checkedCount          int
	documentsInserted     int
	documentsReturned     int
	documentsSkippedStale int
	documentsSkippedDup   int
	emptyCount            int
	consecutiveEmpty      int
	errorsCount           int
	initialEmptyStreak    int
}

type syncBatchResult struct {
	docsInBatch int
	advanced    bool
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

func (s *SyncService) processNSU(ctx context.Context, company *nfse.Company, runID nfse.SyncRunID, cursorLastNSU int64, runState *syncRuntimeState, progress nfse.ProgressFunc) (syncBatchResult, error) {
	resp, err := s.apiClient.FetchDocuments(ctx, adn.DistributionRequest{
		LastNSU:          cursorLastNSU,
		ConsultationCNPJ: company.CNPJ,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return syncBatchResult{}, err
		}
		return syncBatchResult{}, &syncFailure{
			err:        fmt.Errorf("failed to fetch documents at NSU %d: %w", cursorLastNSU, err),
			status:     nfse.SyncStatusFailed,
			stopReason: nfse.SyncStopReasonFetchError,
			code:       "fetch_error",
		}
	}

	runState.checkedCount++
	docsInBatch := len(resp.Docs)
	runState.documentsReturned += docsInBatch

	sort.Slice(resp.Docs, func(i, j int) bool {
		return resp.Docs[i].NSU < resp.Docs[j].NSU
	})

	nextCursorLastNSU := cursorLastNSU

	for _, env := range resp.Docs {
		if env.NSU <= cursorLastNSU {
			runState.documentsSkippedStale++
			s.log.DebugContext(ctx, "Ignoring stale document", slog.Int64("nsu", env.NSU))
			continue
		}

		nextLastFoundNSU := runState.lastFoundNSU
		nextLastFoundNSUValid := runState.lastFoundNSUValid
		if !runState.lastFoundNSUValid || env.NSU > runState.lastFoundNSU {
			nextLastFoundNSU = env.NSU
			nextLastFoundNSUValid = true
		}

		progressParams := nfse.PersistSyncProgressParams{
			CompanyID:             company.ID,
			RunID:                 runID,
			Environment:           company.Environment,
			ConsultationCNPJ:      company.CNPJ,
			LastProcessedNSU:      env.NSU,
			LastFoundNSU:          nextLastFoundNSU,
			LastFoundNSUValid:     nextLastFoundNSUValid,
			LastEmptyStreak:       runState.currentEmptyStreak(),
			CheckedCount:          runState.checkedCount,
			DocumentsFound:        runState.documentsInserted,
			EmptyCount:            runState.emptyCount,
			ConsecutiveEmptyCount: runState.consecutiveEmpty,
			ErrorsCount:           runState.errorsCount,
			MarkSuccess:           true,
		}

		var (
			outcome    nfse.ApplyOutcome
			processErr error
		)
		if env.IsEvent() {
			outcome, processErr = s.processEvent(ctx, company, env, progressParams)
		} else {
			outcome, processErr = s.processDocument(ctx, company, env, progressParams)
		}

		if processErr != nil {
			runState.errorsCount++
			if persistErr := s.store.PersistProgress(ctx, nfse.PersistSyncProgressParams{
				CompanyID:             company.ID,
				RunID:                 runID,
				Environment:           company.Environment,
				ConsultationCNPJ:      company.CNPJ,
				LastProcessedNSU:      runState.lastProcessedNSU,
				LastFoundNSU:          runState.lastFoundNSU,
				LastFoundNSUValid:     runState.lastFoundNSUValid,
				LastEmptyStreak:       runState.currentEmptyStreak(),
				CheckedCount:          runState.checkedCount,
				DocumentsFound:        runState.documentsInserted,
				EmptyCount:            runState.emptyCount,
				ConsecutiveEmptyCount: runState.consecutiveEmpty,
				ErrorsCount:           runState.errorsCount,
				ErrorCode:             "process_error",
				ErrorMessage:          processErr.Error(),
			}); persistErr != nil {
				return syncBatchResult{}, &syncFailure{
					err:        fmt.Errorf("failed to persist checkpoint after processing error: %w", persistErr),
					status:     nfse.SyncStatusFailed,
					stopReason: nfse.SyncStopReasonProcessError,
					code:       "process_error",
				}
			}
			return syncBatchResult{}, &syncFailure{
				err:        fmt.Errorf("failed to process NSU %d: %w", env.NSU, processErr),
				status:     nfse.SyncStatusFailed,
				stopReason: nfse.SyncStopReasonProcessError,
				code:       "process_error",
			}
		}

		if outcome.Inserted {
			runState.documentsInserted++
		} else {
			runState.documentsSkippedDup++
		}
		runState.lastProcessedNSU = env.NSU
		runState.lastFoundNSU = nextLastFoundNSU
		runState.lastFoundNSUValid = nextLastFoundNSUValid
		nextCursorLastNSU = env.NSU
	}

	company.LastNSU = runState.lastProcessedNSU

	if docsInBatch == 0 {
		runState.emptyCount++
		runState.consecutiveEmpty++
		if err := s.store.PersistProgress(ctx, nfse.PersistSyncProgressParams{
			CompanyID:             company.ID,
			RunID:                 runID,
			Environment:           company.Environment,
			ConsultationCNPJ:      company.CNPJ,
			LastProcessedNSU:      runState.lastProcessedNSU,
			LastFoundNSU:          runState.lastFoundNSU,
			LastFoundNSUValid:     runState.lastFoundNSUValid,
			LastEmptyStreak:       runState.currentEmptyStreak(),
			CheckedCount:          runState.checkedCount,
			DocumentsFound:        runState.documentsInserted,
			EmptyCount:            runState.emptyCount,
			ConsecutiveEmptyCount: runState.consecutiveEmpty,
			ErrorsCount:           runState.errorsCount,
			MarkSuccess:           true,
		}); err != nil {
			return syncBatchResult{}, &syncFailure{
				err:        fmt.Errorf("failed to persist sync progress on empty batch: %w", err),
				status:     nfse.SyncStatusFailed,
				stopReason: nfse.SyncStopReasonProcessError,
				code:       "persist_error",
			}
		}
	} else {
		runState.consecutiveEmpty = 0
		if nextCursorLastNSU == cursorLastNSU {
			if err := s.store.PersistProgress(ctx, nfse.PersistSyncProgressParams{
				CompanyID:             company.ID,
				RunID:                 runID,
				Environment:           company.Environment,
				ConsultationCNPJ:      company.CNPJ,
				LastProcessedNSU:      runState.lastProcessedNSU,
				LastFoundNSU:          runState.lastFoundNSU,
				LastFoundNSUValid:     runState.lastFoundNSUValid,
				LastEmptyStreak:       runState.currentEmptyStreak(),
				CheckedCount:          runState.checkedCount,
				DocumentsFound:        runState.documentsInserted,
				EmptyCount:            runState.emptyCount,
				ConsecutiveEmptyCount: runState.consecutiveEmpty,
				ErrorsCount:           runState.errorsCount,
				MarkSuccess:           true,
			}); err != nil {
				return syncBatchResult{}, &syncFailure{
					err:        fmt.Errorf("failed to persist sync progress on non-advancing batch: %w", err),
					status:     nfse.SyncStatusFailed,
					stopReason: nfse.SyncStopReasonProcessError,
					code:       "persist_error",
				}
			}
		}
	}

	s.reportProgress(progress, runState, cursorLastNSU, resp, docsInBatch)
	s.log.DebugContext(ctx, "ADN response observed",
		slog.Int64("requested_last_nsu", cursorLastNSU),
		slog.Int64("ult_nsu", resp.UltNSU),
		slog.Int64("max_nsu", resp.MaxNSU),
		slog.Int("docs_in_batch", docsInBatch),
		slog.Int("documents_inserted", runState.documentsInserted),
		slog.Int("documents_skipped_stale", runState.documentsSkippedStale),
		slog.Int("documents_skipped_duplicate", runState.documentsSkippedDup))

	return syncBatchResult{
		docsInBatch: docsInBatch,
		advanced:    nextCursorLastNSU > cursorLastNSU,
	}, nil
}

func (s *SyncService) reportProgress(progress nfse.ProgressFunc, runState *syncRuntimeState, cursorLastNSU int64, resp *adn.DocumentResponse, docsInBatch int) {
	if progress == nil {
		return
	}
	progress(nfse.ProgressEvent{
		CurrentNSU:        cursorLastNSU,
		MaxNSU:            resp.MaxNSU,
		LastProcessedNSU:  runState.lastProcessedNSU,
		LastFoundNSU:      runState.lastFoundNSU,
		LastFoundNSUValid: runState.lastFoundNSUValid,
		EmptyStreak:       runState.currentEmptyStreak(),
		DocsFound:         runState.documentsInserted,
		DocsInBatch:       docsInBatch,
		Errors:            runState.errorsCount,
		Message:           fmt.Sprintf("cursor=%d fetched=%d ultNSU=%d maxNSU=%d inserted=%d stale=%d duplicate=%d", cursorLastNSU, docsInBatch, resp.UltNSU, resp.MaxNSU, runState.documentsInserted, runState.documentsSkippedStale, runState.documentsSkippedDup),
	})
}

func (r *syncRuntimeState) currentEmptyStreak() int {
	if r.consecutiveEmpty == 0 {
		return 0
	}
	return r.consecutiveEmpty
}

func waitRequestDelay(ctx context.Context) error {
	timer := time.NewTimer(syncRequestDelay)
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

// processDocument handles the decoding, parsing, and saving of a single document.
func (s *SyncService) processDocument(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope, progressParams nfse.PersistSyncProgressParams) (nfse.ApplyOutcome, error) {
	s.log.Log(ctx, slog.Level(-8), "Processando documento", slog.Int64("nsu", env.NSU))

	payload, err := nfse.DecodePayload(env.PayloadBase64(), nfse.PayloadLimits{
		CompressedBytes:   5 * 1024 * 1024,
		UncompressedBytes: 20 * 1024 * 1024,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao decodificar payload do documento", slog.Int64("nsu", env.NSU), slog.String("erro", err.Error()))
		return nfse.ApplyOutcome{}, fmt.Errorf("decode failed: %w", err)
	}

	doc, _, err := nfse.ParseDocumentXML(payload.XML)
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao interpretar XML de documento",
			slog.Int64("nsu", env.NSU),
			slog.String("schema", env.Schema),
			slog.String("tipo_documento", env.DocumentType),
			slog.String("tipo_evento", env.EventType),
			slog.String("xml_preview", xmlPreview(payload.XML)),
			slog.String("erro", err.Error()))
		return nfse.ApplyOutcome{}, fmt.Errorf("parse failed: %w", err)
	}

	doc.ID = nfse.DocumentID(uuid.NewString())
	doc.RawHash = payload.SHA256

	if err := s.fileWriter.Store(doc.RawHash, payload.XML); err != nil {
		return nfse.ApplyOutcome{}, fmt.Errorf("file save failed: %w", err)
	}
	doc.XMLPath = doc.RawHash + ".xml"

	participation := nfse.ClassifyCompanyParticipation(&doc, company.CNPJ)
	outcome, err := s.store.ApplyDocumentAndProgress(ctx, nfse.ApplyDocumentAndProgressParams{
		DocumentParams: nfse.ApplyDocumentParams{
			Document:      doc,
			Participation: participation,
			CompanyID:     company.ID,
			NSU:           env.NSU,
		},
		ProgressParams: progressParams,
	})
	if err != nil {
		return nfse.ApplyOutcome{}, fmt.Errorf("db apply document failed: %w", err)
	}

	return outcome, nil
}

// processEvent handles decoding and saving an Event.
func (s *SyncService) processEvent(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope, progressParams nfse.PersistSyncProgressParams) (nfse.ApplyOutcome, error) {
	s.log.Log(ctx, slog.Level(-8), "Processando evento", slog.Int64("nsu", env.NSU))

	payload, err := nfse.DecodePayload(env.PayloadBase64(), nfse.PayloadLimits{
		CompressedBytes:   5 * 1024 * 1024,
		UncompressedBytes: 20 * 1024 * 1024,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao decodificar payload do evento", slog.Int64("nsu", env.NSU), slog.String("erro", err.Error()))
		return nfse.ApplyOutcome{}, fmt.Errorf("decode event failed: %w", err)
	}

	ev, _, err := nfse.ParseEventXML(payload.XML)
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao interpretar XML de evento",
			slog.Int64("nsu", env.NSU),
			slog.String("schema", env.Schema),
			slog.String("tipo_documento", env.DocumentType),
			slog.String("tipo_evento", env.EventType),
			slog.String("xml_preview", xmlPreview(payload.XML)),
			slog.String("erro", err.Error()))
		return nfse.ApplyOutcome{}, fmt.Errorf("parse event failed: %w", err)
	}

	ev.ID = nfse.GenerateID()
	ev.RawHash = payload.SHA256

	if err := s.fileWriter.Store(ev.RawHash, payload.XML); err != nil {
		return nfse.ApplyOutcome{}, fmt.Errorf("event file save failed: %w", err)
	}
	ev.RawXMLPath = ev.RawHash + ".xml"

	outcome, err := s.store.ApplyEventAndProgress(ctx, nfse.ApplyEventAndProgressParams{
		EventParams: nfse.ApplyEventParams{
			Event:     ev,
			CompanyID: company.ID,
			NSU:       env.NSU,
		},
		ProgressParams: progressParams,
	})
	if err != nil {
		return nfse.ApplyOutcome{}, fmt.Errorf("db apply event failed: %w", err)
	}

	return outcome, nil
}

func xmlPreview(data []byte) string {
	preview := strings.TrimSpace(string(data))
	preview = strings.ReplaceAll(preview, "\r", " ")
	preview = strings.ReplaceAll(preview, "\n", " ")
	preview = strings.ReplaceAll(preview, "\t", " ")
	preview = strings.Join(strings.Fields(preview), " ")
	if len(preview) > 400 {
		return preview[:400] + "...(truncated)"
	}
	return preview
}
