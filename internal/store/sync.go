package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

type SyncRepository struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewSyncRepository(db *sql.DB) *SyncRepository {
	return &SyncRepository{
		db:      db,
		queries: sqlgen.New(db),
	}
}

func (r *SyncRepository) StartRun(ctx context.Context, params nfse.StartRunParams) (nfse.SyncRun, error) {
	now := time.Now().UTC()
	runID := nfse.SyncRunID(nfse.GenerateID())

	err := r.queries.CreateSyncRun(ctx, sqlgen.CreateSyncRunParams{
		ID:                string(runID),
		CompanyID:         string(params.CompanyID),
		CredentialID:      string(params.CredentialID),
		CredentialCnpj:    params.CredentialCNPJ,
		ConsultationCnpj:  params.ConsultationCNPJ,
		ConsultationBasis: string(params.ConsultationBasis),
		StartedAt:         now.Format(time.RFC3339),
		FromNsu:           params.FromNSU,
		ToNsu:             params.ToNSU,
		Status:            string(nfse.SyncStatusRunning),
	})
	if err != nil {
		return nfse.SyncRun{}, err
	}

	return nfse.SyncRun{
		ID:                runID,
		CompanyID:         params.CompanyID,
		CredentialID:      params.CredentialID,
		CredentialCNPJ:    params.CredentialCNPJ,
		ConsultationCNPJ:  params.ConsultationCNPJ,
		ConsultationBasis: params.ConsultationBasis,
		StartedAt:         now,
		FromNSU:           params.FromNSU,
		ToNSU:             params.ToNSU,
		Status:            nfse.SyncStatusRunning,
	}, nil
}

