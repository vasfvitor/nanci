package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

// NFeRepository stores NF-e documents, their events, each company's view of
// them and the manifestações nanci sent.
//
// company_nfe_documents.manifestacao and nfe_documents.situacao are never set
// directly: every write recomputes them from nfe_events.
type NFeRepository struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewNFeRepository(db *sql.DB) *NFeRepository {
	return &NFeRepository{
		db:      db,
		queries: sqlgen.New(db),
	}
}

// ApplyNFeDocumentParams is one distributed resNFe or procNFe for a company.
type ApplyNFeDocumentParams struct {
	Document    nfe.Document
	CompanyID   nfse.CompanyID
	CompanyCNPJ string
	NSU         int64
}

// ApplyNFeEventParams is one distributed resEvento or procEventoNFe.
type ApplyNFeEventParams struct {
	Event nfe.Event
}

// ApplyDocumentTx merges the document with the stored one (a resumo never
// replaces a completa), classifies the company's participation on the merged
// document, links waiting events and recomputes situação and manifestação.
// It runs inside the caller's transaction so the sync checkpoint can commit
// with it. inserted is true when the company did not see this chave before.
func (r *NFeRepository) ApplyDocumentTx(ctx context.Context, tx *sql.Tx, p ApplyNFeDocumentParams) (inserted bool, err error) {
	q := r.queries.WithTx(tx)
	now := time.Now().UTC().Format(time.RFC3339)
	chave := string(p.Document.ChaveAcesso)

	doc := p.Document
	row, err := q.GetNFeDocumentByChave(ctx, chave)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if doc.ID == "" {
			doc.ID = nfse.GenerateID()
		}
	case err != nil:
		return false, fmt.Errorf("read nfe document %s: %w", chave, err)
	default:
		existing, err := documentFromRow(row)
		if err != nil {
			return false, err
		}
		doc = nfe.MergeDocument(existing, doc)
	}

	params, err := upsertDocumentParams(doc, now)
	if err != nil {
		return false, err
	}
	documentID, err := q.UpsertNFeDocument(ctx, params)
	if err != nil {
		return false, fmt.Errorf("upsert nfe document %s: %w", chave, err)
	}

	seen, err := q.CompanyNFeDocumentExists(ctx, sqlgen.CompanyNFeDocumentExistsParams{
		CompanyID:   string(p.CompanyID),
		ChaveAcesso: chave,
	})
	if err != nil {
		return false, fmt.Errorf("check company nfe document %s: %w", chave, err)
	}

	participation := nfe.ClassifyParticipation(&doc, p.CompanyCNPJ)
	err = q.UpsertCompanyNFeDocument(ctx, sqlgen.UpsertCompanyNFeDocumentParams{
		RelationID:       nfse.GenerateID(),
		CompanyID:        string(p.CompanyID),
		NfeDocumentID:    documentID,
		CompanyRole:      string(participation.CompanyRole),
		VisibilityReason: string(participation.VisibilityReason),
		FirstSeenNsu:     sql.NullInt64{Int64: p.NSU, Valid: true},
		LastSeenNsu:      sql.NullInt64{Int64: p.NSU, Valid: true},
		FirstSyncedAt:    now,
		LastSyncedAt:     now,
	})
	if err != nil {
		return false, fmt.Errorf("upsert company nfe document %s: %w", chave, err)
	}

	if err := refreshChave(ctx, q, chave, now); err != nil {
		return false, err
	}
	return seen == 0, nil
}

// ApplyEventTx stores the event (a resumo never replaces a completa) and
// recomputes situação and manifestação for every company related to the
// chave. The document may arrive later; its ApplyDocumentTx links the event
// then. inserted is true when the event was not stored before.
func (r *NFeRepository) ApplyEventTx(ctx context.Context, tx *sql.Tx, p ApplyNFeEventParams) (inserted bool, err error) {
	q := r.queries.WithTx(tx)
	now := time.Now().UTC().Format(time.RFC3339)

	exists, err := eventExists(ctx, q, p.Event)
	if err != nil {
		return false, err
	}
	if err := upsertEvent(ctx, q, p.Event, now); err != nil {
		return false, err
	}
	if err := refreshChave(ctx, q, string(p.Event.ChaveAcesso), now); err != nil {
		return false, err
	}
	return !exists, nil
}

// CompanyDocumentExists reports whether the company already sees the chave.
func (r *NFeRepository) CompanyDocumentExists(ctx context.Context, companyID nfse.CompanyID, chave string) (bool, error) {
	count, err := r.queries.CompanyNFeDocumentExists(ctx, sqlgen.CompanyNFeDocumentExistsParams{
		CompanyID:   string(companyID),
		ChaveAcesso: chave,
	})
	if err != nil {
		return false, fmt.Errorf("check company nfe document %s: %w", chave, err)
	}
	return count > 0, nil
}

