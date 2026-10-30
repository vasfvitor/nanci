package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestFixedWindowTheme(t *testing.T) {
	tests := []struct {
		mode      string
		wantDark  bool
		wantFixed bool
	}{
		{mode: "dark", wantDark: true, wantFixed: true},
		{mode: "light", wantDark: false, wantFixed: true},
		{mode: "system", wantFixed: false},
		{mode: "", wantFixed: false},
		{mode: "auto", wantFixed: false},
	}
	for _, tt := range tests {
		dark, fixed := fixedWindowTheme(tt.mode)
		if dark != tt.wantDark || fixed != tt.wantFixed {
			t.Errorf("fixedWindowTheme(%q) = (%v, %v), want (%v, %v)", tt.mode, dark, fixed, tt.wantDark, tt.wantFixed)
		}
	}
}

func TestKeepFixedThemeCancelsOnlyAFixedTheme(t *testing.T) {
	app := NewApp() // no window: SetWindowTheme only records the mode

	systemThemeChanged := func() bool {
		event := &application.ApplicationEvent{}
		app.keepFixedTheme(event)
		return event.IsCancelled()
	}

	if systemThemeChanged() {
		t.Fatal("cancelled before any theme was picked; Wails must follow the system")
	}
	for _, mode := range []string{"dark", "light"} {
		app.SetWindowTheme(mode)
		if !systemThemeChanged() {
			t.Errorf("SetWindowTheme(%q): system theme change not cancelled", mode)
		}
	}
	app.SetWindowTheme("system")
	if systemThemeChanged() {
		t.Error(`SetWindowTheme("system"): system theme change cancelled`)
	}
}
