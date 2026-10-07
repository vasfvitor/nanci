package store_test

import (
	"context"
	"database/sql"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/store/storetest"
)

// TestSchemaSQLMatchesMigrations guards schema.sql, the sqlc input that no
// code loads, against drifting from what the migrations build. The migrated
// database is the truth: when this fails, fix schema.sql.
func TestSchemaSQLMatchesMigrations(t *testing.T) {
	migrated := schemaObjects(t, storetest.OpenTestDB(t))

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fresh.Close() })
	fresh.SetMaxOpenConns(1)
	if _, err := fresh.ExecContext(context.Background(), string(schema)); err != nil {
		t.Fatalf("execute schema.sql: %v", err)
	}
	declared := schemaObjects(t, fresh)

	for _, name := range sortedUnion(migrated, declared) {
		got, inSchema := declared[name]
		want, inMigrations := migrated[name]
		switch {
		case !inSchema:
			t.Errorf("%s exists after the migrations but not in schema.sql:\n  migrations: %s", name, want)
		case !inMigrations:
			t.Errorf("%s exists in schema.sql but not after the migrations:\n  schema.sql: %s", name, got)
		case got != want:
			t.Errorf("%s differs:\n  migrations: %s\n  schema.sql: %s", name, want, got)
		}
	}
}

// schemaObjects returns the normalized CREATE statement of every table, index,
// trigger and view, keyed by "type name". Automatic indexes have no SQL and
// follow from their table's definition, so they are compared through it.
func schemaObjects(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
		SELECT type, name, sql FROM sqlite_master
		WHERE type IN ('table', 'index', 'trigger', 'view')
		  AND sql IS NOT NULL
		  AND name NOT LIKE 'sqlite_%'
		  AND tbl_name <> 'goose_db_version'
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	objects := map[string]string{}
	for rows.Next() {
		var kind, name, stmt string
		if err := rows.Scan(&kind, &name, &stmt); err != nil {
			t.Fatal(err)
		}
		objects[kind+" "+name] = normalizeSQL(stmt)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return objects
}

// normalizeSQL drops what SQLite stores verbatim from the source text but
// that does not change the schema: comments, whitespace (including around
// parentheses and commas), keyword and identifier case, and identifier quotes
// (RENAME TO and ADD COLUMN rewrite the stored text with quotes and their own
// spacing). String literals are kept as written. Column order is left alone,
// so a column declared in a different position than ADD COLUMN or a rebuild
// put it still fails the comparison.
func normalizeSQL(stmt string) string {
	var b strings.Builder
	pendingSpace := false
	writeSpace := func() {
		if pendingSpace && b.Len() > 0 {
			if s := b.String(); !strings.HasSuffix(s, "(") && !strings.HasSuffix(s, ",") {
				b.WriteByte(' ')
			}
		}
		pendingSpace = false
	}
	for i := 0; i < len(stmt); i++ {
		c := stmt[i]
		switch {
		case c == '\'':
			end := i + 1
			for end < len(stmt) {
				if stmt[end] == '\'' {
					if end+1 < len(stmt) && stmt[end+1] == '\'' {
						end += 2
						continue
					}
					break
				}
				end++
			}
			end = min(end, len(stmt)-1)
			writeSpace()
			b.WriteString(stmt[i : end+1])
			i = end
		case c == '-' && i+1 < len(stmt) && stmt[i+1] == '-':
			for i+1 < len(stmt) && stmt[i+1] != '\n' {
				i++
			}
			pendingSpace = true
		case c == '"' || c == '`' || c == '[' || c == ']':
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			pendingSpace = true
		case c == '(' || c == ')' || c == ',' || c == ';':
			pendingSpace = false
			b.WriteByte(c)
		default:
			writeSpace()
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			b.WriteByte(c)
		}
	}
	return strings.TrimSuffix(b.String(), ";")
}

func sortedUnion(a, b map[string]string) []string {
	seen := map[string]bool{}
	var names []string
	for _, m := range []map[string]string{a, b} {
		for name := range m {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}
