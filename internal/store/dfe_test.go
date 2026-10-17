package store

import (
	"strings"
	"testing"
)

// TestDFeTablesFillEveryPlaceholder guards the shared NF-e and CT-e queries
// against a placeholder that dfeTables.sql does not know, and against one
// source's table leaking into the other's query.
func TestDFeTablesFillEveryPlaceholder(t *testing.T) {
	queries := map[string]string{
		"dfeCompanyDocumentsQuery":   dfeCompanyDocumentsQuery,
		"dfePendingExportJoin":       dfePendingExportJoin,
		"dfePendingExportCondition":  dfePendingExportCondition,
		"eventsByChavesQuery":        eventsByChavesQuery,
		"markExportedQuery":          markExportedQuery,
		"exclusiveDocumentIDsQuery":  exclusiveDocumentIDsQuery,
		"resetExportMarksQuery":      resetExportMarksQuery,
		"resetEventsQuery":           resetEventsQuery,
		"unlinkKeptEventsQuery":      unlinkKeptEventsQuery,
		"resetCompanyDocumentsQuery": resetCompanyDocumentsQuery,
		"resetDocumentsQuery":        resetDocumentsQuery,
	}
	sources := []struct {
		tables dfeTables
		other  string
	}{
		{nfeTables, "cte"},
		{cteTables, "nfe"},
	}
	for _, src := range sources {
		for name, query := range queries {
			got := src.tables.sql(query)
			if strings.ContainsAny(got, "{}") {
				t.Errorf("%s for %s keeps a placeholder:\n%s", name, src.tables.name, got)
			}
			// nfe_chaves is a CT-e column, so look for table names only.
			for _, table := range []string{"_document", "_events", "_export"} {
				if strings.Contains(got, src.other+table) {
					t.Errorf("%s for %s names a %s table:\n%s", name, src.tables.name, src.other, got)
				}
			}
		}
	}
}
