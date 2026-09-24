package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
)

// CTeRepository stores CT-e documents (CT-e, CT-e OS, GTV-e and CT-e
// Simplificado), their events and each company's view of them.
//
// cte_documents.situacao is never set directly by an event: every write
// recomputes it from cte_events.
type CTeRepository struct {
	db      *sql.DB
	queries *sqlgen.Queries
}

func NewCTeRepository(db *sql.DB) *CTeRepository {
	return &CTeRepository{
		db:      db,
		queries: sqlgen.New(db),
	}
}

// ApplyCTeDocumentParams is one distributed transport document for a company.
type ApplyCTeDocumentParams struct {
	Document    cte.Document
	CompanyID   dfe.CompanyID
	CompanyCNPJ string
	NSU         int64
}

// ApplyCTeEventParams is one distributed procEventoCTe.
type ApplyCTeEventParams struct {
	Event cte.Event
}

// ApplyDocumentTx merges the document with the stored one, classifies the
// company's participation on the merged document, links waiting events and
// recomputes the situação. It runs inside the caller's transaction so the
// sync checkpoint can commit with it. inserted is true when the company did
// not see this chave before.
func (r *CTeRepository) ApplyDocumentTx(ctx context.Context, tx *sql.Tx, p ApplyCTeDocumentParams) (inserted bool, err error) {
	q := r.queries.WithTx(tx)
	now := time.Now().UTC().Format(time.RFC3339)
	chave := string(p.Document.ChaveAcesso)

	doc := p.Document
	row, err := q.GetCTeDocumentByChave(ctx, chave)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if doc.ID == "" {
			doc.ID = nfse.GenerateID()
		}
	case err != nil:
		return false, fmt.Errorf("read cte document %s: %w", chave, err)
	default:
		existing, err := cteDocumentFromRow(row)
		if err != nil {
			return false, err
		}
		doc = cte.MergeDocument(existing, doc)
	}

	params, err := upsertCTeDocumentParams(doc, now)
	if err != nil {
		return false, err
	}
	documentID, err := q.UpsertCTeDocument(ctx, params)
	if err != nil {
		return false, fmt.Errorf("upsert cte document %s: %w", chave, err)
	}

	seen, err := q.HasCompanyCTeDocument(ctx, sqlgen.HasCompanyCTeDocumentParams{
		CompanyID:   string(p.CompanyID),
		ChaveAcesso: chave,
	})
	if err != nil {
		return false, fmt.Errorf("check company cte document %s: %w", chave, err)
	}

	participation := cte.ClassifyParticipation(&doc, p.CompanyCNPJ)
	err = q.UpsertCompanyCTeDocument(ctx, sqlgen.UpsertCompanyCTeDocumentParams{
		RelationID:       nfse.GenerateID(),
		CompanyID:        string(p.CompanyID),
		CteDocumentID:    documentID,
		CompanyRole:      string(participation.CompanyRole),
		Papeis:           joinPapeis(participation.Papeis),
		VisibilityReason: string(participation.VisibilityReason),
		FirstSeenNsu:     sql.NullInt64{Int64: p.NSU, Valid: true},
		LastSeenNsu:      sql.NullInt64{Int64: p.NSU, Valid: true},
		FirstSyncedAt:    now,
		LastSyncedAt:     now,
	})
	if err != nil {
		return false, fmt.Errorf("upsert company cte document %s: %w", chave, err)
	}

	if err := refreshCTeChave(ctx, q, chave, now); err != nil {
		return false, err
	}
	return seen == 0, nil
}

// ApplyEventTx stores the event and recomputes the situação of its
// document. The document may arrive later; its ApplyDocumentTx links the
// event then. inserted is true when the event was not stored before.
func (r *CTeRepository) ApplyEventTx(ctx context.Context, tx *sql.Tx, p ApplyCTeEventParams) (inserted bool, err error) {
	q := r.queries.WithTx(tx)
	now := time.Now().UTC().Format(time.RFC3339)
	e := p.Event

	count, err := q.HasCTeEvent(ctx, sqlgen.HasCTeEventParams{
		ChaveAcesso: string(e.ChaveAcesso),
		TpEvento:    e.TpEvento,
		NSeqEvento:  int64(e.NSeqEvento),
	})
	if err != nil {
		return false, fmt.Errorf("check cte event %s %s: %w", e.TpEvento, e.ChaveAcesso, err)
	}
	if err := upsertCTeEvent(ctx, q, e, now); err != nil {
		return false, err
	}
	if err := refreshCTeChave(ctx, q, string(e.ChaveAcesso), now); err != nil {
		return false, err
	}
	return count == 0, nil
}

