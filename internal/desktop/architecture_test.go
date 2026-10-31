package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

// wailsLifecycleMethods are the App methods Wails calls itself and never
// binds (internalServiceMethods in the Wails bindings.go).
var wailsLifecycleMethods = map[string]bool{
	"ServiceName":     true,
	"ServiceStartup":  true,
	"ServiceShutdown": true,
	"ServeHTTP":       true,
}

// TestBoundMethodsCrossOnlyDTOs keeps the desktop boundary of AGENTS.md:
// every method the frontend can call takes and returns desktopapi DTOs, basic
// values, slices and maps of them, an optional leading context.Context and a
// trailing error. DTO fields follow the same rule, plus time.Time for dates
// and time.Duration (a number of nanoseconds in JSON). A domain struct from
// internal/nfe, internal/app or elsewhere would ship its Go field names and
// types to the frontend and tie the screens to the core; add a DTO and a
// mapper in desktopapi instead (see the desktop-method skill).
func TestBoundMethodsCrossOnlyDTOs(t *testing.T) {
	dtoPkg := reflect.TypeFor[desktopapi.ErrorPayload]().PkgPath()
	ctxType := reflect.TypeFor[context.Context]()
	errType := reflect.TypeFor[error]()
	timeTypes := map[reflect.Type]bool{reflect.TypeFor[time.Time](): true, reflect.TypeFor[time.Duration](): true}

	// checked holds DTO structs already walked (true) or being walked, so
	// a DTO that refers to itself ends.
	checked := map[reflect.Type]bool{}
	var allowed func(reflect.Type) bool
	allowed = func(typ reflect.Type) bool {
		switch typ.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			return allowed(typ.Elem())
		case reflect.Map:
			return allowed(typ.Key()) && allowed(typ.Elem())
		}
		if timeTypes[typ] {
			return true
		}
		if typ.PkgPath() == "" {
			// Predeclared: string, bool, numbers, and unnamed composites.
			return typ.Kind() != reflect.Interface && typ.Kind() != reflect.Func && typ.Kind() != reflect.Chan
		}
		if typ.PkgPath() != dtoPkg {
			return false
		}
		if typ.Kind() != reflect.Struct {
			return true
		}
		if ok, seen := checked[typ]; seen {
			return ok
		}
		checked[typ] = true
		for i := range typ.NumField() {
			field := typ.Field(i)
			if !allowed(field.Type) {
				t.Errorf("desktopapi.%s.%s is %s; DTO fields hold basic values, time.Time, time.Duration or other DTOs", typ.Name(), field.Name, field.Type)
				checked[typ] = false
			}
		}
		return checked[typ]
	}

	app := reflect.TypeFor[*App]()
	bound := 0
	for i := range app.NumMethod() {
		method := app.Method(i)
		if wailsLifecycleMethods[method.Name] {
			continue
		}
		bound++
		sig := method.Type
		for j := 1; j < sig.NumIn(); j++ { // In(0) is the receiver.
			in := sig.In(j)
			if j == 1 && in == ctxType {
				continue
			}
			if !allowed(in) {
				t.Errorf("%s: parameter %d is %s; bound methods take desktopapi DTOs or basic values", method.Name, j, in)
			}
		}
		for j := range sig.NumOut() {
			out := sig.Out(j)
			if j == sig.NumOut()-1 && out == errType {
				continue
			}
			if !allowed(out) {
				t.Errorf("%s: result %d is %s; bound methods return desktopapi DTOs or basic values", method.Name, j, out)
			}
		}
	}
	if bound == 0 {
		t.Fatal("found no bound methods on *App")
	}
}
