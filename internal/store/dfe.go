package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// dfeTables names the tables and column lists of one DF-e source, NF-e or
// CT-e. Both sources keep the same shape (documents, the company view, events
// and export marks), so the queries below are written once with {name}
// placeholders that sql fills in. Every value is a constant of this package.
type dfeTables struct {
	name             string // nfe or cte, for error messages
	documents        string
	companyDocuments string
	documentID       string // the column of the company view and export marks that holds documents.id
	events           string
	exportMarks      string
	// companyDocumentColumns is the select list over the aliases cd (company
	// view) and d (documents) that the source's scan function reads.
	companyDocumentColumns string
	// eventColumns is the select list over events that the source's event
	// scan function reads.
	eventColumns string
}

func (t dfeTables) sql(query string) string {
	return strings.NewReplacer(
		"{documents}", t.documents,
		"{company_documents}", t.companyDocuments,
		"{document_id}", t.documentID,
		"{events}", t.events,
		"{export_marks}", t.exportMarks,
		"{company_document_columns}", t.companyDocumentColumns,
		"{event_columns}", t.eventColumns,
	).Replace(query)
}

// rowQuerier is a *sql.DB or a *sql.Tx.
type rowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// queryRows runs query and reads every row with scan.
func queryRows[T any](ctx context.Context, db rowQuerier, scan func(*sql.Rows) (T, error), query string, args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const (
	dfeCompanyDocumentsQuery = `SELECT {company_document_columns}
		FROM {company_documents} cd
		INNER JOIN {documents} d ON d.id = cd.{document_id}`

	dfePendingExportJoin = `
		LEFT JOIN {export_marks} m
			ON m.company_id = cd.company_id AND m.{document_id} = cd.{document_id} AND m.export_kind = ?`

	// A change in the event count since the mark, such as a cancelamento
	// that arrived after the export, makes the document pending again.
	// Counting instead of comparing created_at with exported_at also
	// catches an event stored in the same second.
	dfePendingExportCondition = ` AND (m.exported_at IS NULL OR m.exported_hash != d.raw_hash
			OR m.exported_events != (SELECT COUNT(*) FROM {events} e WHERE e.chave_acesso = d.chave_acesso))`
)

// queryCompanyDocuments runs the single company document query of a source.
// where and whereArgs come from the source's filter builder, over the
// aliases cd and d. A non-empty exportKind keeps only rows pending export for
// that kind. A positive limit caps the rows.
func queryCompanyDocuments[T any](ctx context.Context, db *sql.DB, t dfeTables, where string, whereArgs []any, limit int, exportKind string, scan func(*sql.Rows) (T, error)) ([]T, error) {
	query := t.sql(dfeCompanyDocumentsQuery)
	var args []any
	if exportKind != "" {
		query += t.sql(dfePendingExportJoin)
		args = append(args, exportKind)
	}

	query += " WHERE " + where // #nosec G202 -- constant conditions with ? placeholders from the source's filter builder.
	args = append(args, whereArgs...)
	if exportKind != "" {
		query += t.sql(dfePendingExportCondition)
	}

	query += " ORDER BY d.issue_date DESC, d.chave_acesso DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	docs, err := queryRows(ctx, db, scan, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query %s documents: %w", t.name, err)
	}
	return docs, nil
}

const eventsByChavesQuery = `
		SELECT {event_columns}
		FROM {events}
		WHERE chave_acesso IN (SELECT value FROM json_each(?))
		ORDER BY chave_acesso, COALESCE(registered_at, event_at, created_at), tp_evento, n_seq_evento
	`

// listEventsByChaves returns the event rows of every chave in one query,
// ordered by chave and then oldest first. sqlc cannot type the json_each
// parameter, so the query is written here.
func listEventsByChaves[T any](ctx context.Context, db *sql.DB, t dfeTables, chaves []string, scan func(*sql.Rows) (T, error)) ([]T, error) {
	chavesJSON, _ := json.Marshal(chaves) // a []string always marshals
	rows, err := queryRows(ctx, db, scan, t.sql(eventsByChavesQuery), string(chavesJSON))
	if err != nil {
		return nil, fmt.Errorf("list %s events of %d chaves: %w", t.name, len(chaves), err)
	}
	return rows, nil
}

// exportMark is the state of one document when it was exported.
type exportMark struct {
	documentID  string
	chaveAcesso dfe.AccessKey
	rawHash     string
	events      int
}