// CompanyDocumentExists reports whether the company already sees the chave.
func (r *CTeRepository) CompanyDocumentExists(ctx context.Context, companyID dfe.CompanyID, chave string) (bool, error) {
	count, err := r.queries.HasCompanyCTeDocument(ctx, sqlgen.HasCompanyCTeDocumentParams{
		CompanyID:   string(companyID),
		ChaveAcesso: chave,
	})
	if err != nil {
		return false, fmt.Errorf("check company cte document %s: %w", chave, err)
	}
	return count > 0, nil
}

// ListCompanyDocuments returns the company's CT-e, newest issue date first.
func (r *CTeRepository) ListCompanyDocuments(ctx context.Context, companyID dfe.CompanyID, f cte.DocumentFilter) ([]cte.CompanyDocument, error) {
	return r.listCompanyDocuments(ctx, companyID, f, "")
}

// ListPendingExport returns the rows ListCompanyDocuments would return that
// were never exported with this kind, or whose raw hash changed since.
func (r *CTeRepository) ListPendingExport(ctx context.Context, companyID dfe.CompanyID, f cte.DocumentFilter, kind string) ([]cte.CompanyDocument, error) {
	if kind == "" {
		return nil, errors.New("export kind is required")
	}
	return r.listCompanyDocuments(ctx, companyID, f, kind)
}

// CompanyDocumentByChave returns cte.ErrDocumentNotFound when the company
// does not see the chave.
func (r *CTeRepository) CompanyDocumentByChave(ctx context.Context, companyID dfe.CompanyID, chave string) (*cte.CompanyDocument, error) {
	docs, err := r.listCompanyDocuments(ctx, companyID, cte.DocumentFilter{ChavesAcesso: []string{chave}, Limit: 1}, "")
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, cte.ErrDocumentNotFound
	}
	return &docs[0], nil
}

// ListEventsByChave returns the stored events of a chave, oldest first.
func (r *CTeRepository) ListEventsByChave(ctx context.Context, chave string) ([]cte.Event, error) {
	rows, err := r.queries.ListCTeEventsByChave(ctx, chave)
	if err != nil {
		return nil, fmt.Errorf("list cte events %s: %w", chave, err)
	}
	return cteEventsFromRows(rows)
}