func (r *SyncRepository) ApplyDocument(ctx context.Context, params nfse.ApplyDocumentParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := r.queries.WithTx(tx)

	now := time.Now().UTC().Format(time.RFC3339)

	parseWarnings, err := json.Marshal(params.Document.ParseWarnings)
	if err != nil {
		return err
	}

	canonicalDocumentID, err := q.UpsertDocument(ctx, sqlgen.UpsertDocumentParams{
		ID:                 string(params.Document.ID),
		ChaveAcesso:        string(params.Document.ChaveAcesso),
		IssueDate:          params.Document.IssueDate.Format(time.RFC3339),
		Competence:         params.Document.Competence,
		PrestadorCnpj:      params.Document.PrestadorCNPJ,
		PrestadorName:      params.Document.PrestadorName,
		TomadorCnpj:        params.Document.TomadorCNPJ,
		TomadorName:        params.Document.TomadorName,
		IntermediarioCnpj:  params.Document.IntermediarioCNPJ,
		IntermediarioName:  params.Document.IntermediarioName,
		ServiceValue:       int64(params.Document.ServiceValue),
		IssValue:           int64(params.Document.ISSValue),
		IrrfValue:          int64(params.Document.IRRFValue),
		InssValue:          int64(params.Document.INSSValue),
		PisValue:           int64(params.Document.PISValue),
		CofinsValue:        int64(params.Document.COFINSValue),
		CsllValue:          int64(params.Document.CSLLValue),
		TotalRetentions:    int64(params.Document.TotalRetentions),
		Status:             string(params.Document.Status),
		LayoutVersion:      params.Document.LayoutVersion,
		XmlPath:            params.Document.XMLPath,
		RawHash:            params.Document.RawHash,
		ParseWarnings:      sql.NullString{String: string(parseWarnings), Valid: true},
		NfseNumber:         params.Document.NFSeNumber,
		ServiceDescription: params.Document.ServiceDescription,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return err
	}

	err = q.UpsertCompanyDocument(ctx, sqlgen.UpsertCompanyDocumentParams{
		RelationID:        nfse.GenerateID(),
		CompanyID:         string(params.CompanyID),
		DocumentID:        canonicalDocumentID,
		CompanyRole:       string(params.Participation.CompanyRole),
		VisibilityReason:  string(params.Participation.VisibilityReason),
		FirstSeenNsu:      params.NSU,
		LastSeenNsu:       params.NSU,
		FirstSeenNsuValid: 1,
		LastSeenNsuValid:  1,
		FirstSyncedAt:     now,
		LastSyncedAt:      now,
	})
	if err != nil {
		return err
	}

	if err := q.LinkEventsToDocument(ctx, sqlgen.LinkEventsToDocumentParams{
		DocumentID:  sql.NullString{String: canonicalDocumentID, Valid: true},
		ChaveAcesso: string(params.Document.ChaveAcesso),
	}); err != nil {
		return err
	}

	if err := recomputeDocumentStatus(ctx, q, string(params.Document.ChaveAcesso), now); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SyncRepository) ApplyEvent(ctx context.Context, params nfse.ApplyEventParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := r.queries.WithTx(tx)

	now := time.Now().UTC().Format(time.RFC3339)

	documentID, err := q.GetDocumentIDByAccessKey(ctx, string(params.Event.ChaveAcesso))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var valid int64
	var eventAt sql.NullString
	if params.Event.EventAtValid {
		valid = 1
		eventAt = sql.NullString{String: params.Event.EventAt.Format(time.RFC3339), Valid: true}
	}

	parseWarnings, err := json.Marshal(params.Event.ParseWarnings)
	if err != nil {
		return err
	}

	err = q.InsertEvent(ctx, sqlgen.InsertEventParams{
		ID:                     params.Event.ID,
		DocumentID:             sql.NullString{String: documentID, Valid: documentID != ""},
		ChaveAcesso:            string(params.Event.ChaveAcesso),
		Type:                   string(params.Event.Type),
		EventAt:                eventAt,
		EventAtValid:           valid,
		ReplacementChaveAcesso: params.Event.ReplacementChaveAcesso,
		Description:            params.Event.Description,
		RawXmlPath:             params.Event.RawXMLPath,
		RawHash:                params.Event.RawHash,
		ParseWarnings:          sql.NullString{String: string(parseWarnings), Valid: true},
		CreatedAt:              now,
	})
	if err != nil {
		return err
	}

	if err := recomputeDocumentStatus(ctx, q, string(params.Event.ChaveAcesso), now); err != nil {
		return err
	}

	return tx.Commit()
}

func recomputeDocumentStatus(ctx context.Context, q *sqlgen.Queries, chaveAcesso, updatedAt string) error {
	eventTypes, err := q.ListEventTypesByAccessKey(ctx, chaveAcesso)
	if err != nil {
		return err
	}

	status := nfse.DocumentStatusNormal
	hasCancellation := false
	hasSubstitution := false

	for _, eventType := range eventTypes {
		switch eventType {
		case "substituicao":
			hasSubstitution = true
		case "cancelamento":
			hasCancellation = true
		}
	}

	switch {
	case hasSubstitution:
		status = nfse.DocumentStatusSubstituida
	case hasCancellation:
		status = nfse.DocumentStatusCancelada
	}

	return q.UpdateDocumentStatusByAccessKey(ctx, sqlgen.UpdateDocumentStatusByAccessKeyParams{
		Status:      string(status),
		UpdatedAt:   updatedAt,
		ChaveAcesso: chaveAcesso,
	})
}

func (r *SyncRepository) AdvanceCheckpoint(ctx context.Context, params nfse.AdvanceCheckpointParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := r.queries.WithTx(tx)

	now := time.Now().UTC().Format(time.RFC3339)

	err = q.UpdateCompanyNSU(ctx, sqlgen.UpdateCompanyNSUParams{
		LastNsu:   params.LastNSU,
		UpdatedAt: now,
		ID:        string(params.CompanyID),
	})
	if err != nil {
		return err
	}

	err = q.UpdateSyncRunProgress(ctx, sqlgen.UpdateSyncRunProgressParams{
		ToNsu:          params.LastNSU,
		DocumentsFound: 0, // Ignored logic for now
		ErrorsCount:    0,
		ID:             string(params.RunID),
	})
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SyncRepository) FinishRun(ctx context.Context, params nfse.FinishRunParams) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return r.queries.FinishSyncRun(ctx, sqlgen.FinishSyncRunParams{
		FinishedAt: sql.NullString{String: now, Valid: true},
		Status:     string(params.Status),
		ID:         string(params.RunID),
	})
}
