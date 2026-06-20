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

func (r *SyncRepository) GetOrCreateState(ctx context.Context, params nfse.GetOrCreateSyncStateParams) (*nfse.SyncState, error) {
	state, err := r.getSyncState(ctx, params.CompanyID, params.Environment, params.ConsultationCNPJ)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	lastFoundNSU, lastFoundValid, err := r.legacyLastFoundNSU(ctx, params.CompanyID)
	if err != nil {
		return nil, err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO sync_state (
			company_id, environment, consultation_cnpj,
			last_checked_nsu, last_found_nsu, last_empty_streak,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`,
		string(params.CompanyID),
		string(params.Environment),
		params.ConsultationCNPJ,
		params.LegacyLastNSU,
		nullInt64(lastFoundNSU, lastFoundValid),
		now,
		now,
	)
	if err != nil {
		state, retryErr := r.getSyncState(ctx, params.CompanyID, params.Environment, params.ConsultationCNPJ)
		if retryErr == nil {
			return state, nil
		}
		return nil, err
	}

	return r.getSyncState(ctx, params.CompanyID, params.Environment, params.ConsultationCNPJ)
}

func (r *SyncRepository) StartRun(ctx context.Context, params nfse.StartRunParams) (nfse.SyncRun, error) {
	now := time.Now().UTC()
	runID := nfse.SyncRunID(nfse.GenerateID())

	_, _ = r.db.ExecContext(
		ctx,
		"UPDATE sync_runs SET status = 'interrupted', stop_reason = 'context_canceled', finished_at = ? WHERE company_id = ? AND status = 'running'",
		now.Format(time.RFC3339),
		string(params.CompanyID),
	)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_runs (
			id, company_id, credential_id, environment, credential_cnpj, consultation_cnpj,
			consultation_basis, mode, started_at, from_nsu, to_nsu,
			checked_count, documents_found, empty_count, consecutive_empty_count, errors_count, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, 0, ?)
	`,
		string(runID),
		string(params.CompanyID),
		string(params.CredentialID),
		string(params.Environment),
		params.CredentialCNPJ,
		params.ConsultationCNPJ,
		string(params.ConsultationBasis),
		string(params.Mode),
		now.Format(time.RFC3339),
		params.FromNSU,
		params.ToNSU,
		string(nfse.SyncStatusRunning),
	)
	if err != nil {
		return nfse.SyncRun{}, err
	}

	return nfse.SyncRun{
		ID:                runID,
		CompanyID:         params.CompanyID,
		CredentialID:      params.CredentialID,
		Environment:       params.Environment,
		CredentialCNPJ:    params.CredentialCNPJ,
		ConsultationCNPJ:  params.ConsultationCNPJ,
		ConsultationBasis: params.ConsultationBasis,
		Mode:              params.Mode,
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
	defer func() { _ = tx.Rollback() }()

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
	defer func() { _ = tx.Rollback() }()

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

func (r *SyncRepository) PersistProgress(ctx context.Context, params nfse.PersistSyncProgressParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	lastSuccessAt := sql.NullString{}
	lastErrorAt := sql.NullString{}
	lastErrorCode := sql.NullString{}
	lastErrorMessage := sql.NullString{}

	if params.MarkSuccess {
		lastSuccessAt = sql.NullString{String: now, Valid: true}
	}
	if params.ErrorCode != "" || params.ErrorMessage != "" {
		lastErrorAt = sql.NullString{String: now, Valid: true}
		lastErrorCode = sql.NullString{String: params.ErrorCode, Valid: params.ErrorCode != ""}
		lastErrorMessage = sql.NullString{String: params.ErrorMessage, Valid: params.ErrorMessage != ""}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE sync_state
		SET
			last_checked_nsu = ?,
			last_found_nsu = CASE
				WHEN ? IS NOT NULL THEN ?
				ELSE last_found_nsu
			END,
			last_empty_streak = ?,
			last_success_at = CASE
				WHEN ? IS NOT NULL THEN ?
				ELSE last_success_at
			END,
			last_error_at = CASE
				WHEN ? IS NOT NULL THEN ?
				ELSE last_error_at
			END,
			last_error_code = CASE
				WHEN ? IS NOT NULL THEN ?
				WHEN ? IS NOT NULL THEN NULL
				ELSE last_error_code
			END,
			last_error_message = CASE
				WHEN ? IS NOT NULL THEN ?
				WHEN ? IS NOT NULL THEN NULL
				ELSE last_error_message
			END,
			updated_at = ?
		WHERE company_id = ? AND environment = ? AND consultation_cnpj = ?
	`,
		params.LastProcessedNSU,
		nullInt64(params.LastFoundNSU, params.LastFoundNSUValid),
		nullInt64(params.LastFoundNSU, params.LastFoundNSUValid),
		params.LastEmptyStreak,
		lastSuccessAt,
		lastSuccessAt,
		lastErrorAt,
		lastErrorAt,
		lastErrorCode,
		lastErrorCode,
		lastSuccessAt,
		lastErrorMessage,
		lastErrorMessage,
		lastSuccessAt,
		now,
		string(params.CompanyID),
		string(params.Environment),
		params.ConsultationCNPJ,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE companies
		SET last_nsu = CASE
			WHEN last_nsu < ? THEN ?
			ELSE last_nsu
		END,
		updated_at = ?
		WHERE id = ?
	`,
		params.LastProcessedNSU,
		params.LastProcessedNSU,
		now,
		string(params.CompanyID),
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE sync_runs
		SET
			to_nsu = ?,
			checked_count = ?,
			documents_found = ?,
			empty_count = ?,
			consecutive_empty_count = ?,
			errors_count = ?,
			last_found_nsu = CASE
				WHEN ? IS NOT NULL THEN ?
				ELSE last_found_nsu
			END
		WHERE id = ?
	`,
		params.LastProcessedNSU,
		params.CheckedCount,
		params.DocumentsFound,
		params.EmptyCount,
		params.ConsecutiveEmptyCount,
		params.ErrorsCount,
		nullInt64(params.LastFoundNSU, params.LastFoundNSUValid),
		nullInt64(params.LastFoundNSU, params.LastFoundNSUValid),
		string(params.RunID),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SyncRepository) FinishRun(ctx context.Context, params nfse.FinishRunParams) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE sync_runs
		SET
			finished_at = ?,
			status = ?,
			stop_reason = ?,
			checked_count = ?,
			documents_found = ?,
			empty_count = ?,
			consecutive_empty_count = ?,
			errors_count = ?,
			last_found_nsu = ?
		WHERE id = ?
	`,
		now,
		string(params.Status),
		nullString(string(params.StopReason)),
		params.CheckedCount,
		params.DocumentsFound,
		params.EmptyCount,
		params.ConsecutiveEmptyCount,
		params.ErrorsCount,
		nullInt64(params.LastFoundNSU, params.LastFoundNSUValid),
		string(params.RunID),
	)
	return err
}

func (r *SyncRepository) LatestSyncSnapshot(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, consultationCNPJ string) (nfse.SyncSnapshot, error) {
	var snapshot nfse.SyncSnapshot

	state, err := r.getSyncState(ctx, companyID, environment, consultationCNPJ)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return snapshot, err
	}
	if err == nil {
		snapshot.State = state
	}

	run, err := r.latestRun(ctx, companyID, environment, consultationCNPJ)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return snapshot, err
	}
	if err == nil {
		snapshot.Run = run
	}

	return snapshot, nil
}

func (r *SyncRepository) ResetSyncState(ctx context.Context, params nfse.ResetSyncStateParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_state WHERE company_id = ?`, string(params.CompanyID)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE companies SET last_nsu = 0, updated_at = ? WHERE id = ?`, now, string(params.CompanyID)); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SyncRepository) getSyncState(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, consultationCNPJ string) (*nfse.SyncState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			company_id, environment, consultation_cnpj,
			last_checked_nsu, last_found_nsu, last_empty_streak,
			last_success_at, last_error_at, last_error_code, last_error_message,
			created_at, updated_at
		FROM sync_state
		WHERE company_id = ? AND environment = ? AND consultation_cnpj = ?
	`,
		string(companyID),
		string(environment),
		consultationCNPJ,
	)

	var state nfse.SyncState
	var lastFound sql.NullInt64
	var lastSuccessAt, lastErrorAt, lastErrorCode, lastErrorMessage sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(
		&state.CompanyID,
		&state.Environment,
		&state.ConsultationCNPJ,
		&state.LastProcessedNSU,
		&lastFound,
		&state.LastEmptyStreak,
		&lastSuccessAt,
		&lastErrorAt,
		&lastErrorCode,
		&lastErrorMessage,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	if lastFound.Valid {
		state.LastFoundNSU = lastFound.Int64
		state.LastFoundNSUValid = true
	}
	state.LastSuccessAt = parseNullableTime(lastSuccessAt)
	state.LastErrorAt = parseNullableTime(lastErrorAt)
	state.LastErrorCode = lastErrorCode.String
	state.LastErrorMessage = lastErrorMessage.String
	state.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	state.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &state, nil
}

func (r *SyncRepository) latestRun(ctx context.Context, companyID nfse.CompanyID, environment nfse.Environment, consultationCNPJ string) (*nfse.SyncRun, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, company_id, credential_id, environment, credential_cnpj, consultation_cnpj,
			consultation_basis, mode, started_at, finished_at, from_nsu, to_nsu,
			checked_count, documents_found, empty_count, consecutive_empty_count,
			errors_count, last_found_nsu, status, stop_reason
		FROM sync_runs
		WHERE company_id = ? AND environment = ? AND consultation_cnpj = ?
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`,
		string(companyID),
		string(environment),
		consultationCNPJ,
	)

	var run nfse.SyncRun
	var finishedAt sql.NullString
	var lastFound sql.NullInt64
	var stopReason sql.NullString
	var consultationBasis, mode, status string
	var startedAt string
	if err := row.Scan(
		&run.ID,
		&run.CompanyID,
		&run.CredentialID,
		&run.Environment,
		&run.CredentialCNPJ,
		&run.ConsultationCNPJ,
		&consultationBasis,
		&mode,
		&startedAt,
		&finishedAt,
		&run.FromNSU,
		&run.ToNSU,
		&run.CheckedCount,
		&run.DocumentsFound,
		&run.EmptyCount,
		&run.ConsecutiveEmptyCount,
		&run.ErrorsCount,
		&lastFound,
		&status,
		&stopReason,
	); err != nil {
		return nil, err
	}

	run.ConsultationBasis = nfse.ConsultationBasis(consultationBasis)
	run.Mode = nfse.SyncMode(mode)
	run.Status = nfse.SyncStatus(status)
	if stopReason.Valid {
		run.StopReason = nfse.SyncStopReason(stopReason.String)
	}
	run.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	run.FinishedAt = parseNullableTime(finishedAt)
	if lastFound.Valid {
		run.LastFoundNSU = lastFound.Int64
		run.LastFoundNSUValid = true
	}
	return &run, nil
}

func (r *SyncRepository) legacyLastFoundNSU(ctx context.Context, companyID nfse.CompanyID) (int64, bool, error) {
	var nsu sql.NullInt64
	if err := r.db.QueryRowContext(ctx, `
		SELECT MAX(last_seen_nsu)
		FROM company_documents
		WHERE company_id = ? AND last_seen_nsu_valid = 1
	`, string(companyID)).Scan(&nsu); err != nil {
		return 0, false, err
	}
	if !nsu.Valid {
		return 0, false, nil
	}
	return nsu.Int64, true, nil
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

func nullInt64(val int64, valid bool) sql.NullInt64 {
	return sql.NullInt64{Int64: val, Valid: valid}
}

func nullString(val string) sql.NullString {
	return sql.NullString{String: val, Valid: val != ""}
}