// ListEventsByChaves returns the stored events of every chave in one query,
// ordered by chave and then oldest first. sqlc cannot type the json_each
// parameter, so the query is written here.
func (r *CTeRepository) ListEventsByChaves(ctx context.Context, chaves []string) ([]cte.Event, error) {
	if len(chaves) == 0 {
		return nil, nil
	}
	const query = `
		SELECT id, cte_document_id, chave_acesso, tp_amb, c_orgao, tp_evento, type, n_seq_evento, event_at,
			registered_at, registered, c_stat, x_motivo, protocolo, autor_cnpj, description, justificativa,
			observacao, correcao, condicao_uso, raw_hash, parse_warnings, created_at, updated_at
		FROM cte_events
		WHERE chave_acesso IN (SELECT value FROM json_each(?))
		ORDER BY chave_acesso, COALESCE(registered_at, event_at, created_at), tp_evento, n_seq_evento
	`
	chavesJSON, _ := json.Marshal(chaves) // a []string always marshals
	rows, err := r.db.QueryContext(ctx, query, string(chavesJSON))
	if err != nil {
		return nil, fmt.Errorf("list cte events of %d chaves: %w", len(chaves), err)
	}
	defer func() { _ = rows.Close() }()

	var items []sqlgen.CteEvent
	for rows.Next() {
		var i sqlgen.CteEvent
		err := rows.Scan(
			&i.ID, &i.CteDocumentID, &i.ChaveAcesso, &i.TpAmb, &i.COrgao, &i.TpEvento, &i.Type, &i.NSeqEvento, &i.EventAt,
			&i.RegisteredAt, &i.Registered, &i.CStat, &i.XMotivo, &i.Protocolo, &i.AutorCnpj, &i.Description, &i.Justificativa,
			&i.Observacao, &i.Correcao, &i.CondicaoUso, &i.RawHash, &i.ParseWarnings, &i.CreatedAt, &i.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cte event: %w", err)
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list cte events of %d chaves: %w", len(chaves), err)
	}
	return cteEventsFromRows(items)
}

// CountSummary counts the company's CT-e by primary role. A non-empty tpAmb
// counts only the documents of that environment.
func (r *CTeRepository) CountSummary(ctx context.Context, companyID dfe.CompanyID, tpAmb string) (cte.Counts, error) {
	const query = `
		SELECT cd.company_role, COUNT(*)
		FROM company_cte_documents cd
		INNER JOIN cte_documents d ON d.id = cd.cte_document_id
		WHERE cd.company_id = ? AND (? = '' OR d.tp_amb = ?)
		GROUP BY cd.company_role
	`
	rows, err := r.db.QueryContext(ctx, query, string(companyID), tpAmb, tpAmb)
	if err != nil {
		return cte.Counts{}, fmt.Errorf("count cte documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	counts := cte.Counts{ByRole: make(map[cte.CompanyRole]int)}
	for rows.Next() {
		var role string
		var total int
		if err := rows.Scan(&role, &total); err != nil {
			return cte.Counts{}, fmt.Errorf("scan cte counts: %w", err)
		}
		counts.ByRole[cte.CompanyRole(role)] = total
	}
	if err := rows.Err(); err != nil {
		return cte.Counts{}, fmt.Errorf("iterate cte counts: %w", err)
	}
	return counts, nil
}

// MarkExported records the current raw hash of each document as exported.
func (r *CTeRepository) MarkExported(ctx context.Context, companyID dfe.CompanyID, kind string, docs []cte.CompanyDocument) error {
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
		err := q.MarkCTeExported(ctx, sqlgen.MarkCTeExportedParams{
			CompanyID:     string(companyID),
			CteDocumentID: doc.ID,
			ExportKind:    kind,
			ExportedHash:  doc.RawHash,
			ExportedAt:    now,
		})
		if err != nil {
			return fmt.Errorf("mark cte document %s exported: %w", doc.ChaveAcesso, err)
		}
	}
	return tx.Commit()
}

// ResetCompany removes the company's CT-e view in one transaction: its
// company_cte_documents rows and export marks, the cte_documents no other
// company sees and their events, and the company's own events that have no
// document. Events authored by another registered company are kept, unlinked
// from a removed document. The company's CT-e sync cursor and initial-sync
// flag are reset in the same transaction, so the next pull starts over from
// NSU 0. The XML blobs stay on disk.
func (r *CTeRepository) ResetCompany(ctx context.Context, companyID dfe.CompanyID) (cte.ResetCounts, error) {
	return r.resetCompany(ctx, companyID, true)
}

// PreviewResetCompany returns what ResetCompany would remove, changing
// nothing.
func (r *CTeRepository) PreviewResetCompany(ctx context.Context, companyID dfe.CompanyID) (cte.ResetCounts, error) {
	return r.resetCompany(ctx, companyID, false)
}

// resetCompany runs the reset and commits it only when apply is set, so the
// preview counts come from the same statements.
func (r *CTeRepository) resetCompany(ctx context.Context, companyID dfe.CompanyID, apply bool) (cte.ResetCounts, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return cte.ResetCounts{}, err
	}
	defer func() { _ = tx.Rollback() }()

	id := string(companyID)
	var counts cte.ResetCounts

	// The documents only this company sees go with its rows.
	documentIDs, err := exclusiveCTeDocumentIDs(ctx, tx, id)
	if err != nil {
		return cte.ResetCounts{}, err
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

	if counts.ExportMarks, err = exec("cte export marks", `DELETE FROM company_cte_export_marks WHERE company_id = ?`, id); err != nil {
		return cte.ResetCounts{}, err
	}
	counts.Events, err = exec("cte events", `
		DELETE FROM cte_events
		WHERE autor_cnpj NOT IN (SELECT cnpj FROM companies WHERE id <> ?)
			AND (
				chave_acesso IN (SELECT chave_acesso FROM cte_documents WHERE id IN (SELECT value FROM json_each(?)))
				OR (
					autor_cnpj = (SELECT cnpj FROM companies WHERE id = ?)
					AND NOT EXISTS (SELECT 1 FROM cte_documents d WHERE d.chave_acesso = cte_events.chave_acesso)
				)
			)
	`, id, string(documentIDsJSON), id)
	if err != nil {
		return cte.ResetCounts{}, err
	}
	if _, err := exec("kept cte events", `
		UPDATE cte_events SET cte_document_id = NULL
		WHERE cte_document_id IN (SELECT value FROM json_each(?))
	`, string(documentIDsJSON)); err != nil {
		return cte.ResetCounts{}, err
	}
	if counts.CompanyDocuments, err = exec("company cte documents", `DELETE FROM company_cte_documents WHERE company_id = ?`, id); err != nil {
		return cte.ResetCounts{}, err
	}
	if counts.Documents, err = exec("cte documents", `DELETE FROM cte_documents WHERE id IN (SELECT value FROM json_each(?))`, string(documentIDsJSON)); err != nil {
		return cte.ResetCounts{}, err
	}

	if !apply {
		return counts, nil
	}
	if err := ResetSyncStateTx(ctx, tx, nfse.ResetSyncStateParams{CompanyID: companyID, Source: nfse.SyncSourceCTe}); err != nil {
		return cte.ResetCounts{}, fmt.Errorf("reset cte sync state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return cte.ResetCounts{}, err
	}
	return counts, nil
}

// exclusiveCTeDocumentIDs returns the cte_documents the company sees and no
// other company does.
func exclusiveCTeDocumentIDs(ctx context.Context, tx *sql.Tx, companyID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT cd.cte_document_id FROM company_cte_documents cd
		WHERE cd.company_id = ?
			AND NOT EXISTS (
				SELECT 1 FROM company_cte_documents other
				WHERE other.cte_document_id = cd.cte_document_id AND other.company_id <> cd.company_id
			)
	`, companyID)
	if err != nil {
		return nil, fmt.Errorf("list exclusive cte documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan cte document id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list exclusive cte documents: %w", err)
	}
	return ids, nil
}

// refreshCTeChave links the events of one chave to its document and applies
// the registered ones to the situação. It does nothing while the document
// has not arrived.
func refreshCTeChave(ctx context.Context, q *sqlgen.Queries, chave, now string) error {
	doc, err := q.GetCTeDocumentByChave(ctx, chave)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read cte document %s: %w", chave, err)
	}

	err = q.LinkCTeEventsToDocument(ctx, sqlgen.LinkCTeEventsToDocumentParams{
		CteDocumentID: sql.NullString{String: doc.ID, Valid: true},
		ChaveAcesso:   chave,
	})
	if err != nil {
		return fmt.Errorf("link cte events %s: %w", chave, err)
	}

	rows, err := q.ListCTeEventsByChave(ctx, chave)
	if err != nil {
		return fmt.Errorf("list cte events %s: %w", chave, err)
	}
	var registered []cte.EventType
	for _, row := range rows {
		if row.Registered != 0 {
			registered = append(registered, cte.EventType(row.Type))
		}
	}
	situacao := cte.SituacaoFromEvents(cte.Situacao(doc.Situacao), registered)
	if string(situacao) == doc.Situacao {
		return nil
	}
	err = q.UpdateCTeSituacao(ctx, sqlgen.UpdateCTeSituacaoParams{
		Situacao:    string(situacao),
		UpdatedAt:   now,
		ChaveAcesso: chave,
	})
	if err != nil {
		return fmt.Errorf("update cte situacao %s: %w", chave, err)
	}
	return nil
}

func upsertCTeEvent(ctx context.Context, q *sqlgen.Queries, e cte.Event, now string) error {
	id := e.ID
	if id == "" {
		id = nfse.GenerateID()
	}
	warnings, err := encodeWarnings(e.ParseWarnings)
	if err != nil {
		return err
	}
	err = q.UpsertCTeEvent(ctx, sqlgen.UpsertCTeEventParams{
		ID:            id,
		ChaveAcesso:   string(e.ChaveAcesso),
		TpAmb:         e.TpAmb,
		COrgao:        e.COrgao,
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
		Observacao:    e.Observacao,
		Correcao:      e.Correcao,
		CondicaoUso:   e.CondicaoUso,
		RawHash:       e.RawHash,
		ParseWarnings: warnings,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return fmt.Errorf("upsert cte event %s %s: %w", e.TpEvento, e.ChaveAcesso, err)
	}
	return nil
}

// cteCompanyDocumentColumns is the only column list for company CT-e rows;
// scanCompanyCTeDocument reads it in the same order.
const cteCompanyDocumentColumns = `
	d.id, d.chave_acesso, d.tp_amb, d.modelo, d.tipo_documento, d.serie, d.numero, d.cfop, d.nat_op,
	d.issue_date, d.competence, d.authorized_at, d.protocolo, d.tp_cte, d.tp_serv, d.modal,
	d.mun_ini_codigo, d.mun_ini_nome, d.uf_ini, d.mun_fim_codigo, d.mun_fim_nome, d.uf_fim,
	d.emitente_cnpj, d.emitente_name, d.emitente_ie, d.emitente_uf,
	d.remetente_cnpj, d.remetente_name, d.destinatario_cnpj, d.destinatario_name,
	d.expedidor_cnpj, d.expedidor_name, d.recebedor_cnpj, d.recebedor_name,
	d.tomador_indicador, d.tomador_cnpj, d.tomador_name, d.tomador_ie, d.tomador_uf,
	d.autorizados_cnpj, d.nfe_chaves, d.masked_keys,
	d.total_value, d.receivable_value, d.icms_value, d.tot_trib_value, d.carga_value, d.produto_predominante,
	d.situacao, d.layout_version, d.raw_hash, d.parse_warnings, d.created_at, d.updated_at,
	cd.relation_id, cd.company_id, cd.company_role, cd.papeis, cd.visibility_reason,
	cd.first_seen_nsu, cd.last_seen_nsu, cd.first_synced_at, cd.last_synced_at,
	(SELECT COUNT(*) FROM cte_events e WHERE e.chave_acesso = d.chave_acesso)`

// listCompanyDocuments runs the single company CT-e query. A non-empty
// exportKind keeps only rows pending export for that kind.
func (r *CTeRepository) listCompanyDocuments(ctx context.Context, companyID dfe.CompanyID, f cte.DocumentFilter, exportKind string) ([]cte.CompanyDocument, error) {
	query := `SELECT ` + cteCompanyDocumentColumns + `
		FROM company_cte_documents cd
		INNER JOIN cte_documents d ON d.id = cd.cte_document_id`
	var args []any
	if exportKind != "" {
		query += `
		LEFT JOIN company_cte_export_marks m
			ON m.company_id = cd.company_id AND m.cte_document_id = cd.cte_document_id AND m.export_kind = ?`
		args = append(args, exportKind)
	}

	where, whereArgs := buildCTeFilterSQL(companyID, f)
	query += " WHERE " + where // #nosec G202 -- constant conditions with ? placeholders from buildCTeFilterSQL.
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
		return nil, fmt.Errorf("query cte documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var docs []cte.CompanyDocument
	for rows.Next() {
		doc, err := scanCompanyCTeDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cte documents: %w", err)
	}
	return docs, nil
}

// buildCTeFilterSQL returns the WHERE conditions for f over the aliases cd
// (company_cte_documents) and d (cte_documents), and their arguments. Limit
// is left to the caller. papeis and nfe_chaves are comma-separated lists,
// matched as whole items.
func buildCTeFilterSQL(companyID dfe.CompanyID, f cte.DocumentFilter) (string, []any) {
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
	if f.Role != "" {
		where += " AND (cd.company_role = ? OR instr(',' || cd.papeis || ',', ',' || ? || ',') > 0)"
		args = append(args, string(f.Role), string(f.Role))
	}
	if f.Modelo != "" {
		where += " AND d.modelo = ?"
		args = append(args, f.Modelo)
	}
	if f.EmitenteCNPJ != "" {
		where += " AND d.emitente_cnpj = ?"
		args = append(args, cnpj.Clean(f.EmitenteCNPJ))
	}
	if f.TomadorCNPJ != "" {
		where += " AND d.tomador_cnpj = ?"
		args = append(args, cnpj.Clean(f.TomadorCNPJ))
	}
	if f.NFeChave != "" {
		where += " AND instr(',' || d.nfe_chaves || ',', ',' || ? || ',') > 0"
		args = append(args, strings.ToUpper(strings.TrimSpace(f.NFeChave)))
	}
	if len(f.ChavesAcesso) > 0 {
		where += " AND d.chave_acesso IN (SELECT value FROM json_each(?))"
		chaves, _ := json.Marshal(f.ChavesAcesso) // a []string always marshals
		args = append(args, string(chaves))
	}
	if f.TpAmb != "" {
		where += " AND d.tp_amb = ?"
		args = append(args, f.TpAmb)
	}
	return where, args
}

func scanCompanyCTeDocument(rows *sql.Rows) (cte.CompanyDocument, error) {
	var d sqlgen.CteDocument
	var cd cte.CompanyDocument
	var companyID, role, papeis, visibility string
	var firstSeen, lastSeen sql.NullInt64
	var firstSyncedAt, lastSyncedAt string

	err := rows.Scan(
		&d.ID, &d.ChaveAcesso, &d.TpAmb, &d.Modelo, &d.TipoDocumento, &d.Serie, &d.Numero, &d.Cfop, &d.NatOp,
		&d.IssueDate, &d.Competence, &d.AuthorizedAt, &d.Protocolo, &d.TpCte, &d.TpServ, &d.Modal,
		&d.MunIniCodigo, &d.MunIniNome, &d.UfIni, &d.MunFimCodigo, &d.MunFimNome, &d.UfFim,
		&d.EmitenteCnpj, &d.EmitenteName, &d.EmitenteIe, &d.EmitenteUf,
		&d.RemetenteCnpj, &d.RemetenteName, &d.DestinatarioCnpj, &d.DestinatarioName,
		&d.ExpedidorCnpj, &d.ExpedidorName, &d.RecebedorCnpj, &d.RecebedorName,
		&d.TomadorIndicador, &d.TomadorCnpj, &d.TomadorName, &d.TomadorIe, &d.TomadorUf,
		&d.AutorizadosCnpj, &d.NfeChaves, &d.MaskedKeys,
		&d.TotalValue, &d.ReceivableValue, &d.IcmsValue, &d.TotTribValue, &d.CargaValue, &d.ProdutoPredominante,
		&d.Situacao, &d.LayoutVersion, &d.RawHash, &d.ParseWarnings, &d.CreatedAt, &d.UpdatedAt,
		&cd.RelationID, &companyID, &role, &papeis, &visibility,
		&firstSeen, &lastSeen, &firstSyncedAt, &lastSyncedAt,
		&cd.EventCount,
	)
	if err != nil {
		return cte.CompanyDocument{}, fmt.Errorf("scan cte document: %w", err)
	}

	cd.Document, err = cteDocumentFromRow(d)
	if err != nil {
		return cte.CompanyDocument{}, err
	}
	cd.CompanyID = dfe.CompanyID(companyID)
	cd.CompanyRole = cte.CompanyRole(role)
	cd.Papeis = splitPapeis(papeis)
	cd.VisibilityReason = cte.VisibilityReason(visibility)
	cd.FirstSeenNSU = PtrFromNullInt64(firstSeen)
	cd.LastSeenNSU = PtrFromNullInt64(lastSeen)

	if cd.FirstSyncedAt, err = parseRequiredTime("company cte first_synced_at", firstSyncedAt); err != nil {
		return cte.CompanyDocument{}, err
	}
	if cd.LastSyncedAt, err = parseRequiredTime("company cte last_synced_at", lastSyncedAt); err != nil {
		return cte.CompanyDocument{}, err
	}
	return cd, nil
}

// cteDocumentFromRow rebuilds the document from its row. The IE and UF of
// remetente, destinatário, expedidor and recebedor are not stored.
func cteDocumentFromRow(row sqlgen.CteDocument) (cte.Document, error) {
	doc := cte.Document{
		ID:                  row.ID,
		ChaveAcesso:         dfe.AccessKey(row.ChaveAcesso),
		TpAmb:               row.TpAmb,
		Modelo:              row.Modelo,
		TipoDocumento:       cte.TipoDocumento(row.TipoDocumento),
		Serie:               row.Serie,
		Numero:              row.Numero,
		CFOP:                row.Cfop,
		NatOp:               row.NatOp,
		Competence:          row.Competence,
		Protocolo:           row.Protocolo,
		TpCTe:               row.TpCte,
		TpServ:              row.TpServ,
		Modal:               row.Modal,
		MunIni:              cte.Municipio{Codigo: row.MunIniCodigo, Nome: row.MunIniNome, UF: row.UfIni},
		MunFim:              cte.Municipio{Codigo: row.MunFimCodigo, Nome: row.MunFimNome, UF: row.UfFim},
		Emitente:            cte.Party{CNPJ: row.EmitenteCnpj, Name: row.EmitenteName, IE: row.EmitenteIe, UF: row.EmitenteUf},
		Remetente:           cte.Party{CNPJ: row.RemetenteCnpj, Name: row.RemetenteName},
		Destinatario:        cte.Party{CNPJ: row.DestinatarioCnpj, Name: row.DestinatarioName},
		Expedidor:           cte.Party{CNPJ: row.ExpedidorCnpj, Name: row.ExpedidorName},
		Recebedor:           cte.Party{CNPJ: row.RecebedorCnpj, Name: row.RecebedorName},
		Tomador:             cte.Party{CNPJ: row.TomadorCnpj, Name: row.TomadorName, IE: row.TomadorIe, UF: row.TomadorUf},
		TomadorIndicador:    row.TomadorIndicador,
		AutorizadosCNPJ:     splitCSV(row.AutorizadosCnpj),
		TotalValue:          dfe.Money(row.TotalValue),
		ReceivableValue:     dfe.Money(row.ReceivableValue),
		ICMSValue:           dfe.Money(row.IcmsValue),
		TotTribValue:        dfe.Money(row.TotTribValue),
		CargaValue:          dfe.Money(row.CargaValue),
		ProdutoPredominante: row.ProdutoPredominante,
		NFeChaves:           splitCSV(row.NfeChaves),
		MaskedKeys:          row.MaskedKeys != 0,
		Situacao:            cte.Situacao(row.Situacao),
		LayoutVersion:       row.LayoutVersion,
		RawHash:             row.RawHash,
	}

	var err error
	if row.IssueDate != "" {
		if doc.IssueDate, err = parseRequiredTime("cte issue_date", row.IssueDate); err != nil {
			return cte.Document{}, err
		}
	}
	if doc.AuthorizedAt, err = parseOptionalTime("cte authorized_at", row.AuthorizedAt); err != nil {
		return cte.Document{}, err
	}
	if err := decodeWarnings(row.ParseWarnings, &doc.ParseWarnings); err != nil {
		return cte.Document{}, fmt.Errorf("cte parse_warnings: %w", err)
	}
	return doc, nil
}

func upsertCTeDocumentParams(doc cte.Document, now string) (sqlgen.UpsertCTeDocumentParams, error) {
	warnings, err := encodeWarnings(doc.ParseWarnings)
	if err != nil {
		return sqlgen.UpsertCTeDocumentParams{}, err
	}
	var issueDate string
	if !doc.IssueDate.IsZero() {
		issueDate = doc.IssueDate.Format(time.RFC3339)
	}
	return sqlgen.UpsertCTeDocumentParams{
		ID:                  doc.ID,
		ChaveAcesso:         string(doc.ChaveAcesso),
		TpAmb:               doc.TpAmb,
		Modelo:              doc.Modelo,
		TipoDocumento:       string(doc.TipoDocumento),
		Serie:               doc.Serie,
		Numero:              doc.Numero,
		Cfop:                doc.CFOP,
		NatOp:               doc.NatOp,
		IssueDate:           issueDate,
		Competence:          doc.Competence,
		AuthorizedAt:        nullableTime(doc.AuthorizedAt, time.RFC3339),
		Protocolo:           doc.Protocolo,
		TpCte:               doc.TpCTe,
		TpServ:              doc.TpServ,
		Modal:               doc.Modal,
		MunIniCodigo:        doc.MunIni.Codigo,
		MunIniNome:          doc.MunIni.Nome,
		UfIni:               doc.MunIni.UF,
		MunFimCodigo:        doc.MunFim.Codigo,
		MunFimNome:          doc.MunFim.Nome,
		UfFim:               doc.MunFim.UF,
		EmitenteCnpj:        doc.Emitente.CNPJ,
		EmitenteName:        doc.Emitente.Name,
		EmitenteIe:          doc.Emitente.IE,
		EmitenteUf:          doc.Emitente.UF,
		RemetenteCnpj:       doc.Remetente.CNPJ,
		RemetenteName:       doc.Remetente.Name,
		DestinatarioCnpj:    doc.Destinatario.CNPJ,
		DestinatarioName:    doc.Destinatario.Name,
		ExpedidorCnpj:       doc.Expedidor.CNPJ,
		ExpedidorName:       doc.Expedidor.Name,
		RecebedorCnpj:       doc.Recebedor.CNPJ,
		RecebedorName:       doc.Recebedor.Name,
		TomadorIndicador:    doc.TomadorIndicador,
		TomadorCnpj:         doc.Tomador.CNPJ,
		TomadorName:         doc.Tomador.Name,
		TomadorIe:           doc.Tomador.IE,
		TomadorUf:           doc.Tomador.UF,
		AutorizadosCnpj:     strings.Join(doc.AutorizadosCNPJ, ","),
		NfeChaves:           strings.Join(doc.NFeChaves, ","),
		MaskedKeys:          boolToInt(doc.MaskedKeys),
		TotalValue:          doc.TotalValue.Cents(),
		ReceivableValue:     doc.ReceivableValue.Cents(),
		IcmsValue:           doc.ICMSValue.Cents(),
		TotTribValue:        doc.TotTribValue.Cents(),
		CargaValue:          doc.CargaValue.Cents(),
		ProdutoPredominante: doc.ProdutoPredominante,
		Situacao:            string(doc.Situacao),
		LayoutVersion:       doc.LayoutVersion,
		RawHash:             doc.RawHash,
		ParseWarnings:       warnings,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

func cteEventsFromRows(rows []sqlgen.CteEvent) ([]cte.Event, error) {
	events := make([]cte.Event, 0, len(rows))
	for _, row := range rows {
		e := cte.Event{
			ID:            row.ID,
			ChaveAcesso:   dfe.AccessKey(row.ChaveAcesso),
			TpAmb:         row.TpAmb,
			COrgao:        row.COrgao,
			TpEvento:      row.TpEvento,
			Type:          cte.EventType(row.Type),
			NSeqEvento:    int(row.NSeqEvento),
			Protocolo:     row.Protocolo,
			AutorCNPJ:     row.AutorCnpj,
			Description:   row.Description,
			Justificativa: row.Justificativa,
			Observacao:    row.Observacao,
			Correcao:      row.Correcao,
			CondicaoUso:   row.CondicaoUso,
			Registered:    row.Registered != 0,
			CStat:         row.CStat,
			XMotivo:       row.XMotivo,
			RawHash:       row.RawHash,
		}
		var err error
		if e.EventAt, err = parseOptionalTime("cte event event_at", row.EventAt); err != nil {
			return nil, err
		}
		if e.RegisteredAt, err = parseOptionalTime("cte event registered_at", row.RegisteredAt); err != nil {
			return nil, err
		}
		if err := decodeWarnings(row.ParseWarnings, &e.ParseWarnings); err != nil {
			return nil, fmt.Errorf("cte event parse_warnings: %w", err)
		}
		events = append(events, e)
	}
	return events, nil
}

func joinPapeis(papeis []cte.CompanyRole) string {
	parts := make([]string, len(papeis))
	for i, p := range papeis {
		parts[i] = string(p)
	}
	return strings.Join(parts, ",")
}

func splitPapeis(csv string) []cte.CompanyRole {
	var papeis []cte.CompanyRole
	for _, p := range splitCSV(csv) {
		papeis = append(papeis, cte.CompanyRole(p))
	}
	return papeis
}

// splitCSV splits a comma-separated column, returning nil for an empty one.
func splitCSV(csv string) []string {
	if csv == "" {
		return nil
	}
	return strings.Split(csv, ",")
}
