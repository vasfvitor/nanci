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
	normalRevisitWindow  = 50
	normalEmptyLimit     = 100
	firstSetupEmptyLimit = 500
	requestDelay         = 500 * time.Millisecond
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
		lastProcessedNSU:     state.LastProcessedNSU,
		lastFoundNSU:       state.LastFoundNSU,
		lastFoundNSUValid:  state.LastFoundNSUValid,
		checkedCount:       0,
		documentsFound:     0,
		emptyCount:         0,
		consecutiveEmpty:   0,
		errorsCount:        0,
		initialEmptyStreak: state.LastEmptyStreak,
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
			DocumentsFound:        runState.documentsFound,
			EmptyCount:            runState.emptyCount,
			ConsecutiveEmptyCount: runState.consecutiveEmpty,
			ErrorsCount:           runState.errorsCount,
			LastFoundNSU:          runState.lastFoundNSU,
			LastFoundNSUValid:     runState.lastFoundNSUValid,
		})
	}()

	if mode == nfse.SyncModeNormal {
		revisitStart := maxInt64(1, state.LastProcessedNSU-int64(normalRevisitWindow)+1)
		for nsu := revisitStart; nsu <= state.LastProcessedNSU; nsu++ {
			select {
			case <-ctx.Done():
				finalStatus = nfse.SyncStatusInterrupted
				stopReason = nfse.SyncStopReasonContextCanceled
				return ctx.Err()
			default:
			}

			_, err := s.processNSU(ctx, company, syncRun.ID, nsu+1, true, &runState, progress)
			if err != nil {
				finalStatus, stopReason, errorCode, errorMsg = classifySyncError(err)
				return err
			}

			if err := waitRequestDelay(ctx); err != nil {
				finalStatus = nfse.SyncStatusInterrupted
				stopReason = nfse.SyncStopReasonContextCanceled
				return err
			}
		}
	}

	advanceNSU := int64(1)
	if mode == nfse.SyncModeNormal {
		advanceNSU = state.LastProcessedNSU + 1
	}

	for {
		select {
		case <-ctx.Done():
			finalStatus = nfse.SyncStatusInterrupted
			stopReason = nfse.SyncStopReasonContextCanceled
			return ctx.Err()
		default:
		}

		processedCount, err := s.processNSU(ctx, company, syncRun.ID, advanceNSU, false, &runState, progress)
		if err != nil {
			finalStatus, stopReason, errorCode, errorMsg = classifySyncError(err)
			return err
		}

		if processedCount == 0 {
			break
		}

		if err := waitRequestDelay(ctx); err != nil {
			finalStatus = nfse.SyncStatusInterrupted
			stopReason = nfse.SyncStopReasonContextCanceled
			return err
		}

		advanceNSU = runState.lastProcessedNSU + 1
	}

	return nil
}


type syncRuntimeState struct {
	lastProcessedNSU     int64
	lastFoundNSU       int64
	lastFoundNSUValid  bool
	checkedCount       int
	documentsFound     int
	emptyCount         int
	consecutiveEmpty   int
	errorsCount        int
	initialEmptyStreak int
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

func (s *SyncService) processNSU(ctx context.Context, company *nfse.Company, runID nfse.SyncRunID, advanceNSU int64, isRevisit bool, runState *syncRuntimeState, progress nfse.ProgressFunc) (int, error) {
	resp, err := s.apiClient.FetchDocuments(ctx, adn.DistributionRequest{
		LastNSU:          advanceNSU - 1,
		ConsultationCNPJ: company.CNPJ,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		return 0, &syncFailure{
			err:        fmt.Errorf("failed to fetch documents at NSU %d: %w", advanceNSU-1, err),
			status:     nfse.SyncStatusFailed,
			stopReason: nfse.SyncStopReasonFetchError,
			code:       "fetch_error",
		}
	}

	runState.checkedCount++
	docsInBatch := len(resp.Docs)


	sort.Slice(resp.Docs, func(i, j int) bool {
		return resp.Docs[i].NSU < resp.Docs[j].NSU
	})


	processedCount := 0

	for _, env := range resp.Docs {
		if env.NSU < advanceNSU {
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
			DocumentsFound:        runState.documentsFound + 1,
			EmptyCount:            runState.emptyCount,
			ConsecutiveEmptyCount: runState.consecutiveEmpty,
			ErrorsCount:           runState.errorsCount,
			MarkSuccess:           true,
		}
		if isRevisit {
			progressParams.LastProcessedNSU = runState.lastProcessedNSU
		}

		var processErr error
		if env.IsEvent() {
			processErr = s.processEvent(ctx, company, env, progressParams)
		} else {
			processErr = s.processDocument(ctx, company, env, progressParams)
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
				DocumentsFound:        runState.documentsFound,
				EmptyCount:            runState.emptyCount,
				ConsecutiveEmptyCount: runState.consecutiveEmpty,
				ErrorsCount:           runState.errorsCount,
				ErrorCode:             "process_error",
				ErrorMessage:          processErr.Error(),
			}); persistErr != nil {
				return processedCount, &syncFailure{
					err:        fmt.Errorf("failed to persist checkpoint after processing error: %w", persistErr),
					status:     nfse.SyncStatusFailed,
					stopReason: nfse.SyncStopReasonProcessError,
					code:       "process_error",
				}
			}
			return processedCount, &syncFailure{
				err:        fmt.Errorf("failed to process NSU %d: %w", env.NSU, processErr),
				status:     nfse.SyncStatusFailed,
				stopReason: nfse.SyncStopReasonProcessError,
				code:       "process_error",
			}
		}

		if !isRevisit {
			runState.lastProcessedNSU = env.NSU
		}
		runState.documentsFound++
		processedCount++
		runState.lastFoundNSU = nextLastFoundNSU
		runState.lastFoundNSUValid = nextLastFoundNSUValid
	}

	if !isRevisit {
		company.LastNSU = runState.lastProcessedNSU
	}

	if docsInBatch == 0 {
		runState.emptyCount++
		if !isRevisit {
			runState.consecutiveEmpty++
		}
		
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
			DocumentsFound:        runState.documentsFound,
			EmptyCount:            runState.emptyCount,
			ConsecutiveEmptyCount: runState.consecutiveEmpty,
			ErrorsCount:           runState.errorsCount,
			MarkSuccess:           true,
		}); err != nil {
			return processedCount, &syncFailure{
				err:        fmt.Errorf("failed to persist sync progress on empty batch: %w", err),
				status:     nfse.SyncStatusFailed,
				stopReason: nfse.SyncStopReasonProcessError,
				code:       "persist_error",
			}
		}
	} else if !isRevisit {
		runState.consecutiveEmpty = 0
	}

	s.reportProgress(progress, runState, isRevisit, advanceNSU, resp, docsInBatch)
	s.log.DebugContext(ctx, "ADN response observed",
		slog.Int64("requested_last_nsu", advanceNSU-1),
		slog.Int64("ult_nsu", resp.UltNSU),
		slog.Int64("max_nsu", resp.MaxNSU),
		slog.Int("docs_in_batch", docsInBatch))

	return processedCount, nil
}

