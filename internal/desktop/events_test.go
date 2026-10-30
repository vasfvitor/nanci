package main

import (
	"context"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/vasfvitor/nanci/internal/app"
)

// checkRegistered runs an emitted event through the check Wails applies on
// every emit. The payload must match the type registered in events.go, and a
// payload of another type must be refused, which only happens when the name
// is registered.
func checkRegistered(t *testing.T, got call) {
	t.Helper()
	if len(got.args) != 1 {
		t.Fatalf("%s emitted %d values, want 1", got.method, len(got.args))
	}
	processor := application.NewWailsEventProcessor(func(*application.CustomEvent) {})

	if err := processor.Emit(&application.CustomEvent{Name: got.method, Data: got.args[0]}); err != nil {
		t.Errorf("emit %s: %v", got.method, err)
	}
	if err := processor.Emit(&application.CustomEvent{Name: got.method, Data: struct{}{}}); err == nil {
		t.Errorf("emit %s with struct{}{} succeeded; the event is not registered", got.method)
	}
}

func receive(t *testing.T, events *fakeEvents) call {
	t.Helper()
	select {
	case got := <-events.emitted:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was emitted")
		return call{}
	}
}

func TestEmittedPayloadsMatchRegisteredTypes(t *testing.T) {
	a := NewApp()
	events := &fakeEvents{emitted: make(chan call, 1)}
	a.events = events
	a.cred.events = events

	a.emitLog(desktopLogEvent{Level: "INFO", Msg: "pronto"})
	checkRegistered(t, receive(t, events))

	a.guard.pulls.Add(1)
	a.beforeClose()
	checkRegistered(t, receive(t, events))

	// A canceled context makes GetCertPassword return right after it emits.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = a.cred.GetCertPassword(ctx, app.CertPasswordRequest{RequestID: "req-1"})
	checkRegistered(t, receive(t, events))

	// The Exportar Logs menu item in main.go emits these with a message.
	checkRegistered(t, call{eventNotifySuccess, []any{"Logs exportados"}})
	checkRegistered(t, call{eventNotifyError, []any{"Erro ao exportar logs"}})
}