// ListCompanyDocuments returns the company's NF-e, newest issue date first.
func (r *NFeRepository) ListCompanyDocuments(ctx context.Context, companyID nfse.CompanyID, f nfe.DocumentFilter) ([]nfe.CompanyDocument, error) {
	return r.listCompanyDocuments(ctx, companyID, f, "")
}

// ListPendingExport returns the rows ListCompanyDocuments would return that
// were never exported with this kind, or whose raw hash changed since (for
// example after a resumo was upgraded to completa).
func (r *NFeRepository) ListPendingExport(ctx context.Context, companyID nfse.CompanyID, f nfe.DocumentFilter, kind string) ([]nfe.CompanyDocument, error) {
	if kind == "" {
		return nil, errors.New("export kind is required")
	}
	return r.listCompanyDocuments(ctx, companyID, f, kind)
}

// CompanyDocumentByChave returns ErrNotFound when the company does not see
// the chave.
func (r *NFeRepository) CompanyDocumentByChave(ctx context.Context, companyID nfse.CompanyID, chave string) (*nfe.CompanyDocument, error) {
	docs, err := r.listCompanyDocuments(ctx, companyID, nfe.DocumentFilter{ChavesAcesso: []string{chave}, Limit: 1}, "")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, ErrNotFound
	}
	return &docs[0], nil
}

// ListEventsByChave returns the stored events of a chave, oldest first.
func (r *NFeRepository) ListEventsByChave(ctx context.Context, chave string) ([]nfe.Event, error) {
	rows, err := r.queries.ListNFeEventsByChave(ctx, chave)
	if err != nil {
		return nil, fmt.Errorf("list nfe events %s: %w", chave, err)
	}
	return eventsFromRows(rows)
}

