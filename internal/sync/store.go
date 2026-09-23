package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

type Store struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		queries: sqlgen.New(db),
	}
}

// executor abstracts sql.Tx and sql.DB for the private helpers.
type executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *Store) GetOrCreateState(ctx context.Context, params nfse.GetOrCreateSyncStateParams) (*nfse.SyncState, error) {
	state, err := r.getSyncState(ctx, params.CompanyID, params.Source, params.Environment, params.ConsultationCNPJ)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO sync_state (
			company_id, source, environment, consultation_cnpj,
			last_checked_nsu, last_found_nsu, last_empty_streak,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, 0, NULL, 0, ?, ?)
	`,
		string(params.CompanyID),
		string(params.Source),
		string(params.Environment),
		params.ConsultationCNPJ,
		now,
		now,
	)
	if err != nil {
		state, retryErr := r.getSyncState(ctx, params.CompanyID, params.Source, params.Environment, params.ConsultationCNPJ)
		if retryErr == nil {
			return state, nil
		}
		return nil, err
	}

	return r.getSyncState(ctx, params.CompanyID, params.Source, params.Environment, params.ConsultationCNPJ)
}

func (r *Store) StartRun(ctx context.Context, params nfse.StartRunParams) (nfse.SyncRun, error) {
	now := time.Now().UTC()
	runID := nfse.SyncRunID(nfse.GenerateID())

	_, _ = r.db.ExecContext(
		ctx,
		"UPDATE sync_runs SET status = 'interrupted', stop_reason = 'context_canceled', finished_at = ? WHERE company_id = ? AND source = ? AND status = 'running'",
		now.Format(time.RFC3339),
		string(params.CompanyID),
		string(params.Source),
	)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sync_runs (
			id, company_id, source, credential_id, environment, credential_cnpj, consultation_cnpj,
			consultation_basis, mode, started_at, from_nsu, to_nsu,
			checked_count, documents_found, empty_count, consecutive_empty_count, errors_count, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, 0, ?)
	`,
		string(runID),
		string(params.CompanyID),
		string(params.Source),
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
		Source:            params.Source,
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

func (r *Store) doApplyDocument(ctx context.Context, tx executor, q *sqlgen.Queries, params nfse.ApplyDocumentParams) (nfse.ApplyOutcome, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	parseWarnings, err := json.Marshal(params.Document.ParseWarnings)
	if err != nil {
		return nfse.ApplyOutcome{}, err
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
		return nfse.ApplyOutcome{}, err
	}

	inserted, err := companyDocumentMissing(ctx, tx, string(params.CompanyID), canonicalDocumentID)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	err = q.UpsertCompanyDocument(ctx, sqlgen.UpsertCompanyDocumentParams{
		RelationID:       nfse.GenerateID(),
		CompanyID:        string(params.CompanyID),
		DocumentID:       canonicalDocumentID,
		CompanyRole:      string(params.Participation.CompanyRole),
		VisibilityReason: string(params.Participation.VisibilityReason),
		FirstSeenNsu:     sql.NullInt64{Int64: params.NSU, Valid: true},
		LastSeenNsu:      sql.NullInt64{Int64: params.NSU, Valid: true},
		FirstSyncedAt:    now,
		LastSyncedAt:     now,
	})
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	if err := q.LinkEventsToDocument(ctx, sqlgen.LinkEventsToDocumentParams{
		DocumentID:  sql.NullString{String: canonicalDocumentID, Valid: true},
		ChaveAcesso: string(params.Document.ChaveAcesso),
	}); err != nil {
		return nfse.ApplyOutcome{}, err
	}

	if err := recomputeDocumentStatus(ctx, q, string(params.Document.ChaveAcesso), now); err != nil {
		return nfse.ApplyOutcome{}, err
	}

	return nfse.ApplyOutcome{Inserted: inserted}, nil
}

func (r *Store) doApplyEvent(ctx context.Context, tx executor, q *sqlgen.Queries, params nfse.ApplyEventParams) (nfse.ApplyOutcome, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	documentID, err := q.GetDocumentIDByAccessKey(ctx, string(params.Event.ChaveAcesso))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nfse.ApplyOutcome{}, err
	}

	var eventAt sql.NullString
	if params.Event.EventAt != nil {
		eventAt = sql.NullString{String: params.Event.EventAt.Format(time.RFC3339), Valid: true}
	}

	parseWarnings, err := json.Marshal(params.Event.ParseWarnings)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	inserted, err := eventHashMissing(ctx, tx, params.Event.RawHash)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	err = q.InsertEvent(ctx, sqlgen.InsertEventParams{
		ID:                     params.Event.ID,
		DocumentID:             sql.NullString{String: documentID, Valid: documentID != ""},
		ChaveAcesso:            string(params.Event.ChaveAcesso),
		Type:                   string(params.Event.Type),
		EventAt:                eventAt,
		ReplacementChaveAcesso: params.Event.ReplacementChaveAcesso,
		Description:            params.Event.Description,
		RawXmlPath:             params.Event.RawXMLPath,
		RawHash:                params.Event.RawHash,
		ParseWarnings:          sql.NullString{String: string(parseWarnings), Valid: true},
		CreatedAt:              now,
	})
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	if err := recomputeDocumentStatus(ctx, q, string(params.Event.ChaveAcesso), now); err != nil {
		return nfse.ApplyOutcome{}, err
	}

	return nfse.ApplyOutcome{Inserted: inserted}, nil
}

