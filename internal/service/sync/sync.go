package syncservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
)

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
func (s *SyncService) Sync(ctx context.Context, company *nfse.Company, credential *nfse.Credential, consultationBasis string, progress nfse.ProgressFunc) error {
	s.log.InfoContext(ctx, "Iniciando processo de sincronização",
		slog.String("cnpj", company.CNPJ),
		slog.Int64("from_nsu", company.LastNSU))

	// Create SyncRun record
	syncRun, err := s.store.StartRun(ctx, nfse.StartRunParams{
		CompanyID:         company.ID,
		CredentialID:      credential.ID,
		CredentialCNPJ:    credential.OwnerCNPJ,
		ConsultationCNPJ:  company.CNPJ,
		ConsultationBasis: nfse.ConsultationBasis(consultationBasis),
		FromNSU:           company.LastNSU,
		ToNSU:             company.LastNSU, // Initial
	})
	if err != nil {
		return fmt.Errorf("failed to create sync run: %w", err)
	}

	finalStatus := nfse.SyncStatus("completed")
	finalErrorMsg := ""
	var syncErr error
	defer func() {
		_ = s.finishRun(ctx, nfse.FinishRunParams{
			RunID:    syncRun.ID,
			Status:   finalStatus,
			ErrorMsg: finalErrorMsg,
		})
	}()

	committedNSU := company.LastNSU
	totalDocs := 0
	totalErrors := 0
	emptyConsecutive := 0
	const maxEmptyConsecutive = 500

	for {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			finalStatus = "interrupted"
			syncErr = ctx.Err()
			return syncErr
		default:
		}

		// Fetch documents batch
		requestedNSU := committedNSU

		isThrottled := emptyConsecutive > 0 && emptyConsecutive%50 != 0
		if !isThrottled {
			if emptyConsecutive > 0 {
				s.log.InfoContext(ctx, "Varrendo NSUs vazios...", slog.Int64("requested_nsu", requestedNSU), slog.Int("empty_streak", emptyConsecutive))
			} else {
				s.log.InfoContext(ctx, "Buscando lote de documentos", slog.Int64("requested_nsu", requestedNSU))
			}
		}

		resp, err := s.apiClient.FetchDocuments(ctx, adn.DistributionRequest{
			LastNSU:          requestedNSU,
			ConsultationCNPJ: company.CNPJ,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				finalStatus = "interrupted"
				syncErr = err
				return syncErr
			}
			finalStatus = "failed"
			finalErrorMsg = err.Error()
			syncErr = fmt.Errorf("failed to fetch documents at NSU %d: %w", requestedNSU, err)
			return syncErr
		}

		docsInBatch := len(resp.Docs)
		if docsInBatch > 0 || !isThrottled {
			s.log.DebugContext(ctx, "Lote recebido",
				slog.Int("docs_in_batch", docsInBatch),
				slog.Int64("ult_nsu", resp.UltNSU),
				slog.Int64("max_nsu", resp.MaxNSU))
		}

		// Report progress
		if progress != nil {
			progress(nfse.ProgressEvent{
				CurrentNSU:  committedNSU,
				MaxNSU:      resp.MaxNSU,
				DocsFound:   totalDocs,
				DocsInBatch: docsInBatch,
				Errors:      totalErrors,
				Message:     fmt.Sprintf("Fetched %d documents. ultNSU=%d MaxNSU=%d", docsInBatch, resp.UltNSU, resp.MaxNSU),
			})
		}

		// Process batch
		batchSuccessNSU := committedNSU
		for _, env := range resp.Docs {
			var err error
			if env.IsEvent() {
				err = s.processEvent(ctx, company, env)
			} else {
				err = s.processDocument(ctx, company, env)
			}

			if err != nil {
				totalErrors++
				if progress != nil {
					progress(nfse.ProgressEvent{
						CurrentNSU: batchSuccessNSU,
						MaxNSU:     resp.MaxNSU,
						Errors:     totalErrors,
						Message:    fmt.Sprintf("Error processing NSU %d after committed NSU %d: %v", env.NSU, batchSuccessNSU, err),
					})
				}

				if err := s.advanceCheckpoint(ctx, company, syncRun.ID, batchSuccessNSU); err != nil {
					finalStatus = "failed"
					finalErrorMsg = err.Error()
					syncErr = fmt.Errorf("failed to update company last NSU after item error: %w", err)
					return syncErr
				}

				finalStatus = "failed"
				finalErrorMsg = err.Error()
				syncErr = fmt.Errorf("failed to process NSU %d: %w", env.NSU, err)
				return syncErr
			}

			totalDocs++
			if env.NSU > batchSuccessNSU {
				batchSuccessNSU = env.NSU
			}
		}

		committedNSU = nextNSU(requestedNSU, resp.UltNSU, batchSuccessNSU, docsInBatch == 0)
		if docsInBatch == 0 {
			emptyConsecutive++
			if emptyConsecutive >= maxEmptyConsecutive {
				s.log.InfoContext(ctx, "Atingiu limite de NSUs vazios consecutivos, interrompendo varredura", slog.Int("max_empty", maxEmptyConsecutive))
				break
			}
		} else {
			emptyConsecutive = 0
		}

		if err := s.advanceCheckpoint(ctx, company, syncRun.ID, committedNSU); err != nil {
			finalStatus = "failed"
			finalErrorMsg = err.Error()
			syncErr = fmt.Errorf("failed to update company last NSU: %w", err)
			return syncErr
		}

		// Stop condition
		if committedNSU >= resp.MaxNSU && resp.MaxNSU > 0 {
			break
		}
	}

	return nil
}