// ListEventsByChaves returns the stored events of every chave in one query,
// ordered by chave and then oldest first. sqlc cannot type the json_each
// parameter, so the query is written here.
func (r *NFeRepository) ListEventsByChaves(ctx context.Context, chaves []string) ([]nfe.Event, error) {
	if len(chaves) == 0 {
		return nil, nil
	}
	//nolint:misspell // autor_cnpj is the column name.
	const query = `
		SELECT id, nfe_document_id, chave_acesso, tp_evento, type, n_seq_evento, event_at, registered_at,
			registered, c_stat, x_motivo, protocolo, autor_cnpj, description, justificativa, correcao,
			completeness, sent_by_nanci, raw_hash, parse_warnings, created_at, updated_at
		FROM nfe_events
		WHERE chave_acesso IN (SELECT value FROM json_each(?))
		ORDER BY chave_acesso, COALESCE(registered_at, event_at, created_at), tp_evento, n_seq_evento
	`
	chavesJSON, _ := json.Marshal(chaves) // a []string always marshals
	rows, err := r.db.QueryContext(ctx, query, string(chavesJSON))
	if err != nil {
		return nil, fmt.Errorf("list nfe events of %d chaves: %w", len(chaves), err)
	}
	defer func() { _ = rows.Close() }()

	var items []sqlgen.NfeEvent
	for rows.Next() {
		var i sqlgen.NfeEvent
		err := rows.Scan(
			&i.ID, &i.NfeDocumentID, &i.ChaveAcesso, &i.TpEvento, &i.Type, &i.NSeqEvento, &i.EventAt, &i.RegisteredAt,
			&i.Registered, &i.CStat, &i.XMotivo, &i.Protocolo, &i.AutorCnpj, &i.Description, &i.Justificativa, &i.Correcao,
			&i.Completeness, &i.SentByNanci, &i.RawHash, &i.ParseWarnings, &i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan nfe event: %w", err)
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list nfe events of %d chaves: %w", len(chaves), err)
	}
	return eventsFromRows(items)
}

// CountSummary counts the company's NF-e by role and completeness.
func (r *NFeRepository) CountSummary(ctx context.Context, companyID nfse.CompanyID) (nfe.Counts, error) {
	const query = `
		SELECT
			cd.company_role,
			COUNT(*),
			SUM(d.completeness = 'resumo'),
			SUM(d.completeness = 'completa')
		FROM company_nfe_documents cd
		INNER JOIN nfe_documents d ON d.id = cd.nfe_document_id
		WHERE cd.company_id = ?
		GROUP BY cd.company_role
	`
	rows, err := r.db.QueryContext(ctx, query, string(companyID))
	if err != nil {
		return nfe.Counts{}, fmt.Errorf("count nfe documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := nfe.Counts{ByRole: make(map[nfe.CompanyRole]int)}
	for rows.Next() {
		var role string
		var total, resumos, completas int
		if err := rows.Scan(&role, &total, &resumos, &completas); err != nil {
			return nfe.Counts{}, fmt.Errorf("scan nfe counts: %w", err)
		}
		counts.ByRole[nfe.CompanyRole(role)] = total
		counts.Resumos += resumos
		counts.Completas += completas
	}
	if err := rows.Err(); err != nil {
		return nfe.Counts{}, fmt.Errorf("iterate nfe counts: %w", err)
	}
	return counts, nil
}

// MarkViewed sets viewed_at on the unread rows matching the filter (Limit is
// ignored) and returns how many rows changed.
func (r *NFeRepository) MarkViewed(ctx context.Context, companyID nfse.CompanyID, f nfe.DocumentFilter) (int, error) {
	query := `
		UPDATE company_nfe_documents
		SET viewed_at = ?
		WHERE viewed_at IS NULL AND relation_id IN (
			SELECT cd.relation_id
			FROM company_nfe_documents cd
			INNER JOIN nfe_documents d ON d.id = cd.nfe_document_id
			WHERE `
	where, whereArgs := buildNFeFilterSQL(companyID, f)
	query += where // #nosec G202 -- constant conditions with ? placeholders from buildNFeFilterSQL.
	query += ")"
	args := append([]any{time.Now().UTC().Format(time.RFC3339)}, whereArgs...)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("mark nfe documents viewed: %w", err)
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(changed), nil
}

// MarkExported records the current raw hash of each document as exported.
func (r *NFeRepository) MarkExported(ctx context.Context, companyID nfse.CompanyID, kind string, docs []nfe.CompanyDocument) error {
	if len(docs) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := r.queries.WithTx(tx)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, doc := range docs {
		err := q.MarkNFeExported(ctx, sqlgen.MarkNFeExportedParams{
			CompanyID:     string(companyID),
			NfeDocumentID: doc.ID,
			ExportKind:    kind,
			ExportedHash:  doc.RawHash,
			ExportedAt:    now,
		})
		if err != nil {
			return fmt.Errorf("mark nfe document %s exported: %w", doc.ChaveAcesso, err)
		}
	}
	return tx.Commit()
}

// RecordManifestations stores the outcomes of one lote in one transaction.
// Every item is kept in nfe_manifestations. A registered event is also
// stored in nfe_events as a completa authored by the company; an event SEFAZ
// reports as already registered is stored the same way unless nfe_events
// already has it. Manifestação is then recomputed for the chave.
func (r *NFeRepository) RecordManifestations(ctx context.Context, items []nfe.ManifestationRecord) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := r.queries.WithTx(tx)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range items {
		err := q.InsertNFeManifestation(ctx, sqlgen.InsertNFeManifestationParams{
			ID:              nfse.GenerateID(),
			CompanyID:       string(item.CompanyID),
			ChaveAcesso:     item.ChaveAcesso,
			TpEvento:        item.TpEvento,
			NSeqEvento:      int64(item.NSeqEvento),
			Justificativa:   item.Justificativa,
			IDLote:          item.IDLote,
			Status:          item.Status,
			CStat:           item.CStat,
			XMotivo:         item.XMotivo,
			Protocolo:       item.Protocolo,
			RegisteredAt:    nullableTime(item.RegisteredAt, time.RFC3339),
			RequestRawHash:  nullString(item.RequestRawHash),
			ResponseRawHash: nullString(item.ResponseRawHash),
			CreatedAt:       now,
			TpAmb:           item.TpAmb,
		})
		if err != nil {
			return fmt.Errorf("record manifestação %s %s: %w", item.TpEvento, item.ChaveAcesso, err)
		}

		if item.Status != nfe.ManifestationStatusRegistrada && item.Status != nfe.ManifestationStatusJaRegistrada {
			continue
		}
		event := manifestationEvent(item)
		if item.Status == nfe.ManifestationStatusJaRegistrada {
			exists, err := eventExists(ctx, q, event)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
		}
		if err := upsertEvent(ctx, q, event, now); err != nil {
			return err
		}
		if err := refreshChave(ctx, q, item.ChaveAcesso, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func manifestationEvent(item nfe.ManifestationRecord) nfe.Event {
	return nfe.Event{
		ChaveAcesso:   nfe.AccessKey(item.ChaveAcesso),
		TpEvento:      item.TpEvento,
		Type:          nfe.EventTypeFromTpEvento(item.TpEvento),
		NSeqEvento:    item.NSeqEvento,
		EventAt:       item.EventAt,
		RegisteredAt:  item.RegisteredAt,
		Protocolo:     item.Protocolo,
		AutorCNPJ:     cnpj.Clean(item.CompanyCNPJ),
		Description:   item.Description,
		Justificativa: item.Justificativa,
		Completeness:  nfe.CompletenessCompleta,
		Registered:    true,
		CStat:         item.CStat,
		XMotivo:       item.XMotivo,
		SentByNanci:   true,
		RawHash:       item.ProcEventoRawHash,
	}
}

// refreshChave brings the rows derived from the events of one chave up to
// date: it links events to the document, applies registered events to the
// situação and recomputes the manifestação of every related company. It does
// nothing while the document has not arrived.
func refreshChave(ctx context.Context, q *sqlgen.Queries, chave, now string) error {
	doc, err := q.GetNFeDocumentByChave(ctx, chave)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read nfe document %s: %w", chave, err)
	}

	err = q.LinkNFeEventsToDocument(ctx, sqlgen.LinkNFeEventsToDocumentParams{
		NfeDocumentID: sql.NullString{String: doc.ID, Valid: true},
		ChaveAcesso:   chave,
	})
	if err != nil {
		return fmt.Errorf("link nfe events %s: %w", chave, err)
	}

	rows, err := q.ListNFeEventsByChave(ctx, chave)
	if err != nil {
		return fmt.Errorf("list nfe events %s: %w", chave, err)
	}
	events, err := eventsFromRows(rows)
	if err != nil {
		return err
	}

	var registered []nfe.EventType
	for _, e := range events {
		if e.Registered {
			registered = append(registered, e.Type)
		}
	}
	situacao := nfe.SituacaoFromEvents(nfe.Situacao(doc.Situacao), registered)
	if string(situacao) != doc.Situacao {
		err := q.UpdateNFeSituacao(ctx, sqlgen.UpdateNFeSituacaoParams{
			Situacao:    string(situacao),
			UpdatedAt:   now,
			ChaveAcesso: chave,
		})
		if err != nil {
			return fmt.Errorf("update nfe situacao %s: %w", chave, err)
		}
	}

	relations, err := q.ListCompanyNFeRelationsByChave(ctx, chave)
	if err != nil {
		return fmt.Errorf("list company nfe relations %s: %w", chave, err)
	}
	for _, rel := range relations {
		manifestacao, at := nfe.ManifestacaoAndTimeFromEvents(events, rel.Cnpj)
		err := q.UpdateCompanyNFeManifestacao(ctx, sqlgen.UpdateCompanyNFeManifestacaoParams{
			Manifestacao:   string(manifestacao),
			ManifestacaoAt: nullableTime(at, time.RFC3339),
			RelationID:     rel.RelationID,
		})
		if err != nil {
			return fmt.Errorf("update nfe manifestacao %s: %w", chave, err)
		}
	}
	return nil
}

func eventExists(ctx context.Context, q *sqlgen.Queries, e nfe.Event) (bool, error) {
	count, err := q.NFeEventExists(ctx, sqlgen.NFeEventExistsParams{
		ChaveAcesso: string(e.ChaveAcesso),
		TpEvento:    e.TpEvento,
		NSeqEvento:  int64(e.NSeqEvento),
	})
	if err != nil {
		return false, fmt.Errorf("check nfe event %s %s: %w", e.TpEvento, e.ChaveAcesso, err)
	}
	return count > 0, nil
}

func upsertEvent(ctx context.Context, q *sqlgen.Queries, e nfe.Event, now string) error {
	id := e.ID
	if id == "" {
		id = nfse.GenerateID()
	}
	warnings, err := encodeWarnings(e.ParseWarnings)
	if err != nil {
		return err
	}
	err = q.UpsertNFeEvent(ctx, sqlgen.UpsertNFeEventParams{
		ID:            id,
		ChaveAcesso:   string(e.ChaveAcesso),
		TpEvento:      e.TpEvento,
		Type:          string(e.Type),
		NSeqEvento:    int64(e.NSeqEvento),
		EventAt:       nullableTime(e.EventAt, time.RFC3339),
		RegisteredAt:  nullableTime(e.RegisteredAt, time.RFC3339),
		Registered:    boolToInt(e.Registered),
		CStat:         e.CStat,
		XMotivo:       e.XMotivo,
		Protocolo:     e.Protocolo,
		AutorCnpj:     e.AutorCNPJ,
		Description:   e.Description,
		Justificativa: e.Justificativa,
		Correcao:      e.Correcao,
		Completeness:  string(e.Completeness),
		SentByNanci:   boolToInt(e.SentByNanci),
		RawHash:       e.RawHash,
		ParseWarnings: warnings,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return fmt.Errorf("upsert nfe event %s %s: %w", e.TpEvento, e.ChaveAcesso, err)
	}
	return nil
}

// nfeCompanyDocumentColumns is the only column list for company NF-e rows;
// scanCompanyNFeDocument reads it in the same order.
const nfeCompanyDocumentColumns = `
	d.id, d.chave_acesso, d.modelo, d.serie, d.numero, d.issue_date, d.competence, d.authorized_at, d.protocolo,
	d.emitente_cnpj, d.emitente_name, d.emitente_ie, d.emitente_uf,
	d.destinatario_cnpj, d.destinatario_name, d.transportador_cnpj, d.autorizados_cnpj,
	d.tp_nf, d.fin_nfe, d.nat_op, d.total_value, d.icms_value, d.ipi_value,
	d.situacao, d.completeness, d.layout_version, d.raw_hash, d.resumo_raw_hash, d.parse_warnings,
	cd.relation_id, cd.company_id, cd.company_role, cd.visibility_reason, cd.manifestacao, cd.manifestacao_at,
	cd.first_seen_nsu, cd.last_seen_nsu, cd.first_synced_at, cd.last_synced_at, cd.viewed_at,
	(SELECT COUNT(*) FROM nfe_events e WHERE e.chave_acesso = d.chave_acesso)`

// listCompanyDocuments runs the single company NF-e query. A non-empty
// exportKind keeps only rows pending export for that kind.
func (r *NFeRepository) listCompanyDocuments(ctx context.Context, companyID nfse.CompanyID, f nfe.DocumentFilter, exportKind string) ([]nfe.CompanyDocument, error) {
	query := `SELECT ` + nfeCompanyDocumentColumns + `
		FROM company_nfe_documents cd
		INNER JOIN nfe_documents d ON d.id = cd.nfe_document_id`
	var args []any
	if exportKind != "" {
		query += `
		LEFT JOIN company_nfe_export_marks m
			ON m.company_id = cd.company_id AND m.nfe_document_id = cd.nfe_document_id AND m.export_kind = ?`
		args = append(args, exportKind)
	}

	where, whereArgs := buildNFeFilterSQL(companyID, f)
	query += " WHERE " + where // #nosec G202 -- constant conditions with ? placeholders from buildNFeFilterSQL.
	args = append(args, whereArgs...)
	if exportKind != "" {
		query += " AND (m.exported_at IS NULL OR m.exported_hash != d.raw_hash)"
	}

	query += " ORDER BY d.issue_date DESC, d.chave_acesso DESC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query nfe documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var docs []nfe.CompanyDocument
	for rows.Next() {
		doc, err := scanCompanyNFeDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nfe documents: %w", err)
	}
	return docs, nil
}

// buildNFeFilterSQL returns the WHERE conditions for f over the aliases cd
// (company_nfe_documents) and d (nfe_documents), and their arguments. Limit
// is left to the caller.
func buildNFeFilterSQL(companyID nfse.CompanyID, f nfe.DocumentFilter) (string, []any) {
	where := "cd.company_id = ?"
	args := []any{string(companyID)}

	if f.Competence != "" {
		where += " AND d.competence = ?"
		args = append(args, f.Competence)
	}
	if f.Situacao != "" {
		where += " AND d.situacao = ?"
		args = append(args, string(f.Situacao))
	}
	if f.Completeness != "" {
		where += " AND d.completeness = ?"
		args = append(args, string(f.Completeness))
	}
	if f.Role != "" {
		where += " AND cd.company_role = ?"
		args = append(args, string(f.Role))
	}
	if f.Manifestacao != "" {
		where += " AND cd.manifestacao = ?"
		args = append(args, string(f.Manifestacao))
	}
	if f.EmitenteCNPJ != "" {
		where += " AND d.emitente_cnpj = ?"
		args = append(args, cnpj.Clean(f.EmitenteCNPJ))
	}
	if len(f.ChavesAcesso) > 0 {
		where += " AND d.chave_acesso IN (SELECT value FROM json_each(?))"
		chaves, _ := json.Marshal(f.ChavesAcesso) // a []string always marshals
		args = append(args, string(chaves))
	}
	if f.OnlyUnread {
		where += " AND cd.viewed_at IS NULL"
	}
	if f.IssueDateGTE != nil {
		// issue_date is RFC 3339, so its date prefix compares as text.
		where += " AND d.issue_date >= ?"
		args = append(args, f.IssueDateGTE.Format("2006-01-02"))
	}
	if f.PendingManifestation {
		where += " AND cd.company_role = 'destinatario' AND d.situacao = 'autorizada' AND cd.manifestacao IN ('nenhuma', 'ciencia')"
	}
	return where, args
}

func scanCompanyNFeDocument(rows *sql.Rows) (nfe.CompanyDocument, error) {
	var d sqlgen.NfeDocument
	var cd nfe.CompanyDocument
	var companyID, role, visibility, manifestacao string
	var manifestacaoAt, viewedAt sql.NullString
	var firstSeen, lastSeen sql.NullInt64
	var firstSyncedAt, lastSyncedAt string

	err := rows.Scan(
		&d.ID, &d.ChaveAcesso, &d.Modelo, &d.Serie, &d.Numero, &d.IssueDate, &d.Competence, &d.AuthorizedAt, &d.Protocolo,
		&d.EmitenteCnpj, &d.EmitenteName, &d.EmitenteIe, &d.EmitenteUf,
		&d.DestinatarioCnpj, &d.DestinatarioName, &d.TransportadorCnpj, &d.AutorizadosCnpj,
		&d.TpNf, &d.FinNfe, &d.NatOp, &d.TotalValue, &d.IcmsValue, &d.IpiValue,
		&d.Situacao, &d.Completeness, &d.LayoutVersion, &d.RawHash, &d.ResumoRawHash, &d.ParseWarnings,
		&cd.RelationID, &companyID, &role, &visibility, &manifestacao, &manifestacaoAt,
		&firstSeen, &lastSeen, &firstSyncedAt, &lastSyncedAt, &viewedAt,
		&cd.EventCount,
	)
	if err != nil {
		return nfe.CompanyDocument{}, fmt.Errorf("scan nfe document: %w", err)
	}

	cd.Document, err = documentFromRow(d)
	if err != nil {
		return nfe.CompanyDocument{}, err
	}
	cd.CompanyID = nfse.CompanyID(companyID)
	cd.CompanyRole = nfe.CompanyRole(role)
	cd.VisibilityReason = nfe.VisibilityReason(visibility)
	cd.Manifestacao = nfe.Manifestacao(manifestacao)
	cd.FirstSeenNSU = PtrFromNullInt64(firstSeen)
	cd.LastSeenNSU = PtrFromNullInt64(lastSeen)

	if cd.ManifestacaoAt, err = parseOptionalTime("company nfe manifestacao_at", manifestacaoAt); err != nil {
		return nfe.CompanyDocument{}, err
	}
	if cd.ViewedAt, err = parseOptionalTime("company nfe viewed_at", viewedAt); err != nil {
		return nfe.CompanyDocument{}, err
	}
	if cd.FirstSyncedAt, err = parseRequiredTime("company nfe first_synced_at", firstSyncedAt); err != nil {
		return nfe.CompanyDocument{}, err
	}
	if cd.LastSyncedAt, err = parseRequiredTime("company nfe last_synced_at", lastSyncedAt); err != nil {
		return nfe.CompanyDocument{}, err
	}
	return cd, nil
}

func documentFromRow(row sqlgen.NfeDocument) (nfe.Document, error) {
	doc := nfe.Document{
		ID:                row.ID,
		ChaveAcesso:       nfe.AccessKey(row.ChaveAcesso),
		Modelo:            row.Modelo,
		Serie:             row.Serie,
		Numero:            row.Numero,
		Competence:        row.Competence,
		Protocolo:         row.Protocolo,
		EmitenteCNPJ:      row.EmitenteCnpj,
		EmitenteName:      row.EmitenteName,
		EmitenteIE:        row.EmitenteIe,
		EmitenteUF:        row.EmitenteUf,
		DestinatarioCNPJ:  row.DestinatarioCnpj,
		DestinatarioName:  row.DestinatarioName,
		TransportadorCNPJ: row.TransportadorCnpj,
		TpNF:              row.TpNf,
		FinNFe:            row.FinNfe,
		NatOp:             row.NatOp,
		TotalValue:        nfse.Money(row.TotalValue),
		ICMSValue:         nfse.Money(row.IcmsValue),
		IPIValue:          nfse.Money(row.IpiValue),
		Situacao:          nfe.Situacao(row.Situacao),
		Completeness:      nfe.Completeness(row.Completeness),
		LayoutVersion:     row.LayoutVersion,
		RawHash:           row.RawHash,
		ResumoRawHash:     row.ResumoRawHash.String,
	}
	if row.AutorizadosCnpj != "" {
		doc.AutorizadosCNPJ = strings.Split(row.AutorizadosCnpj, ",")
	}

	var err error
	if row.IssueDate != "" {
		if doc.IssueDate, err = parseRequiredTime("nfe issue_date", row.IssueDate); err != nil {
			return nfe.Document{}, err
		}
	}
	if doc.AuthorizedAt, err = parseOptionalTime("nfe authorized_at", row.AuthorizedAt); err != nil {
		return nfe.Document{}, err
	}
	if err := decodeWarnings(row.ParseWarnings, &doc.ParseWarnings); err != nil {
		return nfe.Document{}, fmt.Errorf("nfe parse_warnings: %w", err)
	}
	return doc, nil
}

func upsertDocumentParams(doc nfe.Document, now string) (sqlgen.UpsertNFeDocumentParams, error) {
	warnings, err := encodeWarnings(doc.ParseWarnings)
	if err != nil {
		return sqlgen.UpsertNFeDocumentParams{}, err
	}
	var issueDate string
	if !doc.IssueDate.IsZero() {
		issueDate = doc.IssueDate.Format(time.RFC3339)
	}
	return sqlgen.UpsertNFeDocumentParams{
		ID:                doc.ID,
		ChaveAcesso:       string(doc.ChaveAcesso),
		Modelo:            doc.Modelo,
		Serie:             doc.Serie,
		Numero:            doc.Numero,
		IssueDate:         issueDate,
		Competence:        doc.Competence,
		AuthorizedAt:      nullableTime(doc.AuthorizedAt, time.RFC3339),
		Protocolo:         doc.Protocolo,
		EmitenteCnpj:      doc.EmitenteCNPJ,
		EmitenteName:      doc.EmitenteName,
		EmitenteIe:        doc.EmitenteIE,
		EmitenteUf:        doc.EmitenteUF,
		DestinatarioCnpj:  doc.DestinatarioCNPJ,
		DestinatarioName:  doc.DestinatarioName,
		TransportadorCnpj: doc.TransportadorCNPJ,
		AutorizadosCnpj:   strings.Join(doc.AutorizadosCNPJ, ","),
		TpNf:              doc.TpNF,
		FinNfe:            doc.FinNFe,
		NatOp:             doc.NatOp,
		TotalValue:        doc.TotalValue.Cents(),
		IcmsValue:         doc.ICMSValue.Cents(),
		IpiValue:          doc.IPIValue.Cents(),
		Situacao:          string(doc.Situacao),
		Completeness:      string(doc.Completeness),
		LayoutVersion:     doc.LayoutVersion,
		RawHash:           doc.RawHash,
		ResumoRawHash:     nullString(doc.ResumoRawHash),
		ParseWarnings:     warnings,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

func eventsFromRows(rows []sqlgen.NfeEvent) ([]nfe.Event, error) {
	events := make([]nfe.Event, 0, len(rows))
	for _, row := range rows {
		e := nfe.Event{
			ID:            row.ID,
			ChaveAcesso:   nfe.AccessKey(row.ChaveAcesso),
			TpEvento:      row.TpEvento,
			Type:          nfe.EventType(row.Type),
			NSeqEvento:    int(row.NSeqEvento),
			Protocolo:     row.Protocolo,
			AutorCNPJ:     row.AutorCnpj,
			Description:   row.Description,
			Justificativa: row.Justificativa,
			Correcao:      row.Correcao,
			Completeness:  nfe.Completeness(row.Completeness),
			Registered:    row.Registered != 0,
			CStat:         row.CStat,
			XMotivo:       row.XMotivo,
			SentByNanci:   row.SentByNanci != 0,
			RawHash:       row.RawHash,
		}
		var err error
		if e.EventAt, err = parseOptionalTime("nfe event event_at", row.EventAt); err != nil {
			return nil, err
		}
		if e.RegisteredAt, err = parseOptionalTime("nfe event registered_at", row.RegisteredAt); err != nil {
			return nil, err
		}
		if err := decodeWarnings(row.ParseWarnings, &e.ParseWarnings); err != nil {
			return nil, fmt.Errorf("nfe event parse_warnings: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}

// parseOptionalTime returns nil for NULL or an empty string.
func parseOptionalTime(field string, value sql.NullString) (*time.Time, error) {
	var result *time.Time
	if value.Valid && value.String != "" {
		t, err := parseRequiredTime(field, value.String)
		if err != nil {
			return nil, err
		}
		result = &t
	}
	return result, nil
}

func encodeWarnings(warnings []string) (sql.NullString, error) {
	if len(warnings) == 0 {
		return sql.NullString{}, nil
	}
	data, err := json.Marshal(warnings)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(data), Valid: true}, nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func boolToInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

// ResetCompany removes the company's NF-e view in one transaction: its
// company_nfe_documents rows and export marks, the nfe_documents no other
// company sees and their events, and the company's own events that have no
// document. Events authored by another registered company are kept, unlinked
// from a removed document. nfe_manifestations are kept as the audit trail,
// and the XML blobs stay on disk. The sync cursor is not touched.
func (r *NFeRepository) ResetCompany(ctx context.Context, companyID nfse.CompanyID) (nfe.ResetCounts, error) {
	return r.resetCompany(ctx, companyID, true)
}

// PreviewResetCompany returns what ResetCompany would remove, changing
// nothing.
func (r *NFeRepository) PreviewResetCompany(ctx context.Context, companyID nfse.CompanyID) (nfe.ResetCounts, error) {
	return r.resetCompany(ctx, companyID, false)
}

// resetCompany runs the reset and commits it only when apply is set, so the
// preview counts come from the same statements.
func (r *NFeRepository) resetCompany(ctx context.Context, companyID nfse.CompanyID, apply bool) (nfe.ResetCounts, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nfe.ResetCounts{}, err
	}
	defer func() { _ = tx.Rollback() }()

	id := string(companyID)
	var counts nfe.ResetCounts
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM nfe_manifestations WHERE company_id = ?`, id).Scan(&counts.ManifestationsKept); err != nil {
		return nfe.ResetCounts{}, fmt.Errorf("count nfe manifestations: %w", err)
	}

	// The documents only this company sees go with its rows.
	documentIDs, err := exclusiveDocumentIDs(ctx, tx, id)
	if err != nil {
		return nfe.ResetCounts{}, err
	}
	documentIDsJSON, _ := json.Marshal(documentIDs) // a []string always marshals

	exec := func(what, query string, args ...any) (int, error) {
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, fmt.Errorf("reset %s: %w", what, err)
		}
		n, err := res.RowsAffected()
		return int(n), err
	}

	if counts.ExportMarks, err = exec("nfe export marks", `DELETE FROM company_nfe_export_marks WHERE company_id = ?`, id); err != nil {
		return nfe.ResetCounts{}, err
	}
	//nolint:misspell // autor_cnpj is the column name.
	counts.Events, err = exec("nfe events", `
		DELETE FROM nfe_events
		WHERE autor_cnpj NOT IN (SELECT cnpj FROM companies WHERE id <> ?)
			AND (
				chave_acesso IN (SELECT chave_acesso FROM nfe_documents WHERE id IN (SELECT value FROM json_each(?)))
				OR (
					autor_cnpj = (SELECT cnpj FROM companies WHERE id = ?)
					AND NOT EXISTS (SELECT 1 FROM nfe_documents d WHERE d.chave_acesso = nfe_events.chave_acesso)
				)
			)
	`, id, string(documentIDsJSON), id)
	if err != nil {
		return nfe.ResetCounts{}, err
	}
	if _, err := exec("kept nfe events", `
		UPDATE nfe_events SET nfe_document_id = NULL
		WHERE nfe_document_id IN (SELECT value FROM json_each(?))
	`, string(documentIDsJSON)); err != nil {
		return nfe.ResetCounts{}, err
	}
	if counts.CompanyDocuments, err = exec("company nfe documents", `DELETE FROM company_nfe_documents WHERE company_id = ?`, id); err != nil {
		return nfe.ResetCounts{}, err
	}
	if counts.Documents, err = exec("nfe documents", `DELETE FROM nfe_documents WHERE id IN (SELECT value FROM json_each(?))`, string(documentIDsJSON)); err != nil {
		return nfe.ResetCounts{}, err
	}

	if !apply {
		return counts, nil
	}
	if err := tx.Commit(); err != nil {
		return nfe.ResetCounts{}, err
	}
	return counts, nil
}

// exclusiveDocumentIDs returns the nfe_documents the company sees and no
// other company does.
func exclusiveDocumentIDs(ctx context.Context, tx *sql.Tx, companyID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT cd.nfe_document_id FROM company_nfe_documents cd
		WHERE cd.company_id = ?
			AND NOT EXISTS (
				SELECT 1 FROM company_nfe_documents other
				WHERE other.nfe_document_id = cd.nfe_document_id AND other.company_id <> cd.company_id
			)
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list exclusive nfe documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan nfe document id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list exclusive nfe documents: %w", err)
	}
	return ids, nil
}
