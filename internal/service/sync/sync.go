package syncservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/adn"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

// SyncService orchestrates the synchronization of documents from the ADN API.
type SyncService struct {
	store      store.Store
	apiClient  documentFetcher
	fileWriter *files.Writer
}

type documentFetcher interface {
	FetchDocuments(ctx context.Context, lastNSU int64) (*adn.DocumentResponse, error)
}

// NewSyncService creates a new SyncService.
func NewSyncService(store store.Store, apiClient documentFetcher, fileWriter *files.Writer) *SyncService {
	return &SyncService{
		store:      store,
		apiClient:  apiClient,
		fileWriter: fileWriter,
	}
}

// Sync starts the synchronization process for a specific company.
func (s *SyncService) Sync(ctx context.Context, company *nfse.Company, progress nfse.ProgressFunc) error {
	// Create SyncRun record
	syncRun := &nfse.SyncRun{
		ID:        uuid.NewString(),
		CompanyID: company.ID,
		StartedAt: time.Now(),
		FromNSU:   company.LastNSU,
		Status:    "running",
	}

	if err := s.store.CreateSyncRun(ctx, syncRun); err != nil {
		return fmt.Errorf("failed to create sync run: %w", err)
	}

	defer func() { //nolint:contextcheck
		now := time.Now()
		syncRun.FinishedAt = &now
		if syncRun.Status == "running" { // if not marked as completed or failed
			syncRun.Status = "interrupted"
		}
		_ = s.store.UpdateSyncRun(context.Background(), syncRun) // Use background context to ensure it saves
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
		resp, err := s.apiClient.FetchDocuments(ctx, requestedNSU)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				syncRun.Status = "interrupted"
				syncRun.ToNSU = committedNSU
				syncRun.DocumentsFound = totalDocs
				syncRun.ErrorsCount = totalErrors
				return err
			}
			syncRun.Status = "failed"
			syncRun.ToNSU = committedNSU
			syncRun.DocumentsFound = totalDocs
			syncRun.ErrorsCount = totalErrors
			return fmt.Errorf("failed to fetch documents at NSU %d: %w", requestedNSU, err)
		}

		if resp.UltNSU < requestedNSU {
			syncRun.Status = "failed"
			syncRun.ToNSU = committedNSU
			syncRun.DocumentsFound = totalDocs
			syncRun.ErrorsCount = totalErrors
			return fmt.Errorf("invalid ADN response: ultNSU %d is behind requested NSU %d", resp.UltNSU, requestedNSU)
		}

		docsInBatch := len(resp.Docs)

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
					if err := s.store.UpdateLastNSU(ctx, company.ID, batchSuccessNSU); err != nil {
						return fmt.Errorf("failed to update company last NSU after item error: %w", err)
					}
					company.LastNSU = batchSuccessNSU
				}

				committedNSU = batchSuccessNSU
				syncRun.ToNSU = committedNSU
				syncRun.DocumentsFound = totalDocs
				syncRun.ErrorsCount = totalErrors
				syncRun.Status = "failed"
				return fmt.Errorf("failed to process NSU %d: %w", env.NSU, err)
			}

			totalDocs++
			if env.NSU > batchSuccessNSU {
				batchSuccessNSU = env.NSU
			}
		}

		committedNSU = resp.UltNSU
		if committedNSU > company.LastNSU {
			if err := s.store.UpdateLastNSU(ctx, company.ID, committedNSU); err != nil {
				return fmt.Errorf("failed to update company last NSU: %w", err)
			}
			company.LastNSU = committedNSU
		}

		// Update SyncRun stats
		syncRun.ToNSU = committedNSU
		syncRun.DocumentsFound = totalDocs
		syncRun.ErrorsCount = totalErrors

		// Stop condition
		if committedNSU >= resp.MaxNSU {
			break
		}
	}

	syncRun.Status = "completed"
	return nil
}

// processDocument handles the decoding, parsing, and saving of a single document.
func (s *SyncService) processDocument(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope) error {
	// 1. Decode Payload
	rawXML, hashHex, err := nfse.DecodeXMLPayload(env.XMLGZipBase64)
	if err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	// 2. Parse XML
	doc, err := nfse.ParseXML(rawXML)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	doc.ID = uuid.NewString()
	doc.RawHash = hashHex

	// 3. Save canonical file
	relPath, err := s.fileWriter.SaveXML(doc.Competence, doc.ChaveAcesso, rawXML)
	if err != nil {
		return fmt.Errorf("file save failed: %w", err)
	}
	doc.XMLPath = relPath

	// 4. Save canonical document
	if err := s.store.UpsertDocument(ctx, doc); err != nil {
		return fmt.Errorf("db save failed: %w", err)
	}

	canonicalDoc, err := s.store.GetDocumentByChave(ctx, doc.ChaveAcesso)
	if err != nil {
		return fmt.Errorf("db fetch canonical failed: %w", err)
	}
	if canonicalDoc == nil {
		return fmt.Errorf("canonical document missing after upsert: %s", doc.ChaveAcesso)
	}

	participation := nfse.ClassifyCompanyParticipation(canonicalDoc, company.CNPJ)
	companyDoc := &nfse.CompanyDocument{
		Document:          *canonicalDoc,
		RelationID:        uuid.NewString(),
		CompanyID:         company.ID,
		DocumentID:        canonicalDoc.ID,
		CompanyRole:       participation.CompanyRole,
		VisibilityReason:  participation.VisibilityReason,
		FirstSeenNSU:      env.NSU,
		LastSeenNSU:       env.NSU,
		FirstSeenNSUValid: true,
		LastSeenNSUValid:  true,
	}

	// 5. Save company relation
	if err := s.store.UpsertCompanyDocument(ctx, companyDoc); err != nil {
		return fmt.Errorf("db save company relation failed: %w", err)
	}

	return nil
}

// processEvent handles decoding and saving an Event.
func (s *SyncService) processEvent(ctx context.Context, _ *nfse.Company, env adn.DocumentEnvelope) error {
	// 1. Decode Payload
	rawXML, hashHex, err := nfse.DecodeXMLPayload(env.XMLGZipBase64)
	if err != nil {
		return fmt.Errorf("decode event failed: %w", err)
	}

	// 2. Parse XML
	event, err := nfse.ParseEvent(rawXML)
	if err != nil {
		return fmt.Errorf("parse event failed: %w", err)
	}

	event.ID = uuid.NewString()
	event.RawHash = hashHex
	relPath, err := s.fileWriter.SaveEventXML(event.ChaveAcesso, hashHex, rawXML)
	if err != nil {
		return fmt.Errorf("event file save failed: %w", err)
	}
	event.RawXMLPath = relPath

	// 3. Save to DB
	if err := s.store.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("db save event failed: %w", err)
	}

	return nil
}
