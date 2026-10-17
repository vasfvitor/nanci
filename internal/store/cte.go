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
	"github.com/vasfvitor/nanci/internal/store/sqlgen"
	"github.com/vasfvitor/nanci/internal/syncstate"
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
	// NSU is the distribution NSU the document came with. nil means it did
	// not come from the distribution (an imported XML, a document fetched
	// by chave): a new row has no first/last seen NSU and an existing row
	// keeps the ones it has.
	NSU *int64
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
			doc.ID = dfe.GenerateID()
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

	var seenNSU sql.NullInt64
	if p.NSU != nil {
		seenNSU = sql.NullInt64{Int64: *p.NSU, Valid: true}
	}
	participation := cte.ClassifyParticipation(&doc, p.CompanyCNPJ)
	err = q.UpsertCompanyCTeDocument(ctx, sqlgen.UpsertCompanyCTeDocumentParams{
		RelationID:       dfe.GenerateID(),
		CompanyID:        string(p.CompanyID),
		CteDocumentID:    documentID,
		CompanyRole:      string(participation.CompanyRole),
		Papeis:           strings.Join(cte.PapeisStrings(participation.Papeis), ","),
		VisibilityReason: string(participation.VisibilityReason),
		FirstSeenNsu:     seenNSU,
		LastSeenNsu:      seenNSU,
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

// ApplyDocument runs ApplyDocumentTx in a transaction of its own, for a
// document stored outside a sync item.
func (r *CTeRepository) ApplyDocument(ctx context.Context, p ApplyCTeDocumentParams) (inserted bool, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	inserted, err = r.ApplyDocumentTx(ctx, tx, p)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return inserted, nil
}

// ApplyEvent runs ApplyEventTx in a transaction of its own, for an event
// stored outside a sync item.
func (r *CTeRepository) ApplyEvent(ctx context.Context, p ApplyCTeEventParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := r.ApplyEventTx(ctx, tx, p); err != nil {
		return err
	}
	return tx.Commit()
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
// were never exported with this kind, whose raw hash changed since, or whose
// event count changed since, such as after a cancelamento.
func (r *CTeRepository) ListPendingExport(ctx context.Context, companyID dfe.CompanyID, f cte.DocumentFilter, kind string) ([]cte.CompanyDocument, error) {
	if kind == "" {
		return nil, errors.New("export kind is required")
	}
	return r.listCompanyDocuments(ctx, companyID, f, kind)
}

// MarkViewed marks the company's CT-e with the given chaves as viewed and
// returns how many were not viewed before.
func (r *CTeRepository) MarkViewed(ctx context.Context, companyID dfe.CompanyID, chaves []string) (int, error) {
	if len(chaves) == 0 {
		return 0, nil
	}
	const query = `
		UPDATE company_cte_documents
		SET viewed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE company_id = ? AND viewed_at IS NULL AND cte_document_id IN (
			SELECT id FROM cte_documents WHERE chave_acesso IN (SELECT value FROM json_each(?))
		)
	`
	return execMarkViewed(ctx, r.db, query, companyID, chaves)
}

// CompanyDocumentByChave returns cte.ErrDocumentNotFound when the company
// does not see the chave in the tpAmb environment. An empty tpAmb matches
// either environment.
func (r *CTeRepository) CompanyDocumentByChave(ctx context.Context, companyID dfe.CompanyID, tpAmb, chave string) (*cte.CompanyDocument, error) {
	docs, err := r.listCompanyDocuments(ctx, companyID, cte.DocumentFilter{ChavesAcesso: []string{chave}, TpAmb: tpAmb, Limit: 1}, "")
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
// ordered by chave and then oldest first.
func (r *CTeRepository) ListEventsByChaves(ctx context.Context, chaves []string) ([]cte.Event, error) {
	if len(chaves) == 0 {
		return nil, nil
	}
	rows, err := listEventsByChaves(ctx, r.db, cteTables, chaves, scanCTeEvent)
	if err != nil {
		return nil, err
	}
	return cteEventsFromRows(rows)
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

// MarkExported records the current raw hash and event count of each
// document as exported. The event count is the one listed with the document,
// so an event stored while the export was written stays pending.
func (r *CTeRepository) MarkExported(ctx context.Context, companyID dfe.CompanyID, kind string, docs []cte.CompanyDocument) error {
	marks := make([]exportMark, 0, len(docs))
	for _, doc := range docs {
		marks = append(marks, exportMark{documentID: doc.ID, chaveAcesso: doc.ChaveAcesso, rawHash: doc.RawHash, events: doc.EventCount})
	}
	return markExported(ctx, r.db, cteTables, companyID, kind, marks)
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

	removed, err := resetCompanyDocuments(ctx, tx, cteTables, companyID)
	if err != nil {
		return cte.ResetCounts{}, err
	}
	counts := cte.ResetCounts{
		CompanyDocuments: removed.CompanyDocuments,
		Documents:        removed.Documents,
		Events:           removed.Events,
		ExportMarks:      removed.ExportMarks,
	}

	if !apply {
		return counts, nil
	}
	if err := ResetSyncStateTx(ctx, tx, syncstate.ResetSyncStateParams{CompanyID: companyID, Source: syncstate.SyncSourceCTe}); err != nil {
		return cte.ResetCounts{}, fmt.Errorf("reset cte sync state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return cte.ResetCounts{}, err
	}
	return counts, nil
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
		id = dfe.GenerateID()
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

// cteTables names the CT-e tables for the queries shared with NF-e.
var cteTables = dfeTables{
	name:                   "cte",
	documents:              "cte_documents",
	companyDocuments:       "company_cte_documents",
	documentID:             "cte_document_id",
	events:                 "cte_events",
	exportMarks:            "company_cte_export_marks",
	companyDocumentColumns: cteCompanyDocumentColumns,
	eventColumns:           cteEventColumns,
}

// cteEventColumns is the column list of ListEventsByChaves; scanCTeEvent
// reads it in the same order.
const cteEventColumns = `
	id, cte_document_id, chave_acesso, tp_amb, c_orgao, tp_evento, type, n_seq_evento, event_at,
	registered_at, registered, c_stat, x_motivo, protocolo, autor_cnpj, description, justificativa,
	observacao, correcao, condicao_uso, raw_hash, parse_warnings, created_at, updated_at`

func scanCTeEvent(rows *sql.Rows) (sqlgen.CteEvent, error) {
	var i sqlgen.CteEvent
	err := rows.Scan(
		&i.ID, &i.CteDocumentID, &i.ChaveAcesso, &i.TpAmb, &i.COrgao, &i.TpEvento, &i.Type, &i.NSeqEvento, &i.EventAt,
		&i.RegisteredAt, &i.Registered, &i.CStat, &i.XMotivo, &i.Protocolo, &i.AutorCnpj, &i.Description, &i.Justificativa,
		&i.Observacao, &i.Correcao, &i.CondicaoUso, &i.RawHash, &i.ParseWarnings, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return sqlgen.CteEvent{}, fmt.Errorf("scan cte event: %w", err)
	}
	return i, nil
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
	cd.first_seen_nsu, cd.last_seen_nsu, cd.first_synced_at, cd.last_synced_at, cd.viewed_at,
	(SELECT COUNT(*) FROM cte_events e WHERE e.chave_acesso = d.chave_acesso)`

// listCompanyDocuments runs the single company CT-e query. A non-empty
// exportKind keeps only rows pending export for that kind.
func (r *CTeRepository) listCompanyDocuments(ctx context.Context, companyID dfe.CompanyID, f cte.DocumentFilter, exportKind string) ([]cte.CompanyDocument, error) {
	where, args := buildCTeFilterSQL(companyID, f)
	return queryCompanyDocuments(ctx, r.db, cteTables, where, args, f.Limit, exportKind, scanCompanyCTeDocument)
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
	if f.OnlyUnread {
		where += " AND cd.viewed_at IS NULL"
	}
	return where, args
}

func scanCompanyCTeDocument(rows *sql.Rows) (cte.CompanyDocument, error) {
	var d sqlgen.CteDocument
	var cd cte.CompanyDocument
	var companyID, role, papeis, visibility string
	var firstSeen, lastSeen sql.NullInt64
	var firstSyncedAt, lastSyncedAt string
	var viewedAt sql.NullString

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
		&firstSeen, &lastSeen, &firstSyncedAt, &lastSyncedAt, &viewedAt,
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
	if cd.ViewedAt, err = parseOptionalTime("company cte viewed_at", viewedAt); err != nil {
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