const markExportedQuery = `
		INSERT INTO {export_marks} (company_id, {document_id}, export_kind, exported_hash, exported_at, exported_events)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(company_id, {document_id}, export_kind) DO UPDATE SET
			exported_hash = excluded.exported_hash,
			exported_at = excluded.exported_at,
			exported_events = excluded.exported_events
	`

// markExported records the marks in one transaction.
func markExported(ctx context.Context, db *sql.DB, t dfeTables, companyID dfe.CompanyID, kind string, marks []exportMark) error {
	if len(marks) == 0 {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	query := t.sql(markExportedQuery)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, m := range marks {
		_, err := tx.ExecContext(ctx, query, string(companyID), m.documentID, kind, m.rawHash, now, int64(m.events))
		if err != nil {
			return fmt.Errorf("mark %s document %s exported: %w", t.name, m.chaveAcesso, err)
		}
	}
	return tx.Commit()
}

// dfeResetCounts is what resetCompanyDocuments removed.
type dfeResetCounts struct {
	CompanyDocuments int
	Documents        int
	Events           int
	ExportMarks      int
}

const (
	exclusiveDocumentIDsQuery = `
		SELECT cd.{document_id} FROM {company_documents} cd
		WHERE cd.company_id = ?
			AND NOT EXISTS (
				SELECT 1 FROM {company_documents} other
				WHERE other.{document_id} = cd.{document_id} AND other.company_id <> cd.company_id
			)
	`
	resetExportMarksQuery = `DELETE FROM {export_marks} WHERE company_id = ?`
	resetEventsQuery      = `
		DELETE FROM {events}
		WHERE autor_cnpj NOT IN (SELECT cnpj FROM companies WHERE id <> ?)
			AND (
				chave_acesso IN (SELECT chave_acesso FROM {documents} WHERE id IN (SELECT value FROM json_each(?)))
				OR (
					autor_cnpj = (SELECT cnpj FROM companies WHERE id = ?)
					AND NOT EXISTS (SELECT 1 FROM {documents} d WHERE d.chave_acesso = {events}.chave_acesso)
				)
			)
	`
	unlinkKeptEventsQuery = `
		UPDATE {events} SET {document_id} = NULL
		WHERE {document_id} IN (SELECT value FROM json_each(?))
	`
	resetCompanyDocumentsQuery = `DELETE FROM {company_documents} WHERE company_id = ?`
	resetDocumentsQuery        = `DELETE FROM {documents} WHERE id IN (SELECT value FROM json_each(?))`
)

// resetCompanyDocuments removes, inside tx, the company's view of one
// source: its company rows and export marks, the documents no other company
// sees and their events, and the company's own events that have no
// document. Events authored by another registered company are kept,
// unlinked from a removed document.
func resetCompanyDocuments(ctx context.Context, tx *sql.Tx, t dfeTables, companyID dfe.CompanyID) (dfeResetCounts, error) {
	id := string(companyID)

	// The documents only this company sees go with its rows.
	documentIDs, err := queryRows(ctx, tx, scanString, t.sql(exclusiveDocumentIDsQuery), id)
	if err != nil {
		return dfeResetCounts{}, fmt.Errorf("list exclusive %s documents: %w", t.name, err)
	}
	if documentIDs == nil {
		documentIDs = []string{} // json_each needs an array, not null
	}
	documentIDsJSON, _ := json.Marshal(documentIDs) // a []string always marshals

	exec := func(what, query string, args ...any) (int, error) {
		res, err := tx.ExecContext(ctx, t.sql(query), args...)
		if err != nil {
			return 0, fmt.Errorf("reset %s %s: %w", t.name, what, err)
		}
		n, err := res.RowsAffected()
		return int(n), err
	}

	var counts dfeResetCounts
	if counts.ExportMarks, err = exec("export marks", resetExportMarksQuery, id); err != nil {
		return dfeResetCounts{}, err
	}
	if counts.Events, err = exec("events", resetEventsQuery, id, string(documentIDsJSON), id); err != nil {
		return dfeResetCounts{}, err
	}
	if _, err := exec("kept events", unlinkKeptEventsQuery, string(documentIDsJSON)); err != nil {
		return dfeResetCounts{}, err
	}
	if counts.CompanyDocuments, err = exec("company documents", resetCompanyDocumentsQuery, id); err != nil {
		return dfeResetCounts{}, err
	}
	if counts.Documents, err = exec("documents", resetDocumentsQuery, string(documentIDsJSON)); err != nil {
		return dfeResetCounts{}, err
	}
	return counts, nil
}

func scanString(rows *sql.Rows) (string, error) {
	var s string
	err := rows.Scan(&s)
	return s, err
}
