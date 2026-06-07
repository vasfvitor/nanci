package syncservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"log/slog"

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
		slog.String("cnpj", string(company.CNPJ)),
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

	defer func() {
		if syncRun.Status == "running" { // if not marked as completed or failed
			_ = s.finishRun(ctx, nfse.FinishRunParams{
				RunID:  syncRun.ID,
				Status: "interrupted",
			})
		}
	}()

	committedNSU := company.LastNSU
	totalDocs := 0
	totalErrors := 0

	for {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			syncRun.Status = "interrupted"
			return ctx.Err()
		default:
		}

		// Fetch documents batch
		requestedNSU := committedNSU
		s.log.InfoContext(ctx, "Buscando lote de documentos", slog.Int64("requested_nsu", requestedNSU))

		resp, err := s.apiClient.FetchDocuments(ctx, adn.DistributionRequest{
			LastNSU:          requestedNSU,
			ConsultationCNPJ: company.CNPJ,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				syncRun.Status = "interrupted"
				_ = s.finishRun(ctx, nfse.FinishRunParams{
					RunID:  syncRun.ID,
					Status: "interrupted",
				})
				return err
			}
			syncRun.Status = "failed"
			_ = s.finishRun(ctx, nfse.FinishRunParams{
				RunID:    syncRun.ID,
				Status:   "failed",
				ErrorMsg: err.Error(),
			})
			return fmt.Errorf("failed to fetch documents at NSU %d: %w", requestedNSU, err)
		}

		if resp.UltNSU < requestedNSU {
			syncRun.Status = "failed"
			_ = s.finishRun(ctx, nfse.FinishRunParams{
				RunID:    syncRun.ID,
				Status:   "failed",
				ErrorMsg: fmt.Sprintf("invalid ADN response: ultNSU %d is behind requested NSU %d", resp.UltNSU, requestedNSU),
			})
			return fmt.Errorf("invalid ADN response: ultNSU %d is behind requested NSU %d", resp.UltNSU, requestedNSU)
		}

		docsInBatch := len(resp.Docs)
		s.log.DebugContext(ctx, "Lote recebido",
			slog.Int("docs_in_batch", docsInBatch),
			slog.Int64("ult_nsu", resp.UltNSU),
			slog.Int64("max_nsu", resp.MaxNSU))

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
			// Check schema to decide if it's a document or an event
			if strings.Contains(env.Schema, "procEvento") {
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

				if batchSuccessNSU > company.LastNSU {
					if err := s.store.AdvanceCheckpoint(ctx, nfse.AdvanceCheckpointParams{
						CompanyID: company.ID,
						RunID:     syncRun.ID,
						LastNSU:   batchSuccessNSU,
					}); err != nil {
						return fmt.Errorf("failed to update company last NSU after item error: %w", err)
					}
					company.LastNSU = batchSuccessNSU
				}

				syncRun.Status = "failed"
				_ = s.finishRun(ctx, nfse.FinishRunParams{
					RunID:    syncRun.ID,
					Status:   "failed",
					ErrorMsg: err.Error(),
				})
				return fmt.Errorf("failed to process NSU %d: %w", env.NSU, err)
			}

			totalDocs++
			if env.NSU > batchSuccessNSU {
				batchSuccessNSU = env.NSU
			}
		}

		committedNSU = resp.UltNSU
		if committedNSU > company.LastNSU {
			if err := s.store.AdvanceCheckpoint(ctx, nfse.AdvanceCheckpointParams{
				CompanyID: company.ID,
				RunID:     syncRun.ID,
				LastNSU:   committedNSU,
			}); err != nil {
				return fmt.Errorf("failed to update company last NSU: %w", err)
			}
			company.LastNSU = committedNSU
		}

		// Stop condition
		if committedNSU >= resp.MaxNSU {
			break
		}
	}

	syncRun.Status = "completed"
	_ = s.finishRun(ctx, nfse.FinishRunParams{
		RunID:  syncRun.ID,
		Status: "completed",
	})
	return nil
}

func (s *SyncService) finishRun(ctx context.Context, params nfse.FinishRunParams) error {
	finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.store.FinishRun(finishCtx, params)
}

// processDocument handles the decoding, parsing, and saving of a single document.
func (s *SyncService) processDocument(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope) error {
	s.log.Log(ctx, slog.Level(-8), "Processando documento", slog.Int64("nsu", env.NSU))
	
	// 1. Decode Payload
	payload, err := nfse.DecodePayload(env.XMLGZipBase64, nfse.PayloadLimits{
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
	payload, err := nfse.DecodePayload(env.XMLGZipBase64, nfse.PayloadLimits{
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
