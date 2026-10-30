package main

import "sync"

// windowTheme remembers the theme picked in Configurações ("dark", "light" or
// "system"). SetWindowTheme writes it from a binding goroutine and the system
// theme hook reads it from the Wails event goroutine.
type windowTheme struct {
	mu   sync.Mutex
	mode string
}

func (t *windowTheme) set(mode string) {
	t.mu.Lock()
	t.mode = mode
	t.mu.Unlock()
}

// fixed reports whether the remembered mode pins the window theme, and to
// which side.
func (t *windowTheme) fixed() (dark, fixed bool) {
	t.mu.Lock()
	mode := t.mode
	t.mu.Unlock()
	return fixedWindowTheme(mode)
}

// fixedWindowTheme reports whether mode pins the window to dark or light.
// "system", and anything else, follows the system theme.
func fixedWindowTheme(mode string) (dark, fixed bool) {
	switch mode {
	case "dark":
		return true, true
	case "light":
		return false, true
	default:
		return false, false
	}
}