func (r *Store) doPersistProgress(ctx context.Context, tx executor, params nfse.PersistSyncProgressParams) error {
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

	_, err := tx.ExecContext(ctx, `
		UPDATE sync_state
		SET
			last_checked_nsu = ?,
			last_found_nsu = CASE
				WHEN ? IS NOT NULL THEN ?
				ELSE last_found_nsu
			END,
			max_nsu = CASE
				WHEN ? IS NOT NULL THEN ?
				ELSE max_nsu
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
		WHERE company_id = ? AND source = ? AND environment = ? AND consultation_cnpj = ?
	`,
		params.LastProcessedNSU,
		store.NullInt64FromPtr(params.LastFoundNSU),
		store.NullInt64FromPtr(params.LastFoundNSU),
		store.NullInt64FromPtr(params.MaxNSU),
		store.NullInt64FromPtr(params.MaxNSU),
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
		string(params.Source),
		string(params.Environment),
		params.ConsultationCNPJ,
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
		store.NullInt64FromPtr(params.LastFoundNSU),
		store.NullInt64FromPtr(params.LastFoundNSU),
		string(params.RunID),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Store) ApplyDocument(ctx context.Context, params nfse.ApplyDocumentParams) (nfse.ApplyOutcome, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}
	defer func() { _ = tx.Rollback() }()

	outcome, err := r.doApplyDocument(ctx, tx, r.queries.WithTx(tx), params)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	if err := tx.Commit(); err != nil {
		return nfse.ApplyOutcome{}, err
	}
	return outcome, nil
}

func (r *Store) ApplyEvent(ctx context.Context, params nfse.ApplyEventParams) (nfse.ApplyOutcome, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}
	defer func() { _ = tx.Rollback() }()

	outcome, err := r.doApplyEvent(ctx, tx, r.queries.WithTx(tx), params)
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}

	if err := tx.Commit(); err != nil {
		return nfse.ApplyOutcome{}, err
	}
	return outcome, nil
}

func (r *Store) PersistProgress(ctx context.Context, params nfse.PersistSyncProgressParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.doPersistProgress(ctx, tx, params); err != nil {
		return err
	}

	return tx.Commit()
}

// ApplyWithProgress runs write and the item's sync checkpoint in one
// transaction. An inserted document (not an event) is added to the run's
// documents_found before the checkpoint is written.
func (r *Store) ApplyWithProgress(ctx context.Context, progress nfse.PersistSyncProgressParams, write func(tx *sql.Tx) (ItemOutcome, error)) (ItemOutcome, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ItemOutcome{}, err
	}
	defer func() { _ = tx.Rollback() }()

	outcome, err := write(tx)
	if err != nil {
		return ItemOutcome{}, err
	}
	if outcome.Inserted && !outcome.IsEvent {
		progress.DocumentsFound++
	}
	if err := r.doPersistProgress(ctx, tx, progress); err != nil {
		return ItemOutcome{}, err
	}

	if err := tx.Commit(); err != nil {
		return ItemOutcome{}, err
	}
	return outcome, nil
}

// ApplyDocumentAndProgress stores a document and its sync checkpoint in one transaction.
func (r *Store) ApplyDocumentAndProgress(ctx context.Context, params nfse.ApplyDocumentAndProgressParams) (nfse.ApplyOutcome, error) {
	outcome, err := r.ApplyWithProgress(ctx, params.ProgressParams, func(tx *sql.Tx) (ItemOutcome, error) {
		applied, err := r.doApplyDocument(ctx, tx, r.queries.WithTx(tx), params.DocumentParams)
		return ItemOutcome{Inserted: applied.Inserted}, err
	})
	if err != nil {
		return nfse.ApplyOutcome{}, err
	}
	return nfse.ApplyOutcome{Inserted: outcome.Inserted}, nil
}

func companyDocumentMissing(ctx context.Context, tx executor, companyID, documentID string) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx,
		`SELECT 1 FROM company_documents WHERE company_id = ? AND document_id = ? LIMIT 1`,
		companyID,
		documentID,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func eventHashMissing(ctx context.Context, tx executor, rawHash string) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx,
		`SELECT 1 FROM events WHERE raw_hash = ? LIMIT 1`,
		rawHash,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func (r *Store) FinishRun(ctx context.Context, params nfse.FinishRunParams) error {
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
		store.NullInt64FromPtr(params.LastFoundNSU),
		string(params.RunID),
	)
	return err
}

