package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestSQLiteStore_EventSchemaMatchesLedgerContract(t *testing.T) {
	store := setupTestStore(t)

	rows, err := store.db.QueryContext(context.Background(), `PRAGMA table_info(events)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(events): %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, pk int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan pragma row: %v", err)
		}
		columns = append(columns, name)
	}

	want := []string{
		"id",
		"document_id",
		"chave_acesso",
		"event_type",
		"event_at",
		"replacement_chave_acesso",
		"description",
		"raw_xml_path",
		"raw_hash",
		"created_at",
	}
	if len(columns) != len(want) {
		t.Fatalf("columns len = %d, want %d (%v)", len(columns), len(want), columns)
	}
	for i, name := range want {
		if columns[i] != name {
			t.Fatalf("column[%d] = %q, want %q", i, columns[i], name)
		}
	}
}

func TestSQLiteStore_SaveCancellationEventUpdatesStatusAndIsIdempotent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	doc := createCanonicalDocumentForEventTest(t, store, ctx, "NFS-CANC")

	event := &nfse.Event{
		ID:          uuid.NewString(),
		ChaveAcesso: doc.ChaveAcesso,
		Type:        "cancelamento",
		EventAt:     time.Date(2026, 6, 4, 13, 0, 0, 0, time.UTC),
		EventAtValid: true,
		Description: "Cancelada pelo emitente",
		RawXMLPath:  "xml/events/NFS-CANC/hash-canc.xml",
		RawHash:     "hash-canc",
	}
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatalf("SaveEvent first: %v", err)
	}

	event.ID = uuid.NewString()
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatalf("SaveEvent second: %v", err)
	}

	savedDoc, err := store.GetDocumentByChave(ctx, doc.ChaveAcesso)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if savedDoc.Status != "cancelada" {
		t.Fatalf("Status = %q, want cancelada", savedDoc.Status)
	}

	events, err := store.ListEventsByDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ListEventsByDocument: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].Type != "cancelamento" || events[0].RawXMLPath != "xml/events/NFS-CANC/hash-canc.xml" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestSQLiteStore_SaveSubstitutionEventStoresReplacementAndWinsStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	doc := createCanonicalDocumentForEventTest(t, store, ctx, "NFS-SUB")

	cancelEvent := &nfse.Event{
		ID:           uuid.NewString(),
		ChaveAcesso:  doc.ChaveAcesso,
		Type:         "cancelamento",
		EventAt:      time.Date(2026, 6, 4, 11, 0, 0, 0, time.UTC),
		EventAtValid: true,
		RawXMLPath:   "xml/events/NFS-SUB/hash-cancel.xml",
		RawHash:      "hash-cancel",
	}
	if err := store.SaveEvent(ctx, cancelEvent); err != nil {
		t.Fatalf("SaveEvent cancel: %v", err)
	}

	subEvent := &nfse.Event{
		ID:                     uuid.NewString(),
		ChaveAcesso:            doc.ChaveAcesso,
		Type:                   "substituicao",
		EventAt:                time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
		EventAtValid:           true,
		ReplacementChaveAcesso: "NFS-SUB-NOVA",
		Description:            "Nota substituida",
		RawXMLPath:             "xml/events/NFS-SUB/hash-sub.xml",
		RawHash:                "hash-sub",
	}
	if err := store.SaveEvent(ctx, subEvent); err != nil {
		t.Fatalf("SaveEvent substitution: %v", err)
	}

	savedDoc, err := store.GetDocumentByChave(ctx, doc.ChaveAcesso)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if savedDoc.Status != "substituida" {
		t.Fatalf("Status = %q, want substituida", savedDoc.Status)
	}

	events, err := store.ListEventsByDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ListEventsByDocument: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2", len(events))
	}
	if events[1].Type != "substituicao" || events[1].ReplacementChaveAcesso != "NFS-SUB-NOVA" {
		t.Fatalf("unexpected substitution event: %+v", events[1])
	}
}

func TestSQLiteStore_UnknownEventDoesNotChangeStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	doc := createCanonicalDocumentForEventTest(t, store, ctx, "NFS-UNK")

	event := &nfse.Event{
		ID:          uuid.NewString(),
		ChaveAcesso: doc.ChaveAcesso,
		Type:        "unknown",
		Description: "Payload nao classificado",
		RawXMLPath:  "xml/events/NFS-UNK/hash-unknown.xml",
		RawHash:     "hash-unknown",
	}
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatalf("SaveEvent: %v", err)
	}

	savedDoc, err := store.GetDocumentByChave(ctx, doc.ChaveAcesso)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if savedDoc.Status != "normal" {
		t.Fatalf("Status = %q, want normal", savedDoc.Status)
	}
}

func TestSQLiteStore_DocumentUpsertDoesNotRevertEventDrivenStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	doc := createCanonicalDocumentForEventTest(t, store, ctx, "NFS-RERUN")

	event := &nfse.Event{
		ID:          uuid.NewString(),
		ChaveAcesso: doc.ChaveAcesso,
		Type:        "cancelamento",
		RawXMLPath:  "xml/events/NFS-RERUN/hash-rerun.xml",
		RawHash:     "hash-rerun",
	}
	if err := store.SaveEvent(ctx, event); err != nil {
		t.Fatalf("SaveEvent: %v", err)
	}

	doc.Status = "normal"
	doc.RawHash = "hash-rerun-doc"
	if err := store.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("UpsertDocument rerun: %v", err)
	}

	savedDoc, err := store.GetDocumentByChave(ctx, doc.ChaveAcesso)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if savedDoc.Status != "cancelada" {
		t.Fatalf("Status after rerun = %q, want cancelada", savedDoc.Status)
	}
}

func createCanonicalDocumentForEventTest(t *testing.T, store *SQLiteStore, ctx context.Context, chave string) *nfse.Document {
	t.Helper()

	doc := &nfse.Document{
		ID:            uuid.NewString(),
		ChaveAcesso:   chave,
		IssueDate:     time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
		Competence:    "2026-06",
		PrestadorCNPJ: "12345678000199",
		TomadorCNPJ:   "99887766000155",
		ServiceValue:  100,
		Status:        "normal",
		XMLPath:       "xml/2026-06/" + chave + ".xml",
		RawHash:       "doc-" + chave,
	}
	if err := store.UpsertDocument(ctx, doc); err != nil {
		t.Fatalf("UpsertDocument: %v", err)
	}

	savedDoc, err := store.GetDocumentByChave(ctx, chave)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if savedDoc == nil {
		t.Fatalf("document %s not found", chave)
	}

	return savedDoc
}