func (s *SyncService) finishRun(ctx context.Context, params nfse.FinishRunParams) error {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.store.FinishRun(finishCtx, params)
}

func (s *SyncService) advanceCheckpoint(ctx context.Context, company *nfse.Company, runID nfse.SyncRunID, lastNSU int64) error {
	if lastNSU <= company.LastNSU {
		return nil
	}
	if err := s.store.AdvanceCheckpoint(ctx, nfse.AdvanceCheckpointParams{
		CompanyID: company.ID,
		RunID:     runID,
		LastNSU:   lastNSU,
	}); err != nil {
		return err
	}
	company.LastNSU = lastNSU
	return nil
}

func nextNSU(requestedNSU, apiUltNSU, batchSuccessNSU int64, emptyBatch bool) int64 {
	if emptyBatch {
		if apiUltNSU <= requestedNSU {
			return requestedNSU + 1
		}
		return apiUltNSU
	}
	if apiUltNSU > batchSuccessNSU {
		return apiUltNSU
	}
	return batchSuccessNSU
}

// processDocument handles the decoding, parsing, and saving of a single document.
func (s *SyncService) processDocument(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope) error {
	s.log.Log(ctx, slog.Level(-8), "Processando documento", slog.Int64("nsu", env.NSU))

	// 1. Decode Payload
	payload, err := nfse.DecodePayload(env.PayloadBase64(), nfse.PayloadLimits{
		CompressedBytes:   5 * 1024 * 1024,
		UncompressedBytes: 20 * 1024 * 1024,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao decodificar payload do documento", slog.Int64("nsu", env.NSU), slog.String("erro", err.Error()))
		return fmt.Errorf("decode failed: %w", err)
	}

	// 2. Parse XML
	doc, _, err := nfse.ParseDocumentXML(payload.XML)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	doc.ID = nfse.DocumentID(uuid.NewString())
	doc.RawHash = payload.SHA256

	// 3. Save canonical file
	err = s.fileWriter.Store(doc.RawHash, payload.XML)
	if err != nil {
		return fmt.Errorf("file save failed: %w", err)
	}
	doc.XMLPath = doc.RawHash + ".xml"

	participation := nfse.ClassifyCompanyParticipation(&doc, company.CNPJ)

	// 4. Apply document (Save document + relation)
	if err := s.store.ApplyDocument(ctx, nfse.ApplyDocumentParams{
		Document:      doc,
		Participation: participation,
		CompanyID:     company.ID,
		NSU:           env.NSU,
	}); err != nil {
		return fmt.Errorf("db apply document failed: %w", err)
	}

	return nil
}

// processEvent handles decoding and saving an Event.
func (s *SyncService) processEvent(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope) error {
	s.log.Log(ctx, slog.Level(-8), "Processando evento", slog.Int64("nsu", env.NSU))

	// 1. Decode Payload
	payload, err := nfse.DecodePayload(env.PayloadBase64(), nfse.PayloadLimits{
		CompressedBytes:   5 * 1024 * 1024,
		UncompressedBytes: 20 * 1024 * 1024,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "Falha ao decodificar payload do evento", slog.Int64("nsu", env.NSU), slog.String("erro", err.Error()))
		return fmt.Errorf("decode event failed: %w", err)
	}

	// 2. Parse XML
	ev, _, err := nfse.ParseEventXML(payload.XML)
	if err != nil {
		return fmt.Errorf("parse event failed: %w", err)
	}

	ev.ID = nfse.GenerateID()
	ev.RawHash = payload.SHA256

	err = s.fileWriter.Store(ev.RawHash, payload.XML)
	if err != nil {
		return fmt.Errorf("event file save failed: %w", err)
	}
	ev.RawXMLPath = ev.RawHash + ".xml"

	// 3. Apply event
	if err := s.store.ApplyEvent(ctx, nfse.ApplyEventParams{
		Event:     ev,
		CompanyID: company.ID,
		NSU:       env.NSU,
	}); err != nil {
		return fmt.Errorf("db apply event failed: %w", err)
	}

	return nil
}
