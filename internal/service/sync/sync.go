package syncservice

import (
	"context"
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
	apiClient  *adn.Client
	fileWriter *files.Writer
}

// NewSyncService creates a new SyncService.
func NewSyncService(store store.Store, apiClient *adn.Client, fileWriter *files.Writer) *SyncService {
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

	defer func() {
		now := time.Now()
		syncRun.FinishedAt = &now
		if syncRun.Status == "running" { // if not marked as completed or failed
			syncRun.Status = "interrupted"
		}
		s.store.UpdateSyncRun(context.Background(), syncRun) // Use background context to ensure it saves
	}()

	currentNSU := company.LastNSU
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
		resp, err := s.apiClient.FetchDocuments(ctx, currentNSU)
		if err != nil {
			syncRun.Status = "failed"
			return fmt.Errorf("failed to fetch documents at NSU %d: %w", currentNSU, err)
		}

		docsInBatch := len(resp.Docs)

		// Report progress
		if progress != nil {
			progress(nfse.ProgressEvent{
				CurrentNSU:  currentNSU,
				MaxNSU:      resp.MaxNSU,
				DocsFound:   totalDocs,
				DocsInBatch: docsInBatch,
				Errors:      totalErrors,
				Message:     fmt.Sprintf("Fetched %d documents. Target MaxNSU: %d", docsInBatch, resp.MaxNSU),
			})
		}

		// Process batch
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
				// We log or handle the error but continue to the next document
				if progress != nil {
					progress(nfse.ProgressEvent{
						CurrentNSU: env.NSU,
						Errors:     totalErrors,
						Message:    fmt.Sprintf("Error processing NSU %d: %v", env.NSU, err),
					})
				}
			} else {
				totalDocs++
			}

			// Update NSU
			if env.NSU > currentNSU {
				currentNSU = env.NSU
			}
		}

		// Update SyncRun stats
		syncRun.ToNSU = currentNSU
		syncRun.DocumentsFound = totalDocs
		syncRun.ErrorsCount = totalErrors

		// Update Company LastNSU in DB
		if currentNSU > company.LastNSU {
			if err := s.store.UpdateLastNSU(ctx, company.ID, currentNSU); err != nil {
				return fmt.Errorf("failed to update company last NSU: %w", err)
			}
			company.LastNSU = currentNSU
		}

		// Stop condition
		if currentNSU >= resp.MaxNSU || docsInBatch == 0 {
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
	doc, err := nfse.ParseXML(rawXML, company.CNPJRoot)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	doc.ID = uuid.NewString()
	doc.CompanyID = company.ID
	doc.NSU = env.NSU
	doc.RawHash = hashHex

	// 3. Save file
	relPath, err := s.fileWriter.SaveXML(company.CNPJ, doc.Competence, doc.Direction, doc.ChaveAcesso, rawXML)
	if err != nil {
		return fmt.Errorf("file save failed: %w", err)
	}
	doc.XMLPath = relPath

	// 4. Save to DB
	if err := s.store.SaveDocument(ctx, doc); err != nil {
		return fmt.Errorf("db save failed: %w", err)
	}

	return nil
}

// processEvent handles decoding and saving an Event.
func (s *SyncService) processEvent(ctx context.Context, company *nfse.Company, env adn.DocumentEnvelope) error {
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
	event.CompanyID = company.ID
	event.RawHash = hashHex

	// 3. Save to DB
	if err := s.store.SaveEvent(ctx, event); err != nil {
		return fmt.Errorf("db save event failed: %w", err)
	}

	return nil
}
