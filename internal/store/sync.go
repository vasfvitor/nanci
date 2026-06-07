package store

import (
	"context"
	"database/sql"
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

	// 1. Upsert Document
	err = q.UpsertDocument(ctx, sqlgen.UpsertDocumentParams{
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
		ParseWarnings:      sql.NullString{String: "[]", Valid: true},
		NfseNumber:         params.Document.NFSeNumber,
		ServiceDescription: params.Document.ServiceDescription,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return err
	}

	// 2. Upsert CompanyDocument Relation
	err = q.UpsertCompanyDocument(ctx, sqlgen.UpsertCompanyDocumentParams{
		RelationID:         string(nfse.GenerateID()),
		CompanyID:          string(params.CompanyID),
		DocumentID:         string(params.Document.ID),
		CompanyRole:        string(params.Participation.CompanyRole),
		VisibilityReason:   string(params.Participation.VisibilityReason),
		FirstSeenNsu:       params.NSU,
		LastSeenNsu:        params.NSU,
		FirstSeenNsuValid:  1,
		LastSeenNsuValid:   1,
		FirstSyncedAt:      now,
		LastSyncedAt:       now,
	})
	if err != nil {
		return err
	}

	// Wait, we need to update company NSU? The plan says AdvanceCheckpoint does that.
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

	var valid int64 = 0
	var eventAt sql.NullString
	if params.Event.EventAtValid {
		valid = 1
		eventAt = sql.NullString{String: params.Event.EventAt.Format(time.RFC3339), Valid: true}
	}

	err = q.InsertEvent(ctx, sqlgen.InsertEventParams{
		ID:                     string(params.Event.ID),
		DocumentID:             sql.NullString{String: string(params.Event.DocumentID), Valid: params.Event.DocumentID != ""},
		ChaveAcesso:            string(params.Event.ChaveAcesso),
		Type:                   string(params.Event.Type),
		EventAt:                eventAt,
		EventAtValid:           valid,
		ReplacementChaveAcesso: params.Event.ReplacementChaveAcesso,
		Description:            params.Event.Description,
		RawXmlPath:             params.Event.RawXMLPath,
		RawHash:                params.Event.RawHash,
		ParseWarnings:          sql.NullString{String: "[]", Valid: true},
		CreatedAt:              now,
	})
	if err != nil {
		return err
	}

	if err := recomputeDocumentStatusTx(ctx, tx, string(params.Event.ChaveAcesso)); err != nil {
		return err
	}

	return tx.Commit()
}

func recomputeDocumentStatusTx(ctx context.Context, tx *sql.Tx, chaveAcesso string) error {
	query := `
		SELECT event_type
		FROM events
		WHERE chave_acesso = ?
	`
	rows, err := tx.QueryContext(ctx, query, chaveAcesso)
	if err != nil {
		return err
	}
	defer rows.Close()

	status := "normal"
	hasCancellation := false
	hasSubstitution := false

	for rows.Next() {
		var eventType string
		if err := rows.Scan(&eventType); err != nil {
			return err
		}
		switch eventType {
		case "substituicao":
			hasSubstitution = true
		case "cancelamento":
			hasCancellation = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	switch {
	case hasSubstitution:
		status = "substituida"
	case hasCancellation:
		status = "cancelada"
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE documents SET status = ?, updated_at = ? WHERE chave_acesso = ?`,
		status,
		time.Now().UTC().Format(time.RFC3339),
		chaveAcesso,
	)
	return err
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