func (s *SyncService) reportProgress(progress nfse.ProgressFunc, runState *syncRuntimeState, isRevisit bool, advanceNSU int64, resp *adn.DocumentResponse, docsInBatch int) {
	if progress == nil {
		return
	}
	phase := "advance"
	if isRevisit {
		phase = "revisit"
	}
	progress(nfse.ProgressEvent{
		CurrentNSU:        advanceNSU - 1,
		MaxNSU:            resp.MaxNSU,
		LastProcessedNSU:  runState.lastProcessedNSU,
		LastFoundNSU:      runState.lastFoundNSU,
		LastFoundNSUValid: runState.lastFoundNSUValid,
		EmptyStreak:       runState.currentEmptyStreak(),
		DocsFound:         runState.documentsFound,
		DocsInBatch:       docsInBatch,
		Errors:            runState.errorsCount,
		Message:           fmt.Sprintf("phase=%s fetched=%d ultNSU=%d maxNSU=%d", phase, docsInBatch, resp.UltNSU, resp.MaxNSU),
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
func (s *SyncService) processDocument(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope, progressParams nfse.PersistSyncProgressParams) error {
	s.log.Log(ctx, slog.Level(-8), "Processando documento", slog.Int64("nsu", env.NSU))

	payload, err := nfse.DecodePayload(env.PayloadBase64(), nfse.PayloadLimits{
		CompressedBytes:   5 * 1024 * 1024,
		UncompressedBytes: 20 * 1024 * 1024,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao decodificar payload do documento", slog.Int64("nsu", env.NSU), slog.String("erro", err.Error()))
		return fmt.Errorf("decode failed: %w", err)
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
		return fmt.Errorf("parse failed: %w", err)
	}

	doc.ID = nfse.DocumentID(uuid.NewString())
	doc.RawHash = payload.SHA256

	if err := s.fileWriter.Store(doc.RawHash, payload.XML); err != nil {
		return fmt.Errorf("file save failed: %w", err)
	}
	doc.XMLPath = doc.RawHash + ".xml"

	participation := nfse.ClassifyCompanyParticipation(&doc, company.CNPJ)
	if err := s.store.ApplyDocumentAndProgress(ctx, nfse.ApplyDocumentAndProgressParams{
		DocumentParams: nfse.ApplyDocumentParams{
			Document:      doc,
			Participation: participation,
			CompanyID:     company.ID,
			NSU:           env.NSU,
		},
		ProgressParams: progressParams,
	}); err != nil {
		return fmt.Errorf("db apply document failed: %w", err)
	}

	return nil
}

// processEvent handles decoding and saving an Event.
func (s *SyncService) processEvent(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope, progressParams nfse.PersistSyncProgressParams) error {
	s.log.Log(ctx, slog.Level(-8), "Processando evento", slog.Int64("nsu", env.NSU))

	payload, err := nfse.DecodePayload(env.PayloadBase64(), nfse.PayloadLimits{
		CompressedBytes:   5 * 1024 * 1024,
		UncompressedBytes: 20 * 1024 * 1024,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao decodificar payload do evento", slog.Int64("nsu", env.NSU), slog.String("erro", err.Error()))
		return fmt.Errorf("decode event failed: %w", err)
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
		return fmt.Errorf("parse event failed: %w", err)
	}

	ev.ID = nfse.GenerateID()
	ev.RawHash = payload.SHA256

	if err := s.fileWriter.Store(ev.RawHash, payload.XML); err != nil {
		return fmt.Errorf("event file save failed: %w", err)
	}
	ev.RawXMLPath = ev.RawHash + ".xml"

	if err := s.store.ApplyEventAndProgress(ctx, nfse.ApplyEventAndProgressParams{
		EventParams: nfse.ApplyEventParams{
			Event:     ev,
			CompanyID: company.ID,
			NSU:       env.NSU,
		},
		ProgressParams: progressParams,
	}); err != nil {
		return fmt.Errorf("db apply event failed: %w", err)
	}

	return nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
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
