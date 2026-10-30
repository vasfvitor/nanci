package main

import (
	"sync/atomic"
)

// closeGuard keeps the window open while a sync runs, until the user
// confirms. Pull, PullNFe and PullCTe block until their sync finishes, so
// pulls counts the syncs in flight.
type closeGuard struct {
	pulls atomic.Int32
	allow atomic.Bool
}

// trackPull counts a sync until the returned func runs.
func (a *App) trackPull() (done func()) {
	a.guard.pulls.Add(1)
	return func() { a.guard.pulls.Add(-1) }
}

// beforeClose runs for every close: the title-bar X, Ctrl+Q and the menu
// "Sair" reach it through Options.ShouldQuit, the OS close through the
// window's WindowClosing hook (main.go). Returning true keeps the window
// open. During a sync it asks the frontend to confirm through the
// app:before-close event, which carries the number of syncs in flight.
// ShouldQuit runs on the UI thread, so the event is sent from a goroutine
// and nothing here waits.
func (a *App) beforeClose() bool {
	if a.guard.allow.Load() {
		return false
	}
	pulls := a.guard.pulls.Load()
	if pulls == 0 {
		return false
	}
	go a.events.Emit(eventBeforeClose, int(pulls))
	return true
}

// ConfirmQuit closes the app after the user chose to leave during a sync.
func (a *App) ConfirmQuit() {
	a.guard.allow.Store(true)
	a.quit()
}