func (r *Store) LatestSyncSnapshot(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource, environment nfse.Environment, consultationCNPJ string) (nfse.SyncSnapshot, error) {
	var snapshot nfse.SyncSnapshot

	state, err := r.getSyncState(ctx, companyID, source, environment, consultationCNPJ)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return snapshot, err
	}
	if err == nil {
		snapshot.State = state
	}

	run, err := r.latestRun(ctx, companyID, source, environment, consultationCNPJ)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return snapshot, err
	}
	if err == nil {
		snapshot.Run = run
	}

	return snapshot, nil
}

// ResetSyncState deletes the source's sync cursor and clears its initial-sync
// flag, so the next pull starts over under the company start policy. It keeps
// blocked_until: a local reset does not lift a wait imposed by the tax authority.
func (r *Store) ResetSyncState(ctx context.Context, params nfse.ResetSyncStateParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM sync_state WHERE company_id = ? AND source = ?`,
		string(params.CompanyID), string(params.Source),
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE company_sync_sources
		SET initial_sync_completed_at = NULL, updated_at = ?
		WHERE company_id = ? AND source = ?
	`, now, string(params.CompanyID), string(params.Source)); err != nil {
		return err
	}
	if params.Source == nfse.SyncSourceNFSe {
		if _, err := tx.ExecContext(ctx, `UPDATE companies SET initial_sync_completed_at = NULL, updated_at = ? WHERE id = ?`, now, string(params.CompanyID)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// HasSyncState reports whether the company has a sync cursor. An empty
// params.Source matches a cursor of any source.
func (r *Store) HasSyncState(ctx context.Context, params nfse.HasSyncStateParams) (bool, error) {
	query := `SELECT 1 FROM sync_state WHERE company_id = ? LIMIT 1`
	args := []any{string(params.CompanyID)}
	if params.Source != "" {
		query = `SELECT 1 FROM sync_state WHERE company_id = ? AND source = ? LIMIT 1`
		args = append(args, string(params.Source))
	}

	var exists int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SourceState holds the per (company, source) facts that outlive a single run.
type SourceState struct {
	InitialSyncDoneAt *time.Time
	BlockedUntil      *time.Time
	BlockedReason     nfse.SyncStopReason
}

// SourceState returns the company's facts for the source. A company that
// never synced the source gets the zero SourceState.
func (r *Store) SourceState(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource) (SourceState, error) {
	var initialSyncDoneAt, blockedUntil, blockedReason sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT initial_sync_completed_at, blocked_until, blocked_reason
		FROM company_sync_sources
		WHERE company_id = ? AND source = ?
	`, string(companyID), string(source)).Scan(&initialSyncDoneAt, &blockedUntil, &blockedReason)
	if errors.Is(err, sql.ErrNoRows) {
		return SourceState{}, nil
	}
	if err != nil {
		return SourceState{}, err
	}

	return SourceState{
		InitialSyncDoneAt: store.ParseNullableTime(initialSyncDoneAt),
		BlockedUntil:      store.ParseNullableTime(blockedUntil),
		BlockedReason:     nfse.SyncStopReason(blockedReason.String),
	}, nil
}

// MarkInitialSyncCompleted records the first time the source caught up for
// the company. For NFS-e it also sets companies.initial_sync_completed_at,
// which the company list and the desktop start-policy lock still read.
func (r *Store) MarkInitialSyncCompleted(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (company_id, source) DO UPDATE SET
			initial_sync_completed_at = COALESCE(initial_sync_completed_at, excluded.initial_sync_completed_at),
			updated_at = excluded.updated_at
	`, string(companyID), string(source), now, now); err != nil {
		return err
	}
	if source == nfse.SyncSourceNFSe {
		if _, err := tx.ExecContext(ctx, `
			UPDATE companies
			SET initial_sync_completed_at = COALESCE(initial_sync_completed_at, ?),
				updated_at = ?
			WHERE id = ?
		`, now, now, string(companyID)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SetBlockedUntil records that the source must not be queried for the
// company before until, and why.
func (r *Store) SetBlockedUntil(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource, until time.Time, reason nfse.SyncStopReason) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO company_sync_sources (company_id, source, blocked_until, blocked_reason, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (company_id, source) DO UPDATE SET
			blocked_until = excluded.blocked_until,
			blocked_reason = excluded.blocked_reason,
			updated_at = excluded.updated_at
	`, string(companyID), string(source), until.UTC().Format(time.RFC3339), string(reason), now)
	return err
}

// requestRetention is how long sync_requests rows are kept. It only has to
// cover the widest budget window.
const requestRetention = 24 * time.Hour

// RecordRequest logs one outbound distribution request for the source's
// rolling budget and prunes rows older than requestRetention.
func (r *Store) RecordRequest(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource, at time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sync_requests (company_id, source, requested_at) VALUES (?, ?, ?)`,
		string(companyID), string(source), at.UTC().Format(time.RFC3339),
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM sync_requests WHERE requested_at < ?`,
		at.UTC().Add(-requestRetention).Format(time.RFC3339),
	); err != nil {
		return err
	}

	return tx.Commit()
}

// RequestsSince counts the source's requests for the company at or after
// since, and returns the oldest of them (nil when there are none).
func (r *Store) RequestsSince(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource, since time.Time) (int, *time.Time, error) {
	var count int
	var oldest sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), MIN(requested_at)
		FROM sync_requests
		WHERE company_id = ? AND source = ? AND requested_at >= ?
	`, string(companyID), string(source), since.UTC().Format(time.RFC3339)).Scan(&count, &oldest)
	if err != nil {
		return 0, nil, err
	}
	return count, store.ParseNullableTime(oldest), nil
}

// RecordItemFailure counts one decode or parse failure of the item at nsu
// and returns how many times in a row that NSU has failed. A failure at a
// different NSU starts the count over.
func (r *Store) RecordItemFailure(ctx context.Context, key nfse.GetOrCreateSyncStateParams, nsu int64) (int, error) {
	var attempts int
	err := r.db.QueryRowContext(ctx, `
		UPDATE sync_state
		SET
			failed_nsu_attempts = CASE WHEN failed_nsu = ? THEN failed_nsu_attempts + 1 ELSE 1 END,
			failed_nsu = ?,
			updated_at = ?
		WHERE company_id = ? AND source = ? AND environment = ? AND consultation_cnpj = ?
		RETURNING failed_nsu_attempts
	`,
		nsu,
		nsu,
		time.Now().UTC().Format(time.RFC3339),
		string(key.CompanyID),
		string(key.Source),
		string(key.Environment),
		key.ConsultationCNPJ,
	).Scan(&attempts)
	return attempts, err
}

func (r *Store) CompanyDocumentExistsByAccessKey(ctx context.Context, companyID nfse.CompanyID, chave string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1
		FROM company_documents cd
		JOIN documents d ON d.id = cd.document_id
		WHERE cd.company_id = ? AND d.chave_acesso = ?
		LIMIT 1
	`, string(companyID), chave).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Store) getSyncState(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource, environment nfse.Environment, consultationCNPJ string) (*nfse.SyncState, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			company_id, source, environment, consultation_cnpj,
			last_checked_nsu, last_found_nsu, max_nsu, last_empty_streak,
			last_success_at, last_error_at, last_error_code, last_error_message,
			created_at, updated_at
		FROM sync_state
		WHERE company_id = ? AND source = ? AND environment = ? AND consultation_cnpj = ?
	`,
		string(companyID),
		string(source),
		string(environment),
		consultationCNPJ,
	)

	var state nfse.SyncState
	var lastFound, maxNSU sql.NullInt64
	var lastSuccessAt, lastErrorAt, lastErrorCode, lastErrorMessage sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(
		&state.CompanyID,
		&state.Source,
		&state.Environment,
		&state.ConsultationCNPJ,
		&state.LastProcessedNSU,
		&lastFound,
		&maxNSU,
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

	state.LastFoundNSU = store.PtrFromNullInt64(lastFound)
	state.MaxNSU = store.PtrFromNullInt64(maxNSU)
	state.LastSuccessAt = store.ParseNullableTime(lastSuccessAt)
	state.LastErrorAt = store.ParseNullableTime(lastErrorAt)
	state.LastErrorCode = lastErrorCode.String
	state.LastErrorMessage = lastErrorMessage.String
	state.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	state.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &state, nil
}

func (r *Store) latestRun(ctx context.Context, companyID nfse.CompanyID, source nfse.SyncSource, environment nfse.Environment, consultationCNPJ string) (*nfse.SyncRun, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, company_id, source, credential_id, environment, credential_cnpj, consultation_cnpj,
			consultation_basis, mode, started_at, finished_at, from_nsu, to_nsu,
			checked_count, documents_found, empty_count, consecutive_empty_count,
			errors_count, last_found_nsu, status, stop_reason
		FROM sync_runs
		WHERE company_id = ? AND source = ? AND environment = ? AND consultation_cnpj = ?
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`,
		string(companyID),
		string(source),
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
		&run.Source,
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
	run.FinishedAt = store.ParseNullableTime(finishedAt)
	run.LastFoundNSU = store.PtrFromNullInt64(lastFound)
	return &run, nil
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

func nullString(val string) sql.NullString {
	return sql.NullString{String: val, Valid: val != ""}
}
